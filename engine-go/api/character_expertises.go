package api

import (
	"context"
	"fmt"

	"t20engine/db/sqlcgen"
	"t20engine/sheet"
)

// expertiseNames mirrors t20-data EXPERTISE_NAMES — the builtin perícias. A
// custom expertise may not reuse one, and only these can be edited via PATCH.
var expertiseNames = sheet.ToStringSet([]string{
	"Acrobacia", "Adestramento", "Atletismo", "Atuação", "Cavalgar", "Conhecimento",
	"Cura", "Diplomacia", "Enganação", "Fortitude", "Furtividade", "Guerra",
	"Iniciativa", "Intimidação", "Intuição", "Investigação", "Jogatina", "Ladinagem",
	"Luta", "Misticismo", "Nobreza", "Ofício", "Percepção", "Pilotagem", "Pontaria",
	"Reflexos", "Religião", "Sobrevivência", "Vontade",
})

// saveNewCraft é a regra de quem pode virar ofício, e ela é a MESMA para a
// API JSON e para a ficha em Datastar (ALE-272).
//
// Três recusas, e a do meio é a que importa: um ofício não pode ROUBAR o nome de
// uma das 29 do livro, porque a ficha passaria a ter duas linhas com o mesmo
// nome e a decomposição de uma cairia sobre a outra.
func (sr sheetRules) saveNewCraft(ctx context.Context, characterID int64, nome string) error {
	if nome == "" {
		return fmt.Errorf("dê um nome ao ofício")
	}
	if expertiseNames[nome] {
		return fmt.Errorf("%q é uma perícia do livro — escolha outro nome", nome)
	}
	_, err := sr.queries.GetExpertiseMeta(ctx, sqlcgen.GetExpertiseMetaParams{
		Characterid: characterID, Name: nome,
	})
	if err == nil {
		return fmt.Errorf("esta ficha já tem %q", nome)
	}
	return nil
}

func expertiseDTO(name, attribute string, trained, custom int64) sheet.ExpertiseDTO {
	return sheet.ExpertiseDTO{Name: name, Attribute: attribute, Trained: trained != 0, Custom: custom != 0}
}
