package main

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"t20engine/cmd/internal/pyfmt"
	"unicode/utf8"

	"t20engine/domain/catalog/transcript"
)

// pageEntry é um verbete "Nome (Abrev). Definição." de uma página só.
type pageEntry struct{ name, abbreviation, text string }

var (
	blankLine = regexp.MustCompile(`\n` + pySpace + `*\n`)
)

// entriesOfPage corta uma página impressa em verbetes "Nome (Abrev). Definição.".
//
// Os TRÊS cortes, cada um de um defeito medido na tela:
//
//   - no NOME seguinte, que é o corte óbvio;
//   - na LINHA EM BRANCO, porque depois do último verbete vem a mobília da
//     página — em p172, uma citação decorativa que saiu colada na Transmutação;
//   - no RODAPÉ, porque o texto de uma página de duas colunas continua na seção
//     vizinha depois do número.
//
// E o HÍFEN de quebra é emendado antes de tudo: sem isso "convoca-\nções" vira
// "convoca- ções" na tela.
func (b *book) entriesOfPage(printed int, names []string) ([]pageEntry, error) {
	joined := hyphenBreak.ReplaceAllString(b.rawPage(printed), "")

	type mark struct {
		start, end        int
		name, parenthesis string
	}
	var marks []mark
	for _, name := range names {
		// O nome aparece duas vezes: no texto de abertura e abrindo o verbete. O
		// verbete é o que vem seguido de "(" ou de ".".
		m := regexp.MustCompile(regexp.QuoteMeta(name) + pySpace + `*(\([^)]*\))?\.`).FindStringSubmatchIndex(joined)
		if m == nil {
			return nil, fatal{fmt.Sprintf("não achei %s na p%d", pyfmt.Repr(name), printed)}
		}
		parenthesis := ""
		if m[2] >= 0 {
			parenthesis = joined[m[2]:m[3]]
		}
		marks = append(marks, mark{m[0], m[1], name, parenthesis})
	}
	sort.Slice(marks, func(i, j int) bool { return marks[i].start < marks[j].start })

	footer := regexp.MustCompile(pySpace + strconv.Itoa(printed) + pySpace)
	var out []pageEntry
	for i, m := range marks {
		until := len(joined)
		if i+1 < len(marks) {
			until = marks[i+1].start
		}
		// Linha em branco e rodapé, nesta ordem: o que vier depois de qualquer
		// um dos dois não é definição.
		raw := beforeFirst(footer, beforeFirst(blankLine, joined[m.end:until]))
		text := pyStrip(anySpaces.ReplaceAllString(raw, " "))
		text = pyStrip(trailingNumber.ReplaceAllString(text, ""))
		if n := utf8.RuneCountInString(text); n > 700 {
			return nil, fatal{fmt.Sprintf("a definição de %s veio com %d caracteres — o corte falhou", pyfmt.Repr(m.name), n)}
		}
		if text == "" {
			return nil, fatal{fmt.Sprintf("a definição de %s veio vazia — o corte comeu tudo", pyfmt.Repr(m.name))}
		}
		abbreviation := ""
		if m.parenthesis != "" {
			abbreviation = m.parenthesis[1 : len(m.parenthesis)-1]
		}
		out = append(out, pageEntry{m.name, abbreviation, text})
	}
	return out, nil
}

const schoolsPage = 172

// schoolNames na ordem em que o livro as imprime; como nos tipos de efeito, a
// lista existe para o extrator saber onde CORTAR.
var schoolNames = []string{
	"Abjuração", "Adivinhação", "Convocação", "Encantamento",
	"Evocação", "Ilusão", "Necromancia", "Transmutação",
}

// rebuildSpellSchools recria as escolas de magia a partir do texto da p172. A
// magia CITA a escola e não havia para onde o elo apontar. O `id` é o nome
// dobrado, que é o que a magia guarda em `school`; magia com escola sem verbete
// PARA o script.
//
// A ABREVIATURA vem junto porque o livro a imprime ("Abjuração (Abjur)"). Ilusão
// não tem — o livro não deu —, e ausência aqui é ausência.
func (b *book) rebuildSpellSchools() error {
	entries, err := b.entriesOfPage(schoolsPage, schoolNames)
	if err != nil {
		return err
	}
	kept := map[string]string{}
	for _, s := range transcript.SpellSchools {
		kept[s.ID] = s.UID
	}
	var schools []transcript.SpellSchool
	known := map[string]bool{}
	for _, e := range entries {
		id := fold(e.name)
		known[id] = true
		schools = append(schools, transcript.SpellSchool{ID: id, Name: e.name, Description: e.text,
			BookPage: schoolsPage, Abbreviation: e.abbreviation, UID: kept[id]})
	}
	orphans := map[string]bool{}
	for _, s := range transcript.Spells {
		if s.School != "" && !known[s.School] {
			orphans[s.School] = true
		}
	}
	if len(orphans) > 0 {
		return fatal{"magias com escola sem verbete: " + pyStrList(sortedKeys(orphans))}
	}
	transcript.SpellSchools = schools
	fmt.Printf("catálogo de escolas de magia: %d · nenhuma magia órfã\n", len(schools))
	return nil
}
