package sheet

import (
	"encoding/json"
	"slices"
	"sort"

	"t20engine/domain/book"
	"t20engine/infra/db/sqlcgen"
)

// AS ESCOLHAS POR NÍVEL (ALE-556).
//
// A tabela `character_level_choices` é a única fonte do que o personagem escolheu, e guarda
// em que nível do personagem cada coisa aconteceu. Quem lê a ficha continua recebendo os
// dois textos de sempre, `ClassPowers` e `PowerChoices`, porque o motor e as cenas perguntam
// "o que ele tem?", e essa resposta não depende de quando.

// Os tipos de linha, como a coluna `kind` os grava.
const (
	LevelKindClass     = "class"
	LevelKindPower     = "power"
	LevelKindPick      = "pick"
	LevelKindAutomatic = "automatic"
)

// LevelChoiceDTO é uma linha. Level zero é a que ainda não foi situada.
type LevelChoiceDTO struct {
	ID        int64
	Level     int64
	Kind      string
	Ref       string
	Pick      string
	Estimated bool
}

// LevelChoicesFrom traduz as linhas do banco.
func LevelChoicesFrom(rows []sqlcgen.CharacterLevelChoice) []LevelChoiceDTO {
	out := make([]LevelChoiceDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, levelChoiceFrom(row))
	}
	return out
}

func levelChoiceFrom(row sqlcgen.CharacterLevelChoice) LevelChoiceDTO {
	return LevelChoiceDTO{
		ID: row.ID, Level: row.Level.Int64, Kind: row.Kind, Ref: row.Ref, Pick: row.Pick,
		Estimated: row.Estimated != 0,
	}
}

// WithLevelChoices põe as linhas na ficha e monta os dois textos que todo mundo lê.
func WithLevelChoices(dto CharacterDTO, rows []LevelChoiceDTO) CharacterDTO {
	dto.LevelChoices = rows
	dto.ClassPowers, dto.PowerChoices = choiceBlobsOf(rows)
	return dto
}

// choiceBlobsOf monta os textos na ordem em que as linhas foram gravadas.
func choiceBlobsOf(rows []LevelChoiceDTO) (string, string) {
	powers := []string{}
	picks := map[string][]string{}
	for _, row := range rows {
		switch row.Kind {
		case LevelKindPower:
			powers = append(powers, row.Ref)
		case LevelKindPick:
			picks[row.Ref] = append(picks[row.Ref], row.Pick)
		}
	}
	encoded, err := json.Marshal(powers)
	if err != nil {
		encoded = []byte("[]")
	}
	return string(encoded), marshalPowerPicks(picks)
}

// LevelChoiceDiff é o que muda nas linhas quando uma escrita traz os textos novos.
type LevelChoiceDiff struct {
	RemoveIDs []int64
	Add       []LevelChoiceDTO
}

// DiffChoiceBlobs compara as linhas com os textos que o gesto montou. Quem continua fica com
// o nível dele, e quem entra nasce sem nível, para o plano situá-lo. Texto nulo não muda nada
// daquele tipo.
func DiffChoiceBlobs(rows []LevelChoiceDTO, classPowers, powerChoices *string) LevelChoiceDiff {
	diff := LevelChoiceDiff{}
	if classPowers != nil {
		wanted := map[string]int{}
		for _, id := range UnmarshalStrings(*classPowers) {
			wanted[id]++
		}
		for _, row := range rows {
			if row.Kind != LevelKindPower {
				continue
			}
			if wanted[row.Ref] > 0 {
				wanted[row.Ref]--
				continue
			}
			diff.RemoveIDs = append(diff.RemoveIDs, row.ID)
		}
		for _, id := range UnmarshalStrings(*classPowers) {
			if wanted[id] > 0 {
				wanted[id]--
				diff.Add = append(diff.Add, LevelChoiceDTO{Kind: LevelKindPower, Ref: id})
			}
		}
	}
	if powerChoices != nil {
		picks := PowerPicks(*powerChoices)
		wanted := map[[2]string]int{}
		for id, options := range picks {
			for _, option := range options {
				wanted[[2]string{id, option}]++
			}
		}
		for _, row := range rows {
			if row.Kind != LevelKindPick {
				continue
			}
			key := [2]string{row.Ref, row.Pick}
			if wanted[key] > 0 {
				wanted[key]--
				continue
			}
			diff.RemoveIDs = append(diff.RemoveIDs, row.ID)
		}
		ids := make([]string, 0, len(picks))
		for id := range picks {
			ids = append(ids, id)
		}
		slices.Sort(ids)
		for _, id := range ids {
			for _, option := range picks[id] {
				key := [2]string{id, option}
				if wanted[key] > 0 {
					wanted[key]--
					diff.Add = append(diff.Add, LevelChoiceDTO{Kind: LevelKindPick, Ref: id, Pick: option})
				}
			}
		}
	}
	return diff
}

