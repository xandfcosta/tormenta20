// Command specificitems confere os ITENS ESPECÍFICOS do catálogo contra o
// livro (ALE-448).
//
// São DUAS TABELAS e UM auditor, como nos encantos: a 8-9 é de armas (p337) e a
// 8-11 de armaduras e escudos (p340), com a mesma forma — `d%` + nome + PREÇO na
// tabela, `Nome. Regra` no verbete.
//
// A terceira âncora é o ITÁLICO, e é a razão deste auditor existir: um item
// específico é um item BASE com sobreposições já postas, e o livro escreve
// QUAIS em itálico — "este machado de guerra de gelo eterno congelante
// formidável" (Avalanche, p337) é a base, o material e dois encantos. Aqui a
// composição é COLHIDA, e o `composedOf` do catálogo conferido contra ela. As
// faixas de `d%` tilam 1..100: uma leitura que perca uma linha não fecha.
//
//	go run ./cmd/audit/specificitems            # as duas tabelas
//	go run ./cmd/audit/specificitems protecao   # uma só
package main

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"t20engine/cmd/internal/pyfmt"

	"t20engine/domain/catalog/transcript"
	"t20engine/infra/bookpdf"
)

const (
	pdfOffset = 6
	slack     = 3.0
)

// profile é uma das duas tabelas de item específico, com onde ela mora.
type profile struct {
	key, table   string
	tablePage    int
	entryPages   []int
	diceX        [2]float64
	nameX        float64
	priceX       [2]float64
	proseColumns map[int][]bookpdf.ColumnPair
	tableRegion  map[int][]bookpdf.Region
	entriesEnd   []string
}

var profileOrder = []string{"arma", "protecao"}

var profiles = map[string]profile{
	// A 8-9 mora na coluna direita da p337, e os verbetes atravessam a p337
	// (a direita retoma ABAIXO da tabela) e a p338 só na esquerda.
	"arma": {
		key: "arma", table: "8-9",
		tablePage: 343, entryPages: []int{342, 343, 344},
		diceX: [2]float64{313.0, 320.0}, nameX: 348.7, priceX: [2]float64{480.0, 490.0},
		// A MARGEM MUDA DE PÁGINA PAR PARA ÍMPAR: 51 na par e 70,9 na ímpar.
		proseColumns: map[int][]bookpdf.ColumnPair{
			342: {{Body: 51.0, Indent: 68.0}, {Body: 289.1, Indent: 306.1}},
			343: {{Body: 70.9, Indent: 87.9}, {Body: 309.0, Indent: 326.0}},
			344: {{Body: 51.0, Indent: 68.0}},
		},
		// Duas tabelas a evitar, em LADOS DIFERENTES da página (a 8-8 à
		// esquerda da p336, a 8-9 no alto da direita da p337): por isso a
		// região é um RETÂNGULO, e não um corte por "x menor que".
		tableRegion: map[int][]bookpdf.Region{
			342: {{X0: 0.0, X1: 260.0, Y0: 0.0, Y1: 530.0}},
			343: {{X0: 300.0, X1: 581.0, Y0: 0.0, Y1: 380.0}},
		},
		entriesEnd: []string{"Armaduras"},
	},
	// A 8-11 mora na coluna esquerda da p340; os verbetes vêm abaixo dela e na
	// direita. Medido: o d% em 59,1 (56,9 na linha colapsada), o nome em 90,7 e
	// o preço entre 223,7 e 225,9 — alinhado à direita.
	"protecao": {
		key: "protecao", table: "8-11",
		tablePage: 346, entryPages: []int{346},
		diceX: [2]float64{56.0, 60.0}, nameX: 90.7, priceX: [2]float64{222.0, 228.0},
		proseColumns: map[int][]bookpdf.ColumnPair{346: {{Body: 51.0, Indent: 68.0}, {Body: 289.1, Indent: 306.1}}},
		tableRegion:  map[int][]bookpdf.Region{346: {{X0: 0.0, X1: 250.0, Y0: 0.0, Y1: 292.0}}},
		// DOIS títulos no meio da lista; sem eles o título entra no verbete
		// anterior e a cobertura dele cai sem motivo.
		entriesEnd: []string{"Armaduras Específicas", "Escudos Específicos"},
	},
}

