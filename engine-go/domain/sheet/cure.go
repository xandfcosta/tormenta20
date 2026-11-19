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
	if !IsMundaneCure(catalogID) || computed.MundaneHealing == nil {
		return 0, ""
	}
	return computed.MundaneHealing.Total, strings.Join(namedParts(computed.MundaneHealing), ", ")
}

// IsMundaneCure diz se o item é cura MUNDANA (o bálsamo, p160), a única que a ficha soma por cima.
// Quem só precisa saber disso não calcula a ficha: a poção cura sem ela.
func IsMundaneCure(catalogID string) bool {
	item := book.ItemByID(catalogID)
	return item != nil && item.Consumable != nil && item.Consumable.MundaneCure
}
