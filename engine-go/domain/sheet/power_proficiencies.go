package sheet

import (
	"fmt"
	"slices"

	"t20engine/domain/book"
)

// AS PROFICIÊNCIAS DO PODER PROFICIÊNCIA (p129, ALE-510): "Escolha uma
// proficiência: armas marciais, armas de fogo, armaduras pesadas ou escudos (se
// for proficiente em armas marciais, você também pode escolher armas
// exóticas). Você recebe essa proficiência."
//
// A escolha mora em `powerChoices`, e a proficiência é gravada na coluna ao ser
// escolhida, como a da classe na criação. O poder é uma ORIGEM dela no painel.

// proficiencyPower é o id do poder geral.
const proficiencyPower = "proficiencia"

// PowerProficiencies são as categorias que as escolhas do poder dão, na ordem do
// livro. A armadura pesada traz a leve, pela mesma decisão de produto que vale
// para a classe (ver o `book.GrantedProficiencies`).
//
//	PowerProficiencies(`{"proficiencia":["armaduras-pesadas"]}`) // [armaduras-leves armaduras-pesadas]
func PowerProficiencies(blob string) []string {
	picks := PowerPicks(blob)[proficiencyPower]
	granted := map[string]bool{}
	for _, pick := range picks {
		granted[pick] = true
		if pick == "armaduras-pesadas" {
			granted["armaduras-leves"] = true
		}
	}
	out := []string{}
	for _, cat := range book.ProficiencyCategories {
		if granted[cat.Key] {
			out = append(out, cat.Key)
		}
	}
	return out
}

// proficiencyPicksFit confere as escolhas do poder: cada vez uma proficiência
// diferente, e as exóticas só para quem sabe usar marciais.
func proficiencyPicksFit(dto CharacterDTO, power book.ClassPower, picks []string) error {
	for i, pick := range picks {
		if slices.Contains(picks[:i], pick) {
			return fmt.Errorf("%s já escolheu %s; cada vez é uma proficiência diferente", power.Name, book.ProficiencyLabel(pick))
		}
		if pick == "armas-exoticas" && !slices.Contains(UnmarshalStrings(dto.Proficiencies), "armas-marciais") {
			return fmt.Errorf("%s só dá armas exóticas a quem sabe usar armas marciais (p129)", power.Name)
		}
	}
	return nil
}
