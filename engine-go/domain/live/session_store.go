package live

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"t20engine/infra/platform"

	"github.com/google/uuid"

	"t20engine/infra/db/sqlcgen"
	"t20engine/infra/events"
)

// NewUUID gera um id v4 aleatório para entrada de iniciativa. Injetado no store
// para o teste trocar por um gerador determinístico.
func NewUUID() string { return uuid.NewString() }

// SessionStore guarda o estado vivo de cada sessão em memória: protege as
// mutações puras (session_state.go) com um mutex e grava em
// `Session.runtimeState`. Reiniciar o servidor zera os rastreadores até o
// primeiro `Load()` reidratar do banco, e cada mutação devolve um instantâneo
// para quem chama serializar e transmitir fora da trava.
type SessionStore struct {
	Mu sync.Mutex
	// ficha é a PORTA para o contexto da ficha: o regime escreve PV e PM de
	// personagem, mas as REGRAS dessa escrita são de lá. Nulo é caminho normal em
	// teste de regime puro — quem tem personagem na fila injeta o implementador.
	ficha  SheetVitals
	States map[int64]*SessionRuntimeState
	Dirty  map[int64]bool
	// seqs numera as mutações de cada sessão, para o hub reconhecer quadro
	// atrasado. Mora aqui e não no estado: hidratar do banco troca o estado, e um
	// contador que vivesse nele voltaria a zero.
	seqs  map[int64]uint64
	newID func() string
	q     *sqlcgen.Queries
	// persistMus guarda um mutex por sessão (sessionID → *sync.Mutex) serializando
	// as gravações do estado daquela sessão, para mutações concorrentes não
	// chegarem fora de ordem — SEM acoplar a latência entre sessões.
	persistMus sync.Map
	// bus é por onde as mutações desta sessão viram notícia.
	//
	// Quem publica é o `apply`, DEPOIS de soltar a trava, e o evento diz o que
	// aconteceu em vez de só tocar o sino.
	//
	// Ponteiro e não valor porque ele é COMPARTILHADO com o tabuleiro e com o
	// servidor: um barramento por store devolveria a quem escuta o trabalho de
	// juntar as peças de novo. Nulo EXPLODE, e é para explodir: quem monta um store
	// à mão sem barramento descobre no primeiro `apply`, e não numa tela que não
	// atualiza.
	bus *events.Bus
}

// persistLock devolve o mutex de gravação daquela sessão, criando-o no primeiro
// uso.
func (st *SessionStore) persistLock(sessionID int64) *sync.Mutex {
	m, _ := st.persistMus.LoadOrStore(sessionID, &sync.Mutex{})
	return m.(*sync.Mutex)
}

// NewSessionStore recebe a PORTA da ficha por parâmetro — injetada e não
// importada, que é o que impede o regime de conhecer as regras da ficha.
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

// cloneState copia o estado para o broadcast: as entradas vão por VALOR e a
// fatia é recriada. Os vitais `*int64` são compartilhados mas nunca mutados no
// lugar (patch e delta sempre atribuem ponteiro novo), então o instantâneo é
// seguro de serializar fora da trava.
//
// Por VALOR e não campo a campo: listar campos é uma lista que envelhece, e o
// campo novo que ficasse de fora continuaria compilando e passaria a zerar o
// valor em silêncio — na cópia que vai para o fio e para o banco.
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

// LiveSessionsWithCharacter devolve as sessões EM MEMÓRIA que têm este
// personagem na fila.
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

// GetState devolve um instantâneo do estado atual (rastreador vazio quando a
// sessão nunca foi carregada).
func (st *SessionStore) GetState(sessionID int64) *SessionRuntimeState {
	st.Mu.Lock()
	defer st.Mu.Unlock()
	return cloneState(st.getOrCreateLocked(sessionID))
}

// apply roda uma mutação pura sob a trava, publica o evento e devolve o
// instantâneo para o broadcast.
//
// O EVENTO É PARÂMETRO OBRIGATÓRIO, e é isso que substitui uma promessa por uma
// garantia: não dá para mutar sem dizer O QUE aconteceu, e o compilador cobra.
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
// A `seq` nasce AQUI DENTRO e não na publicação: ela numera as mutações para o
// hub reconhecer quadro atrasado, e decidir a sequência e entregar têm de ser
// atômicos.
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

// DeltaCharacterVitals move os vitais de um PERSONAGEM, esteja ele na fila ou
// não.
//
// O irmão dele, o `DeltaVitals`, entra pela ENTRADA da fila, e é o caminho do
// COMBATE. Este entra pelo personagem, e é o caminho do ELENCO — onde metade da
// gente não tem linha na iniciativa durante a maior parte da sessão, que é a
// razão de o elenco existir separado da fila.
//
// Quem manda é a FICHA nos dois, e é isso que impede as duas telas de
// divergirem sobre o mesmo herói. A fila ESPELHA quando existe linha; quando não
// existe, não há o que espelhar e a ficha é a única a mudar — devolver o estado
// como está é a resposta certa, e não um erro, porque "não está na fila" é o
// caso comum aqui e não uma falha.
func (st *SessionStore) DeltaCharacterVitals(sessionID, characterID int64, hpDelta, mpDelta *int64) (*SessionRuntimeState, error) {
	hp, mp, err := st.ficha.ApplyDelta(context.Background(), characterID, hpDelta, mpDelta)
	if err != nil {
		return nil, err
	}
	entryID := st.entryIDForCharacter(sessionID, characterID)
	if entryID == "" {
		return st.GetState(sessionID), nil
	}
	return st.apply(sessionID, vitalsEvent(sessionID, entryID, &characterID),
		func(s *SessionRuntimeState) error { return patchEntryVitals(s, entryID, hp, mp) })
}

