package sheet

import (
	"fmt"
	"slices"
	"sort"

	"t20engine/domain/book"
)

// AS ESCOLHAS CONTADAS POR NÍVEL (ALE-503): o Explorador do caçador ganha um
// terreno no 3º nível e mais um a cada quatro (p50), e repetir um terreno é
// escolha válida — soma +2 nele. Não ocupam vaga.
//
// A ficha que sairia da conta — descer o nível com terrenos a mais — é
// RECUSADA, como a do Especialista (decisão do dono).

// levelChoiceLimit é quantas escolhas o nível da classe dá, e o nível.
func levelChoiceLimit(dto CharacterDTO, power book.ClassPower) (int, int) {
	level := 0
	for _, c := range dto.Classes {
		if c.ClassName == power.ClassName {
			level = int(c.Level)
		}
	}
	limit := 0
	for _, at := range power.Choice.CountLevels {
		if level >= at {
			limit++
		}
	}
	return limit, level
}

// LevelCountedChoicesFit confere as escolhas contadas por nível: cabem no que o
// nível dá, e cada opção vale no nível da classe (a área de Tormenta, no 11º).
func LevelCountedChoicesFit(dto CharacterDTO) error {
	picks := PowerPicks(dto.PowerChoices)
	ids := make([]string, 0, len(picks))
	for id := range picks {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		power, known := book.ClassPowers()[id]
		if !known || power.Choice == nil || len(power.Choice.CountLevels) == 0 {
			continue
		}
		limit, level := levelChoiceLimit(dto, power)
		if len(picks[id]) > limit {
			return fmt.Errorf("%s dá %d escolhas no nível %d de %s, e ficariam %d: tire uma do %s antes",
				power.Name, limit, level, power.ClassName, len(picks[id]), power.Name)
		}
		for _, pick := range picks[id] {
			i := slices.IndexFunc(power.Choice.Options, func(o book.PowerChoiceOption) bool { return o.ID == pick })
			if i >= 0 && power.Choice.Options[i].MinLevel > level {
				o := power.Choice.Options[i]
				return fmt.Errorf("%s: %s só a partir do %dº nível de %s, e a ficha está no %dº",
					power.Name, o.Name, o.MinLevel, power.ClassName, level)
			}
		}
	}
	return nil
}
