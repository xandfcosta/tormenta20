package sheet

import (
	"slices"

	"t20engine/domain/book"
)

// A HABILIDADE EMPRESTADA (ALE-504): "você pode ser um osteon de outra raça
// humanoide que não humano. Neste caso, você ganha uma habilidade dessa raça a
// sua escolha" (Memória Póstuma, p29).
//
// A Memória guarda o id da habilidade emprestada; a raça sai dele. Quando a
// emprestada pede escolha própria — o elemento da Resistência Elemental, o
// Ofício do Vanguardista —, ela é guardada no MESMO blob, pela chave da
// emprestada, como se a habilidade fosse do osteon.

// variantChoice é a escolha sintetizada da habilidade de VARIANTE (a
// Resistência Elemental): uma opção, entre as variantes dela.
const variantChoice = "variant"

// borrowableAbility acha a habilidade que a Memória pode emprestar: de raça
// humanoide, que não humano. Devolve a raça junto.
func borrowableAbility(id string) (book.RaceAbility, book.RaceForScreen, bool) {
	for key, race := range book.RaceTraitsByKey() {
		if key != race.ID || race.Type != "humanoide" || race.ID == "humano" {
			continue
		}
		for _, ability := range race.Abilities {
			if ability.ID == id {
				return ability, race, true
			}
		}
	}
	return book.RaceAbility{}, book.RaceForScreen{}, false
}

// BorrowableRaces são as raças que a Memória pode imitar, em ordem de nome.
func BorrowableRaces() []book.RaceForScreen {
	out := []book.RaceForScreen{}
	for key, race := range book.RaceTraitsByKey() {
		if key == race.ID && race.Type == "humanoide" && race.ID != "humano" {
			out = append(out, race)
		}
	}
	slices.SortFunc(out, func(a, b book.RaceForScreen) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})
	return out
}

// borrowedAbilities são as habilidades que a ficha tomou de outra raça, pelo
// id, com a raça que as tomou (a do osteon).
func borrowedAbilities(dto CharacterDTO) map[string]borrowed {
	out := map[string]borrowed{}
	picks := RaceAbilityPicks(dto)
	for _, r := range dto.Races {
		traits, found := book.RaceTraitsByKey()[r.Race]
		if !found {
			continue
		}
		for _, ability := range traits.Abilities {
			if ability.Choice == nil || ability.Choice.Kind != trainedOrPower {
				continue
			}
			for _, pick := range picks[ability.ID] {
				if b, origin, ok := borrowableAbility(pick); ok {
					out[pick] = borrowed{ability: withChoice(b), by: r.Race, from: origin}
				}
			}
		}
	}
	return out
}

type borrowed struct {
	ability book.RaceAbility
	by      string
	from    book.RaceForScreen
}

// withChoice dá à habilidade de VARIANTE uma escolha de uma opção, para ela
// passar pelo mesmo caminho das outras.
func withChoice(ability book.RaceAbility) book.RaceAbility {
	if ability.Choice == nil && len(ability.Variants) > 0 {
		ability.Choice = &book.PowerChoice{Kind: variantChoice, Count: 1}
	}
	return ability
}
