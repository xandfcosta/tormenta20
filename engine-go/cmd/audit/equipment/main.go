// Command equipment confere o EQUIPAMENTO do app contra o livro, item por item
// (ALE-393).
//
// Rodar é ATO DELIBERADO, como o `genoracle` e os auditores irmãos: a
// ferramenta PROPÕE, o diff se revisa contra o PDF, e é a revisão que decide.
// Ela não escreve.
//
// As âncoras são o NOME e o PREÇO: o capítulo 3 põe todo item numa tabela —
// Armas (p144-145), Armaduras & Escudos (p153), Itens Gerais (p156-157),
// Melhorias (p165) —, e toda tabela traz o preço que o catálogo também afirma.
// Item que não aparece em tabela nenhuma sai como NÃO MEDIDO, que não é a mesma
// coisa que "correto".
//
// A armadilha: as tabelas de Itens Gerais são de duas colunas, e uma linha traz
// dois itens lado a lado — `['Óleo', 'T$ 0,1', '0,5', 'Tabardo', 'T$ 10', '1']`.
// Ler o primeiro nome e o primeiro preço da linha audita SÓ a coluna da
// esquerda; o bloco de um item vai do NOME dele até o próximo nome.
//
//	go run ./cmd/audit/equipment
package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"t20engine/domain/catalog/transcript"
	"t20engine/infra/bookpdf"
)

// pdfOffset: página do PDF = página do livro + 6.
const pdfOffset = 6

// itemTable é uma tabela de ITEM do capítulo 3, com as páginas do LIVRO. Ficam
// de fora a 3-1 (dinheiro inicial), a 3-2 (dano por tamanho), a 3-4 (munições),
// a 3-7 e a 3-9 (tabelas de PREÇO de melhoria e material, que não são verbetes).
type itemTable struct {
	title string
	pages []int
	label string
}

var tables = []itemTable{
	{"Tabela 3-3", []int{144, 145}, "Armas"},
	{"Tabela 3-5", []int{153}, "Armaduras & Escudos"},
	{"Tabela 3-6", []int{156, 157}, "Itens Gerais"},
	{"Tabela 3-8", []int{165}, "Melhorias"},
}

var bookPriceRe = regexp.MustCompile(`^T\$\s*([\d.,]+)$`)

// cell é uma célula de linha de tabela: o x e o texto.
type cell struct {
	x    float64
	text string
}

// divergence é um item cujo preço o livro diz outro.
type divergence struct {
	table, name   string
	catalog, book float64
}

