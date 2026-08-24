package api

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/google/uuid"

	"t20engine/db/sqlcgen"
)

// newUUID generates a random v4 UUID string for initiative entry ids (randomUUID() in the
// service). Injected into the store so tests can swap a deterministic generator.
func newUUID() string { return uuid.NewString() }

// sessionStore holds each session's in-memory runtime state, guarding the pure mutations
// (session_state.go) with a mutex and persisting fire-and-forget to Session.runtimeState.
// Mirrors SessionStateService — a server restart wipes trackers until the first load()
// re-hydrates from the DB. Mutation methods return a snapshot (deep-enough copy) so the
// gateway can serialize/broadcast it without racing a concurrent message on the session.
type sessionStore struct {
	mu     sync.Mutex
	states map[int64]*SessionRuntimeState
	dirty  map[int64]bool
	// seqs numera as mutações de cada sessão, para o hub reconhecer quadro
	// atrasado (ALE-238). Mora aqui e não no estado: hidratar do banco troca o
	// estado, e um contador que vivesse nele voltaria a zero.
	seqs  map[int64]uint64
	newID func() string
	q     *sqlcgen.Queries
	// persistMus holds a per-session mutex (sessionID → *sync.Mutex) serializing that
	// session's runtime-state writes so concurrent mutations can't land out of order —
	// WITHOUT coupling latency across sessions.
	persistMus sync.Map
	// ouvintes são os streams SSE acordados a cada mutação (piloto ALE-219, em
	// mesa_watch.go). Guardados sob o MESMO `mu` das mutações, que é o que faz o
	// aviso sair junto com a mudança e não depois dela.
	ouvintes map[int64][]chan struct{}
}

// persistLock returns the per-session DB-write mutex, creating it on first use.
func (st *sessionStore) persistLock(sessionID int64) *sync.Mutex {
	m, _ := st.persistMus.LoadOrStore(sessionID, &sync.Mutex{})
	return m.(*sync.Mutex)
}

func newSessionStore(q *sqlcgen.Queries, newID func() string) *sessionStore {
	return &sessionStore{
		states: map[int64]*SessionRuntimeState{},
		dirty:  map[int64]bool{},
		seqs:   map[int64]uint64{},
		newID:  newID,
		q:      q,
	}
}

// cloneState copies the state for broadcast. The entry structs are copied by value; their
// *int64 vitals are shared but never mutated in place (patch/delta always assign a fresh
// pointer), so the snapshot is safe to serialize outside the lock.
// cloneState copia por VALOR e só depois recria a fatia. A versão anterior
// listava os campos um a um, e listar campos é uma lista que envelhece: ao
// entrar o `TurnsTaken` (ALE-142) a cópia continuou compilando e passou a zerar
// o contador em silêncio — e é a cópia que vai para o socket e para o banco, de
// modo que o valor certo existia só na memória do servidor. Assim, campo novo
// vem junto sem ninguém precisar lembrar.
// proximaSeq devolve a ordem da próxima mutação desta sessão. Chamada SEMPRE
// com `st.mu` seguro, que é o que faz a numeração coincidir com a ordem real
// das mutações.
//
// O contador mora na LOJA e não no estado: hidratar do banco substitui o estado
// e zeraria um contador que vivesse nele, e aí o hub descartaria todo quadro
// seguinte por achá-los atrasados. O contador só reinicia com o processo, que é
// quando o hub também esquece o que já mandou.
func (st *sessionStore) proximaSeqLocked(sessionID int64) uint64 {
	st.seqs[sessionID]++
	return st.seqs[sessionID]
}

func cloneState(s *SessionRuntimeState) *SessionRuntimeState {
	out := *s
	out.Initiative = make([]InitiativeEntry, len(s.Initiative))
	copy(out.Initiative, s.Initiative)
	return &out
}

func (st *sessionStore) getOrCreateLocked(sessionID int64) *SessionRuntimeState {
	s := st.states[sessionID]
	if s == nil {
		s = emptyRuntimeState()
		st.states[sessionID] = s
	}
	return s
}

