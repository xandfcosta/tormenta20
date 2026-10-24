package main

import (
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"t20engine/domain/catalog/transcript"
	"t20engine/infra/bookpdf"
)

const (
	firstPage, lastPage = 25, 37 // PDF
	pdfOffset           = 6      // livro = PDF - 6
	extrasSection       = "Raças Extras"
	maxTitleLength      = 25 // caracteres; um título de raça é curto, a prosa não
	regionFallback      = 1200
)

var attributes = []struct{ book, id string }{
	{"Força", "strength"}, {"Destreza", "dexterity"}, {"Constituição", "constitution"},
	{"Inteligência", "intelligence"}, {"Sabedoria", "wisdom"}, {"Carisma", "charisma"},
}

func attributeID(book string) string {
	for _, a := range attributes {
		if a.book == book {
			return a.id
		}
	}
	return ""
}

const (
	minus = `–−—`
)

var (
	attrNames = func() string {
		var names []string
		for _, a := range attributes {
			names = append(names, a.book)
		}
		return strings.Join(names, "|")
	}()
	signed  = `[+` + minus + `-]\d+`
	fixedRe = regexp.MustCompile(`(?:(?:` + attrNames + `) ` + signed + `(?: \(` + bookpdf.PyWord + `+\))?[,;]? ?)+\.?`)
	pairRe  = regexp.MustCompile(`(` + attrNames + `) (` + signed + `)(?: \((` + bookpdf.PyWord + `+)\))?`)
	// "+1 em três atributos diferentes", com a exceção e a penalidade opcionais
	// do Lefou e do Osteon na mesma frase.
	floatingBody    = `\+1 em [Tt]rês [Aa]tributos [Dd]iferentes(?: \(exceto (` + attrNames + `)\))?(?:, (` + attrNames + `) (` + signed + `))?\.?`
	floatingRe      = regexp.MustCompile(floatingBody)
	floatingStartRe = regexp.MustCompile(`^` + floatingBody)
	sizeRe          = regexp.MustCompile(`[Ss]eu tamanho é (Minúsculo|Pequeno|Médio|Grande|Enorme)`)
	// O resto do deslocamento depois da palavra; o "não é de natação" fica em
	// `displacement`, porque o RE2 não tem lookahead.
	displacementTailRe = regexp.MustCompile(`^` + bookpdf.PySpace + `*(?:é|de)?` + bookpdf.PySpace + `*(\d+)m`)
)

const defaultDisplacement, defaultSize = 9, "Médio"

// otherDisplacements NÃO são o deslocamento da raça: a Sereia imprime
// "deslocamento de natação 12m" e "(deslocamento 9m)" na mesma habilidade, e
// ler o primeiro número trocaria um pelo outro.
var otherDisplacements = []string{" de natação", " de voo", " de escalada", " de escavação"}

// displacement é o primeiro deslocamento DA RAÇA que o texto imprime.
func displacement(text string) (string, bool) {
	const word = "deslocamento"
	for from := 0; ; {
		i := strings.Index(text[from:], word)
		if i < 0 {
			return "", false
		}
		after := from + i + len(word)
		other := false
		for _, o := range otherDisplacements {
			if strings.HasPrefix(text[after:], o) {
				other = true
			}
		}
		if !other {
			if m := displacementTailRe.FindStringSubmatch(text[after:]); m != nil {
				return m[1], true
			}
		}
		from = from + i + 1
	}
}

// chapter é o texto corrido do capítulo, com as posições em CARACTERE — as do
// Python —, porque a região de uma habilidade sem sucessora tem 1200 deles.
type chapter struct {
	text  string
	runes []rune
	index []int // byte → caractere
}

func readChapter(book *bookpdf.Book) chapter {
	var lines []string
	for page := firstPage; page <= lastPage; page++ {
		lines = append(lines, book.PageLines(page)...)
	}
	text := bookpdf.Join(lines)
	index := make([]int, len(text)+1)
	n := 0
	for i := range text {
		index[i] = n
		n++
	}
	index[len(text)] = n
	for i := len(text) - 1; i >= 0; i-- {
		if !utf8.RuneStart(text[i]) {
			index[i] = index[i+1]
		}
	}
	return chapter{text, []rune(text), index}
}