// italicThePageLost é o itálico que a página perdeu, nomeado e com a prova: o
// Escudo do Conjurador italiza "escudo leve" e deixa "defensor" em redondo — o
// estilo não atravessa a quebra de linha (`font="4"` numa, `font="3"` na
// seguinte). É deslize de tipografia: os outros doze trazem o encanto DENTRO do
// itálico, e o preço acompanha (T$ 45.000 contra 50.000 do escudo PESADO
// defensor). A exceção sai no relatório, e continua conferida.
var italicThePageLost = map[string][]string{
	"Escudo do conjurador": {"escudo-leve", "encanto-defensor"},
}

// coverageFloor: a descrição é TRANSCRITA e nasce em 100%; o piso prende o dia
// em que alguém "melhorar" a frase e ela passar a dizer outra coisa.
const coverageFloor = 0.95

func main() {
	chosen := os.Args[1:]
	if len(chosen) == 0 {
		chosen = profileOrder
	}
	var unknown []string
	for _, c := range chosen {
		if _, ok := profiles[c]; !ok {
			unknown = append(unknown, c)
		}
	}
	if len(unknown) > 0 {
		fmt.Printf("perfil desconhecido: %s. Conheço %s.\n", pyfmt.List(unknown), pyfmt.List(profileOrder))
		os.Exit(2)
	}
	book, err := bookpdf.Open()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	total := 0
	for _, c := range chosen {
		total += audit(book, profiles[c])
	}
	os.Exit(min(1, total))
}

// audit audita UMA tabela e devolve quantas falhas achou.
func audit(book *bookpdf.Book, p profile) int {
	fmt.Printf("\n══ TABELA %s (%s) %s\n", p.table, p.key, strings.Repeat("═", 40))
	table := book.LootRows(p.tablePage, p.diceX, p.nameX, p.priceX, slack)

	// ÂNCORA 1 contra si mesma, antes de o catálogo ser mencionado.
	if gap := bookpdf.RangesThatDontClose(table); gap != "" {
		fmt.Printf("A TABELA %s NÃO FECHA: %s\n", p.table, gap)
		fmt.Println("Nada do que vem abaixo vale — o auditor leu a página errado.")
		return 1
	}

	// ÂNCORA 1 contra ÂNCORA 2.
	names := make([]string, len(table))
	for i, r := range table {
		names[i] = r.Name
	}
	entries := book.EntriesByName(p.entryPages, p.proseColumns, p.tableRegion, names, p.entriesEnd, pdfOffset, slack)
	var failures []string
	for _, r := range table {
		if _, ok := entries.ByName[r.Name]; !ok {
			failures = append(failures, fmt.Sprintf("%s: está na tabela %s e não tem verbete", r.Name, p.table))
		}
	}

	pieces := knownPieces(transcript.Items)
	byName := map[string]transcript.Item{}
	for _, it := range transcript.Items {
		byName[bookpdf.Key(it.Name)] = it
	}

	fmt.Printf("%-26s %7s %10s  catálogo\n", "item", "d%", "preço")
	transcribed := 0
	for _, r := range table {
		span := strconv.Itoa(r.Start)
		if r.End != r.Start {
			span = fmt.Sprintf("%d-%d", r.Start, r.End)
		}
		entry, ok := byName[bookpdf.Key(r.Name)]
		if !ok {
			fmt.Printf("%-26s %7s %10s  — AUSENTE\n", r.Name, span, price(r.Price))
			continue
		}
		transcribed++
		failures = append(failures, checkEntry(book, p, entry, entries.ByName[r.Name], r, pieces)...)
		fmt.Printf("%-26s %7s %10s  %s\n", r.Name, span, price(r.Price), entry.ID)
	}

	fmt.Printf("\n%d/%d transcritos\n", transcribed, len(table))
	for _, f := range failures {
		fmt.Printf("  ✗ %s\n", f)
	}
	return len(failures)
}

func price(p *int) string {
	if p == nil {
		return "None"
	}
	return strconv.Itoa(*p)
}