func main() {
	flag.Parse()
	book, err := bookpdf.Open()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	items, elsewhere := catalogByKey()
	seen := map[string]bool{}
	checked := 0
	var divergent []divergence

	for _, t := range tables {
		priced, unpriced := 0, 0
		for _, block := range blocksOfTable(book, t.pages, t.title, items) {
			k := nameKey(block[0].text)
			seen[k] = true
			price, ok := firstBookPrice(block)
			// Uma linha da tabela pode ser de VÁRIOS verbetes — as cinco Coleções de livros são uma linha só —, e o
			// preço dela vale para cada um.
			for _, item := range items[k] {
				if !ok {
					unpriced++
					continue
				}
				priced++
				checked++
				if math.Abs(price-item.Price) > 0.001 {
					divergent = append(divergent, divergence{t.label, item.Name, item.Price, price})
				}
			}
		}
		fmt.Printf("  %-22s preço conferido: %3d   sem preço na tabela: %d\n", t.label, priced, unpriced)
	}

	var unseen []string
	mine := 0
	for k, group := range items {
		mine += len(group)
		if seen[k] {
			continue
		}
		for _, it := range group {
			unseen = append(unseen, it.Name)
		}
	}
	sort.Strings(unseen)
	// A SOMA ANTES DAS PARCELAS: os itens deste auditor e os dos irmãos têm de dar o catálogo inteiro. Um verbete
	// que caísse fora das duas contas — um mapa de um item por nome engole os outros do mesmo nome — sumiria sem
	// virar "não medido", e o relatório sairia limpo com a medida furada.
	others := 0
	for _, names := range elsewhere {
		others += len(names)
	}
	if mine+others != len(transcript.Items) {
		fmt.Fprintf(os.Stderr, "a conta não fecha: %d deste auditor + %d dos irmãos, e o catálogo tem %d itens\n",
			mine, others, len(transcript.Items))
		os.Exit(1)
	}
	fmt.Printf("\nitens no catálogo:        %d\n", len(transcript.Items))
	fmt.Printf("  das tabelas do cap. 3:  %d\n", mine)
	fmt.Printf("  com preço conferido:    %d\n", checked)
	fmt.Printf("  NÃO MEDIDOS:            %d\n", len(unseen))
	fmt.Printf("  divergem do livro:      %d\n", len(divergent))
	for _, d := range divergent {
		name := []rune(d.name)
		if len(name) > 28 {
			name = name[:28]
		}
		fmt.Printf("    PREÇO  %-20s %-30s catálogo=%s livro=%s\n", d.table, string(name), optionalNumber(d.catalog), pyFloat(d.book))
	}
	if len(unseen) > 0 {
		fmt.Println("\nNÃO MEDIDOS (do capítulo 3, e não aparecem em tabela nenhuma dele):")
		for _, n := range unseen {
			fmt.Printf("    %s\n", n)
		}
	}
	// O QUE É DE OUTRO AUDITOR sai contado, e não somado aos não medidos: "não medido" quer dizer "deste capítulo e
	// sem âncora". O "nenhum" é o item que auditor nenhum alcança, e esse sai pelo nome.
	auditors := make([]string, 0, len(elsewhere))
	for auditor := range elsewhere {
		auditors = append(auditors, auditor)
	}
	sort.Strings(auditors)
	fmt.Println("\nde outro auditor:")
	for _, auditor := range auditors {
		fmt.Printf("    %-16s %d\n", auditor, len(elsewhere[auditor]))
	}
	if orphans := elsewhere["nenhum"]; len(orphans) > 0 {
		sort.Strings(orphans)
		fmt.Println("\nDE AUDITOR NENHUM:")
		for _, n := range orphans {
			fmt.Printf("    %s\n", n)
		}
	}
}

// catalogByKey são os itens DESTE auditor pelo nome comparável (`nameKey`), e cada nome guarda TODOS os seus
// verbetes: sem o qualificador, as cinco Coleções de livros têm um nome só, e um mapa de um item por nome mediria
// uma e esqueceria quatro. O resto vai pelo nome, agrupado pelo auditor que o mede (`measuredBy`).
func catalogByKey() (map[string][]*transcript.Item, map[string][]string) {
	mine := map[string][]*transcript.Item{}
	elsewhere := map[string][]string{}
	for i := range transcript.Items {
		it := &transcript.Items[i]
		if auditor := measuredBy(*it); auditor != "" {
			elsewhere[auditor] = append(elsewhere[auditor], it.Name)
			continue
		}
		k := nameKey(it.Name)
		mine[k] = append(mine[k], it)
	}
	return mine, elsewhere
}

func firstBookPrice(block []cell) (float64, bool) {
	for _, c := range block {
		m := bookPriceRe.FindStringSubmatch(c.text)
		if m == nil {
			continue
		}
		v, err := strconv.ParseFloat(strings.ReplaceAll(strings.ReplaceAll(m[1], ".", ""), ",", "."), 64)
		if err != nil {
			panic(fmt.Sprintf("preço do livro %q não é número no formato \"T$ 1.234,5\": %v", c.text, err))
		}
		return v, true
	}
	return 0, false
}

