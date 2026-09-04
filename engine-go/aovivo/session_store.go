package aovivo

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"t20engine/plataforma"

	"github.com/google/uuid"

	"t20engine/db/sqlcgen"
	"t20engine/events"
)

// NewUUID generates a random v4 UUID string for initiative entry ids (randomUUID() in the
// service). Injected into the store so tests can swap a deterministic generator.
func NewUUID() string { return uuid.NewString() }

// SessionStore holds each session's in-memory runtime state, guarding the pure mutations
// (session_state.go) with a mutex and persisting fire-and-Forget to Session.runtimeState.
// Mirrors SessionStateService — a server restart wipes trackers until the first Load()
// re-hydrates from the DB. Mutation methods return a snapshot (deep-enough copy) so the
// gateway can serialize/broadcast it without racing a concurrent message on the session.
type SessionStore struct {
	Mu sync.Mutex
	// ficha é a PORTA para o contexto da ficha (ALE-254): o regime escreve PV e
	// PM de personagem, mas as REGRAS dessa escrita são de lá. Nulo é caminho
	// normal em teste de regime puro — quem tem personagem na fila injeta o
	// implementador.
	ficha  SheetVitals
	States map[int64]*SessionRuntimeState
	Dirty  map[int64]bool
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
	// bus é por onde as mutações desta sessão viram notícia (ALE-279).
	//
	// Aqui morava `ouvintes map[int64][]chan struct{}`, guardado sob o MESMO `mu`
	// das mutações — o aviso saía de dentro da trava, junto com a mudança. O
	// barramento não precisa disso: quem publica é o `apply`, DEPOIS de soltar a
	// trava, e o evento diz o que aconteceu em vez de só tocar o sino.
	//
	// Ponteiro e não valor porque ele é COMPARTILHADO com o tabuleiro e com o
	// servidor: um barramento por store devolveria o problema que esta issue
	// veio resolver, que é quem escuta ter de juntar as peças de novo. Nulo
	// EXPLODE, e é para explodir: quem monta um store à mão sem barramento
	// descobre no primeiro `apply`, e não numa tela que não atualiza.
	bus *events.Bus
}

// persistLock returns the per-session DB-write mutex, creating it on first use.
func (st *SessionStore) persistLock(sessionID int64) *sync.Mutex {
	m, _ := st.persistMus.LoadOrStore(sessionID, &sync.Mutex{})
	return m.(*sync.Mutex)
}

