package api

import (
	"fmt"
	"sort"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

// INITIATIVE_MAX_ENTRIES — hard ceiling on combatants in one tracker (runaway-add guard;
// the UI paginates poorly past ~20 anyway). Mirrors session-state.service.ts.
const initiativeMaxEntries = 50

// InitiativeEntry is one combatant row in a session's initiative tracker. The optional
// numeric fields use pointers + omitempty so the JSON matches the socket.io-client shape
// (absent ⇒ undefined) that the frontend InitiativeEntry expects.
type InitiativeEntry struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Initiative  int    `json:"initiative"`
	Type        string `json:"type"` // "character" | "npc"
	CharacterID *int64 `json:"characterId,omitempty"`
	// MonsterID liga a linha ao verbete do bestiário, para o mestre abrir o bloco
	// do monstro sem procurar no catálogo. Ausente em NPC digitado à mão — e é
	// por isso que é ponteiro: "sem bloco" é diferente de "bloco vazio".
	MonsterID *string `json:"monsterId,omitempty"`
	// CreatureID liga a linha ao bloco de criatura que o MESTRE escreveu
	// (ALE-137). Diferente do `MonsterID`, que aponta para o verbete imutável do
	// livro: este é editável e pertence à campanha, e é o que responde "o ogro
	// que eu modifiquei". Uma linha tem um ou outro, nunca os dois.
	CreatureID *int64 `json:"creatureId,omitempty"`
	// Conditions são as condições do livro ativas nesta linha (p394-395).
	// Moram na LINHA e não no bloco de criatura pelo mesmo motivo que os PV
	// atuais: condição é estado de combate, e o vilão recorrente não volta na
	// semana seguinte ainda caído. Para PC a fonte continua sendo a ficha —
	// aqui é o caminho do NPC, que ficha não tem (ALE-122).
	Conditions []string `json:"conditions,omitempty"`
	// HpHidden esconde os PV desta linha dos JOGADORES: saber que o ogro está com
	// 12 de 130 muda a decisão de quem está na mesa, e essa é a informação do
	// mestre. Ponteiro porque a maioria das linhas não decide nada a respeito.
	HpHidden  *bool  `json:"hpHidden,omitempty"`
	HpCurrent *int64 `json:"hpCurrent,omitempty"`
	HpMax     *int64 `json:"hpMax,omitempty"`
	MpCurrent *int64 `json:"mpCurrent,omitempty"`
	MpMax     *int64 `json:"mpMax,omitempty"`
}

// SessionRuntimeState is the live per-session tracker: a DESC-sorted initiative list, the
// current round, and the index of the combatant on turn (-1 before combat / after reset).
// Mirrors the frontend/backend SessionRuntimeState — the shape persisted in
// Session.runtimeState and broadcast on `session-state`.
type SessionRuntimeState struct {
	Initiative []InitiativeEntry `json:"initiative"`
	Round      int               `json:"round"`
	TurnIndex  int               `json:"turnIndex"`
	// TurnsTaken conta os turnos desde o começo do combate, e é CONTADO em vez
	// de derivado: rodada × tamanho da lista mente assim que alguém entra ou
	// morre no meio do combate, que é o normal numa mesa (ALE-142).
	TurnsTaken int `json:"turnsTaken"`
	// SceneActive é a CENA como estado explícito (ALE-210): o mestre liga e
	// desliga, e a mesa só recebe a fila enquanto ela está ligada.
	//
	// É campo NOVO e não `TurnIndex >= 0` mal nomeado, que era a resposta mais
	// barata que a issue levantava. Ela não sobrevive ao PRIMEIRO instante do
	// fluxo: iniciar a cena abre a gaveta para o mestre montar a ordem, então
	// "cena iniciada, fila vazia" é obrigatório — e `advanceTurn` não tem para
	// onde ir com a lista vazia, de modo que `TurnIndex` nunca chegaria a 0 ali.
	//
	// A recíproca também acontecia: hoje o mestre pode ter oito linhas na fila
	// com `TurnIndex` −1, montando a briga antes de começar. Um campo só não
	// consegue dizer "montando" e "fora de cena" ao mesmo tempo.
	SceneActive bool `json:"sceneActive"`
}

// emptyRuntimeState is a fresh mutable tracker. Each call returns a new slice so different
// sessions never share the initiative array.
func emptyRuntimeState() *SessionRuntimeState {
	return &SessionRuntimeState{Initiative: []InitiativeEntry{}, Round: 0, TurnIndex: -1}
}

