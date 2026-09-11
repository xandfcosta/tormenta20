package board

// Andaime dos testes deste pacote (ALE-254).
//
// Mesma razão do irmão em `api/`: teste não exporta para o vizinho, e andaime
// compartilhado entre pacotes é dependência que ninguém declarou. Duplicar
// ANDAIME é barato; duplicar REGRA não seria.

import "t20engine/live"

func labels(st *live.SessionRuntimeState) []string {
	out := make([]string, len(st.Initiative))
	for i, e := range st.Initiative {
		out[i] = e.Label
	}
	return out
}

func npc(label string, init int) live.InitiativeEntry {
	return live.InitiativeEntry{Label: label, Initiative: init, Type: "npc"}
}
