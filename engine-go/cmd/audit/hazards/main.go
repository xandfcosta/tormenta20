// Command hazards confere os perigos do catálogo (p317-319) contra o livro: as armadilhas campo a campo, as doenças
// pelo contágio, pela CD e pelos efeitos, e o ambiente pelos dados e pelas páginas que a paráfrase cita.
//
// As armadilhas se leem pela FICHA, que o livro escreve numa frase só terminada em "ND x." — cortar o texto nesses
// pontos dá uma ficha por pedaço, na ordem do livro. A armadilha MÁGICA é a do nome em itálico (p317), e o itálico
// vem do PDF em pedaços: a capitular sai numa altura e o resto noutra, então os pedaços da mesma faixa de altura se
// juntam pela posição horizontal antes de comparar.
//
//	go run ./cmd/audit/hazards
package main

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"t20engine/domain/catalog/transcript"
	"t20engine/infra/bookpdf"
)

// pdfOffset: página do PDF = página do livro + 6.
const pdfOffset = 6

var (
	trapEnd   = regexp.MustCompile(`; ND (1/4|1/2|\d+)\.`)
	saveRe    = regexp.MustCompile(`(Reflexos|Fortitude|Vontade) CD (\d+)`)
	searchOne = regexp.MustCompile(`Investigação/ ?Ladinagem CD (\d+)`)
	searchTwo = regexp.MustCompile(`Investigação CD (\d+), Ladinagem CD (\d+)`)
	diceRe    = regexp.MustCompile(`\d+d\d+(?:\+\d+)?`)
	citedPage = regexp.MustCompile(`página \d+`)
	diseaseRe = regexp.MustCompile(`(Contato|Inalação), CD (\d+), vítima (?:fica )?([^.]*)\.`)
)

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
	byKind := map[string][]transcript.Hazard{}
	for _, h := range transcript.Hazards {
		byKind[h.Kind] = append(byKind[h.Kind], h)
	}

	traps := auditTraps(pdf, byKind["armadilha"], fail)
	diseases := auditDiseases(pdf, byKind["doenca"], fail)
	ambient := auditAmbient(pdf, byKind["ambiente"], fail)

	fmt.Printf("\nFALHAS: %d   (medidos: %d armadilhas, %d doenças, %d do ambiente)\n", failures, traps, diseases, ambient)
	if failures > 0 {
		os.Exit(1)
	}
}

// runningFoot é o cabeçalho corrido do pé da página ("Ameaças 317", "Capítulo Sete 318"), que a leitura cola no
// meio da ficha que atravessa a página.
var runningFoot = regexp.MustCompile(`(?:Ameaças|Capítulo Sete) \d{3}`)

// pageText é o texto da página do livro em ordem de leitura, sem o rodapé.
func pageText(pdf *bookpdf.Book, bookPage int) string {
	var kept []string
	for _, l := range pdf.PageLines(bookPage + pdfOffset) {
		if !strings.Contains(l, "@") {
			kept = append(kept, l)
		}
	}
	return bookpdf.Collapse(runningFoot.ReplaceAllString(bookpdf.Join(kept), ""))
}

func auditTraps(pdf *bookpdf.Book, catalog []transcript.Hazard, fail func(string, ...any)) int {
	text := pageText(pdf, 317) + " " + pageText(pdf, 318)
	start := strings.Index(text, "nível de desafio de uma criatura.")
	end := strings.Index(text, "Clima, calor e Frio")
	if start < 0 || end < start {
		fail("não achei a seção das armadilhas na p317-318: defeito de LEITURA")
		return 0
	}
	section := text[start+len("nível de desafio de uma criatura.") : end]
	magical := magicalNames(pdf)

	measured := 0
	cursor := 0
	for _, m := range trapEnd.FindAllStringSubmatchIndex(section, -1) {
		entry := strings.TrimSpace(section[cursor:m[0]])
		nd := section[m[2]:m[3]]
		cursor = m[1]
		trap, found := trapOf(entry, catalog)
		if !found {
			fail("a ficha %q do livro não casa nenhuma armadilha do catálogo", firstWords(entry))
			continue
		}
		measured++
		t := trap.Trap
		if ndOf(nd) != t.ND {
			fail("%s: ND %v no catálogo, %s no livro", trap.Name, t.ND, nd)
		}
		if save := saveRe.FindStringSubmatch(entry); save != nil {
			if t.Save == nil || t.Save.Test != save[1] || fmt.Sprint(t.Save.DC) != save[2] {
				fail("%s: a resistência do livro é %s CD %s, e o catálogo diz %+v", trap.Name, save[1], save[2], t.Save)
			}
		} else if t.Save != nil {
			fail("%s: o catálogo tem resistência e o livro não", trap.Name)
		}
		inv, thv := searchCDs(entry)
		if inv != t.Investigation || thv != t.Thievery {
			fail("%s: Investigação %d e Ladinagem %d no livro, %d e %d no catálogo", trap.Name, inv, thv, t.Investigation, t.Thievery)
		}
		for _, d := range t.Damage {
			if diceRe.MatchString(d.Dice) && !strings.Contains(entry, d.Dice) {
				fail("%s: o dado %s do catálogo não está na ficha do livro", trap.Name, d.Dice)
			}
		}
		if italic := inItalic(magical, trap.Name); italic != t.Magical {
			fail("%s: mágica no livro (nome em itálico) %v, no catálogo %v", trap.Name, italic, t.Magical)
		}
	}
	fmt.Printf("ARMADILHAS       livro: %d fichas · catálogo: %d\n", measured, len(catalog))
	// O DENOMINADOR: uma leitura que perdesse fichas conferiria só as que achou.
	if measured != len(catalog) {
		fail("o livro rendeu %d fichas de armadilha e o catálogo tem %d", measured, len(catalog))
	}
	return measured
}