// entryPatch is a partial update of an entry (Partial<Omit<InitiativeEntry,'id'>>): only
// the non-nil fields are applied. Kept separate from InitiativeEntry so "leave unchanged"
// (nil) is distinct from "set to zero".
type entryPatch struct {
	Label       *string `json:"label"`
	Initiative  *int    `json:"initiative"`
	Type        *string `json:"type"`
	CharacterID *int64  `json:"characterId"`
	HpCurrent   *int64  `json:"hpCurrent"`
	HpMax       *int64  `json:"hpMax"`
	MpCurrent   *int64  `json:"mpCurrent"`
	MpMax       *int64  `json:"mpMax"`
	HpHidden    *bool   `json:"hpHidden"`
	// Conditions substitui a lista inteira, como o endpoint da ficha faz: um
	// patch por condição exigiria dizer "some" e "tire", e a tela sempre sabe o
	// conjunto final.
	Conditions *[]string `json:"conditions"`
	// CreatureID liga a linha ao bloco de criatura do mestre depois de a linha
	// já existir — é o "detalhar este NPC" (ALE-137), que cria o bloco e o
	// prende ao combatente que já estava na mesa.
	CreatureID *int64 `json:"creatureId"`
}

// sortInitiative keeps the list DESC by initiative, ties broken by label using pt-BR
// collation (accent-aware, e.g. "Ávila" < "Bravo"), which is what the client's
// String.localeCompare. The collator is created per call — used only within this single
// sort goroutine, so no sharing/concurrency concern (Collator isn't concurrency-safe).
func sortInitiative(st *SessionRuntimeState) {
	c := collate.New(language.BrazilianPortuguese)
	sort.SliceStable(st.Initiative, func(i, j int) bool {
		a, b := st.Initiative[i], st.Initiative[j]
		if a.Initiative != b.Initiative {
			return a.Initiative > b.Initiative
		}
		return c.CompareString(a.Label, b.Label) < 0
	})
}

func findEntryIndex(st *SessionRuntimeState, entryID string) int {
	for i := range st.Initiative {
		if st.Initiative[i].ID == entryID {
			return i
		}
	}
	return -1
}

// turnEntryID returns the id of the entry currently on turn, or "" when none — so a
// re-sort can restore turnIndex to the same combatant regardless of index shuffles.
func turnEntryID(st *SessionRuntimeState) string {
	if st.TurnIndex < 0 || st.TurnIndex >= len(st.Initiative) {
		return ""
	}
	return st.Initiative[st.TurnIndex].ID
}

// restoreTurn points turnIndex back at the entry that was on turn before a re-sort.
func restoreTurn(st *SessionRuntimeState, id string) {
	if id == "" {
		return
	}
	if idx := findEntryIndex(st, id); idx >= 0 {
		st.TurnIndex = idx
	}
}

// addEntry appends a combatant (a fresh id is assigned via newID), re-sorts, and preserves
// who is on turn. Errors when the tracker is full.
func addEntry(st *SessionRuntimeState, input InitiativeEntry, newID func() string) error {
	if len(st.Initiative) >= initiativeMaxEntries {
		return fmt.Errorf("Initiative tracker is full (max %d entries)", initiativeMaxEntries)
	}
	onTurn := turnEntryID(st)
	input.ID = newID()
	st.Initiative = append(st.Initiative, input)
	sortInitiative(st)
	restoreTurn(st, onTurn)
	return nil
}

// upsertCharacterEntry adds a character's entry, or — if that character is already in the
// tracker — updates only its initiative (a re-roll), keeping mid-combat hp/mp. Preserves
// who is on turn.
func upsertCharacterEntry(st *SessionRuntimeState, input InitiativeEntry, newID func() string) error {
	idx := -1
	if input.CharacterID != nil {
		for i := range st.Initiative {
			if st.Initiative[i].CharacterID != nil && *st.Initiative[i].CharacterID == *input.CharacterID {
				idx = i
				break
			}
		}
	}
	if idx < 0 {
		return addEntry(st, input, newID)
	}
	onTurn := turnEntryID(st)
	st.Initiative[idx].Initiative = input.Initiative
	sortInitiative(st)
	restoreTurn(st, onTurn)
	return nil
}

