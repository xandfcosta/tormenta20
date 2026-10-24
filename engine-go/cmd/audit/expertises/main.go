// Command expertises confere as perícias contra o Capítulo 2 do livro
// (p115-123). Seção da ALE-391.
//
// Mede os quatro campos que o catálogo afirma por perícia — atributo, só
// treinada, penalidade de armadura e página —, e não a prosa dos usos.
//
// São DUAS leituras independentes, e elas têm de concordar ANTES de se comparar
// com o catálogo: a Tabela 2-1 (p115, coluna da direita, células lidas por Y) e
// a linha de estado de cada VERBETE (`Des • Treinada • Armadura`, à direita do
// nome), que dá também a página. Um instrumento que passou a ler errado
// discorda do outro, com o nome da perícia.
//
// O estado se liga ao título por PALAVRA, e em três condições, cada uma paga:
//
//   - por palavra, não por linha: na p119 um `<line>` traz `Furtividade Des •
//     Armadura Iniciativa` — título, estado, e o título da coluna VIZINHA;
//
//   - na mesma COLUNA: sem isso o estado de Religião liga em Percepção;
//
//   - sem distância fixa título→estado: a largura da coluna varia (111pt na
//     p115, 195pt na p116).
//
//     go run ./cmd/audit/expertises
package main

import (
	"fmt"
	"math"
	"os"
	"slices"
	"sort"
	"strconv"
	"t20engine/cmd/internal/pyfmt"

	"t20engine/domain/catalog/transcript"
	"t20engine/infra/bookpdf"
)

// firstPage e lastPage são o capítulo no PDF; o livro é PDF − 6.
const (
	firstPage = 121
	lastPage  = 129
)

const pdfOffset = 6

const tablePage = 121

// tableStartsAtX: a Tabela 2-1 divide a p115 com o primeiro verbete.
const tableStartsAtX = 300

var abbreviations = map[string]string{
	"For": "strength", "Des": "dexterity", "Con": "constitution",
	"Int": "intelligence", "Sab": "wisdom", "Car": "charisma",
}

var marks = []string{"Treinada", "Armadura"}

// sameHeight é a folga de y: a marca desce um fio em relação à abreviação.
const sameHeight = 4.0

// O título nasce na mesma altura do estado; a folga para baixo cabe o título de
// seção da p115, que empurra o primeiro verbete.
const (
	slackAbove = 30.0
	slackBelow = 5.0
)

const minLinesPerColumn = 5

// maxIndent: acima disto é outra coluna, abaixo é recuo de parágrafo.
const maxIndent = 40.0

// reading é o que uma das duas leituras diz de uma perícia. `attribute` "" é o
// `None` do script de origem: abreviação que a tabela não reconhece.
type reading struct {
	attribute    string
	trainedOnly  bool
	armorPenalty bool
	bookPage     int
}

// columnStartsByLine são os x onde as colunas começam, por frequência de LINHA.
// Contando linhas aparece um x que não é coluna — o RECUO de primeira linha,
// 17pt à direita —, e fundir o que está a menos de `maxIndent` de um começo já
// aceito é o que separa recuo de coluna.
func columnStartsByLine(lines []bookpdf.Positioned) []float64 {
	count := map[float64]int{}
	for _, l := range lines {
		count[bookpdf.Round1(l.X)]++
	}
	var frequent []float64
	for x, n := range count {
		if n >= minLinesPerColumn {
			frequent = append(frequent, x)
		}
	}
	sort.Float64s(frequent)
	var starts []float64
	for _, x := range frequent {
		if len(starts) == 0 || x-starts[len(starts)-1] > maxIndent {
			starts = append(starts, x)
		}
	}
	if len(starts) > 0 {
		return starts
	}
	least := 0.0
	for i, l := range lines {
		if i == 0 || l.X < least {
			least = l.X
		}
	}
	return []float64{least}
}

type cell struct {
	x    float64
	text string
}

// readTable lê a Tabela 2-1, célula por célula agrupada por Y: as colunas são
// blocos DIFERENTES, e o que diz que "Acrobacia", "Des", "—" e "sim" são a
// mesma linha é o y das quatro ser igual. Devolve também as alturas que ela
// ocupa.
func readTable(book *bookpdf.Book, names map[string]string) (map[string]reading, map[float64]bool) {
	rows := map[float64][]cell{}
	var order []float64
	for _, l := range book.PositionedLines(tablePage) {
		if l.X <= tableStartsAtX {
			continue
		}
		y := bookpdf.Round1(l.Y)
		if _, seen := rows[y]; !seen {
			order = append(order, y)
		}
		rows[y] = append(rows[y], cell{l.X, l.Text})
	}
	out, heights := map[string]reading{}, map[float64]bool{}
	for _, y := range order {
		cells := rows[y]
		sort.Slice(cells, func(i, j int) bool {
			if cells[i].x != cells[j].x {
				return cells[i].x < cells[j].x
			}
			return cells[i].text < cells[j].text
		})
		id, ok := names[bookpdf.Key(cells[0].text)]
		if !ok || len(cells) != 4 {
			continue
		}
		out[id] = reading{abbreviations[cells[1].text], cells[2].text == "sim", cells[3].text == "sim", 0}
		heights[y] = true
	}
	return out, heights
}

