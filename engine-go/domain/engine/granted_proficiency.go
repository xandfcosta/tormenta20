package engine

import "slices"

// A PROFICIÊNCIA QUE VEM DE PODER E DE HABILIDADE DE RAÇA (ALE-649).
//
// Ela é CALCULADA pelo que a ficha tem, e não gravada na coluna: entra com o poder e sai com ele, sem passar
// pela escolha nem pela troca de nível. E nem toda é uma categoria inteira — o Armeiro é "armas marciais
// corpo a corpo" (p68), e o anão sabe usar o machado de batalha sem saber as marciais (p20) —, e a coluna só
// guarda categoria. A coluna continua sendo da classe, da raça na criação e do ajuste à mão.

// ProficiencyReach é o que uma concessão alcança: uma categoria inteira (`armas-de-fogo`), uma categoria num
// propósito (`armas-marciais` corpo a corpo, "melee", ou à distância, "ranged", que é o disparo e o
// arremesso, p142), ou armas nomeadas (o item específico delas também, pela arma base). Label é como a ficha
// a diz quando ela não é a categoria inteira ("as corpo a corpo").
type ProficiencyReach struct {
	Category string   `json:"category,omitempty"`
	Purpose  string   `json:"purpose,omitempty"`
	Weapons  []string `json:"weapons,omitempty"`
	Label    string   `json:"label,omitempty"`
}

// Whole diz se a concessão é a categoria inteira, que o painel mostra como marcada.
func (g ProficiencyReach) Whole() bool { return g.Purpose == "" && len(g.Weapons) == 0 }

// GrantedProficiency é uma concessão e quem a dá, pelo nome que a ficha mostra.
type GrantedProficiency struct {
	Source string
	ProficiencyReach
}

// Proficiency é tudo o que diz se a pessoa sabe usar um item: a coluna gravada e as concessões.
type Proficiency struct {
	Saved   map[string]bool
	Granted []GrantedProficiency
}

// reaches diz se a concessão alcança o item, que pede a categoria `required`.
func (g ProficiencyReach) reaches(item *CatalogItem, required string) bool {
	if len(g.Weapons) > 0 {
		base := ""
		if item.ComposedOf != nil {
			base = item.ComposedOf.Base
		}
		return slices.Contains(g.Weapons, item.ID) || (base != "" && slices.Contains(g.Weapons, base))
	}
	if g.Category != required {
		return false
	}
	switch g.Purpose {
	case "melee":
		return item.Weapon != nil && item.Weapon.Purpose == "melee"
	case "ranged":
		return item.Weapon != nil && (item.Weapon.Purpose == "ranged" || item.Weapon.Purpose == "thrown")
	}
	return true
}

// ProficiencyOf é a proficiência da ficha: a coluna e o que os poderes de classe e as habilidades de raça
// concedem, inclusive a habilidade tomada pela Memória Póstuma (p29).
//
//	world.ProficiencyOf(armeiro).Holds(espadaLonga, &wielded) // true
func (r *Ruleset) ProficiencyOf(ch Character) Proficiency {
	out := Proficiency{Saved: parseProficiencySet(ch.Proficiencies)}
	chosen := parseChoiceSet(ch.ClassPowers)
	choices := parseClassChoices(ch.ClassChoices)
	for _, entry := range ch.Classes {
		for _, power := range r.ownedClassPowers(entry.ClassName, entry.Level, chosen.has, choices[entry.ClassName]) {
			for _, reach := range power.ProficientWith {
				out.Granted = append(out.Granted, GrantedProficiency{Source: power.Name, ProficiencyReach: reach})
			}
		}
	}
	for _, applied := range r.appliedRaces(ch) {
		abilities := slices.Clone(applied.race.Abilities)
		for _, borrowed := range r.borrowedOf(applied.race, applied.choice.abilityPicks) {
			abilities = append(abilities, borrowed.ability)
		}
		for _, ability := range abilities {
			for _, reach := range ability.ProficientWith {
				out.Granted = append(out.Granted, GrantedProficiency{Source: ability.Name, ProficiencyReach: reach})
			}
		}
	}
	return out
}