// updateEntry applies a partial patch to an entry (re-sorting + preserving turn only when
// initiative changes). Errors if the entry is gone.
func updateEntry(st *SessionRuntimeState, entryID string, patch entryPatch) error {
	idx := findEntryIndex(st, entryID)
	if idx < 0 {
		return fmt.Errorf("Entry %s not found", entryID)
	}
	e := &st.Initiative[idx]
	if patch.Label != nil {
		e.Label = *patch.Label
	}
	if patch.Type != nil {
		e.Type = *patch.Type
	}
	if patch.CharacterID != nil {
		e.CharacterID = patch.CharacterID
	}
	if patch.HpCurrent != nil {
		e.HpCurrent = patch.HpCurrent
	}
	if patch.HpMax != nil {
		e.HpMax = patch.HpMax
	}
	if patch.MpCurrent != nil {
		e.MpCurrent = patch.MpCurrent
	}
	if patch.MpMax != nil {
		e.MpMax = patch.MpMax
	}
	if patch.HpHidden != nil {
		e.HpHidden = patch.HpHidden
	}
	if patch.CreatureID != nil {
		e.CreatureID = patch.CreatureID
	}
	if patch.Conditions != nil {
		e.Conditions = *patch.Conditions
	}
	if patch.Initiative != nil {
		e.Initiative = *patch.Initiative
		onTurn := turnEntryID(st)
		sortInitiative(st)
		restoreTurn(st, onTurn)
	}
	return nil
}

// removeEntry drops an entry and fixes turnIndex: shift left when a row before the current
// turn leaves; wrap to a new round when the row on turn was the tail.
func removeEntry(st *SessionRuntimeState, entryID string) error {
	idx := findEntryIndex(st, entryID)
	if idx < 0 {
		return fmt.Errorf("Entry %s not found", entryID)
	}
	st.Initiative = append(st.Initiative[:idx], st.Initiative[idx+1:]...)
	if len(st.Initiative) == 0 {
		st.TurnIndex = -1
		return nil
	}
	if idx < st.TurnIndex {
		st.TurnIndex--
	} else if idx == st.TurnIndex && st.TurnIndex >= len(st.Initiative) {
		st.TurnIndex = 0
		st.Round++
	}
	return nil
}

// advanceTurn moves to the next combatant, wrapping to index 0 and bumping the round.
// From the pre-combat state (turnIndex -1) it puts the first combatant on turn without
// bumping the round.
//
// Sem CENA não avança (ALE-210). A guarda não é defensiva: ela é o que dá ao
// estado uma direção única — turno só existe dentro de cena —, e é dela que
// `parseRuntimeBlob` tira o direito de deduzir a cena de um turno em curso.
func advanceTurn(st *SessionRuntimeState) {
	if !st.SceneActive || len(st.Initiative) == 0 {
		return
	}
	if st.TurnIndex < 0 {
		st.TurnIndex = 0
		if st.Round < 1 {
			st.Round = 1
		}
		st.TurnsTaken++
		return
	}
	st.TurnIndex++
	st.TurnsTaken++
	if st.TurnIndex >= len(st.Initiative) {
		st.TurnIndex = 0
		st.Round++
	}
}

// rewindTurn desfaz um "Próximo turno" — o erro mais comum da mesa, e cujo único
// conserto até aqui era dar a volta na iniciativa inteira, o que empurrava a
// rodada junto. Cruzar a virada de volta devolve a rodada; desfazer o primeiro
// turno devolve ao pré-combate (turnIndex -1) sem zerar a rodada, porque a
// rodada 1 JÁ começou e voltar não desfaz isso.
func rewindTurn(st *SessionRuntimeState) {
	if len(st.Initiative) == 0 || st.TurnIndex < 0 {
		return
	}
	// Desfazer um turno desconta um turno: o contador é o que JÁ aconteceu, e
	// voltar diz que não aconteceu.
	if st.TurnsTaken > 0 {
		st.TurnsTaken--
	}
	if st.TurnIndex > 0 {
		st.TurnIndex--
		return
	}
	if st.Round > 1 {
		st.TurnIndex = len(st.Initiative) - 1
		st.Round--
		return
	}
	st.TurnIndex = -1
}

// startScene liga a cena, e só isso: a ordem se monta DEPOIS. É por esse gesto
// que a fila passa a existir para a mesa (ALE-210).
func startScene(st *SessionRuntimeState) {
	st.SceneActive = true
}

// endScene desliga a cena e devolve o combate ao começo — mas GUARDA a fila.
//
// Encerrar não é reiniciar, e essa é a única diferença entre os dois: quem
// esvazia é o resetInitiative. O mestre que encerra a briga do castelo não pode
// pagar oito goblins digitados de novo para recomeçá-la.
func endScene(st *SessionRuntimeState) {
	st.SceneActive = false
	st.Round = 0
	st.TurnIndex = -1
	st.TurnsTaken = 0
}

