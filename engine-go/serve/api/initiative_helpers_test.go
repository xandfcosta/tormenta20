package api

import (
	"strconv"

	"t20engine/domain/live"
)

// O andaime da FILA nos testes deste pacote.
//
// Ele mora num `_test.go` de propósito: num arquivo comum ele entrava no binário
// de produção sem chamador nenhum lá. E ele não é compartilhado com o `live/`,
// que tem o seu: teste não exporta para o vizinho, e duplicar andaime é barato
// onde duplicar regra não seria.

func npc(label string, init int) live.InitiativeEntry {
	return live.InitiativeEntry{Label: label, Initiative: init, Type: "npc"}
}

// idCounter gera ids previsíveis ("e1", "e2", …) para os testes deste pacote.
func idCounter() func() string {
	n := 0
	return func() string {
		n++
		return "e" + strconv.Itoa(n)
	}
}

func sheetCombatant(label string, init int, charID int64) live.InitiativeEntry {
	c := charID
	return live.InitiativeEntry{Label: label, Initiative: init, Type: "character", CharacterID: &c}
}
