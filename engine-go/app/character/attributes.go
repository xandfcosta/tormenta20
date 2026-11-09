package character

import (
	"context"
	"fmt"
	"slices"

	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// SetAttribute grava o valor livre de um atributo.
//
// A escolha é livre (p17), e a única recusa é a Inteligência que ficaria abaixo das perícias escolhidas no
// Especialista (p73). A Constituição mexe no PV máximo: o máximo é derivado e o atual é `máximo − dano`,
// então o teto se move sozinho, e quem chega a 0 ou menos cai (p236) pela mesma regra da queda de nível
// (`sheet.FollowMaximumChange`).
func (p Plays) SetAttribute(ctx context.Context, row sqlcgen.Character, key string, value int) error {
	if !slices.Contains(engine.AttributeKeys, key) {
		return fmt.Errorf("atributo %q não existe: são os seis do livro", key)
	}
	spread := map[string]int64{
		"strength": row.Strength, "dexterity": row.Dexterity, "constitution": row.Constitution,
		"intelligence": row.Intelligence, "wisdom": row.Wisdom, "charisma": row.Charisma,
	}
	spread[key] = int64(value)
	dto, err := sheet.Load(ctx, p.queries, p.catalogs, row)
	if err != nil {
		return err
	}
	withSpread := func(d *sheet.CharacterDTO) {
		d.Strength, d.Dexterity, d.Constitution = spread["strength"], spread["dexterity"], spread["constitution"]
		d.Intelligence, d.Wisdom, d.Charisma = spread["intelligence"], spread["wisdom"], spread["charisma"]
	}
	if sheet.HasExpertiseChoice(dto.PowerChoices) || sheet.HasRaceAbilityPicks(row.Raceattributechoices, row.Secondaryracechoices) {
		if refused := sheet.ChangeKeepsChosenExpertises(dto, withSpread); refused != nil {
			return refused
		}
	}
	if err := p.queries.SetCharacterAttributes(ctx, sqlcgen.SetCharacterAttributesParams{
		Strength: spread["strength"], Dexterity: spread["dexterity"], Constitution: spread["constitution"],
		Intelligence: spread["intelligence"], Wisdom: spread["wisdom"], Charisma: spread["charisma"],
		UpdatedAt: dbvalue.NowISO(), ID: row.ID,
	}); err != nil {
		return fmt.Errorf("gravar os atributos da ficha %d: %w", row.ID, err)
	}
	projected := dto
	withSpread(&projected)
	return sheet.FollowMaximumChange(ctx, p.queries, dto, projected)
}