// getState returns a snapshot of the current state (an empty tracker if never loaded).
// liveSessionsWithCharacter devolve as sessões EM MEMÓRIA que têm este
// personagem na fila (ALE-245).
//
// Só as vivas, e isso é o ponto: o aviso serve para atualizar tela aberta. Mesa
// que ninguém está olhando não precisa ser avisada — quem entrar depois busca o
// estado do zero.
func (st *sessionStore) liveSessionsWithCharacter(characterID int64) []int64 {
	st.mu.Lock()
	defer st.mu.Unlock()
	var out []int64
	for sessionID, state := range st.states {
		for _, entry := range state.Initiative {
			if entry.CharacterID != nil && *entry.CharacterID == characterID {
				out = append(out, sessionID)
				break
			}
		}
	}
	return out
}

func (st *sessionStore) getState(sessionID int64) *SessionRuntimeState {
	st.mu.Lock()
	defer st.mu.Unlock()
	return cloneState(st.getOrCreateLocked(sessionID))
}

// apply runs a pure mutation under the lock and returns a snapshot for broadcast.
func (st *sessionStore) apply(sessionID int64, fn func(*SessionRuntimeState) error) (*SessionRuntimeState, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	s := st.getOrCreateLocked(sessionID)
	if err := fn(s); err != nil {
		return nil, err
	}
	// AS DUAS COISAS, e não uma escolha entre elas — elas resolvem problemas
	// diferentes que o merge só empilhou no mesmo lugar.
	//
	// O aviso (ALE-219) é para o stream do piloto: ele é um SINO, não carrega
	// estado, e quem o ouve relê. A `seq` (ALE-253) é para o hub SSE, que
	// publica o CLONE e por isso precisa de ordem — decidir a sequência e
	// entregar têm de ser atômicos, e é por isso que ela nasce aqui dentro da
	// trava e não na publicação.
	st.avisarLocked(sessionID)
	clone := cloneState(s)
	clone.seq = st.proximaSeqLocked(sessionID)
	return clone, nil
}

func (st *sessionStore) addInitiativeEntry(sessionID int64, e InitiativeEntry) (*SessionRuntimeState, error) {
	return st.apply(sessionID, func(s *SessionRuntimeState) error { return addEntry(s, e, st.newID) })
}

func (st *sessionStore) upsertInitiativeEntry(sessionID int64, e InitiativeEntry) (*SessionRuntimeState, error) {
	return st.apply(sessionID, func(s *SessionRuntimeState) error { return upsertCharacterEntry(s, e, st.newID) })
}

func (st *sessionStore) updateInitiativeEntry(sessionID int64, entryID string, patch entryPatch) (*SessionRuntimeState, error) {
	return st.apply(sessionID, func(s *SessionRuntimeState) error { return updateEntry(s, entryID, patch) })
}

func (st *sessionStore) removeInitiativeEntry(sessionID int64, entryID string) (*SessionRuntimeState, error) {
	return st.apply(sessionID, func(s *SessionRuntimeState) error { return removeEntry(s, entryID) })
}

func (st *sessionStore) nextTurn(sessionID int64) (*SessionRuntimeState, error) {
	return st.apply(sessionID, func(s *SessionRuntimeState) error { advanceTurn(s); return nil })
}

func (st *sessionStore) previousTurn(sessionID int64) (*SessionRuntimeState, error) {
	return st.apply(sessionID, func(s *SessionRuntimeState) error { rewindTurn(s); return nil })
}

func (st *sessionStore) reset(sessionID int64) (*SessionRuntimeState, error) {
	return st.apply(sessionID, func(s *SessionRuntimeState) error { resetInitiative(s); return nil })
}

func (st *sessionStore) startScene(sessionID int64) (*SessionRuntimeState, error) {
	return st.apply(sessionID, func(s *SessionRuntimeState) error { startScene(s); return nil })
}

func (st *sessionStore) endScene(sessionID int64) (*SessionRuntimeState, error) {
	return st.apply(sessionID, func(s *SessionRuntimeState) error { endScene(s); return nil })
}

