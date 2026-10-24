package bookpdf

import (
	"html"
	"regexp"
	"sort"
	"strconv"
)

// ── o ESTILO, de outro leitor ────────────────────────────────────────────────
//
// O `-bbox` do `pdftotext` não marca estilo nenhum, e neste livro o estilo é
// DADO: o título de uma magia é `Tormenta20` a 24, e o itálico de um item
// específico é a composição dele (p337). Ele vem do `pdftohtml -xml`, e o x e o
// y voltam convertidos para a escala do `-bbox` — a razão entre as alturas que
// os dois declaram —, senão quem misturasse as leituras compararia réguas
// diferentes.

var (
	htmlPageRe   = regexp.MustCompile(`<page number="\d+"[^>]*height="([\d.]+)" width="([\d.]+)"`)
	htmlTextRe   = regexp.MustCompile(`(?s)<text top="([\d.-]+)" left="([\d.-]+)"[^>]*>(.*?)</text>`)
	italicRe     = regexp.MustCompile(`(?s)<i>(.*?)</i>`)
	fontSpecRe   = regexp.MustCompile(`<fontspec id="(\d+)" size="(\d+)" family="([^"]+)"`)
	textFontRe   = regexp.MustCompile(`(?s)<text top="(-?[\d.]+)" left="(-?[\d.]+)"[^>]*font="(\d+)"[^>]*>(.*?)</text>`)
	boldLinkRe   = regexp.MustCompile(`(?s)<a href="[^"#]*#(\d+)"><b>(.*?)</b></a>`)
	diceRangeRe  = regexp.MustCompile(`^(\d{1,3})(?:-(\d{1,3}))?(?:` + PySpace + `+(.*))?$`)
	bookPriceRe  = regexp.MustCompile(`^T\$` + PySpace + `*([\d.]+)`)
	powerHeadKey = "Poder"
	prereqKey    = "Pré-requisitos"
)

func (b *Book) htmlScale(page int, xml string) (float64, bool) {
	m := htmlPageRe.FindStringSubmatch(xml)
	if m == nil {
		return 0, false
	}
	height, err := b.PageHeight(page)
	if err != nil || height == 0 {
		return 0, false
	}
	return parseFloat(m[1]) / height, true
}

func cleanHTML(s string) string { return Collapse(html.UnescapeString(tag.ReplaceAllString(s, ""))) }

// Italics são os trechos em ITÁLICO, ordenados por (y, x). O trecho vem
// QUEBRADO como a linha quebra; quem junta é quem lê.
func (b *Book) Italics(page int) []Positioned {
	xml := b.htmlXML(page)
	scale, ok := b.htmlScale(page, xml)
	if !ok {
		return nil
	}
	var out []Positioned
	for _, m := range htmlTextRe.FindAllStringSubmatch(xml, -1) {
		for _, it := range italicRe.FindAllStringSubmatch(m[3], -1) {
			if text := cleanHTML(it[1]); text != "" {
				out = append(out, Positioned{parseFloat(m[2]) / scale, parseFloat(m[1]) / scale, text})
			}
		}
	}
	sortByYX(out)
	return out
}

func sortByYX(out []Positioned) {
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Y != out[j].Y {
			return out[i].Y < out[j].Y
		}
		return out[i].X < out[j].X
	})
}

// StyledRun é um pedaço de texto com o ESTILO em que o livro o desenhou.
type StyledRun struct {
	X, Y   float64
	Size   int
	Family string
	Text   string
}

// StyledRuns são os trechos da página com TAMANHO e FAMÍLIA do tipo, por (y, x).
//
//	for _, r := range book.StyledRuns(184) { if r.Size == 24 { … } } // as magias da p178
func (b *Book) StyledRuns(page int) []StyledRun {
	xml := b.htmlXML(page)
	scale, ok := b.htmlScale(page, xml)
	if !ok {
		return nil
	}
	type font struct {
		size   int
		family string
	}
	fonts := map[string]font{}
	for _, m := range fontSpecRe.FindAllStringSubmatch(xml, -1) {
		size, _ := strconv.Atoi(m[2])
		fonts[m[1]] = font{size, m[3]}
	}
	var out []StyledRun
	for _, m := range textFontRe.FindAllStringSubmatch(xml, -1) {
		f := fonts[m[3]]
		if text := cleanHTML(m[4]); text != "" {
			out = append(out, StyledRun{parseFloat(m[2]) / scale, parseFloat(m[1]) / scale, f.size, f.family, text})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Y != out[j].Y {
			return out[i].Y < out[j].Y
		}
		return out[i].X < out[j].X
	})
	return out
}

// BoldLink é um link em NEGRITO e a página do PDF para onde ele aponta.
type BoldLink struct {
	Text string
	Page int
}

// BoldLinks são os links em negrito da página — o índice de magias das
// p174-177. Negrito SEM link não entra: o cabeçalho de página também é negrito,
// e contá-lo inflou o índice de 198 para 203 (ALE-456).
func (b *Book) BoldLinks(page int) []BoldLink {
	var out []BoldLink
	for _, m := range boldLinkRe.FindAllStringSubmatch(b.htmlXML(page), -1) {
		if text := cleanHTML(m[2]); text != "" {
			target, _ := strconv.Atoi(m[1])
			out = append(out, BoldLink{text, target})
		}
	}
	return out
}
