package sheetui

import (
	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
)

// consumableUse descreve o consumo, ou nil quando o item não se usa.
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

// offerCureBonus diz, no campo que pergunta o dado da cura, o que a ficha soma a ela (o Médico de
// Campo, p89): quem rola sabe o número que vai curar antes de confirmar.
func offerCureBonus(entry *itemSheet, item sheet.ItemDTO, computed engine.ComputedSheet) {
	if entry.Consumable == nil || entry.Consumable.HpDice == "" || item.CatalogID == nil {
		return
	}
	if _, written := sheet.MundaneCureBonus(*item.CatalogID, computed); written != "" {
		entry.Consumable.HpBonus = " · " + written
	}
}
