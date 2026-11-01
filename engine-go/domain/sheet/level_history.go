package sheet

import (
	"slices"

	"t20engine/domain/book"
)

// LevelEntry é um nível do personagem, como a seção "Por nível" o mostra.
type LevelEntry struct {
	Level      int64
	ClassName  string
	ClassLevel int64
	Tier       string
	Lines      []LevelLine
	// Pending diz que este nível dá uma vaga de poder e ela ainda está livre.
	Pending bool
}

// LevelLine é o que aconteceu naquele nível. Só a escolha tem RowID, e é o que se troca.
type LevelLine struct {
	RowID     int64
	Name      string
	Detail    string
	Automatic bool
	Estimated bool
}

// LevelHistory são os níveis do mais recente para o mais antigo, porque quem sobe de nível
// quer ver primeiro o que acabou de ganhar.
func LevelHistory(dto CharacterDTO) []LevelEntry {
	classLevel := map[string]int64{}
	entries := []LevelEntry{}
	levelRows := map[int64]LevelChoiceDTO{}
	classRows := []LevelChoiceDTO{}
	for _, row := range dto.LevelChoices {
		if row.Kind == LevelKindClass {
			levelRows[row.Level] = row
			classRows = append(classRows, row)
		}
	}
	slices.SortFunc(classRows, func(a, b LevelChoiceDTO) int { return int(a.Level - b.Level) })
	taken := map[int64]bool{}
	for _, row := range dto.LevelChoices {
		if consumesSlot(row) {
			taken[row.Level] = true
		}
	}
	for _, row := range classRows {
		classLevel[row.Ref]++
		tier, _ := characterTier(row.Level)
		entry := LevelEntry{
			Level: row.Level, ClassName: row.Ref, ClassLevel: classLevel[row.Ref], Tier: tier,
			Pending: classLevel[row.Ref] >= levelWithPowerFirst && !taken[row.Level],
		}
		for _, choice := range dto.LevelChoices {
			if choice.Level == row.Level {
				if line, shown := levelLineOf(dto, choice); shown {
					entry.Lines = append(entry.Lines, line)
				}
			}
		}
		entries = append(entries, entry)
	}
	slices.Reverse(entries)
	return entries
}

// levelLineOf escreve a linha. A do poder que pede escolha não aparece quando há escolha: quem
// aparece, e se troca, é a escolha.
func levelLineOf(dto CharacterDTO, row LevelChoiceDTO) (LevelLine, bool) {
	name, description, options := powerShownAs(row.Ref)
	switch row.Kind {
	case LevelKindAutomatic:
		return LevelLine{Name: name, Detail: description, Automatic: true}, true
	case LevelKindPick:
		label := row.Pick
		for _, option := range options {
			if option.ID == row.Pick {
				label = option.Name
			}
		}
		return LevelLine{RowID: row.ID, Name: name, Detail: label, Estimated: row.Estimated}, true
	case LevelKindPower:
		if len(PowerPicks(dto.PowerChoices)[row.Ref]) > 0 {
			return LevelLine{}, false
		}
		return LevelLine{RowID: row.ID, Name: name, Detail: description, Estimated: row.Estimated}, true
	}
	return LevelLine{}, false
}

func powerShownAs(id string) (string, string, []book.PowerChoiceOption) {
	if power, found := book.ClassPowers()[id]; found {
		if power.Choice != nil {
			return power.Name, power.Description, power.Choice.Offered()
		}
		return power.Name, power.Description, nil
	}
	if general, found := book.GeneralPowers()[id]; found {
		if general.Choice != nil {
			return general.Name, general.Description, general.Choice.Offered()
		}
		return general.Name, general.Description, nil
	}
	return id, "", nil
}
