// Command threatstats confere a Tabela 7-2 (p323) e os pacotes de atributos dos patamares (p322) do catálogo contra o
// livro, linha a linha e campo a campo.
//
// A tabela se lê pelas PALAVRAS com posição, agrupadas pela altura: o leitor de linhas quebra a tabela em colunas, e
// o rótulo do patamar ("Veterano") sai grudado numa linha qualquer pela diagramação — por isso ele é descartado, e
// os limites de cada patamar são os da p35, que o catálogo já guarda.
//
//	go run ./cmd/audit/threatstats
package main

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"t20engine/domain/book"
	"t20engine/domain/catalog/transcript"
	"t20engine/infra/bookpdf"
)

// pdfOffset: página do PDF = página do livro + 6.
const pdfOffset = 6

// tablePage e tiersPage são as páginas do LIVRO.
const (
	tablePage = 323
	tiersPage = 322
)

// tableRow é uma linha da tabela como o livro a escreve: "5 +18, x2 1d12+11 28 150 +14/+5 20", com o rótulo do
// patamar à frente, às vezes.
var tableRow = regexp.MustCompile(`^(?:[A-Za-zÀ-ú]+ )?(1/2|\d+) \+(\d+)(?:, x(\d))? (\d+d\d+\+\d+) (\d+) (\d+) \+(\d+)/\+(\d+) (\d+)$`)

// tierPackage é "Iniciante: 3, 2, 1, 1, 0, –1" — o menos do livro é o travessão.
var tierPackage = regexp.MustCompile(`(Iniciante|Veterano|Campeão|Lenda): ((?:[–-]?\d, ){5}[–-]?\d)`)

func main() {
	pdf, err := bookpdf.Open()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	failures := 0
	fail := func(format string, args ...any) {
		failures++
		fmt.Printf("  FALHA "+format+"\n", args...)
	}

	fromBook := bookRows(pdf)
	catalog := map[float64]transcript.ThreatStatsRow{}
	for _, r := range transcript.ThreatStats.Rows {
		catalog[r.ND] = r
	}
	fmt.Printf("TABELA 7-2       livro: %d linhas · catálogo: %d linhas\n", len(fromBook), len(catalog))
	// O DENOMINADOR: o ND 1/2 e os vinte inteiros. Uma leitura que perdesse linhas conferiria só as que achou.
	if len(fromBook) != 21 {
		fail("o livro rendeu %d linhas, e a Tabela 7-2 tem 21 (ND 1/2 e 1 a 20): defeito de LEITURA", len(fromBook))
	}
	for _, b := range fromBook {
		c, ok := catalog[b.ND]
		if !ok {
			fail("ND %s está no livro e não no catálogo", book.CRWritten(b.ND))
			continue
		}
		if c != b {
			fail("ND %s: o catálogo diz %+v e o livro %+v", book.CRWritten(b.ND), c, b)
		}
		delete(catalog, b.ND)
	}
	for nd := range catalog {
		fail("ND %s está no catálogo e não no livro", book.CRWritten(nd))
	}

	packages := bookPackages(pdf)
	fmt.Printf("PATAMARES        livro: %d pacotes · catálogo: %d patamares\n", len(packages), len(transcript.ThreatStats.Tiers))
	if len(packages) != 4 {
		fail("o livro rendeu %d pacotes de atributos, e a p322 tem 4", len(packages))
	}
	for _, tier := range transcript.ThreatStats.Tiers {
		want, ok := packages[tier.Name]
		if !ok {
			fail("o patamar %s não tem pacote no livro", tier.Name)
			continue
		}
		if fmt.Sprint(want) != fmt.Sprint(tier.Attributes) {
			fail("o pacote do %s: o catálogo diz %v e o livro %v", tier.Name, tier.Attributes, want)
		}
	}

	fmt.Printf("\nFALHAS: %d   (medidos: %d linhas da tabela, %d pacotes)\n", failures, len(fromBook), len(packages))
	if failures > 0 {
		os.Exit(1)
	}
}

// bookRows são as linhas da Tabela 7-2, pelas palavras agrupadas na mesma altura.
func bookRows(book *bookpdf.Book) []transcript.ThreatStatsRow {
	byY := map[float64][]bookpdf.Positioned{}
	for _, w := range book.Words(tablePage + pdfOffset) {
		y := math.Round(w.Y)
		byY[y] = append(byY[y], w)
	}
	ys := make([]float64, 0, len(byY))
	for y := range byY {
		ys = append(ys, y)
	}
	sort.Float64s(ys)
	var out []transcript.ThreatStatsRow
	for _, y := range ys {
		words := byY[y]
		sort.Slice(words, func(i, j int) bool { return words[i].X < words[j].X })
		texts := make([]string, len(words))
		for i, w := range words {
			texts[i] = w.Text
		}
		m := tableRow.FindStringSubmatch(strings.Join(texts, " "))
		if m == nil {
			continue
		}
		nd := 0.5
		if m[1] != "1/2" {
			nd = float64(number(m[1]))
		}
		attacks := 1
		if m[3] != "" {
			attacks = number(m[3])
		}
		out = append(out, transcript.ThreatStatsRow{
			ND: nd, Attack: number(m[2]), Attacks: attacks, Damage: m[4], Defense: number(m[5]), HP: number(m[6]),
			FocusedSkills: number(m[7]), OtherSkills: number(m[8]), DC: number(m[9]),
		})
	}
	return out
}

// bookPackages são os pacotes de atributos dos quatro patamares, pelo nome.
func bookPackages(book *bookpdf.Book) map[string][]int {
	out := map[string][]int{}
	text := strings.Join(book.PageLines(tiersPage+pdfOffset), " ")
	for _, m := range tierPackage.FindAllStringSubmatch(text, -1) {
		var values []int
		for _, v := range strings.Split(m[2], ", ") {
			values = append(values, number(strings.ReplaceAll(v, "–", "-")))
		}
		out[m[1]] = values
	}
	return out
}

func number(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		panic(fmt.Sprintf("o número %q do livro não é inteiro: %v", s, err))
	}
	return n
}