// redactForPlayers devolve uma CÓPIA do estado sem os PV das linhas que o mestre
// escondeu. A flag continua na cópia de propósito: o jogador precisa saber que
// existe vida ali e que ela está oculta — sem isso, "sem barra" e "escondido"
// viram a mesma coisa na tela, e o segundo é informação.
//
// Fora de cena o jogador não recebe fila NENHUMA (ALE-210). A trava mora aqui, e
// não numa condição de render, porque não mandar é diferente de não desenhar: a
// primeira é segurança, a segunda é UX. E mora nesta função em particular porque
// ela é o gargalo pelo qual os DOIS caminhos do estado passam — o broadcast por
// sala de papel e o ack do `get-session-state` (ALE-122).
func redactForPlayers(st *SessionRuntimeState) *SessionRuntimeState {
	if !st.SceneActive {
		// Rastreador limpo e não `cloneState` com a lista zerada: a rodada e o
		// contador de turnos também são da cena, e "rodada 7, ninguém na fila"
		// é uma contradição que o jogador leria como defeito.
		return emptyRuntimeState()
	}
	out := cloneState(st)
	for i := range out.Initiative {
		e := &out.Initiative[i]
		if e.HpHidden != nil && *e.HpHidden {
			e.HpCurrent, e.HpMax = nil, nil
		}
	}
	return out
}

// stateForRole é o que UM socket pode ver. Existe porque o broadcast não é o
// único caminho do estado até a tela: o `ack` do `get-state` hidrata o cliente e
// responde a quem pediu. Papel desconhecido cai em jogador — errar para o lado
// que mostra seria vazar por omissão (ALE-122).
func stateForRole(role string, st *SessionRuntimeState) *SessionRuntimeState {
	if role == "gm" {
		return st
	}
	return redactForPlayers(st)
}

// resetInitiative clears the tracker but keeps the session tracked. Desliga a
// cena junto (ALE-210): reiniciar é voltar ao ponto de partida, e o ponto de
// partida é fora de cena com a fila vazia. Deixar a cena ligada aqui produziria
// o estado "em cena, ninguém na fila" sem ninguém ter pedido por ele.
func resetInitiative(st *SessionRuntimeState) {
	st.Initiative = []InitiativeEntry{}
	st.Round = 0
	st.TurnIndex = -1
	st.TurnsTaken = 0
	st.SceneActive = false
}

// patchEntryVitals sets absolute hp/mp on an entry, clamped to its max when present.
// (the DB write-through lives in the store layer).
func patchEntryVitals(st *SessionRuntimeState, entryID string, hpCurrent, mpCurrent *int64) error {
	idx := findEntryIndex(st, entryID)
	if idx < 0 {
		return fmt.Errorf("Entry %s not found", entryID)
	}
	e := &st.Initiative[idx]
	if hpCurrent != nil {
		e.HpCurrent = ptrInt64(clampVital(*hpCurrent, e.HpMax))
	}
	if mpCurrent != nil {
		e.MpCurrent = ptrInt64(clampVital(*mpCurrent, e.MpMax))
	}
	return nil
}

// deltaEntryVitals applies an hp/mp delta ("sofreu 10 de dano" ⇒ hpDelta -10). Absent
// current counts as 0.
func deltaEntryVitals(st *SessionRuntimeState, entryID string, hpDelta, mpDelta *int64) error {
	idx := findEntryIndex(st, entryID)
	if idx < 0 {
		return fmt.Errorf("Entry %s not found", entryID)
	}
	e := &st.Initiative[idx]
	if hpDelta != nil {
		e.HpCurrent = ptrInt64(clampVital(derefOr(e.HpCurrent, 0)+*hpDelta, e.HpMax))
	}
	if mpDelta != nil {
		e.MpCurrent = ptrInt64(clampVital(derefOr(e.MpCurrent, 0)+*mpDelta, e.MpMax))
	}
	return nil
}

// clampVital floors a vital at 0 and caps it at max when present. Negative floor is 0: the
// tracker never displays below 0 (a character at 0 is unconscious/dying, handled
// narratively). Mirrors the clampVital in session-state.service.ts.
func clampVital(value int64, max *int64) int64 {
	floored := value
	if floored < 0 {
		floored = 0
	}
	if max == nil {
		return floored
	}
	if floored > *max {
		return *max
	}
	return floored
}

func ptrInt64(v int64) *int64 { return &v }

func derefOr(p *int64, def int64) int64 {
	if p == nil {
		return def
	}
	return *p
}
