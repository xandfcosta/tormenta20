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
		use.HpBonus = fixedBonusBeside(use.HpDice, immediate.HP)
		use.MpBonus = fixedBonusBeside(use.MpDice, immediate.MP)
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
		entry.Consumable.HpBonus = besideTheDice(entry.Consumable.HpBonus, written)
	}
}

// fixedBonusBeside é o bônus fixo do item dito ao lado do dado (" · +2 do item"), e nada quando não há
// dado a perguntar ou bônus a somar.
func fixedBonusBeside(dice string, gain *book.GainRoll) string {
	if dice == "" || gain == nil || gain.Bonus <= 0 {
		return ""
	}
	return besideTheDice("", book.WithSign(gain.Bonus)+" do item")
}

// besideTheDice acrescenta uma parcela ao que já vai ao lado do dado: a primeira abre com " · ", as
// seguintes entram com vírgula.
func besideTheDice(written, part string) string {
	if written == "" {
		return " · " + part
	}
	return written + ", " + part
}
