package main

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"t20engine/domain/catalog/transcript"
	"t20engine/infra/bookpdf"
)

// O ÍNDICE, p174-177: cada magia é um LINK EM NEGRITO para o verbete.
const (
	indexFirstPage = 180
	indexLastPage  = 183
)

// O TÍTULO do verbete é `Tormenta20` a 24 pontos — o corpo é Iowan Old Style a
// 13, a linha de círculo é 14 e "Descrição das magias" é 39.
const titleSize = 24
const titleFamily = "Tormenta20"

// Título longo quebra em duas linhas ("Abençoar" / "Alimentos"). Junta-se o
// trecho seguinte quando ele está na MESMA coluna e na linha de baixo: dois
// títulos de colunas vizinhas na mesma altura não são um título — foi esse o
// erro que fez uma contagem dar 247 (ALE-456).
const sameColumn = 30.0
const nextLine = 32.0
const columnWidth = 120.0

// anchored é um nome do livro e a página (do livro) onde ele foi lido.
type anchored struct {
	name string
	page int
}

// bookIndex são as magias das duas listas do índice, por chave. Por nome e não
// por contagem: 28 magias estão nas DUAS listas.
func bookIndex(book *bookpdf.Book) map[string]anchored {
	out := map[string]anchored{}
	for page := indexFirstPage; page <= indexLastPage; page++ {
		for _, link := range book.BoldLinks(page) {
			name := bookpdf.Strip(strings.TrimRight(link.Text, "."))
			if name == "" {
				continue
			}
			if _, seen := out[bookpdf.Key(name)]; !seen {
				out[bookpdf.Key(name)] = anchored{name, link.Page - pdfOffset}
			}
		}
	}
	return out
}

// entryTitles são os títulos dos verbetes (p178-211), lidos pelo TIPO em que o
// livro os desenha. É a segunda âncora, independente do índice: se uma leitura
// quebrar, a outra a denuncia.
func entryTitles(book *bookpdf.Book) map[string]anchored {
	out := map[string]anchored{}
	keep := func(title string, page int) {
		if _, seen := out[bookpdf.Key(title)]; !seen {
			out[bookpdf.Key(title)] = anchored{title, page - pdfOffset}
		}
	}
	for page := firstPage; page <= lastPage; page++ {
		var runs []bookpdf.StyledRun
		for _, r := range book.StyledRuns(page) {
			if r.Size == titleSize && strings.Contains(r.Family, titleFamily) {
				runs = append(runs, r)
			}
		}
		// round() do Python arredonda o empate para o par.
		sort.SliceStable(runs, func(i, j int) bool {
			ci, cj := math.RoundToEven(runs[i].X/columnWidth), math.RoundToEven(runs[j].X/columnWidth)
			if ci != cj {
				return ci < cj
			}
			return runs[i].Y < runs[j].Y
		})
		current, open := "", false
		var anchor bookpdf.StyledRun
		for _, run := range runs {
			joined := open && math.Abs(run.X-anchor.X) < sameColumn &&
				run.Y-anchor.Y > 0 && run.Y-anchor.Y < nextLine
			if joined {
				current += " " + run.Text
			} else {
				if open && current != "" {
					keep(current, page)
				}
				current, open = run.Text, true
			}
			anchor = run
		}
		if open && current != "" {
			keep(current, page)
		}
	}
	return out
}

func sortedKeys(m map[string]anchored) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// coverageReport mede quantas magias o LIVRO tem contra quantas entraram, e
// devolve o número de falhas.
func coverageReport(book *bookpdf.Book, catalog map[string]*transcript.Spell) int {
	index := bookIndex(book)
	entries := entryTitles(book)
	inCatalog := map[string]string{}
	for _, s := range transcript.Spells {
		inCatalog[bookpdf.Key(s.Name)] = s.ID
	}
	var failures []string

	fmt.Println("══ COBERTURA " + strings.Repeat("═", 46))
	fmt.Printf("  índice do livro (p%d-%d, links em negrito): %d\n",
		indexFirstPage-pdfOffset, indexLastPage-pdfOffset, len(index))
	fmt.Printf("  verbetes (p%d-%d, título a %dpt):        %d\n",
		firstPage-pdfOffset, lastPage-pdfOffset, titleSize, len(entries))
	fmt.Printf("  no catálogo:                                 %d\n", len(catalog))

	// AS DUAS ÂNCORAS PRIMEIRO, uma contra a outra: se elas discordam, a
	// leitura está torta e nada do que vem depois vale.
	for _, k := range sortedKeys(index) {
		if _, ok := entries[k]; !ok {
			failures = append(failures, index[k].name+": está no índice e não tem título de verbete — "+
				"uma das duas leituras está torta")
		}
	}
	for _, k := range sortedKeys(entries) {
		if _, ok := index[k]; !ok {
			failures = append(failures, entries[k].name+": tem verbete e não está no índice — "+
				"uma das duas leituras está torta")
		}
	}
	if len(failures) > 0 {
		fmt.Println("\n  AS DUAS ÂNCORAS DISCORDAM, e por isso o catálogo nem foi conferido:")
		for _, f := range failures {
			fmt.Printf("   ✗ %s\n", f)
		}
		return len(failures)
	}

	// A PÁGINA SAI DO TÍTULO, e NÃO do destino do link: o destino aponta para o
	// começo do QUADRO, e quando o verbete nasce no pé de uma coluna ele cai na
	// página seguinte (Dificultar Detecção, Esculpir Sons). O link vale para a
	// PRESENÇA, que é o que ele mede bem.
	for _, k := range sortedKeys(index) {
		id, ok := inCatalog[k]
		if !ok {
			failures = append(failures, index[k].name+": o livro tem e o catálogo não")
			continue
		}
		page := entries[k].page
		if catalog[id].BookPage != page {
			failures = append(failures, fmt.Sprintf("%s: o catálogo cita p%d e o título do verbete está desenhado na p%d",
				index[k].name, catalog[id].BookPage, page))
		}
	}
	var catalogKeys []string
	for k := range inCatalog {
		catalogKeys = append(catalogKeys, k)
	}
	sort.Strings(catalogKeys)
	for _, k := range catalogKeys {
		if _, ok := index[k]; !ok {
			failures = append(failures, catalog[inCatalog[k]].Name+": está no catálogo e o livro não tem")
		}
	}

	if len(failures) == 0 {
		fmt.Printf("\n  as três fecham em %d: nada falta e nada sobra\n", len(catalog))
	}
	for _, f := range failures {
		fmt.Printf("   ✗ %s\n", f)
	}
	return len(failures)
}
