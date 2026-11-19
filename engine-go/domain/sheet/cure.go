package sheet

import (
	"strings"

	"t20engine/domain/book"
	"t20engine/domain/engine"
)

// MundaneCureBonus é o que a ficha soma aos PV da cura MUNDANA deste item (o Médico de Campo soma a
// Sabedoria, p89), e as parcelas como a frase as diz. Zero e "" no item que não é cura mundana — a
// poção é mágica, e a dela é a Cura Gentil (p133). É o mesmo número para o Usar que cura e para o
// campo que pergunta o dado.
//
//	MundaneCureBonus("balsamo-restaurador", computed) // 3, "+3 Origem: Curandeiro"
func MundaneCureBonus(catalogID string, computed engine.ComputedSheet) (bonus int, written string) {
	item := book.ItemByID(catalogID)
	if item == nil || item.Consumable == nil || !item.Consumable.MundaneCure || computed.MundaneHealing == nil {
		return 0, ""
	}
	return computed.MundaneHealing.Total, strings.Join(namedParts(computed.MundaneHealing), ", ")
}
