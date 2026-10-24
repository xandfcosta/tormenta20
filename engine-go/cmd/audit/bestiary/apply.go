package main

import (
	"fmt"
	"regexp"
	"strings"
	"t20engine/infra/bookpdf"
	"unicode"
	"unicode/utf8"

	"t20engine/domain/catalog/transcript"
)

// applyCorrections grava as correções e os campos lidos no catálogo, e
// reescreve os arquivos de dados em `dir` na forma canônica.
func applyCorrections(fixes []correction, r auditResult, dir string) error {
	byName := map[string]*transcript.Creature{}
	for i := range transcript.Bestiary {
		byName[transcript.Bestiary[i].Name] = &transcript.Bestiary[i]
	}
	for _, c := range fixes {
		if err := setNumericField(byName[c.name], c.field, c.to); err != nil {
			return err
		}
	}
	truncated := map[string]bool{}
	for _, name := range r.truncated {
		truncated[name] = true
	}
	for _, t := range r.trusted {
		target := byName[t.creature.Name]
		fill(target, t.read, !truncated[t.creature.Name])
		target.SpecialAbilities = withoutFieldAbilities(target.SpecialAbilities)
		// O `treasureXp` que o script de origem apagava não existe no tipo:
		// era `nd * 1000` em todos os 80, a mesma conta do XP por ND.
	}
	return transcript.WriteSource(dir)
}

// fill grava os campos que o verbete passa a ter. `skills` e `equipamento`
// entram SEMPRE, vazios quando o livro não dá — o bloco foi lido inteiro e não
// tinha. `pm` só existe em conjurador: um zero diria "tem mana e está sem".
//
// O tesouro de um bloco TRUNCADO não se grava: o do Reishid é o valor mais a
// descrição da adaga, e sobrescrever trocaria um texto conferido por um pela
// metade a cada execução.
func fill(c *transcript.Creature, read readBlock, withTreasure bool) {
	c.Skills = []transcript.CreatureSkill{}
	if read.hasSkills {
		c.Skills = read.skills
	}
	c.Equipment = read.equipment
	if read.has("iniciativa") {
		c.Initiative = *read.number("iniciativa")
	}
	if read.has("percepcao") {
		c.Perception = *read.number("percepcao")
	}
	if read.hasTreasure && withTreasure {
		c.Treasure = read.treasure
	}
	if read.has("pm") {
		c.PM = *read.number("pm")
	}
}

// setNumericField grava um campo conferido. O `null` do livro ("—") só cabe em
// Força e Inteligência; nos outros o tipo não o representa, e isso é erro e
// não silêncio.
func setNumericField(c *transcript.Creature, field string, v *int) error {
	switch field {
	case "forca":
		c.Strength = v
		return nil
	case "inteligencia":
		c.Intelligence = v
		return nil
	}
	if v == nil {
		return fmt.Errorf("%s: o livro diz \"—\" em %s, e o catálogo só aceita nulo em forca e inteligencia", c.Name, field)
	}
	*numericField(c, field) = *v
	return nil
}

var (
	looseSkills = regexp.MustCompile(`^Perícias`)
	loosePM     = regexp.MustCompile(`^PM` + bookpdf.PySpace + `+\d+` + bookpdf.PySpace + `*[;.]` + bookpdf.PySpace + `*`)
)

// withoutFieldAbilities tira das habilidades o que virou CAMPO: a frase
// "Perícias: Furtividade +5." (agora é `skills`, e duas cópias divergiriam) e o
// `PM 20` grudado noutra coisa ("PM 20; Medo de Altura." no Centauro Xamã).
func withoutFieldAbilities(abilities []string) []string {
	out := []string{}
	for _, h := range abilities {
		if looseSkills.MatchString(h) && !wordAt(h, len("Perícias")) {
			continue
		}
		rest := strings.TrimFunc(loosePM.ReplaceAllString(h, ""), bookpdf.IsPySpace)
		if rest == "" {
			continue
		}
		first, size := utf8.DecodeRuneInString(rest)
		out = append(out, string(unicode.ToUpper(first))+rest[size:])
	}
	return out
}
