package sheet

import (
	"slices"

	"t20engine/domain/book"
)

// LastLevelLoss é o que o `−` tira: o último nível, a classe dele, e o que foi escolhido nele.
type LastLevelLoss struct {
	Level     int64
	ClassName string
	RemoveIDs []int64
	Removed   []LevelChoiceDTO
	Lines     []string
}

// LastLevelOf calcula a perda de descer um nível. O automático não entra em `RemoveIDs` porque
// ele se refaz sozinho, mas entra em `Lines`, porque também some.
func LastLevelOf(dto CharacterDTO) (LastLevelLoss, bool) {
	var last LevelChoiceDTO
	for _, row := range dto.LevelChoices {
		if row.Kind == LevelKindClass && row.Level > last.Level {
			last = row
		}
	}
	if last.Level == 0 {
		return LastLevelLoss{}, false
	}
	loss := LastLevelLoss{Level: last.Level, ClassName: last.Ref, RemoveIDs: []int64{last.ID}}
	gone := map[int64]bool{}
	for _, row := range dto.LevelChoices {
		if row.Level != last.Level || row.Kind == LevelKindClass {
			continue
		}
		if line, shown := levelLineOf(dto, row); shown {
			text := line.Name
			if row.Kind == LevelKindPick {
				text += ", " + line.Detail
			}
			loss.Lines = append(loss.Lines, text)
		}
		if row.Kind == LevelKindPower && isRepeatablePower(row.Ref) {
			continue
		}
		if row.Kind == LevelKindPower || row.Kind == LevelKindPick {
			loss.RemoveIDs = append(loss.RemoveIDs, row.ID)
			loss.Removed = append(loss.Removed, row)
			gone[row.ID] = true
		}
	}
	for _, row := range excessCountedPicks(dto, last, gone) {
		line, _ := levelLineOf(dto, row)
		loss.Lines = append(loss.Lines, line.Name+", "+line.Detail)
		loss.RemoveIDs = append(loss.RemoveIDs, row.ID)
		loss.Removed = append(loss.Removed, row)
		gone[row.ID] = true
	}
	// O poder repetível sai quando a última escolha dele sai.
	for _, row := range dto.LevelChoices {
		if row.Kind != LevelKindPower || !isRepeatablePower(row.Ref) {
			continue
		}
		left := slices.ContainsFunc(dto.LevelChoices, func(pick LevelChoiceDTO) bool {
			return pick.Kind == LevelKindPick && pick.Ref == row.Ref && !gone[pick.ID]
		})
		if !left {
			loss.RemoveIDs = append(loss.RemoveIDs, row.ID)
		}
	}
	return loss, true
}

// excessCountedPicks são as escolhas contadas que o nível novo já não dá, mesmo situadas mais
// abaixo: a ficha antiga guardou os terrenos todos no nível do poder. As mais antigas ficam.
func excessCountedPicks(dto CharacterDTO, last LevelChoiceDTO, gone map[int64]bool) []LevelChoiceDTO {
	lowered := dto
	lowered.Classes = slices.Clone(dto.Classes)
	for i := range lowered.Classes {
		if lowered.Classes[i].ClassName == last.Ref {
			lowered.Classes[i].Level--
		}
	}
	kept := map[string]int{}
	out := []LevelChoiceDTO{}
	for _, row := range dto.LevelChoices {
		power, known := book.ClassPowers()[row.Ref]
		if row.Kind != LevelKindPick || gone[row.ID] || !known || power.Choice == nil || len(power.Choice.CountLevels) == 0 {
			continue
		}
		limit, level := levelChoiceLimit(lowered, power)
		if kept[row.Ref] >= limit || optionMinLevel(power, row.Pick) > level {
			out = append(out, row)
			continue
		}
		kept[row.Ref]++
	}
	return out
}

func optionMinLevel(power book.ClassPower, pick string) int {
	for _, option := range power.Choice.Options {
		if option.ID == pick {
			return option.MinLevel
		}
	}
	return 0
}

// LoweredBy é a ficha como fica sem o nível: é ela que a conferência da descida olha.
func LoweredBy(dto CharacterDTO, loss LastLevelLoss) CharacterDTO {
	after := []LevelChoiceDTO{}
	for _, row := range dto.LevelChoices {
		if !slices.Contains(loss.RemoveIDs, row.ID) {
			after = append(after, row)
		}
	}
	lowered := WithLevelChoices(dto, after)
	lowered.Classes = slices.Clone(dto.Classes)
	for i := range lowered.Classes {
		if lowered.Classes[i].ClassName == loss.ClassName {
			lowered.Classes[i].Level--
		}
	}
	return lowered
}
