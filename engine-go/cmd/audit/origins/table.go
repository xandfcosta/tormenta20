package main

import (
	"fmt"
	"sort"
	"t20engine/cmd/internal/pyfmt"

	"t20engine/infra/bookpdf"
)

const (
	firstPage, lastPage = 91, 101 // PDF; o livro abre a seção na p85 e fecha na p95
	pdfOffset           = 6       // livro = PDF - 6
	nameHeader          = "Origem"
	benefitsHeader      = "Benefícios"
	tableTitle          = "Tabela 1-19: Origens"
	sameColumn          = 2.0 // pontos de tolerância no x de uma célula da tabela
)

// consumedLine é uma linha que a tabela leu, com o (x, y) EXATO: é ela que a
// busca por verbete tem de pular, ou o rodapé da coluna dá ao "Trabalhador" o
// verbete do Artista.
type consumedLine struct {
	page int
	x, y float64
}

// A ordem de leitura é o `Reading` do bookpdf: colunas com separação mínima de
// 40 pontos, porque na p87 a célula de nome da Tabela 1-19 (x=74,9) e a prosa
// (x=70,9) passam as duas o piso de frequência, e a página sai fora de ordem
// sem cara de desordem — o `Benefícios.` do Artista depois do `(poderes).` dele.

func tablePage(book *bookpdf.Book) (int, bool) {
	for page := firstPage; page <= lastPage; page++ {
		for _, l := range book.Reading(page) {
			if l.Text == tableTitle {
				return page, true
			}
		}
	}
	return 0, false
}

type tableCell struct {
	name, benefits string
}

// tableCells são as duas colunas da Tabela 1-19 agrupadas pelo y — o que diz
// que um nome e uma lista são a MESMA linha. Parear por índice erraria na
// primeira que quebra em duas, e a do Assistente de Laboratório quebra nas DUAS.
func tableCells(book *bookpdf.Book, page int) (map[float64]*tableCell, []consumedLine, []string) {
	lines := book.Reading(page)
	nameX, headerY, found := 0.0, 0.0, false
	for _, l := range lines {
		if l.Text == nameHeader {
			nameX, headerY, found = l.X, l.Y, true
			break
		}
	}
	if !found {
		return nil, nil, []string{fmt.Sprintf("PROBLEMA a coluna %s da %s não foi achada na p%d",
			pyfmt.Repr(nameHeader), tableTitle, page-pdfOffset)}
	}
	benefitsX, found := 0.0, false
	for _, l := range lines {
		if abs(l.Y-headerY) < sameColumn && l.Text == benefitsHeader {
			benefitsX, found = l.X, true
			break
		}
	}
	if !found {
		return nil, nil, []string{fmt.Sprintf("PROBLEMA a coluna %s da %s não foi achada na p%d",
			pyfmt.Repr(benefitsHeader), tableTitle, page-pdfOffset)}
	}
	cells := map[float64]*tableCell{}
	var consumed []consumedLine
	for _, l := range lines {
		if l.Y <= headerY {
			continue
		}
		isName := abs(l.X-nameX) < sameColumn
		if !isName && abs(l.X-benefitsX) >= sameColumn {
			continue
		}
		y := bookpdf.Round1(l.Y)
		if cells[y] == nil {
			cells[y] = &tableCell{}
		}
		if isName {
			cells[y].name = l.Text
		} else {
			cells[y].benefits = l.Text
		}
		consumed = append(consumed, consumedLine{page, l.X, l.Y})
	}
	return cells, consumed, nil
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// readTable é a âncora 1: o texto de benefícios de cada origem na Tabela 1-19,
// as notas com o denominador, e as linhas que a tabela consumiu.
func readTable(book *bookpdf.Book, names nameIndex) (map[string]string, []string, []consumedLine) {
	page, ok := tablePage(book)
	if !ok {
		return map[string]string{}, []string{fmt.Sprintf("PROBLEMA %s não foi achado entre as páginas %d-%d",
			pyfmt.Repr(tableTitle), firstPage-pdfOffset, lastPage-pdfOffset)}, nil
	}
	cells, consumed, problems := tableCells(book, page)
	if len(problems) > 0 {
		return map[string]string{}, problems, nil
	}

	fromTable := map[string]string{}
	var notes []string
	slug, haveSlug := "", false
	// O nome de uma linha pode ocupar DUAS linhas de texto. A que não casa
	// sozinha fica pendente, e a seguinte decide o que ela era: nome acumulado
	// de uma linha NOVA, ou continuação da linha de cima.
	var pending *tableCell
	var ys []float64
	for y := range cells {
		ys = append(ys, y)
	}
	sort.Float64s(ys)
	for _, y := range ys {
		name, benefits := cells[y].name, cells[y].benefits
		if s, ok := names.byKey[bookpdf.Key(name)]; ok {
			slug, haveSlug, pending = s, true, nil
			fromTable[slug] = benefits
			continue
		}
		if pending != nil {
			if s, ok := names.byKey[bookpdf.Key(pending.name+" "+name)]; ok {
				slug, haveSlug = s, true
				fromTable[slug] = bookpdf.Strip(pending.benefits + " " + benefits)
				notes = append(notes, fmt.Sprintf("%s: o nome ocupa duas linhas da tabela (%s + %s) — as duas "+
					"metades foram juntadas nas DUAS colunas", names.name[slug], pyfmt.Repr(pending.name), pyfmt.Repr(name)))
				pending = nil
				continue
			}
		}
		if name != "" {
			pending = &tableCell{name, benefits}
			continue
		}
		if !haveSlug {
			notes = append(notes, "PROBLEMA célula da tabela antes da primeira origem: "+pyfmt.Repr(benefits))
			continue
		}
		fromTable[slug] = bookpdf.Strip(fromTable[slug] + " " + benefits)
	}
	if pending != nil {
		notes = append(notes, fmt.Sprintf("PROBLEMA nome de linha da tabela sem origem no catálogo: %s / %s",
			pyfmt.Repr(pending.name), pyfmt.Repr(pending.benefits)))
	}
	notes = append(notes, fmt.Sprintf("grupos de y na tabela: %d → linhas casadas com uma origem: %d "+
		"(a diferença são as quebras nomeadas acima)", len(cells), len(fromTable)))
	return fromTable, notes, consumed
}