// checkEntry faz as quatro perguntas a um item específico já transcrito.
func checkEntry(book *bookpdf.Book, p profile, entry transcript.Item, found *bookpdf.Entry,
	row bookpdf.LootRow, pieces map[string][]candidate) []string {
	var failures []string
	name := row.Name
	if (row.Price == nil && entry.Price != 0) || (row.Price != nil && entry.Price != float64(*row.Price)) {
		failures = append(failures, fmt.Sprintf("%s: o catálogo diz T$ %s e a tabela %s diz T$ %s",
			name, pyfmt.Optional(entry.Price), p.table, price(row.Price)))
	}
	if found == nil {
		return failures
	}
	if entry.BookPage != found.Page {
		failures = append(failures, fmt.Sprintf("%s: o catálogo cita p%s e o verbete está na p%d",
			name, pyfmt.Optional(float64(entry.BookPage)), found.Page))
	}

	// A COMPOSIÇÃO, que é a razão deste auditor existir.
	_, composed := entryComposition(book, found, pieces)
	fromBook := make([]string, len(composed.pieces))
	for i, pc := range composed.pieces {
		fromBook[i] = pc.id
	}
	if manual, ok := italicThePageLost[name]; ok {
		fromBook = manual
		fmt.Printf("  ! %s: usando a composição ESCRITA À MÃO — a página perde o itálico no meio da frase (ver o comentário)\n", name)
	}
	declared := declaredComposition(entry)
	entryText := bookpdf.Join(found.Body)
	if len(fromBook) > 0 {
		// CONJUNTO e não lista: a ordem é a da PROSA e varia de verbete para
		// verbete; o catálogo guarda a composição, não a ordem.
		if !slices.Equal(sortedCopy(fromBook), sortedCopy(declared)) {
			failures = append(failures, fmt.Sprintf("%s: o livro compõe %s e o catálogo declara %s",
				name, pyfmt.List(sortedCopy(fromBook)), pyfmt.List(sortedCopy(declared))))
		}
	} else {
		// SEM ITÁLICO NÃO É SEM BASE: a azagaia dos relâmpagos continua sendo
		// uma azagaia, dita em prosa. O que se proíbe é SOBREPOSIÇÃO declarada
		// sem o livro ter escrito nenhuma.
		failures = append(failures, baseWithoutItalic(name, declared, entryText)...)
	}
	if len(composed.leftover) > 0 {
		failures = append(failures, fmt.Sprintf("%s: o itálico tem palavra que não é peça conhecida: %s", name, pyfmt.List(composed.leftover)))
	}

	ratio, _, _ := bookpdf.Coverage(entryText, entry.Description)
	if ratio < coverageFloor {
		failures = append(failures, fmt.Sprintf("%s: a descrição cobre %s do verbete (piso %s)", name, pyfmt.Percent(ratio), pyfmt.Percent(coverageFloor)))
	}
	return failures
}

// baseWithoutItalic é o que se cobra de um item que o livro não compôs em
// itálico.
func baseWithoutItalic(name string, declared []string, text string) []string {
	if len(declared) > 1 {
		return []string{fmt.Sprintf("%s: o catálogo declara %s e o verbete não escreve sobreposição nenhuma em itálico", name, pyfmt.List(declared[1:]))}
	}
	if len(declared) == 0 {
		return []string{name + ": o catálogo não diz de que item ele é feito"}
	}
	var base *transcript.Item
	for i := range transcript.Items {
		if transcript.Items[i].ID == declared[0] {
			base = &transcript.Items[i]
			break
		}
	}
	if base == nil {
		return []string{fmt.Sprintf("%s: a base %s não existe no catálogo", name, declared[0])}
	}
	if !strings.Contains(bookpdf.NormalizePhrase(text), bookpdf.NormalizePhrase(base.Name)) {
		return []string{fmt.Sprintf("%s: o catálogo diz que a base é %s, e o verbete não a nomeia", name, pyfmt.Repr(base.Name))}
	}
	return nil
}

// declaredComposition é a composição do catálogo na ORDEM em que o livro a
// escreve: base, depois o que vem em itálico depois dela.
func declaredComposition(entry transcript.Item) []string {
	c := entry.ComposedOf
	if c == nil {
		return nil
	}
	out := append([]string{c.Base}, c.Improvements...)
	if c.Material != "" {
		out = append(out, c.Material)
	}
	return append(out, c.Enchants...)
}

func sortedCopy(xs []string) []string {
	out := slices.Clone(xs)
	slices.Sort(out)
	return out
}