// PlanLevelRows devolve as linhas como elas têm de ficar: os níveis em sequência, as
// habilidades automáticas, e toda escolha situada num nível.
//
// A conta é idempotente, e por isso serve para tudo: subir, descer, escolher, e preencher as
// fichas que vieram da migração.
//
// As automáticas se refazem a cada vez a partir do catálogo, e é de propósito: se o catálogo
// for corrigido, o motor segue o catálogo, e a linha guardada não pode contar outra história.
//
// Linha com ID zero é nova, e a linha que não voltou é para apagar.
func PlanLevelRows(dto CharacterDTO) []LevelChoiceDTO {
	levels, moved := classLevelsOf(dto)
	out := make([]LevelChoiceDTO, 0, len(dto.LevelChoices)+len(levels))
	out = append(out, levels...)
	pending := []LevelChoiceDTO{}
	for _, row := range dto.LevelChoices {
		if row.Kind != LevelKindPower && row.Kind != LevelKindPick {
			continue
		}
		if row.Level > 0 {
			if level, known := moved[row.Level]; known {
				row.Level = level
			} else {
				row.Level = 0
			}
		}
		pending = append(pending, row)
	}
	automatic := automaticLevelRows(dto, levels)
	out = append(out, automatic...)
	return append(out, situate(pending, levels, automatic)...)
}

// classLevelsOf põe os níveis em sequência: a linha de classe que ainda não tem nível vai para
// o fim, e o buraco de um nível que saiu fecha. Devolve também para onde foi cada nível antigo,
// e o que estava num nível que saiu fica sem nível.
func classLevelsOf(dto CharacterDTO) ([]LevelChoiceDTO, map[int64]int64) {
	rows := []LevelChoiceDTO{}
	for _, row := range dto.LevelChoices {
		if row.Kind == LevelKindClass {
			rows = append(rows, row)
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if (rows[i].Level == 0) != (rows[j].Level == 0) {
			return rows[j].Level == 0
		}
		return rows[i].Level < rows[j].Level
	})
	moved := map[int64]int64{}
	for i := range rows {
		next := int64(i + 1)
		if rows[i].Level > 0 {
			moved[rows[i].Level] = next
		}
		rows[i].Level = next
	}
	return rows, moved
}

// automaticLevelRows são as habilidades automáticas, no nível do personagem em que a classe
// as recebeu.
func automaticLevelRows(dto CharacterDTO, levels []LevelChoiceDTO) []LevelChoiceDTO {
	existing := map[string]LevelChoiceDTO{}
	for _, row := range dto.LevelChoices {
		if row.Kind == LevelKindAutomatic {
			existing[row.Ref] = row
		}
	}
	out := []LevelChoiceDTO{}
	for _, owned := range automaticClassPowersOf(dto) {
		power := book.ClassPowers()[owned.ID]
		classLevel := int64(1)
		if power.GrantedAtLevel != nil {
			classLevel = int64(*power.GrantedAtLevel)
		}
		row := existing[owned.ID]
		row.Kind, row.Ref = LevelKindAutomatic, owned.ID
		row.Level = levelWhereClassReached(levels, owned.Owner, classLevel)
		out = append(out, row)
	}
	return out
}

// levelWhereClassReached é o nível do personagem em que a classe chegou àquele nível dela.
func levelWhereClassReached(levels []LevelChoiceDTO, class string, classLevel int64) int64 {
	var count int64
	for _, row := range levels {
		if row.Ref == class {
			count++
			if count == classLevel {
				return row.Level
			}
		}
	}
	return 1
}

