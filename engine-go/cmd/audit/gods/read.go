package main

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"t20engine/cmd/internal/pyfmt"
	"unicode/utf8"

	"t20engine/infra/bookpdf"
)

const (
	firstPage, lastPage = 102, 111 // PDF
	pdfOffset           = 6        // livro = PDF - 6
	tablePage           = 103      // p97, onde a Tabela 1-20 divide a página com o verbete
	tableEndsAtY        = 440.0
	opensEntry          = "Crenças e Objetivos."
	closesEntry         = "Obrigações & Restrições"
	maxTitleLength      = 18 // caracteres; nome de deus é curto, prosa não
	tableRowGap         = 12.0
	columnWidth         = 200.0
)

// fields são os campos ROTULADOS do verbete, na ordem em que o livro os imprime.
var fields = []string{"Símbolo Sagrado", "Canalizar Energia", "Arma Preferida", "Devotos", "Poderes Concedidos"}

// tableRow é a linha de um deus na Tabela 1-20.
type tableRow struct {
	energy string
	powers []string
}

type cellText struct {
	x    float64
	text string
}

// readTable lê a Tabela 1-20 por chave do deus.
//
// A célula de poderes QUEBRA em duas linhas quando a lista é longa (Aharadak,
// Hyninn), e a continuação não tem nome na coluna da esquerda. Ela pertence à
// linha de cima, e é o y que diz isso.
func readTable(book *bookpdf.Book) map[string]tableRow {
	var cells []bookpdf.Positioned
	for _, l := range book.PositionedLines(tablePage) {
		if l.Y < tableEndsAtY {
			cells = append(cells, l)
		}
	}
	sort.SliceStable(cells, func(i, j int) bool {
		if cells[i].Y != cells[j].Y {
			return cells[i].Y < cells[j].Y
		}
		return cells[i].X < cells[j].X
	})
	type row struct {
		y     float64
		cells []cellText
	}
	var rows []row
	for _, c := range cells {
		if len(rows) > 0 && c.Y-rows[len(rows)-1].y < tableRowGap {
			last := &rows[len(rows)-1]
			last.cells = append(last.cells, cellText{c.X, c.Text})
		} else {
			rows = append(rows, row{c.Y, []cellText{{c.X, c.Text}}})
		}
	}
	out := map[string]tableRow{}
	for _, r := range rows {
		sort.Slice(r.cells, func(i, j int) bool {
			if r.cells[i].x != r.cells[j].x {
				return r.cells[i].x < r.cells[j].x
			}
			return r.cells[i].text < r.cells[j].text
		})
		if len(r.cells) < 3 {
			continue
		}
		name, energy := r.cells[0].text, r.cells[1].text
		if name == "Divindade" {
			continue
		}
		var powerTexts []string
		for _, c := range r.cells[2:] {
			powerTexts = append(powerTexts, c.text)
		}
		out[bookpdf.Key(name)] = tableRow{
			strings.ToLower(strings.TrimRight(bookpdf.Strip(energy), ".")),
			splitPowers(strings.Join(powerTexts, " "), true),
		}
	}
	return out
}

