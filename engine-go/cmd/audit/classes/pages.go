package main

import (
	"math"
	"sort"
	"strconv"
	"strings"

	"t20engine/infra/bookpdf"
)

// levelTable é a tabela da classe: {nível: texto da linha}, na ordem em que os
// níveis foram lidos — é a ordem em que o relatório os lista.
type levelTable struct {
	levels []int
	text   map[int]string
}

func (t *levelTable) set(level int, text string) {
	if _, ok := t.text[level]; !ok {
		t.levels = append(t.levels, level)
	}
	t.text[level] = text
}

// classTable lê a tabela da classe pareando por Y: o nível e a habilidade são
// blocos DIFERENTES do PDF, e parear por índice erra na primeira habilidade que
// quebra em duas linhas — a do Bárbaro quebra no 11º. A linha que não casa com
// nível nenhum é CONTINUAÇÃO da anterior.
//
// A coluna da habilidade não é "tudo à direita do nível": em página cuja tabela
// fica no meio, isso apanha a PROSA. Ela é o x mais FREQUENTE entre as linhas
// que pareiam com um nível — aprender custou cinco falsos "não nomeado na
// tabela" que pareciam defeito de catálogo.
func classTable(book *bookpdf.Book, class string, first, last int) levelTable {
	for page := first + pdfOffset; page <= last+pdfOffset; page++ {
		lines := bookpdf.PositionedWithoutJunk(book.PositionedLines(page))
		titleY, haveTitle := 0.0, false
		for _, l := range lines {
			if strings.HasPrefix(l.Text, "Tabela") && strings.Contains(bookpdf.Key(l.Text), bookpdf.Key(class)) {
				titleY, haveTitle = l.Y, true
				break
			}
		}
		if !haveTitle {
			continue
		}
		var body []bookpdf.Positioned
		for _, l := range lines {
			if l.Y > titleY {
				body = append(body, l)
			}
		}
		// {round(y): nível}, na ordem de inserção do dict do Python.
		var levelYs []float64
		levelAt := map[float64]int{}
		levelX, haveLevel := 0.0, false
		for _, l := range body {
			m := levelCell.FindStringSubmatch(l.Text)
			if m == nil {
				continue
			}
			y := math.RoundToEven(l.Y)
			if _, ok := levelAt[y]; !ok {
				levelYs = append(levelYs, y)
			}
			levelAt[y], _ = strconv.Atoi(m[1])
			if !haveLevel || l.X > levelX {
				levelX, haveLevel = l.X, true
			}
		}
		if len(levelYs) == 0 {
			continue
		}
		levelNear := func(y float64) (int, bool) {
			for _, yy := range levelYs {
				if math.Abs(yy-y) <= 2 {
					return levelAt[yy], true
				}
			}
			return 0, false
		}
		var candidates []bookpdf.Positioned
		for _, l := range body {
			if l.X > levelX+5 {
				candidates = append(candidates, l)
			}
		}
		var xs []float64
		count := map[float64]int{}
		for _, l := range candidates {
			if _, ok := levelNear(l.Y); !ok {
				continue
			}
			x := bookpdf.Round1(l.X)
			if count[x] == 0 {
				xs = append(xs, x)
			}
			count[x]++
		}
		if len(xs) == 0 {
			continue
		}
		abilityX := xs[0]
		for _, x := range xs[1:] {
			if count[x] > count[abilityX] {
				abilityX = x
			}
		}

		var column []bookpdf.Positioned
		for _, l := range candidates {
			if math.Abs(l.X-abilityX) <= 3 {
				column = append(column, l)
			}
		}
		sort.SliceStable(column, func(i, j int) bool { return column[i].Y < column[j].Y })
		out := levelTable{text: map[int]string{}}
		current, haveCurrent := 0, false
		for _, l := range column {
			if n, ok := levelNear(l.Y); ok {
				current, haveCurrent = n, true
				out.set(current, l.Text)
			} else if haveCurrent {
				out.set(current, out.text[current]+" "+l.Text)
			}
		}
		return out
	}
	return levelTable{text: map[int]string{}}
}

// classProse é {chave do nome: [linhas do corpo]} das entradas em prosa.
func classProse(book *bookpdf.Book, first, last int) map[string][]string {
	var lines []string
	for page := first + pdfOffset; page <= last+pdfOffset; page++ {
		lines = append(lines, book.PageLines(page)...)
	}
	out := map[string][]string{}
	current := ""
	for _, line := range lines {
		stripped := bookpdf.Strip(line)
		if m := proseHead.FindStringSubmatch(stripped); m != nil {
			current = bookpdf.Key(m[1])
			if _, ok := out[current]; !ok {
				out[current] = []string{m[2]}
			}
		} else if current != "" {
			if _, ok := out[current]; ok {
				out[current] = append(out[current], stripped)
			}
		}
	}
	return out
}