// situate dá nível a quem ainda não tem, na ordem em que foram gravados.
//
// Cada nível de classe a partir do 2º dá uma vaga de poder (p33). A escolha vai para a vaga
// livre mais baixa, e o Aumento de Atributo evita o patamar em que aquele atributo já foi
// escolhido (p38). Sem vaga, ela fica no último nível: a ficha antiga pode ter mais do que
// cabe, e isto só situa; quem recusa é a conferência das escolhas.
func situate(rows []LevelChoiceDTO, levels, automatic []LevelChoiceDTO) []LevelChoiceDTO {
	slots := powerSlotLevels(levels)
	taken := map[int64]bool{}
	tiersWithAttribute := map[string]map[string]bool{}
	note := func(row LevelChoiceDTO) {
		if consumesSlot(row) {
			taken[row.Level] = true
		}
		if isAttributeIncrease(row) {
			if tiersWithAttribute[row.Pick] == nil {
				tiersWithAttribute[row.Pick] = map[string]bool{}
			}
			tier, _ := characterTier(row.Level)
			tiersWithAttribute[row.Pick][tier] = true
		}
	}
	for _, row := range rows {
		if row.Level > 0 {
			note(row)
		}
	}
	last := int64(len(levels))
	if last == 0 {
		last = 1
	}
	freeSlot := func(row LevelChoiceDTO) int64 {
		first := int64(0)
		for _, level := range slots {
			if taken[level] {
				continue
			}
			if first == 0 {
				first = level
			}
			tier, _ := characterTier(level)
			if !isAttributeIncrease(row) || !tiersWithAttribute[row.Pick][tier] {
				return level
			}
		}
		if first != 0 {
			return first
		}
		return last
	}

	// Primeiro quem ocupa vaga, porque os outros seguem o nível dele.
	for i := range rows {
		if rows[i].Level == 0 && consumesSlot(rows[i]) {
			rows[i].Level = freeSlot(rows[i])
			note(rows[i])
		}
	}
	levelOf := map[string]int64{}
	for _, row := range automatic {
		levelOf[row.Ref] = row.Level
	}
	for _, row := range rows {
		if row.Level == 0 || row.Kind == LevelKindPower && !isRepeatablePower(row.Ref) {
			if row.Kind == LevelKindPower && row.Level > 0 {
				levelOf[row.Ref] = row.Level
			}
			continue
		}
		if existing, known := levelOf[row.Ref]; !known || row.Level < existing {
			levelOf[row.Ref] = row.Level
		}
	}
	for i := range rows {
		if rows[i].Level != 0 {
			continue
		}
		rows[i].Level = last
		if level, known := levelOf[rows[i].Ref]; known {
			rows[i].Level = level
		}
	}
	return rows
}

// powerSlotLevels são os níveis do personagem que dão vaga de poder: cada classe, a partir do
// 2º nível dela (p33).
func powerSlotLevels(levels []LevelChoiceDTO) []int64 {
	count := map[string]int{}
	out := []int64{}
	for _, row := range levels {
		count[row.Ref]++
		if count[row.Ref] >= levelWithPowerFirst {
			out = append(out, row.Level)
		}
	}
	return out
}

// consumesSlot diz se a linha ocupa uma vaga. No poder repetível, quem ocupa é cada escolha, e
// a linha do poder segue a primeira delas: é a conta do `UsedPowerSlots`.
func consumesSlot(row LevelChoiceDTO) bool {
	if row.Kind == LevelKindPower {
		return !isRepeatablePower(row.Ref)
	}
	return row.Kind == LevelKindPick && isRepeatablePower(row.Ref)
}

func isRepeatablePower(id string) bool {
	power, chooses := book.PowerWithChoice(id)
	return chooses && power.Choice.Repeatable
}

func isAttributeIncrease(row LevelChoiceDTO) bool {
	if row.Kind != LevelKindPick {
		return false
	}
	power, chooses := book.PowerWithChoice(row.Ref)
	return chooses && power.Name == attributeIncreasePower
}