// splitPowers corta uma lista de poderes nas vírgulas; `dropEmpty` é o `if
// p.strip()` que o Python põe em umas leituras e não em outras.
func splitPowers(s string, dropEmpty bool) []string {
	out := []string{}
	for _, p := range strings.Split(s, ",") {
		p = bookpdf.Strip(p)
		if dropEmpty && p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

// readingByColumn é a ordem de leitura deste capítulo: coluna de 200 pontos,
// depois y. Não é o `Reading` do bookpdf — é a conta que este auditor sempre fez.
func readingByColumn(book *bookpdf.Book, page int) []bookpdf.Positioned {
	lines := book.PositionedLines(page)
	sort.SliceStable(lines, func(i, j int) bool {
		ci, cj := math.Floor(lines[i].X/columnWidth), math.Floor(lines[j].X/columnWidth)
		if ci != cj {
			return ci < cj
		}
		return lines[i].Y < lines[j].Y
	})
	return bookpdf.PositionedWithoutJunk(lines)
}

// godBlock é o verbete ligado a um deus.
type godBlock struct {
	god  string
	body string
	page int
}

type godTitle struct {
	position int
	god      string
	page     int
}

// chapterBlocks são os verbetes em ordem de leitura.
//
// O capítulo se corta em `Crenças e Objetivos.`, que abre todo verbete. A
// ligação bloco→deus é GEOMÉTRICA — o último título de deus antes do corte —, e
// a ordem alfabética é o CONTROLE, não a regra: como regra ela funcionaria hoje
// e calaria no dia em que o livro mudasse de ordem.
func chapterBlocks(book *bookpdf.Book, names map[string]string) ([]godBlock, []string) {
	var flow []string
	var titles []godTitle
	for page := firstPage; page <= lastPage; page++ {
		for _, l := range readingByColumn(book, page) {
			if page == tablePage && l.Y < tableEndsAtY {
				continue
			}
			if name, ok := names[bookpdf.Key(l.Text)]; ok && utf8.RuneCountInString(l.Text) <= maxTitleLength {
				titles = append(titles, godTitle{len(flow), name, page - pdfOffset})
			}
			flow = append(flow, l.Text)
		}
	}

	var openings []int
	for i, line := range flow {
		if strings.HasPrefix(line, opensEntry) {
			openings = append(openings, i)
		}
	}
	var blocks []godBlock
	var complaints []string
	for n, i := range openings {
		end := len(flow)
		if n+1 < len(openings) {
			end = openings[n+1]
		}
		body := bookpdf.Join(flow[i:end])
		cut := strings.Index(body, closesEntry)
		if cut < 0 {
			// A seção "Características dos deuses" EXPLICA os rótulos e não tem
			// Obrigações. Cortar até o PRÓXIMO `Crenças` é o que a distingue:
			// fatiar até o fim do capítulo faria todo bloco conter o rótulo de
			// alguém, e a explicação passaria por um deus sem nome.
			continue
		}
		var before []godTitle
		for _, t := range titles {
			if t.position < i {
				before = append(before, t)
			}
		}
		if len(before) == 0 {
			complaints = append(complaints, fmt.Sprintf("bloco na linha %d sem título de deus antes dele", i))
			continue
		}
		last := before[len(before)-1]
		blocks = append(blocks, godBlock{last.god, body[:cut], last.page})
	}
	return blocks, complaints
}

// fieldValue é um campo do verbete; `found` separa "o livro não imprimiu" de
// "imprimiu vazio", que o Python mostra como `None` e `”`.
type fieldValue struct {
	text  string
	found bool
}

func (f fieldValue) repr() string {
	if !f.found {
		return "None"
	}
	return pyfmt.Repr(f.text)
}

var fieldRes = func() map[string]*regexp.Regexp {
	out := map[string]*regexp.Regexp{}
	for i, field := range fields {
		next := closesEntry
		if i+1 < len(fields) {
			next = fields[i+1]
		}
		out[field] = regexp.MustCompile(`(?s)` + regexp.QuoteMeta(field) + `\. (.+?)(?: ` + regexp.QuoteMeta(next) + `\.|$)`)
	}
	return out
}()

// readEntry lê os campos rotulados de um verbete.
func readEntry(body string) map[string]fieldValue {
	out := map[string]fieldValue{}
	for _, field := range fields {
		if m := fieldRes[field].FindStringSubmatch(body); m != nil {
			out[field] = fieldValue{strings.TrimRight(bookpdf.Strip(m[1]), "."), true}
		} else {
			out[field] = fieldValue{}
		}
	}
	return out
}

var availableRe = regexp.MustCompile(`deus disponível para (druidas|paladinos) \(([^)]+)\)`)
var availableSeparator = regexp.MustCompile(`,| ou `)

// pagesAfterClass é onde a regra de devoção cai: na abertura da classe.
const pagesAfterClass = 3

// godsByClass lê, da frase que a CLASSE imprime, quais deuses ela aceita.
//
// A elegibilidade NÃO se deriva dos "Devotos" do deus, e a Valkaria é a prova:
// ela aceita "membros de todas as classes", mas a p61 diz que o druida é devoto
// "de um deus disponível para druidas (Allihanna, Megalokk ou Oceano)". Quem
// restringe é a classe. As páginas saem do catálogo de classes, e não de um
// número escrito aqui.
func godsByClass(book *bookpdf.Book, classPages map[string]int) (map[string]map[string]bool, []string) {
	out := map[string]map[string]bool{}
	var complaints []string
	for _, c := range []struct{ class, who string }{{"Druida", "druidas"}, {"Paladino", "paladinos"}} {
		start := classPages[c.class] + pdfOffset
		found := false
		for page := start; page < start+pagesAfterClass; page++ {
			var texts []string
			for _, l := range readingByColumn(book, page) {
				texts = append(texts, l.Text)
			}
			if m := availableRe.FindStringSubmatch(bookpdf.Join(texts)); m != nil {
				gods := map[string]bool{}
				for _, n := range availableSeparator.Split(m[2], -1) {
					gods[bookpdf.Strip(n)] = true
				}
				out[c.who] = gods
				found = true
				break
			}
		}
		if !found {
			complaints = append(complaints, fmt.Sprintf("não achei a frase \"deus disponível para %s\" nas páginas "+
				"%d-%d — %s ficaram sem medição",
				c.who, classPages[c.class], classPages[c.class]+pagesAfterClass-1, c.who))
		}
	}
	return out, complaints
}