// patchVitals fixa os vitais de uma entrada. Mesma regra do delta sobre quem é a
// fonte; valor absoluto NÃO drena pool temporário, porque é uma afirmação sobre
// o total e não uma pancada.
func (st *sessionStore) patchVitals(sessionID int64, entryID string, hpCurrent, mpCurrent *int64) (*SessionRuntimeState, error) {
	charID := st.characterIDOf(sessionID, entryID)
	if charID == nil {
		return st.apply(sessionID, func(s *SessionRuntimeState) error { return patchEntryVitals(s, entryID, hpCurrent, mpCurrent) })
	}
	hp, mp, err := st.applyCharacterVitals(context.Background(), *charID, hpCurrent, mpCurrent)
	if err != nil {
		return nil, err
	}
	return st.apply(sessionID, func(s *SessionRuntimeState) error { return patchEntryVitals(s, entryID, hp, mp) })
}

// deltaVitals move os vitais de uma entrada. Se há personagem atrás dela, quem
// manda é a FICHA: o delta é aplicado na linha do personagem (dano drenando PV
// temporários, como o endpoint de dano) e a entrada espelha o resultado
// (ALE-122). NPC não tem ficha — ali o rastreador é o registro.
func (st *sessionStore) deltaVitals(sessionID int64, entryID string, hpDelta, mpDelta *int64) (*SessionRuntimeState, error) {
	charID := st.characterIDOf(sessionID, entryID)
	if charID == nil {
		return st.apply(sessionID, func(s *SessionRuntimeState) error { return deltaEntryVitals(s, entryID, hpDelta, mpDelta) })
	}
	hp, mp, err := st.applyCharacterDelta(context.Background(), *charID, hpDelta, mpDelta)
	if err != nil {
		return nil, err
	}
	return st.apply(sessionID, func(s *SessionRuntimeState) error { return patchEntryVitals(s, entryID, hp, mp) })
}

// load hydrates the session from Session.runtimeState on first access, then serves the
// cached copy./hydrate.
func (st *sessionStore) load(ctx context.Context, sessionID int64) (*SessionRuntimeState, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if s := st.states[sessionID]; s != nil {
		return cloneState(s), nil
	}
	sess, err := st.q.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	s := parseRuntimeBlob(sess.Runtimestate)
	st.states[sessionID] = s
	return cloneState(s), nil
}

// parseRuntimeBlob parses a persisted blob, falling back to an empty tracker on malformed
// input (mirrors the zod safeParse fallback). Blobs are always full (our Marshal + the
// column default carry round/turnIndex), so partial-blob defaulting isn't needed.
func parseRuntimeBlob(blob string) *SessionRuntimeState {
	if blob == "" {
		return emptyRuntimeState()
	}
	var parsed SessionRuntimeState
	if err := json.Unmarshal([]byte(blob), &parsed); err != nil {
		return emptyRuntimeState()
	}
	if parsed.Initiative == nil {
		parsed.Initiative = []InitiativeEntry{}
	}
	// Sessão gravada antes da ALE-210 volta sem `sceneActive`, e o zero de um
	// bool é `false`: uma mesa que parou na rodada 3 reabriria "fora de cena" e a
	// fila sumiria para os jogadores até o mestre clicar em iniciar. Um turno em
	// curso é PROVA de que a cena estava ligada — depois desta issue não existe
	// turno sem cena (`advanceTurn`), então isto não é remendo de migração: é a
	// invariante afirmada onde o estado entra no processo.
	if parsed.TurnIndex >= 0 {
		parsed.SceneActive = true
	}
	return &parsed
}

