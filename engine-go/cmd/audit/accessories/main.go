// Command accessories confere os ACESSÓRIOS do catálogo contra as Tabelas
// 8-13/14/15 (ALE-452).
//
// São TRÊS tabelas e UMA lista de verbetes: o livro separa os acessórios por
// magnitude — menores (p342), médios e maiores (p343) —, mas os verbetes vêm
// numa lista só, em ordem alfabética, atravessando as páginas 342 a 345. Por
// isso as três tabelas são lidas juntas: o que procura verbete é o conjunto dos
// 65 nomes, e cada um volta sabendo de que tabela veio.
//
// Confere-se o que o livro imprime duas vezes ou declara: o nome está na tabela
// E tem verbete, o preço bate, a página citada é a do verbete, e a descrição
// cobre o texto do livro. NÃO se confere a regra: um acessório não tem
// composição em itálico, e o que ele faz é prosa — quem decide se aquilo virou
// modificador, gesto ou `unmodeled` é o guarda do catálogo.
//
//	go run ./cmd/audit/accessories
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"t20engine/cmd/internal/pyfmt"

	"t20engine/domain/catalog/transcript"
	"t20engine/infra/bookpdf"
)

const (
	pdfOffset     = 6
	slack         = 3.0
	coverageFloor = 0.95
)

// lootTable é uma das três tabelas, com as coordenadas medidas na página. O x
// do `d%` é FAIXA porque a célula é alinhada à direita — a borda de baixo já
// custou a última linha da 8-13 ("99-100" começa 2 pontos à esquerda de
// "97-98").
type lootTable struct {
	number, label string
	page          int
	diceX         [2]float64
	nameX         float64
	priceX        [2]float64
}

var tables = []lootTable{
	{"8-13", "menores", 348, [2]float64{293.0, 302.0}, 330.0, [2]float64{460.0, 470.0}},
	{"8-14", "médios", 349, [2]float64{76.0, 84.0}, 111.7, [2]float64{242.0, 250.0}},
	{"8-15", "maiores", 349, [2]float64{313.0, 322.0}, 348.7, [2]float64{480.0, 492.0}},
}

// Os verbetes atravessam QUATRO páginas, e a margem MUDA de par para ímpar.
//
// A lista acaba na p345, com a Vassoura voadora — o que vem depois são os
// ARTEFATOS, que o livro tira da geração aleatória de propósito. O título deles
// é CENTRALIZADO, fora das duas colunas, então nenhuma sentinela de texto o
// alcança: quem para a leitura é a última página da lista.
var entryPages = []int{348, 349, 350, 351}

var (
	evenColumns = []bookpdf.ColumnPair{{Body: 51.0, Indent: 68.0}, {Body: 289.1, Indent: 306.1}}
	oddColumns  = []bookpdf.ColumnPair{{Body: 70.9, Indent: 87.9}, {Body: 309.0, Indent: 326.0}}
)

// As regiões de tabela a pular, e a p349 tem DUAS — a 8-14 à esquerda e a
// 8-15 à direita —, por isso a região é uma lista.
var regions = map[int][]bookpdf.Region{
	348: {{X0: 290.0, X1: 520.0, Y0: 50.0, Y1: 390.0}},
	349: {{X0: 70.0, X1: 270.0, Y0: 50.0, Y1: 395.0}, {X0: 310.0, X1: 520.0, Y0: 50.0, Y1: 395.0}},
}

// tableRow é uma linha de uma das três tabelas, sabendo de qual veio.
type tableRow struct {
	table *lootTable
	row   bookpdf.LootRow
}

func main() {
	book, err := bookpdf.Open()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(min(1, audit(book)))
}

// rowsOfTables são as linhas das três tabelas, com a tilagem conferida.
func rowsOfTables(book *bookpdf.Book) []tableRow {
	var out []tableRow
	for i := range tables {
		t := &tables[i]
		rows := book.LootRows(t.page, t.diceX, t.nameX, t.priceX, slack)
		if hole := bookpdf.RangesThatDontClose(rows); hole != "" {
			fmt.Fprintf(os.Stderr, "A TABELA %s NÃO FECHA: %s\nNada do que viria abaixo valeria — a leitura está torta.\n", t.number, hole)
			os.Exit(1)
		}
		for _, r := range rows {
			out = append(out, tableRow{t, r})
		}
	}
	return out
}

