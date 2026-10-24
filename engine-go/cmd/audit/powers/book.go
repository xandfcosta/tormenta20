package main

import (
	"slices"
	"strconv"
	"strings"

	"t20engine/infra/bookpdf"
)

const firstPDFPage, lastPDFPage = 130, 143 // o livro abre a seção na p124 e fecha na p137
const pdfOffset = 6

// tablePages são as páginas que têm TABELA. A da p127 é dos Poderes Concedidos
// (ALE-408) e não se lê aqui, mas as células dela saem da busca por verbete do
// mesmo jeito.
var tablePages = []int{132, 133, 134}

// tableRow é uma linha da tabela: o nome impresso, o pré-requisito, e o PAI na
// árvore ("" na raiz).
type tableRow struct {
	name, prereq, parent string
}

// readTable é a âncora 1: `{chave: linha}` das tabelas de p126 e p128.
//
// O `parent` é o poder acima na árvore — a indentação é o que diz "o
// pré-requisito deste é aquele", e o `—` de uma linha indentada significa
// "nenhum ALÉM do pai", não "nenhum".
func readTable(book *bookpdf.Book) map[string]tableRow {
	out := map[string]tableRow{}
	roots := map[int]string{}
	for _, page := range []int{132, 134} {
		for _, t := range book.PowerTables(page) {
			for _, pair := range t.Pairs {
				rows, _ := bookpdf.PairRows(t.Body, pair.PowerX, pair.PrereqX)
				for _, r := range rows {
					roots[r.Level] = r.Power
					parent := ""
					if r.Level != 0 {
						parent = roots[r.Level-1]
					}
					out[bookpdf.Key(r.Power)] = tableRow{r.Power, r.Prereq, parent}
				}
			}
		}
	}
	return out
}

// bookEntry é um verbete achado: o nome, a página do LIVRO e o corpo.
type bookEntry struct {
	name string
	page int
	body string
}

type pageLine struct {
	page int
	text string
}

// readEntries é a âncora 2: `{chave: verbete}`, e quantas vezes cada título
// foi achado, na ordem em que apareceram.
//
// O nome aparece uma TERCEIRA vez no capítulo, como pré-requisito de outro
// poder, e quando a quebra de linha o deixa sozinho ele tem cara de título. O
// discriminante está na PRÓPRIA linha: um título não termina em ponto, e a
// menção termina, porque fecha a frase. Olhar a linha ANTERIOR não serve: ela
// quase sempre termina com o `Pré-requisito: …` do verbete de cima, e esse
// filtro descartou 26 verbetes legítimos.
func readEntries(book *bookpdf.Book, names []string) (map[string]bookEntry, []countEntry) {
	byKey := map[string]string{}
	for _, n := range names {
		byKey[bookpdf.Key(n)] = n
	}
	var lines []pageLine
	for page := firstPDFPage; page <= lastPDFPage; page++ {
		consumed := map[bookpdf.Cell]bool{}
		if slices.Contains(tablePages, page) {
			consumed = book.TableCells(page)
		}
		for _, l := range book.Reading(page) {
			if !consumed[bookpdf.Cell{X: bookpdf.Round1(l.X), Y: bookpdf.Round1(l.Y)}] {
				lines = append(lines, pageLine{page, l.Text})
			}
		}
	}

	type found struct {
		index, page int
		name        string
	}
	var hits []found
	for i, l := range lines {
		if name, ok := byKey[bookpdf.Key(l.text)]; ok && !strings.HasSuffix(strings.TrimRightFunc(l.text, bookpdf.IsPySpace), ".") {
			hits = append(hits, found{i, l.page, name})
		}
	}

	entries := map[string]bookEntry{}
	var seen []countEntry
	for n, h := range hits {
		end := len(lines)
		if n+1 < len(hits) {
			end = hits[n+1].index
		}
		if i := slices.IndexFunc(seen, func(c countEntry) bool { return c.key == h.name }); i >= 0 {
			seen[i].n++
		} else {
			seen = append(seen, countEntry{h.name, 1})
		}
		if _, ok := entries[bookpdf.Key(h.name)]; ok {
			continue
		}
		var body []string
		for _, l := range lines[h.index:end] {
			body = append(body, l.text)
		}
		entries[bookpdf.Key(h.name)] = bookEntry{h.name, h.page - pdfOffset, bookpdf.Join(body)}
	}
	return entries, seen
}

func itoa(n int) string { return strconv.Itoa(n) }
