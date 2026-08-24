package aovivo

// Quem está ATRÁS de um combatente: a ficha, ou ninguém.
//
// É pura leitura do estado em memória, então é do regime — veio do
// `session_character_vitals.go` quando o `aovivo/` nasceu (ALE-254). O que
// ficou lá é o que ESCREVE na ficha, que é outro contexto e agora atravessa
// uma porta.
//
// Devolver nil é a resposta para NPC, e ela decide qual é o registro: com ficha
// atrás, a linha do personagem manda (ALE-122); sem ficha, o rastreador é a
// única verdade que existe.

// characterIDOf reports which character backs an entry, or nil for an NPC —
// which is what decides whether the sheet or the tracker is the record.
func (st *SessionStore) CharacterIDOf(sessionID int64, entryID string) *int64 {
	st.Mu.Lock()
	defer st.Mu.Unlock()
	state := st.States[sessionID]
	if state == nil {
		return nil
	}
	idx := FindEntryIndex(state, entryID)
	if idx < 0 {
		return nil
	}
	return state.Initiative[idx].CharacterID
}