// blocksOfTable é um bloco por ITEM: do nome dele até o próximo nome da mesma
// linha, abaixo do título da tabela.
func blocksOfTable(book *bookpdf.Book, pages []int, title string, items map[string][]*transcript.Item) [][]cell {
	var out [][]cell
	for _, page := range pages {
		lines := book.PositionedLines(page + pdfOffset)
		titleY, found := 0.0, false
		for _, l := range lines {
			if strings.HasPrefix(l.Text, title) {
				titleY, found = l.Y, true
				break
			}
		}
		if !found {
			continue
		}
		// O y agrupa pelo `round()` do Python, que arredonda o meio para o PAR.
		byY := map[float64][]cell{}
		for _, l := range lines {
			if l.Y > titleY {
				y := math.RoundToEven(l.Y)
				byY[y] = append(byY[y], cell{l.X, l.Text})
			}
		}
		ys := make([]float64, 0, len(byY))
		for y := range byY {
			ys = append(ys, y)
		}
		sort.Float64s(ys)
		for _, y := range ys {
			cells := byY[y]
			sort.SliceStable(cells, func(i, j int) bool {
				if cells[i].x != cells[j].x {
					return cells[i].x < cells[j].x
				}
				return cells[i].text < cells[j].text
			})
			var starts []int
			for i, c := range cells {
				if k := nameKey(c.text); k != "" && len(items[k]) > 0 {
					starts = append(starts, i)
				}
			}
			for n, i := range starts {
				end := len(cells)
				if n+1 < len(starts) {
					end = starts[n+1]
				}
				out = append(out, cells[i:end])
			}
		}
	}
	return out
}

// optionalNumber é como o Python imprime o `get('price')`: no dado tipado
// "ausente" e "0" são o mesmo, e o ausente é o que o Python via — `None`. O
// preço inteiro chega do JSON como `int`, e o fracionário como `float`.
func optionalNumber(x float64) string {
	if x == 0 {
		return "None"
	}
	if x == math.Trunc(x) {
		return strconv.FormatFloat(x, 'f', 0, 64)
	}
	return pyFloat(x)
}

// pyFloat é o `repr()` de um float do Python: os dígitos mais curtos que voltam
// ao mesmo número, em notação fixa entre 1e-4 e 1e16 (com `.0` quando inteiro),
// e científica fora disso.
func pyFloat(x float64) string {
	sci := strconv.FormatFloat(x, 'e', -1, 64)
	mantissa, expText, _ := strings.Cut(sci, "e")
	exp, _ := strconv.Atoi(expText)
	if exp < -4 || exp >= 16 {
		sign := "+"
		if exp < 0 {
			sign, exp = "-", -exp
		}
		return fmt.Sprintf("%se%s%02d", mantissa, sign, exp)
	}
	fixed := strconv.FormatFloat(x, 'f', -1, 64)
	if !strings.Contains(fixed, ".") {
		fixed += ".0"
	}
	return fixed
}

// As páginas do capítulo 3 no LIVRO: o que está fora delas é de outro auditor.
const (
	chapterThreeFirst = 138
	chapterThreeLast  = 167
)

// measuredBy é o auditor que mede o item: "" é este, o das tabelas do capítulo 3; o resto é o nome do irmão em
// `cmd/audit/`, e "nenhum" é o item que nenhum auditor alcança — o achado. A página decide primeiro, porque o
// capítulo 3 também tem `apparel` e o capítulo 8 também tem arma; dentro do capítulo 8, o tipo do verbete diz qual
// tabela o tem, que é como cada irmão escolhe os seus.
//
//	measuredBy(transcript.Item{Category: "material", BookPage: 166}) // "materials"
func measuredBy(it transcript.Item) string {
	if it.BookPage >= chapterThreeFirst && it.BookPage <= chapterThreeLast {
		if it.Category == "material" {
			return "materials"
		}
		return ""
	}
	switch {
	case it.Category == "weapon-enchant" || it.Category == "armor-enchant":
		return "enchants"
	case it.Artifact:
		return "artifacts"
	case it.ComposedOf != nil:
		return "specificitems"
	case it.Consumable != nil && it.Consumable.Spell != "":
		return "potions"
	case it.Category == "apparel":
		return "accessories"
	}
	return "nenhum"
}

// qualifier é o qualificador entre parênteses no fim do nome — "(10 m)", "(Guerra)", "(por dia)".
var qualifier = regexp.MustCompile(`\s*\([^)]*\)\s*$`)

// nameKey é o nome comparável, SEM o qualificador dos dois lados: o catálogo o guarda porque ele importa na mesa
// ("Corda (10 m)", decisão do dono), e a tabela do livro tem o seu ("Ração de viagem (por dia)"). Comparar com ele
// deixava esses itens sem medida. Uma célula só de qualificador vira chave vazia, e chave vazia não casa nada.
//
//	nameKey("Corda (10 m)") == nameKey("Corda") // true
func nameKey(name string) string {
	return bookpdf.Key(qualifier.ReplaceAllString(name, ""))
}