// persist serializes the current state to Session.runtimeState. Fire-and-forget: never
// returns an error — it returns (dirty, changed), where `changed` is true only when the
// persistence health flipped since the last persist, so the gateway can broadcast
// `persistence-warning` exactly on the transitions. The store is the single owner of the
// dirty flag (pruned by forget) — the gateway no longer tracks it. Code-review finding.
//
// Serialized so overlapping persists for one session write in order: whichever runs last
// snapshots the newest state, so the DB converges to the latest instead of a stale capture.
func (st *sessionStore) persist(ctx context.Context, sessionID int64) (dirty, changed bool) {
	pm := st.persistLock(sessionID)
	pm.Lock()
	defer pm.Unlock()

	st.mu.Lock()
	s := st.states[sessionID]
	if s == nil {
		st.mu.Unlock()
		return false, false
	}
	blob, _ := json.Marshal(cloneState(s))
	st.mu.Unlock()

	err := st.q.ResetSessionTracker(ctx, sqlcgen.ResetSessionTrackerParams{
		RuntimeState: string(blob), UpdatedAt: nowISO(), ID: sessionID,
	})

	st.mu.Lock()
	defer st.mu.Unlock()
	prev := st.dirty[sessionID] // absent ⇒ false (healthy)
	dirty = err != nil
	changed = prev != dirty
	if dirty {
		st.dirty[sessionID] = true
		log.Printf("session %d: persist failed (%v); marked dirty for retry", sessionID, err)
	} else {
		delete(st.dirty, sessionID)
	}
	return dirty, changed
}

func (st *sessionStore) isDirty(sessionID int64) bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.dirty[sessionID]
}

// forget drops a session's in-memory tracker (e.g. on clear-tracker). It does NOT clear
// the dirty flag: that would swallow the dirty→healthy recovery — a session left dirty
// must still emit persistence-warning{dirty:false} on the next successful persist. The
// dirty map self-prunes on that success, so it stays small (only currently-dirty sessions).
func (st *sessionStore) forget(sessionID int64) {
	st.mu.Lock()
	defer st.mu.Unlock()
	delete(st.states, sessionID)
}

// refreshCharacterMaxes refreshes hpMax/mpMax on every entry carrying a characterId from
// the DB rows (ceilings only; current untouched) so a mid-session level-up isn't capped at
// the stale max. Best-effort: a DB blip logs and returns the current
// snapshot rather than failing the get-session-state pull.
func (st *sessionStore) refreshCharacterMaxes(ctx context.Context, sessionID int64) *SessionRuntimeState {
	st.mu.Lock()
	ids := uniqueCharacterIDs(st.getOrCreateLocked(sessionID))
	st.mu.Unlock()
	if len(ids) == 0 {
		return st.getState(sessionID)
	}
	rows, err := st.q.ListCharacterMaxes(ctx, ids)
	if err != nil {
		log.Printf("session %d: hpMax refresh failed (%v)", sessionID, err)
		return st.getState(sessionID)
	}
	maxes := make(map[int64]sqlcgen.ListCharacterMaxesRow, len(rows))
	for _, r := range rows {
		maxes[r.ID] = r
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	s := st.getOrCreateLocked(sessionID)
	for i := range s.Initiative {
		e := &s.Initiative[i]
		if e.CharacterID == nil {
			continue
		}
		if fresh, ok := maxes[*e.CharacterID]; ok {
			e.HpMax = ptrInt64(fresh.Hpmax)
			e.MpMax = ptrInt64(fresh.Mpmax)
			// O máximo pode ter ENCOLHIDO (nível abaixado, CON caída): sem aparar,
			// a barra mostra 9/5 e a ficha se contradiz na tela — o mesmo par que
			// a criação e o PATCH de vitais recusam.
			clampCurrentTo(&e.HpCurrent, fresh.Hpmax)
			clampCurrentTo(&e.MpCurrent, fresh.Mpmax)
		}
	}
	return cloneState(s)
}

// clampCurrentTo apara o valor atual no novo máximo, deixando o ponteiro nulo
// como está (entrada sem aquele recurso não ganha um zero do nada).
func clampCurrentTo(current **int64, max int64) {
	if *current == nil || **current <= max {
		return
	}
	*current = ptrInt64(max)
}

func uniqueCharacterIDs(s *SessionRuntimeState) []int64 {
	seen := map[int64]bool{}
	ids := []int64{}
	for _, e := range s.Initiative {
		if e.CharacterID != nil && !seen[*e.CharacterID] {
			seen[*e.CharacterID] = true
			ids = append(ids, *e.CharacterID)
		}
	}
	return ids
}