func nearAny(y float64, heights map[float64]bool) bool {
	for h := range heights {
		if math.Abs(y-h) < sameHeight {
			return true
		}
	}
	return false
}

// readEntries lê a linha de estado de cada verbete: a abreviação é a semente, as
// marcas vêm à direita na mesma altura, e o título é a palavra de nome de
// perícia mais próxima À ESQUERDA, na MESMA COLUNA.
func readEntries(book *bookpdf.Book, names map[string]string, tableHeights map[float64]bool) (map[string]reading, []string) {
	out := map[string]reading{}
	var complaints []string
	for page := firstPage; page <= lastPage; page++ {
		// As colunas se acham na PROSA: as colunas internas da Tabela 2-1 poriam
		// o título do Adestramento, logo abaixo dela, noutra coluna.
		var prose []bookpdf.Positioned
		for _, l := range book.PositionedLines(page) {
			if !nearAny(l.Y, tableHeights) {
				prose = append(prose, l)
			}
		}
		starts := columnStartsByLine(prose)
		words := book.Words(page)
		for _, w := range words {
			attribute, ok := abbreviations[w.Text]
			if !ok {
				continue
			}
			if page == tablePage && nearAny(w.Y, tableHeights) {
				continue // célula da Tabela 2-1, já lida pelo outro instrumento
			}
			found := map[string]bool{}
			column := bookpdf.ColumnOf(starts, w.X)
			var candidates []bookpdf.Positioned
			for _, p := range words {
				if slices.Contains(marks, p.Text) && p.X > w.X && math.Abs(p.Y-w.Y) < sameHeight {
					found[p.Text] = true
				}
				if _, isName := names[bookpdf.Key(p.Text)]; isName && p.X < w.X &&
					bookpdf.ColumnOf(starts, p.X) == column && w.Y-slackAbove <= p.Y && p.Y <= w.Y+slackBelow {
					candidates = append(candidates, p)
				}
			}
			if len(candidates) == 0 {
				complaints = append(complaints, fmt.Sprintf("estado %s na p%d sem título de perícia à esquerda na mesma coluna",
					pyfmt.Repr(w.Text), page-pdfOffset))
				continue
			}
			title := candidates[0]
			for _, c := range candidates[1:] {
				if math.Abs(c.Y-w.Y) < math.Abs(title.Y-w.Y) {
					title = c
				}
			}
			out[names[bookpdf.Key(title.Text)]] = reading{attribute, found["Treinada"], found["Armadura"], page - pdfOffset}
		}
	}
	return out, complaints
}

func main() {
	book, err := bookpdf.Open()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	names := map[string]string{}
	for _, e := range transcript.Expertises {
		names[bookpdf.Key(e.Name)] = e.ID
	}
	table, heights := readTable(book, names)
	entries, complaints := readEntries(book, names, heights)

	// O CONTROLE: os dois instrumentos antes do catálogo.
	var both []string
	for id := range table {
		if _, ok := entries[id]; ok {
			both = append(both, id)
		}
	}
	sort.Strings(both)
	disagreements := 0
	for _, id := range both {
		for _, f := range fieldsOf(table[id], entries[id]) {
			if f.a != f.b {
				fmt.Printf("  INSTRUMENTOS DISCORDAM %s.%s: tabela %s × verbete %s\n", id, f.name, f.a, f.b)
				disagreements++
			}
		}
	}

	diverge := 0
	for _, e := range transcript.Expertises {
		read, ok := entries[e.ID]
		if !ok {
			complaints = append(complaints, e.Name+": sem linha de estado no capítulo")
			continue
		}
		catalog := reading{e.Attribute, e.TrainedOnly, e.ArmorPenalty, e.BookPage}
		for _, f := range append(fieldsOf(catalog, read), field{"bookPage", strconv.Itoa(catalog.bookPage), strconv.Itoa(read.bookPage)}) {
			if f.a != f.b {
				fmt.Printf("  %s: %s: catálogo %s × livro %s\n", e.Name, f.name, f.a, f.b)
				diverge++
			}
		}
		if _, ok := table[e.ID]; !ok {
			complaints = append(complaints, e.Name+": sem linha na Tabela 2-1")
		}
	}

	for _, c := range complaints {
		fmt.Printf("  NÃO MEDIDO %s\n", c)
	}
	fmt.Printf("\nperícias: %d | lidas na Tabela 2-1: %d | lidas no verbete: %d | instrumentos discordam: %d | divergem do livro: %d\n",
		len(transcript.Expertises), len(table), len(entries), disagreements, diverge)
	if diverge > 0 || disagreements > 0 || len(complaints) > 0 {
		os.Exit(1)
	}
}

// field é um campo comparado, já no texto em que o Python o imprimia.
type field struct{ name, a, b string }

// fieldsOf são os três campos que as duas leituras e o catálogo afirmam.
func fieldsOf(a, b reading) []field {
	return []field{
		{"attribute", pyOptionalStr(a.attribute), pyOptionalStr(b.attribute)},
		{"soTreinada", pyfmt.Bool(a.trainedOnly), pyfmt.Bool(b.trainedOnly)},
		{"penalidadeDeArmadura", pyfmt.Bool(a.armorPenalty), pyfmt.Bool(b.armorPenalty)},
	}
}
