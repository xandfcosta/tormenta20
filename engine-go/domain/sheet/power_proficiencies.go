package sheet

import (
	"fmt"
	"slices"

	"t20engine/domain/book"
	"t20engine/domain/engine"
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

// ProficienciesTheChoicesNeed são as proficiências que as escolhas de arma
// pedem — o pré-requisito do Foco em Arma, "proficiência com a arma" (p128) —,
// com o nome do poder que as pede. Elas não se desmarcam na mão enquanto a
// escolha fica, como o treino que o Foco em Perícia pede (ALE-510).
//
//	ProficienciesTheChoicesNeed(dto) // {"armas-exoticas": "Foco em Arma"}
func ProficienciesTheChoicesNeed(dto CharacterDTO) map[string]string {
	out := map[string]string{}
	for id, picks := range PowerPicks(dto.PowerChoices) {
		power, chooses := book.PowerWithChoice(id)
		if !chooses || !power.Choice.RequiresProficiency {
			continue
		}
		for _, pick := range picks {
			if key := weaponProficiency(pick); key != "" {
				out[key] = power.Name
			}
		}
	}
	return out
}

// weaponProficiency é a proficiência que a arma do catálogo pede, pela MESMA
// tabela que decide a penalidade do motor.
func weaponProficiency(itemID string) string {
	item := book.ItemByID(itemID)
	if item == nil {
		return ""
	}
	return engine.RequiredProficiency(&engine.CatalogItem{Category: item.Category})
}

// weaponPicksFit confere o pré-requisito da arma escolhida: "proficiência com
// a arma" (Foco em Arma, p128).
func weaponPicksFit(dto CharacterDTO, power book.ClassPower, picks []string) error {
	has := UnmarshalStrings(dto.Proficiencies)
	for _, pick := range picks {
		key := weaponProficiency(pick)
		if key != "" && !slices.Contains(has, key) {
			return fmt.Errorf("%s pede proficiência com a arma, e esta ficha não sabe usar %s", power.Name, book.ProficiencyLabel(key))
		}
	}
	return nil
}
