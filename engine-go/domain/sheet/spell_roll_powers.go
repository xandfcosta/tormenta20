package sheet

import (
	"fmt"
	"slices"

	"t20engine/domain/book"
	"t20engine/domain/engine"
)

// OS PODERES QUE MEXEM NO DADO DA MAGIA e entram como PARCELA PRÓPRIA, com o nome deles (ver GLOSSARY, **dado da
// magia**). Os que se pagam no diálogo de conjurar (o Espírito dos Solstícios, a Comunhão Vital, a Magia Sagrada)
// são aprimoramentos de poder e agem no `Cast`; estes valem sempre, sem PM.

// Os poderes que somam ao dado da magia sem custo.
const (
	BattleArcanePower = "class.arcanista.arcano-de-batalha"
	WinterAspectPower = "class.druida.aspecto-do-inverno"
)

// SpellBonus é uma parcela fixa que um poder soma ao dado da magia: o número, o tipo quando ele é de um, e o que a
// faixa diz além do nome.
type SpellBonus struct {
	Name   string
	Amount int
	Kind   string
	Note   string
}

// SpellRollPowers são os poderes da ficha que somam ao dado das magias dela.
type SpellRollPowers struct {
	// battleArcane é o atributo-chave do arcanista, com o Arcano de Batalha; zero sem ele.
	battleArcane int
	winter       bool
}

// SpellRollPowersOf lê da ficha os poderes que somam ao dado das magias: o Arcano de Batalha, com o atributo-chave
// do arcanista computado, e o Aspecto do Inverno.
//
//	powers := sheet.SpellRollPowersOf(dto, computed)
func SpellRollPowersOf(dto CharacterDTO, computed engine.ComputedSheet) SpellRollPowers {
	owned := OwnedPowers(dto)
	has := func(id string) bool {
		return slices.ContainsFunc(owned, func(p OwnedPower) bool { return p.ID == id })
	}
	powers := SpellRollPowers{winter: has(WinterAspectPower)}
	if has(BattleArcanePower) {
		if attribute, found := computed.SpellAttributeByClass["Arcanista"]; found {
			powers.battleArcane = computed.Attributes[attribute].Total
		}
	}
	return powers
}

// CastBonuses são as parcelas que os poderes somam ao dado da CONJURAÇÃO: o Arcano de Batalha, uma vez, se o dado
// causa dano — "quando lança uma magia, você soma seu atributo-chave na rolagem de dano" (p38) —, e o Aspecto do
// Inverno.
func (p SpellRollPowers) CastBonuses(parts []book.SpellRoll) []SpellBonus {
	var bonuses []SpellBonus
	if p.battleArcane != 0 && slices.ContainsFunc(parts, book.SpellRoll.IsDamage) {
		bonuses = append(bonuses, SpellBonus{Name: "Arcano de Batalha", Amount: p.battleArcane})
	}
	return append(bonuses, p.LaterBonuses(parts)...)
}

// LaterBonuses são as parcelas que os poderes somam ao dado de DEPOIS da conjuração — o dado da vez, a ação da
// magia, a marca na fila: só o Aspecto do Inverno, "suas magias que causam dano de frio causam +1 ponto de dano por
// dado" (p61). O Arcano de Batalha é de quando se lança.
func (p SpellRollPowers) LaterBonuses(parts []book.SpellRoll) []SpellBonus {
	if !p.winter {
		return nil
	}
	frost := 0
	for _, part := range parts {
		if part.Kind != "frio" {
			continue
		}
		if dice, err := engine.ParseDiceSum(part.Dice); err == nil {
			frost += dice.Count
		}
	}
	if frost == 0 {
		return nil
	}
	return []SpellBonus{{Name: "Aspecto do Inverno", Amount: frost, Kind: "frio", Note: fmt.Sprintf("%d dados de frio", frost)}}
}