// NewSessionStore recebe a PORTA da ficha por parâmetro (ALE-254) — injetada e
// não importada, que é o que impede o regime de conhecer as regras da ficha.
func NewSessionStore(q *sqlcgen.Queries, newID func() string, ficha SheetVitals, bus *events.Bus) *SessionStore {
	return &SessionStore{
		States: map[int64]*SessionRuntimeState{},
		Dirty:  map[int64]bool{},
		seqs:   map[int64]uint64{},
		newID:  newID,
		ficha:  ficha,
		q:      q,
		bus:    bus,
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
// nextSeqLocked devolve a ordem da próxima mutação desta sessão. Chamada SEMPRE
// com `st.Mu` seguro, que é o que faz a numeração coincidir com a ordem real
// das mutações.
//
// O contador mora na LOJA e não no estado: hidratar do banco substitui o estado
// e zeraria um contador que vivesse nele, e aí o hub descartaria todo quadro
// seguinte por achá-los atrasados. O contador só reinicia com o processo, que é
// quando o hub também esquece o que já mandou.
func (st *SessionStore) nextSeqLocked(sessionID int64) uint64 {
	st.seqs[sessionID]++
	return st.seqs[sessionID]
}

func cloneState(s *SessionRuntimeState) *SessionRuntimeState {
	out := *s
	out.Initiative = make([]InitiativeEntry, len(s.Initiative))
	copy(out.Initiative, s.Initiative)
	return &out
}

func (st *SessionStore) getOrCreateLocked(sessionID int64) *SessionRuntimeState {
	s := st.States[sessionID]
	if s == nil {
		s = EmptyRuntimeState()
		st.States[sessionID] = s
	}
	return s
}

// GetState returns a snapshot of the current state (an empty tracker if never loaded).
// LiveSessionsWithCharacter devolve as sessões EM MEMÓRIA que têm este
// personagem na fila (ALE-245).
//
// Só as vivas, e isso é o ponto: o aviso serve para atualizar tela aberta. Mesa
// que ninguém está olhando não precisa ser avisada — quem entrar depois busca o
// estado do zero.
func (st *SessionStore) LiveSessionsWithCharacter(characterID int64) []int64 {
	st.Mu.Lock()
	defer st.Mu.Unlock()
	var out []int64
	for sessionID, state := range st.States {
		for _, entry := range state.Initiative {
			if entry.CharacterID != nil && *entry.CharacterID == characterID {
				out = append(out, sessionID)
				break
			}
		}
	}
	return out
}

func (st *SessionStore) GetState(sessionID int64) *SessionRuntimeState {
	st.Mu.Lock()
	defer st.Mu.Unlock()
	return cloneState(st.getOrCreateLocked(sessionID))
}

// apply runs a pure mutation under the lock, publishes the event, and returns a
// snapshot for broadcast.
//
// O EVENTO É PARÂMETRO OBRIGATÓRIO, e é isso que substitui uma promessa por uma
// garantia (ALE-279). Antes o aviso era uma linha aqui dentro, e o comentário
// prometia que ninguém escapava porque `apply` é o funil das treze mutações —
// uma promessa que vale enquanto ninguém escrever a décima quarta por fora.
// Agora não dá para mutar sem dizer O QUE aconteceu: o compilador cobra.
//
// A publicação sai FORA da trava. O barramento é folha e poderia ser chamado de
// dentro (ver `events.Bus.Publish`), mas quem acorda agora sabe o que houve e
// pode ler o estado na hora — publicar sob a trava faria esse leitor esperar
// pelo escritor no exato instante em que foi acordado para ler.
func (st *SessionStore) apply(sessionID int64, ev events.Event, fn func(*SessionRuntimeState) error) (*SessionRuntimeState, error) {
	clone, err := st.applyLocked(sessionID, fn)
	if err != nil {
		return nil, err
	}
	st.bus.Publish(ev)
	return clone, nil
}

// applyLocked é a parte que precisa da trava: mutar e tirar o retrato.
//
// A `seq` (ALE-253) nasce AQUI DENTRO e não na publicação: ela numera as
// mutações para o hub reconhecer quadro atrasado, e decidir a sequência e
// entregar têm de ser atômicos.
func (st *SessionStore) applyLocked(sessionID int64, fn func(*SessionRuntimeState) error) (*SessionRuntimeState, error) {
	st.Mu.Lock()
	defer st.Mu.Unlock()
	s := st.getOrCreateLocked(sessionID)
	if err := fn(s); err != nil {
		return nil, err
	}
	clone := cloneState(s)
	clone.Seq = st.nextSeqLocked(sessionID)
	return clone, nil
}

func (st *SessionStore) AddInitiativeEntry(sessionID int64, e InitiativeEntry) (*SessionRuntimeState, error) {
	return st.apply(sessionID, events.CombatantJoined{SessionID: sessionID, EntryID: e.ID},
		func(s *SessionRuntimeState) error { return AddEntry(s, e, st.newID) })
}

func (st *SessionStore) UpsertInitiativeEntry(sessionID int64, e InitiativeEntry) (*SessionRuntimeState, error) {
	return st.apply(sessionID, events.CombatantJoined{SessionID: sessionID, EntryID: e.ID},
		func(s *SessionRuntimeState) error { return upsertCharacterEntry(s, e, st.newID) })
}

func (st *SessionStore) UpdateInitiativeEntry(sessionID int64, entryID string, patch EntryPatch) (*SessionRuntimeState, error) {
	return st.apply(sessionID, events.CombatantChanged{SessionID: sessionID, EntryID: entryID},
		func(s *SessionRuntimeState) error { return UpdateEntry(s, entryID, patch) })
}

func (st *SessionStore) RemoveInitiativeEntry(sessionID int64, entryID string) (*SessionRuntimeState, error) {
	return st.apply(sessionID, events.CombatantLeft{SessionID: sessionID, EntryID: entryID},
		func(s *SessionRuntimeState) error { return RemoveEntry(s, entryID) })
}

func (st *SessionStore) NextTurn(sessionID int64) (*SessionRuntimeState, error) {
	return st.apply(sessionID, events.TurnAdvanced{SessionID: sessionID},
		func(s *SessionRuntimeState) error { advanceTurn(s); return nil })
}

func (st *SessionStore) PreviousTurn(sessionID int64) (*SessionRuntimeState, error) {
	return st.apply(sessionID, events.TurnAdvanced{SessionID: sessionID},
		func(s *SessionRuntimeState) error { rewindTurn(s); return nil })
}

func (st *SessionStore) Reset(sessionID int64) (*SessionRuntimeState, error) {
	return st.apply(sessionID, events.InitiativeReset{SessionID: sessionID},
		func(s *SessionRuntimeState) error { resetInitiative(s); return nil })
}

func (st *SessionStore) StartScene(sessionID int64) (*SessionRuntimeState, error) {
	return st.apply(sessionID, events.SceneStarted{SessionID: sessionID},
		func(s *SessionRuntimeState) error { StartScene(s); return nil })
}

func (st *SessionStore) EndScene(sessionID int64) (*SessionRuntimeState, error) {
	return st.apply(sessionID, events.SceneEnded{SessionID: sessionID},
		func(s *SessionRuntimeState) error { EndScene(s); return nil })
}

// PatchVitals fixa os vitais de uma entrada. Mesma regra do delta sobre quem é a
// fonte; valor absoluto NÃO drena pool temporário, porque é uma afirmação sobre
// o total e não uma pancada.
func (st *SessionStore) PatchVitals(sessionID int64, entryID string, hpCurrent, mpCurrent *int64) (*SessionRuntimeState, error) {
	charID := st.CharacterIDOf(sessionID, entryID)
	if charID == nil {
		return st.apply(sessionID, vitalsEvent(sessionID, entryID, nil),
			func(s *SessionRuntimeState) error { return patchEntryVitals(s, entryID, hpCurrent, mpCurrent) })
	}
	hp, mp, err := st.ficha.ApplyAbsolute(context.Background(), *charID, hpCurrent, mpCurrent)
	if err != nil {
		return nil, err
	}
	return st.apply(sessionID, vitalsEvent(sessionID, entryID, charID),
		func(s *SessionRuntimeState) error { return patchEntryVitals(s, entryID, hp, mp) })
}

// DeltaVitals move os vitais de uma entrada. Se há personagem atrás dela, quem
// manda é a FICHA: o delta é aplicado na linha do personagem (dano drenando PV
// temporários, como o endpoint de dano) e a entrada espelha o resultado
// (ALE-122). NPC não tem ficha — ali o rastreador é o registro.
func (st *SessionStore) DeltaVitals(sessionID int64, entryID string, hpDelta, mpDelta *int64) (*SessionRuntimeState, error) {
	charID := st.CharacterIDOf(sessionID, entryID)
	if charID == nil {
		return st.apply(sessionID, vitalsEvent(sessionID, entryID, nil),
			func(s *SessionRuntimeState) error { return deltaEntryVitals(s, entryID, hpDelta, mpDelta) })
	}
	hp, mp, err := st.ficha.ApplyDelta(context.Background(), *charID, hpDelta, mpDelta)
	if err != nil {
		return nil, err
	}
	return st.apply(sessionID, vitalsEvent(sessionID, entryID, charID),
		func(s *SessionRuntimeState) error { return patchEntryVitals(s, entryID, hp, mp) })
}

// Load hydrates the session from Session.runtimeState on first access, then serves the
// cached copy./hydrate.
func (st *SessionStore) Load(ctx context.Context, sessionID int64) (*SessionRuntimeState, error) {
	st.Mu.Lock()
	defer st.Mu.Unlock()
	if s := st.States[sessionID]; s != nil {
		return cloneState(s), nil
	}
	sess, err := st.q.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	s := parseRuntimeBlob(sess.Runtimestate)
	st.States[sessionID] = s
	return cloneState(s), nil
}

// parseRuntimeBlob parses a persisted blob, falling back to an empty tracker on malformed
// input (mirrors the zod safeParse fallback). Blobs are always full (our Marshal + the
// column default carry round/turnIndex), so partial-blob defaulting isn't needed.
func parseRuntimeBlob(blob string) *SessionRuntimeState {
	if blob == "" {
		return EmptyRuntimeState()
	}
	var parsed SessionRuntimeState
	if err := json.Unmarshal([]byte(blob), &parsed); err != nil {
		return EmptyRuntimeState()
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

// Persist serializes the current state to Session.runtimeState. Fire-and-Forget: never
// returns an error — it returns (Dirty, changed), where `changed` is true only when the
// persistence health flipped since the last Persist, so the gateway can broadcast
// `persistence-warning` exactly on the transitions. The store is the single owner of the
// Dirty flag (pruned by Forget) — the gateway no longer tracks it. Code-review finding.
//
// Serialized so overlapping persists for one session write in order: whichever runs last
// snapshots the newest state, so the DB converges to the latest instead of a stale capture.

// SaveFailed diz se a última gravação do estado desta sessão falhou.
//
// ESTADO e não notícia, e a diferença é o que faz o aviso servir: ele vale
// enquanto durar, então quem abre a aba dez minutos depois da primeira falha
// merece vê-lo. Um evento perdido é um evento que não existiu (ALE-288). O irmão dele é o `BoardStore.SaveFailed`.
//
// Sob a trava porque o `Dirty` é escrito pelo `Persist`, que roda em goroutine.
func (st *SessionStore) SaveFailed(sessionID int64) bool {
	st.Mu.Lock()
	defer st.Mu.Unlock()
	return st.Dirty[sessionID]
}

func (st *SessionStore) Persist(ctx context.Context, sessionID int64) (Dirty, changed bool) {
	pm := st.persistLock(sessionID)
	pm.Lock()
	defer pm.Unlock()

	st.Mu.Lock()
	s := st.States[sessionID]
	if s == nil {
		st.Mu.Unlock()
		return false, false
	}
	blob, _ := json.Marshal(cloneState(s))
	st.Mu.Unlock()

	err := st.q.ResetSessionTracker(ctx, sqlcgen.ResetSessionTrackerParams{
		RuntimeState: string(blob), UpdatedAt: plataforma.NowISO(), ID: sessionID,
	})

	st.Mu.Lock()
	defer st.Mu.Unlock()
	prev := st.Dirty[sessionID] // absent ⇒ false (healthy)
	Dirty = err != nil
	changed = prev != Dirty
	if Dirty {
		st.Dirty[sessionID] = true
		log.Printf("session %d: Persist failed (%v); marked Dirty for retry", sessionID, err)
	} else {
		delete(st.Dirty, sessionID)
	}
	return Dirty, changed
}

func (st *SessionStore) IsDirty(sessionID int64) bool {
	st.Mu.Lock()
	defer st.Mu.Unlock()
	return st.Dirty[sessionID]
}

// Forget drops a session's in-memory tracker (e.g. on clear-tracker). It does NOT clear
// the Dirty flag: that would swallow the Dirty→healthy recovery — a session left Dirty
// must still Emit persistence-warning{Dirty:false} on the next successful Persist. The
// Dirty map self-prunes on that success, so it stays small (only currently-Dirty sessions).
func (st *SessionStore) Forget(sessionID int64) {
	st.Mu.Lock()
	defer st.Mu.Unlock()
	delete(st.States, sessionID)
}

// RefreshCharacterMaxes refreshes hpMax/mpMax on every entry carrying a characterId from
// the DB rows (ceilings only; current untouched) so a mid-session level-up isn't capped at
// the stale max. Best-effort: a DB blip logs and returns the current
// snapshot rather than failing the get-session-state pull.
func (st *SessionStore) RefreshCharacterMaxes(ctx context.Context, sessionID int64) *SessionRuntimeState {
	st.Mu.Lock()
	ids := uniqueCharacterIDs(st.getOrCreateLocked(sessionID))
	st.Mu.Unlock()
	if len(ids) == 0 {
		return st.GetState(sessionID)
	}
	rows, err := st.q.ListCharacterMaxes(ctx, ids)
	if err != nil {
		log.Printf("session %d: hpMax refresh failed (%v)", sessionID, err)
		return st.GetState(sessionID)
	}
	maxes := make(map[int64]sqlcgen.ListCharacterMaxesRow, len(rows))
	for _, r := range rows {
		maxes[r.ID] = r
	}
	st.Mu.Lock()
	defer st.Mu.Unlock()
	s := st.getOrCreateLocked(sessionID)
	for i := range s.Initiative {
		e := &s.Initiative[i]
		if e.CharacterID == nil {
			continue
		}
		if fresh, ok := maxes[*e.CharacterID]; ok {
			e.HpMax = PtrInt64(fresh.Hpmax)
			e.MpMax = PtrInt64(fresh.Mpmax)
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
	*current = PtrInt64(max)
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

// vitalsEvent monta o evento do dano ou da cura.
//
// O `CharacterID` só entra quando HÁ ficha atrás da linha, e o zero do NPC é
// significativo: ele é o que impede o dano num ogro de acordar toda ficha do
// processo, porque um alvo zero casaria com um interesse zero. Ver
// `TestNpcVitalsWakeNoSheet`.
func vitalsEvent(sessionID int64, entryID string, charID *int64) events.VitalsChanged {
	ev := events.VitalsChanged{SessionID: sessionID, EntryID: entryID}
	if charID != nil {
		ev.CharacterID = *charID
	}
	return ev
}
