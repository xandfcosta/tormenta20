package session

import "context"

import "t20engine/domain/live"

// Quem está ATRÁS de um combatente: a ficha, ou ninguém.
//
// É pura leitura do estado em memória, então é do regime — veio do
// `session_character_vitals.go` quando o `live/` nasceu (ALE-254). O que
// ficou lá é o que ESCREVE na ficha, que é outro contexto e agora atravessa
// uma porta.
//
// Devolver nil é a resposta para NPC, e ela decide qual é o registro: com ficha
// atrás, a linha do personagem manda (ALE-122); sem ficha, o rastreador é a
// única verdade que existe.

// CharacterIDOf reports which character backs an entry, or nil for an NPC —
// which is what decides whether the sheet or the tracker is the record.
//
// Lida do banco: a cópia em memória respondia "NPC" para a sessão que ainda não tinha carregado, e o
// dano de um personagem caía no rastreador. Sessão que não se lê também é "sem ficha" — o gesto que
// pergunta ainda vai ler a mesa, e é ali que o erro sobe.
func (st *Store) CharacterIDOf(ctx context.Context, sessionID int64, entryID string) *int64 {
	state, err := st.State(ctx, sessionID)
	if err != nil {
		return nil
	}
	idx := live.FindEntryIndex(state, entryID)
	if idx < 0 {
		return nil
	}
	return state.Initiative[idx].CharacterID
}