// entryIDForCharacter é o inverso do `CharacterIDOf`, e devolve "" para quem não
// está na fila.
//
// Vazio e não erro: no elenco, estar FORA da iniciativa é o estado normal — o
// mestre cura a Arwen entre duas brigas —, e tratar isso como falha faria o
// gesto recusar exatamente o caso que ele veio atender.
func (st *SessionStore) entryIDForCharacter(sessionID, characterID int64) string {
	for _, e := range st.GetState(sessionID).Initiative {
		if e.CharacterID != nil && *e.CharacterID == characterID {
			return e.ID
		}
	}
	return ""
}

// DeltaVitals move os vitais de uma entrada. Se há personagem atrás dela, quem
// manda é a FICHA: o delta é aplicado na linha do personagem (dano drenando PV
// temporários, como o endpoint de dano) e a entrada espelha o resultado. NPC não
// tem ficha — ali o rastreador é o registro.
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

// Load hidrata a sessão de `Session.runtimeState` no primeiro acesso e depois
// serve a cópia em memória.
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

// parseRuntimeBlob lê um blob gravado e cai para um rastreador vazio quando ele
// vem malformado. Os blobs são sempre completos (o nosso Marshal e o default da
// coluna carregam rodada e turno), então não há campo parcial a preencher.
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
	// Sessão gravada antes de `sceneActive` existir volta sem ele, e o zero de um
	// bool é `false`: uma mesa que parou na rodada 3 reabriria "fora de cena" e a
	// fila sumiria para os jogadores até o mestre clicar em iniciar. Um turno em
	// curso é PROVA de que a cena estava ligada — não existe turno sem cena
	// (`advanceTurn`), então isto não é remendo de migração: é a invariante
	// afirmada onde o estado entra no processo.
	if parsed.TurnIndex >= 0 {
		parsed.SceneActive = true
	}
	return &parsed
}

// SaveFailed diz se a última gravação do estado desta sessão falhou.
//
// ESTADO e não notícia, e a diferença é o que faz o aviso servir: ele vale
// enquanto durar, então quem abre a aba dez minutos depois da primeira falha
// merece vê-lo. Um evento perdido é um evento que não existiu. O irmão dele é o
// `BoardStore.SaveFailed`.
//
// Sob a trava porque o `Dirty` é escrito pelo `Persist`, que roda em goroutine.
func (st *SessionStore) SaveFailed(sessionID int64) bool {
	st.Mu.Lock()
	defer st.Mu.Unlock()
	return st.Dirty[sessionID]
}

// Persist serializa o estado atual em `Session.runtimeState`. Dispara e esquece:
// nunca devolve erro — devolve (Dirty, changed), com `changed` verdadeiro só
// quando a saúde da gravação VIROU desde o último `Persist`, para quem chama
// avisar a mesa exatamente nas transições. O store é o dono único da marca.
//
// Serializado para que gravações sobrepostas da mesma sessão cheguem em ordem: a
// última a rodar retrata o estado mais novo, e o banco converge para ele em vez
// de para uma captura velha.
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
		RuntimeState: string(blob), UpdatedAt: platform.NowISO(), ID: sessionID,
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

// Forget descarta o rastreador em memória de uma sessão. Ele NÃO limpa o
// `Dirty`: isso engoliria a recuperação suja→saudável — uma sessão deixada suja
// ainda precisa avisar `persistence-warning{Dirty:false}` no próximo `Persist`
// bem-sucedido, e o mapa se poda sozinho nesse sucesso.
func (st *SessionStore) Forget(sessionID int64) {
	st.Mu.Lock()
	defer st.Mu.Unlock()
	delete(st.States, sessionID)
}

// SessionDeleted é o `Forget` de uma sessão que deixou de EXISTIR, e a diferença
// entre os dois é uma linha: esta apaga o `Dirty` também.
//
// O argumento do `Forget` acima — *"não limpa o Dirty: isso engoliria a
// recuperação suja→saudável"* — depende de haver um próximo `Persist` que avise
// que voltou a gravar. Com a sessão apagada não há: a marca ficaria acesa até o
// processo reiniciar, sobre uma mesa que ninguém quer gravar.
func (st *SessionStore) SessionDeleted(sessionID int64) {
	st.Mu.Lock()
	defer st.Mu.Unlock()
	delete(st.States, sessionID)
	delete(st.Dirty, sessionID)
}

// RefreshCharacterMaxes relê do banco o PV/PM MÁXIMO de toda entrada com
// personagem atrás (só os tetos; o atual não é tocado), para um nível subido no
// meio da sessão não ficar preso ao máximo velho. Melhor esforço: uma piscada do
// banco vira log e devolve o instantâneo atual em vez de derrubar a leitura.
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