// trapOf é a armadilha cujo nome abre a ficha. A comparação é pela chave, porque o nome mágico sai do PDF com a
// capitular separada ("S ímbolo do M edo").
func trapOf(entry string, catalog []transcript.Hazard) (transcript.Hazard, bool) {
	key := bookpdf.Key(entry)
	best, found := transcript.Hazard{}, false
	for _, h := range catalog {
		if strings.HasPrefix(key, bookpdf.Key(h.Name)) && len(h.Name) > len(best.Name) {
			best, found = h, true
		}
	}
	return best, found
}

func searchCDs(entry string) (int, int) {
	if m := searchTwo.FindStringSubmatch(entry); m != nil {
		return number(m[1]), number(m[2])
	}
	if m := searchOne.FindStringSubmatch(entry); m != nil {
		return number(m[1]), number(m[1])
	}
	return 0, 0
}

// inItalic diz se o nome está numa das linhas em itálico. CONTÉM, e não igual: a faixa de altura junta também o
// itálico da outra coluna que cai na mesma linha (o "Sono" da magia citada), e a chave da linha passa do nome.
func inItalic(rows []string, name string) bool {
	key := bookpdf.Key(name)
	for _, row := range rows {
		if strings.Contains(row, key) {
			return true
		}
	}
	return false
}

// magicalNames são as linhas em itálico das p317-318, pela chave: os pedaços da mesma faixa de altura juntos pela
// posição horizontal.
func magicalNames(pdf *bookpdf.Book) []string {
	var out []string
	for _, page := range []int{317, 318} {
		// AGRUPADOS POR PROXIMIDADE, e não por arredondamento: a capitular fica uns pontos acima do resto, e um
		// arredondamento corta o par ao meio sempre que os dois caem dos dois lados de uma fronteira.
		var rows [][]bookpdf.Positioned
		for _, it := range pdf.Italics(page + pdfOffset) {
			if n := len(rows); n > 0 && it.Y-rows[n-1][len(rows[n-1])-1].Y <= 6 {
				rows[n-1] = append(rows[n-1], it)
				continue
			}
			rows = append(rows, []bookpdf.Positioned{it})
		}
		for _, row := range rows {
			sort.Slice(row, func(i, j int) bool { return row[i].X < row[j].X })
			var text strings.Builder
			for _, it := range row {
				text.WriteString(it.Text)
			}
			out = append(out, bookpdf.Key(text.String()))
		}
	}
	return out
}

func auditDiseases(pdf *bookpdf.Book, catalog []transcript.Hazard, fail func(string, ...any)) int {
	text := pageText(pdf, 318)
	measured := 0
	for _, h := range catalog {
		at := strings.Index(text, h.Name+".")
		if at < 0 {
			fail("a doença %s não está na p318", h.Name)
			continue
		}
		m := diseaseRe.FindStringSubmatch(text[at:])
		if m == nil {
			fail("%s: não achei o contágio e a CD na ficha do livro", h.Name)
			continue
		}
		measured++
		d := h.Disease
		if !strings.EqualFold(m[1], d.Contagion) || fmt.Sprint(d.DC) != m[2] {
			fail("%s: o livro diz %s CD %s, o catálogo %s CD %d", h.Name, m[1], m[2], d.Contagion, d.DC)
		}
		if steps := strings.Count(m[3], "/") + 1; steps != len(d.Effects) {
			fail("%s: o livro tem %d efeitos (%s), o catálogo %d", h.Name, steps, m[3], len(d.Effects))
		}
	}
	fmt.Printf("DOENÇAS          medidas: %d de %d\n", measured, len(catalog))
	return measured
}

func auditAmbient(pdf *bookpdf.Book, catalog []transcript.Hazard, fail func(string, ...any)) int {
	measured := 0
	texts := map[int]string{}
	for _, h := range catalog {
		if _, ok := texts[h.BookPage]; !ok {
			texts[h.BookPage] = pageText(pdf, h.BookPage)
		}
		page := texts[h.BookPage]
		if !strings.Contains(bookpdf.Key(page), bookpdf.Key(h.Name)) {
			fail("%s não está na p%d", h.Name, h.BookPage)
			continue
		}
		measured++
		for _, d := range diceRe.FindAllString(h.Description, -1) {
			if !strings.Contains(page, d) {
				fail("%s: a paráfrase cita %s, e a p%d não tem esse dado", h.Name, d, h.BookPage)
			}
		}
		for _, cited := range citedPage.FindAllString(h.Description, -1) {
			if !strings.Contains(page, cited) {
				fail("%s: a paráfrase diz %q, e a p%d não cita essa página", h.Name, cited, h.BookPage)
			}
		}
	}
	fmt.Printf("AMBIENTE         medidos: %d de %d\n", measured, len(catalog))
	return measured
}

func ndOf(s string) float64 {
	switch s {
	case "1/4":
		return 0.25
	case "1/2":
		return 0.5
	}
	return float64(number(s))
}

func number(s string) int {
	n, err := bookpdf.Num(s)
	if err != nil {
		panic(fmt.Sprintf("o número %q do livro não é inteiro: %v", s, err))
	}
	return n
}

func firstWords(s string) string {
	words := strings.Fields(s)
	if len(words) > 5 {
		words = words[:5]
	}
	return strings.Join(words, " ")
}
