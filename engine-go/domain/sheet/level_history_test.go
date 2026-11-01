package sheet

import (
	"slices"
	"testing"
)

const explorerPower = "class.cacador.explorador"

// hunterRows são os níveis de um caçador, já situados.
func hunterRows(level int64) []LevelChoiceDTO {
	rows := []LevelChoiceDTO{}
	for at := int64(1); at <= level; at++ {
		rows = append(rows, LevelChoiceDTO{ID: at, Level: at, Kind: LevelKindClass, Ref: "Caçador"})
	}
	return rows
}

func hunter(level int64, picks ...LevelChoiceDTO) CharacterDTO {
	return WithLevelChoices(CharacterDTO{Classes: []ClassDTO{{ClassName: "Caçador", Level: level}}},
		append(hunterRows(level), picks...))
}

// O segundo terreno do Explorador é do 7º nível do caçador (p50), mesmo escolhido depois.
func TestACountedPickIsSituatedAtTheLevelThatGaveIt(t *testing.T) {
	dto := hunter(7,
		LevelChoiceDTO{ID: 100, Kind: LevelKindPick, Ref: explorerPower, Pick: "floresta"},
		LevelChoiceDTO{ID: 101, Kind: LevelKindPick, Ref: explorerPower, Pick: "urbano"},
	)
	levels := []int64{}
	for _, row := range PlanLevelRows(dto) {
		if row.Kind == LevelKindPick {
			levels = append(levels, row.Level)
		}
	}
	if !slices.Equal(levels, []int64{3, 7}) {
		t.Errorf("os terrenos ficaram nos níveis %v, e o livro os dá no 3º e no 7º", levels)
	}
}
