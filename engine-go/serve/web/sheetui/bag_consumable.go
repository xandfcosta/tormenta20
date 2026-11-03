package sheetui

import "t20engine/domain/book"

// consumableUse descreve a dose, ou nil quando o item não se usa.
func consumableUse(catalog book.Item) *consumeChoice {
	// O item de dose só se usa carregando (decisão do dono): ver `itemSheetDoses`.
	if catalog.Consumable == nil || catalog.Consumable.Dose != nil {
		return nil
	}
	use := &consumeChoice{Scope: writtenScope(catalog.Consumable.Scope)}
	if immediate := catalog.Consumable.Instant; immediate != nil && catalog.Consumable.Scope == "instant" {
		use.HpDice = rollThatAsksANumber(immediate.HP)
		use.MpDice = rollThatAsksANumber(immediate.MP)
	}
	return use
}

var consumableScopes = map[string]string{
	"instant": "imediato", "scene": "1 cena", "day": "1 dia",
}

func writtenScope(scope string) string {
	if name, found := consumableScopes[scope]; found {
		return name
	}
	return scope
}

// rollThatAsksANumber é o dado que a MESA rola, ou "" quando o ganho é fixo.
//
// Ganho fixo não pergunta nada: perguntar o resultado de um dado que não existe
// é pedir que a pessoa invente um número.
func rollThatAsksANumber(gain *book.GainRoll) string {
	if gain == nil || gain.Dice == "" || gain.Dice == "0" {
		return ""
	}
	return gain.Dice
}
