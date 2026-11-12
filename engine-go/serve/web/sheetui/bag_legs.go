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
//
// A ARMA SEM EFEITO continua na tira, porque esconder o item seria pior, mas a linha diz por quê e não
// promete o dano: o motor já a ignora, e a pata ataca como arma natural.
func legPositionsOf(items []sheet.ItemDTO, card func(sheet.ItemDTO) *equippedCard, armable bool) ([]equippedPosition, int) {
	positions := []equippedPosition{}
	for i, item := range items {
		label := "Pata insetoide"
		if i >= engine.LegsLimit {
			label = "Acima do limite"
		}
		shown := card(item)
		if why := whyTheLegIsIdle(armable, item); why != "" {
			idle := *shown
			idle.Chips = nil
			shown, label = &idle, label+" · sem efeito: "+why
		}
		positions = append(positions, equippedPosition{Card: shown, Label: label})
	}
	return positions, len(items)
}

// whyTheLegIsIdle é o motivo de a arma na pata não valer, ou vazio quando ela vale.
func whyTheLegIsIdle(armable bool, item sheet.ItemDTO) string {
	if !armable {
		return "falta a Ambidestria ou o Estilo de Duas Armas"
	}
	catalog := catalogItem(item)
	if catalog == nil || catalog.Weapon == nil || !engine.FitsALeg(catalog.Weapon.Purpose, catalog.Weapon.Hand) {
		return "a pata empunha só arma leve"
	}
	return ""
}

// keptByTheWildShape é a tira na Forma Selvagem: "na forma selvagem você não pode […] empunhar itens"
// (p63). O item continua na posição dele, dizendo que a forma o guardou, e sem o chip de dano — a carta
// dele já não ataca. As posições livres saem: não há o que empunhar. O esotérico com a Magia Natural
// (p62) fica como está, porque continua na mão (`held`).
func keptByTheWildShape(positions []equippedPosition, held map[int64]bool) []equippedPosition {
	kept := []equippedPosition{}
	for _, p := range positions {
		if p.Card == nil {
			continue
		}
		if held[p.Card.ID] {
			kept = append(kept, p)
			continue
		}
		idle := *p.Card
		idle.Chips = nil
		kept = append(kept, equippedPosition{Card: &idle, Label: p.Label + " · guardada pela Forma Selvagem", Wide: p.Wide})
	}
	return kept
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

// heldInTheShape são as linhas da Mochila que continuam na mão da forma: os esotéricos, com a Magia
// Natural (p62).
func heldInTheShape(dto sheet.CharacterDTO) map[int64]bool {
	held := map[int64]bool{}
	ec, err := sheet.EngineCharacterFrom(dto)
	if err != nil || dto.Ruleset == nil {
		return held
	}
	for _, it := range ec.Items {
		if dto.Ruleset.HeldInTheShape(ec, it) {
			held[it.ID] = true
		}
	}
	return held
}
