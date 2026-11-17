package sheetui

import (
	"t20engine/domain/book"
	"t20engine/domain/sheet"
)

// A ORIGEM NO ESCOLHER: os benefícios, um interruptor cada.

// originChoiceCard são os benefícios da origem, com o teto de dois.
type originChoiceCard struct {
	Name    string
	Options []filterOption
	Chosen  []string
	// Ceiling é quantos benefícios a origem cobra (`originBenefitCeiling`).
	Ceiling int
	Left    int
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
