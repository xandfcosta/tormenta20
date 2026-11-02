package forge

import (
	"context"
	"database/sql"
	"fmt"
	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
)

// OS ATRIBUTOS DA FORJA. Eles moram no formulário como o resto da folha, e o total vem da mesma conta da
// ficha, feita sobre um herói que ainda não existe: é isso que faz a raça aparecer na linha antes de
// forjar, sem uma segunda cópia do que cada raça soma.
//
// O campo é texto e não `type="number"`: o `@post` com `contentType: 'form'` roda `checkValidity()`, e um
// número fora de um `min`/`max` não mandaria o pedido nem diria por quê. Quem recusa é o servidor.

// attributeRow é uma das seis linhas.
type attributeRow struct {
	Key   string
	Label string
	// Written é o que está no campo, que pode ser o que a recusa cita.
	Written string
	Total   int
	// Sum é de onde vem o total, em palavras; vazio quando nada soma ao valor escrito.
	Sum string
	// Warning é o efeito que o livro prevê abaixo de −5.
	Warning string
}

// attributeValues lê os seis campos, e o que não é valor vira recusa com o nome do atributo.
func (a forgeAnswers) attributeValues() (map[string]int, []string) {
	values := map[string]int{}
	var refusals []string
	for _, attribute := range book.AttributeOrder {
		value, err := sheet.AttributeValueRead(a.AttributesWritten[attribute.Key])
		if err != nil {
			refusals = append(refusals, book.AttributeLabel(attribute.Key)+": "+err.Error()+".")
			continue
		}
		values[attribute.Key] = value
	}
	return values, refusals
}

// attributeRowsOf monta as seis linhas, com o total que o motor dá para estas respostas.
func (s Scene) attributeRowsOf(ctx context.Context, answers forgeAnswers) ([]attributeRow, []string, error) {
	values, refusals := answers.attributeValues()
	dto := sheet.CharacterDTO{
		Strength: int64(values["strength"]), Dexterity: int64(values["dexterity"]),
		Constitution: int64(values["constitution"]), Intelligence: int64(values["intelligence"]),
		Wisdom: int64(values["wisdom"]), Charisma: int64(values["charisma"]),
		Level: totalLevelOf(answers.Classes), Classes: classesOnTheSheet(answers.Classes),
		Proficiencies: "[]", RaceAttributeChoices: "{}", SecondaryRaceChoices: "[]", OriginChoices: "[]",
		ClassPowers: "[]", ClassChoices: "{}", PowerChoices: "{}", ActiveConditions: "[]",
	}
	if race := raceBySlug(answers.Race); race != nil {
		dto.Races = []sheet.RaceDTO{{Race: race.ID}}
	}
	ruleset, err := sheet.RulesetFor(ctx, s.deps.Queries(), s.deps.Catalogs(), sql.NullInt64{})
	if err != nil {
		return nil, nil, fmt.Errorf("o mundo da forja: %w", err)
	}
	dto.Ruleset = ruleset
	computed, err := sheet.Compute(dto)
	if err != nil {
		return nil, nil, fmt.Errorf("a conta da forja: %w", err)
	}
	rows := make([]attributeRow, 0, len(book.AttributeOrder))
	for _, attribute := range book.AttributeOrder {
		key, total := attribute.Key, computed.Attributes[attribute.Key]
		written, typed := answers.AttributesWritten[key]
		if !typed || written == "" {
			written = "0"
		}
		rows = append(rows, attributeRow{
			Key: key, Label: book.AttributeLabel(key), Written: written, Total: total.Total,
			Sum:     sumWhenThereIsOne(values[key], total.Contributions),
			Warning: sheet.AttributeMinimumEffect(key, total.Total),
		})
	}
	return rows, refusals, nil
}

// sumWhenThereIsOne é vazia quando nada soma ao valor livre: ele já é o total, e a linha só
// repetiria o número.
func sumWhenThereIsOne(base int, contributions []engine.SourceAmount) string {
	if len(contributions) == 0 {
		return ""
	}
	return sheet.AttributeSum(base, contributions)
}

// attributeKnown diz se a chave é um dos seis.
func attributeKnown(key string) bool {
	for _, attribute := range book.AttributeOrder {
		if attribute.Key == key {
			return true
		}
	}
	return false
}
