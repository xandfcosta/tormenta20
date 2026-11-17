package sheetui

import (
	"t20engine/domain/book"
	"t20engine/domain/sheet"
)

// A ORIGEM NO ESCOLHER: os benefícios, um interruptor cada, e o poder que um benefício
// marcado deixa escolher ("um poder de combate a sua escolha", p91).

// originChoiceCard são os benefícios da origem, com o teto de dois.
type originChoiceCard struct {
	Name    string
	Options []filterOption
	Chosen  []string
	// Ceiling é quantos benefícios a origem cobra (`originBenefitCeiling`).
	Ceiling int
	Left    int
}

// originPickChoice é o poder que um benefício marcado da origem deixa escolher: os poderes
// gerais do tipo dele, e o escolhido.
type originPickChoice struct {
	BenefitID string
	Name      string
	// Kind é o tipo de poder que o benefício dá (`combate`, `tormenta`): o rótulo da lista.
	Kind   string
	Picker pickerChoice
}

// originChoice monta o cartão da origem.
func originChoice(dto sheet.CharacterDTO) *originChoiceCard {
	origin, found := book.Origins()[dto.Origin]
	if !found {
		return nil
	}
	chosen := sheet.UnmarshalStrings(dto.OriginChoices)
	ceiling := originBenefitCeiling(origin)
	card := &originChoiceCard{
		Name: origin.Name, Chosen: chosen,
		Ceiling: ceiling, Left: ceiling - len(chosen),
	}
	for _, b := range sheet.OriginBenefitsOf(origin) {
		card.Options = append(card.Options, filterOption{
			Value: b.ID, Label: b.Name, Active: contemTraco(chosen, b.ID),
		})
	}
	return card
}

// originPicksOf monta um seletor por benefício marcado que deixa escolher um poder, com os
// poderes gerais do tipo dele.
func originPicksOf(dto sheet.CharacterDTO) []originPickChoice {
	var out []originPickChoice
	for _, pick := range sheet.OriginPowerPicks(dto) {
		choice := originPickChoice{
			BenefitID: pick.Benefit.ID, Name: pick.Benefit.Name, Kind: pick.Benefit.PowerPick,
			Picker: pickerChoice{Chosen: pick.Chosen},
		}
		for _, power := range sheet.GeneralPowersOfKind(pick.Benefit.PowerPick) {
			choice.Picker.Options = append(choice.Picker.Options, filterOption{
				Value: power.ID, Label: power.Name, Active: power.ID == pick.Chosen,
			})
		}
		out = append(out, choice)
	}
	return out
}