func audit(book *bookpdf.Book) int {
	rows := rowsOfTables(book)
	names := make([]string, len(rows))
	for i, r := range rows {
		names[i] = r.row.Name
	}
	columns := map[int][]bookpdf.ColumnPair{}
	for _, p := range entryPages {
		columns[p] = oddColumns
		if p%2 == 0 {
			columns[p] = evenColumns
		}
	}
	entries := book.EntriesByName(entryPages, columns, regions, names, nil, pdfOffset, slack)
	byName := apparelByKey()
	var failures []string
	matched := 0

	for i := range tables {
		t := &tables[i]
		fmt.Printf("\n══ TABELA %s (acessórios %s) %s\n", t.number, t.label, strings.Repeat("═", 20))
		fmt.Printf("%7s %-34s %8s  catálogo\n", "d%", "acessório", "preço")
		for _, r := range rows {
			if r.table != t {
				continue
			}
			span := strconv.Itoa(r.row.Start)
			if r.row.End != r.row.Start {
				span = fmt.Sprintf("%d-%d", r.row.Start, r.row.End)
			}
			item := byName[bookpdf.Key(r.row.Name)]
			label := "— AUSENTE"
			if item != nil {
				label = item.ID
			}
			fmt.Printf("%7s %-34s %8s  %s\n", span, r.row.Name, priceText(r.row.Price), label)
			entry := entries.ByName[r.row.Name]
			if entry == nil {
				failures = append(failures, fmt.Sprintf("%s: está na tabela %s e não tem verbete", r.row.Name, t.number))
			}
			if item != nil {
				matched++
				failures = append(failures, check(item, r.row, entry)...)
			}
		}
	}

	fmt.Printf("\n%d/%d transcritos\n", matched, len(rows))
	for _, f := range failures {
		fmt.Printf("  ✗ %s\n", f)
	}
	return len(failures)
}

// apparelByKey são os acessórios pelo nome normalizado; o último de um nome
// repetido vence, como no dicionário do auditor original.
func apparelByKey() map[string]*transcript.Item {
	out := map[string]*transcript.Item{}
	for i := range transcript.Items {
		if it := &transcript.Items[i]; it.Category == "apparel" {
			out[bookpdf.Key(it.Name)] = it
		}
	}
	return out
}

func check(item *transcript.Item, row bookpdf.LootRow, entry *bookpdf.Entry) []string {
	var failures []string
	want := 0.0
	if row.Price != nil {
		want = float64(*row.Price)
	}
	if item.Price != want {
		failures = append(failures, fmt.Sprintf("%s: o catálogo diz T$ %s e a tabela diz T$ %s",
			row.Name, optionalNumber(item.Price), priceText(row.Price)))
	}
	if entry == nil {
		return failures
	}
	if item.BookPage != entry.Page {
		failures = append(failures, fmt.Sprintf("%s: o catálogo cita p%s e o verbete está na p%d",
			row.Name, optionalNumber(float64(item.BookPage)), entry.Page))
	}
	ratio, _, _ := bookpdf.Coverage(bookpdf.Join(entry.Body), item.Description)
	if ratio < coverageFloor {
		failures = append(failures, fmt.Sprintf("%s: a descrição cobre %s do verbete (piso %s)",
			row.Name, pyfmt.Percent(ratio), pyfmt.Percent(coverageFloor)))
	}
	return failures
}

func priceText(price *int) string {
	if price == nil {
		return "None"
	}
	return strconv.Itoa(*price)
}

// optionalNumber é como o Python imprime um `get()` de número: no dado tipado
// "ausente" e "0" são o mesmo, e o ausente é o que o Python via — `None`.
func optionalNumber(x float64) string {
	if x == 0 {
		return "None"
	}
	return strconv.FormatFloat(x, 'f', -1, 64)
}