func (c chapter) slice(from, to int) string {
	to = min(to, len(c.runes))
	if from >= to {
		return ""
	}
	return string(c.runes[from:to])
}

// titlePages é em que página do LIVRO cada raça abre, e onde abre a seção
// "Raças Extras".
//
// O título é a única coisa da raça que fica na página que o catálogo cita: o
// bloco de regras dela pode terminar na página seguinte (o Hynne termina), e
// conferir `bookPage` pela linha de atributos acusaria a p28 para uma raça que
// o livro abre na p27.
func titlePages(book *bookpdf.Book, races []transcript.Race) (map[string]int, int, bool) {
	byKey := map[string]string{}
	for _, r := range races {
		byKey[bookpdf.Key(r.Name)] = r.ID
	}
	pages := map[string]int{}
	extras, haveExtras := 0, false
	for page := firstPage; page <= lastPage; page++ {
		for _, line := range book.PageLines(page) {
			if bookpdf.Strip(line) == extrasSection && !haveExtras {
				extras, haveExtras = page-pdfOffset, true
			}
			if utf8.RuneCountInString(line) > maxTitleLength {
				continue
			}
			if id, ok := byKey[bookpdf.Key(line)]; ok {
				if _, again := pages[id]; !again {
					pages[id] = page - pdfOffset
				}
			}
		}
	}
	return pages, extras, haveExtras
}

func num(s string) int {
	n, _ := bookpdf.Num(s)
	return n
}

// readAttributeLine lê a linha impressa na forma do `atributoMod` do catálogo.
//
// O livro escreve o atributo ANTES do sinal ("Constituição +2"), ao contrário
// do bestiário, e marca a subraça entre parênteses ("Sabedoria +2 (Aggelus)").
func readAttributeLine(raw string) pyDict {
	if m := floatingStartRe.FindStringSubmatch(raw); m != nil {
		d := pyDict{{"kind", pyText("floating")}, {"count", pyInt(3)}, {"value", pyInt(1)}}
		if m[1] != "" {
			d.set("exclude", pyText(attributeID(m[1])))
		}
		if m[2] != "" {
			d.set("penalty", pyNested(pyDict{{"attribute", pyText(attributeID(m[2]))}, {"value", pyInt(num(m[3]))}}))
		}
		return d
	}
	// Uma linha ROTULADA carrega as duas subraças de uma vez, e o rótulo vem
	// DEPOIS do grupo que ele nomeia: acumular até o rótulo aparecer é o que
	// separa os dois conjuntos.
	var variants, pending pyDict
	for _, m := range pairRe.FindAllStringSubmatch(raw, -1) {
		pending.set(attributeID(m[1]), pyInt(num(m[2])))
		if m[3] != "" {
			variants.set(strings.ToLower(m[3]), pyNested(pending))
			pending = nil
		}
	}
	if len(variants) > 0 {
		return pyDict{{"kind", pyText("subraca-gated")}, {"variants", pyNested(variants)}}
	}
	return pyDict{{"kind", pyText("fixed")}, {"mods", pyNested(pending)}}
}

// attributeLine é uma linha de atributos achada no capítulo, em caracteres.
type attributeLine struct {
	start, end int
	text       string
}

func findAttributeLines(c chapter) []attributeLine {
	var found []attributeLine
	for _, m := range floatingRe.FindAllStringIndex(c.text, -1) {
		found = append(found, attributeLine{c.index[m[0]], c.index[m[1]], c.text[m[0]:m[1]]})
	}
	for _, m := range fixedRe.FindAllStringIndex(c.text, -1) {
		start := c.index[m[0]]
		// A penalidade do Lefou ("…Diferentes (exceto Carisma), Carisma –1.")
		// também casa como linha fixa; ela já pertence à linha livre.
		inside := false
		for _, f := range found {
			if f.start <= start && start < f.end {
				inside = true
			}
		}
		if !inside {
			found = append(found, attributeLine{start, c.index[m[1]], c.text[m[0]:m[1]]})
		}
	}
	sort.Slice(found, func(i, j int) bool {
		a, b := found[i], found[j]
		if a.start != b.start {
			return a.start < b.start
		}
		if a.end != b.end {
			return a.end < b.end
		}
		return a.text < b.text
	})
	return found
}
