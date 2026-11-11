package sheetui

import (
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
)

// AS PATAS ARMADAS da Membros Extras na Mochila (p137): a tira "Patas", o "Empunhar na pata" da folha da
// arma leve, e a recusa do gesto. A regra de quem pode e do que cabe é do motor (`CanArmTheLegs`,
// `FitsALeg`); aqui mora só o que a tela escreve.

// legsOfTheSheet diz se a ficha arma as patas e o dano da pata natural, que a posição livre escreve.
// Sem regras carregadas, a resposta é não: a tira some, e o gesto ainda recusa pelo servidor.
func legsOfTheSheet(dto sheet.CharacterDTO) (armable bool, naturalDamage string) {
	if dto.Ruleset == nil {
		return false, ""
	}
	ec, err := sheet.EngineCharacterFrom(dto)
	if err != nil || !dto.Ruleset.CanArmTheLegs(ec) {
		return false, ""
	}
	for _, card := range dto.Ruleset.ComputeWeaponCards(ec, sheet.ToStringSet(dto.Conditionals)) {
		if card.Natural != nil && card.Natural.SourceID == engine.ExtraLimbsPowerID && card.ItemID == 0 {
			naturalDamage = card.Damage
		}
	}
	return true, naturalDamage
}

// legPositionsOf são as patas ocupadas, na ordem, e quantas.
func legPositionsOf(items []sheet.ItemDTO, card func(sheet.ItemDTO) *equippedCard) ([]equippedPosition, int) {
	positions := []equippedPosition{}
	for i, item := range items {
		label := "Pata insetoide"
		if i >= engine.LegsLimit {
			label = "Acima do limite"
		}
		positions = append(positions, equippedPosition{Card: card(item), Label: label})
	}
	return positions, len(items)
}

// legsFreeText é a linha das patas livres: elas continuam atacando como arma natural.
func legsFreeText(used int, naturalDamage string) string {
	free := freePlaces(used, engine.LegsLimit, "pata livre", "patas livres")
	if free == "" {
		return ""
	}
	attacks := " · ataca com a pata insetoide"
	if engine.LegsLimit-used > 1 {
		attacks = " · atacam com as patas insetoides"
	}
	if naturalDamage != "" {
		attacks += ", " + naturalDamage
	}
	return free + attacks
}

// legChoice é o lugar novo na folha da arma: só a arma leve corpo a corpo, e só para quem arma as patas.
func legChoice(armable bool, item sheet.ItemDTO) []equipChoice {
	catalog := catalogItem(item)
	if !armable || catalog == nil || catalog.Weapon == nil || equippedSlotOf(item) == engine.LegSlot ||
		!engine.FitsALeg(catalog.Weapon.Purpose, catalog.Weapon.Hand) {
		return nil
	}
	return []equipChoice{{Slot: engine.LegSlot, Label: "Empunhar na pata"}}
}
