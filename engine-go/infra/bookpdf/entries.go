package bookpdf

import (
	"regexp"
	"strings"
)

// ── os VERBETES por nome conhecido ───────────────────────────────────────────

// ColumnPair é uma coluna de verbetes: o x do corpo e o do recuo da primeira
// linha do parágrafo — que é justamente a que abre o verbete.
type ColumnPair struct{ Body, Indent float64 }

// Region é um retângulo (x0, x1, y0, y1) de tabela a pular.
type Region struct{ X0, X1, Y0, Y1 float64 }

// Entry é um verbete achado: o corpo, a página do LIVRO, e onde ele começa.
type Entry struct {
	Body   []string
	Page   int
	PDF    int
	Column ColumnPair
	Y0, Y1 float64
}

// Entries são os verbetes achados, por nome e na ordem em que foram achados.
type Entries struct {
	ByName map[string]*Entry
	Order  []string
}

// EntriesByName são os verbetes `Nome. Regra` das páginas, PROCURADOS por nome
// conhecido — um nome com preposição no meio ("Garrafa da Fumaça Eterna") não
// se acha com "maiúscula seguida de ponto". O título de seção corta o verbete em
// andamento, senão ele vira a última frase do anterior.
func (b *Book) EntriesByName(pages []int, columns map[int][]ColumnPair, regions map[int][]Region,
	names []string, titles []string, offset int, slack float64) Entries {
	byKey := map[string]string{}
	for _, n := range names {
		byKey[Key(n)] = n
	}
	found := Entries{ByName: map[string]*Entry{}}
	current := ""
	for _, page := range pages {
		for _, l := range b.Reading(page) {
			column, ok := ColumnOfPair(l.X, columns[page], slack)
			if !ok || InsideAny(l.X, l.Y, regions[page]) {
				continue
			}
			if startsWithAny(l.Text, titles) {
				current = ""
				continue
			}
			head, rest, dot := strings.Cut(l.Text, ". ")
			name := ""
			if dot {
				name = byKey[Key(head)]
			}
			if name != "" {
				current = name
				if _, again := found.ByName[name]; !again {
					found.Order = append(found.Order, name)
				}
				found.ByName[name] = &Entry{Body: []string{rest}, Page: page - offset, PDF: page, Column: column, Y0: l.Y, Y1: l.Y}
			} else if current != "" {
				e := found.ByName[current]
				e.Body = append(e.Body, l.Text)
				if page == e.PDF && column == e.Column {
					e.Y1 = l.Y
				}
			}
		}
	}
	return found
}

func startsWithAny(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// ColumnOfPair é o par em que esta linha está.
func ColumnOfPair(x float64, columns []ColumnPair, slack float64) (ColumnPair, bool) {
	for _, c := range columns {
		if abs(x-c.Body) < slack || abs(x-c.Indent) < slack {
			return c, true
		}
	}
	return ColumnPair{}, false
}

// InsideAny diz se o ponto cai em algum retângulo de tabela.
func InsideAny(x, y float64, regions []Region) bool {
	for _, r := range regions {
		if r.X0 <= x && x <= r.X1 && r.Y0 <= y && y <= r.Y1 {
			return true
		}
	}
	return false
}

// ── a COBERTURA de uma frase ─────────────────────────────────────────────────

// stopWords são artigo, preposição, pronome e verbo de ligação: estão em toda
// frase, e contá-los daria a qualquer par uma semelhança de base.
var stopWords = func() map[string]bool {
	m := map[string]bool{}
	for _, w := range strings.Fields("a o as os um uma uns umas de do da dos das em no na nos nas e ou que se por " +
		"para com sem seu sua seus suas voce ele ela isso este esta esse essa ao aos " +
		"nao mas como mesmo ate entre sobre pelo pela mais menos tambem cada qualquer " +
		"todos todas outro outra sao ser tem pode podem quando onde apenas alem disso " +
		"muito seja ja la") {
		m[w] = true
	}
	return m
}()

var alnumRun = regexp.MustCompile(`[a-z0-9]+`)

// ContentWords são as palavras que carregam a regra, sem as vazias e sem acento.
func ContentWords(text string) map[string]bool {
	out := map[string]bool{}
	for _, w := range alnumRun.FindAllString(fold(text), -1) {
		if len(w) > 2 && !stopWords[w] {
			out[w] = true
		}
	}
	return out
}

// Coverage é quanto das palavras de conteúdo do LIVRO a frase do catálogo
// repete. O denominador é o livro: a pergunta é "a regra está aqui?".
func Coverage(fromBook, fromCatalog string) (float64, int, int) {
	book := ContentWords(fromBook)
	if len(book) == 0 {
		return 1.0, 0, 0
	}
	catalog := ContentWords(fromCatalog)
	shared := 0
	for w := range book {
		if catalog[w] {
			shared++
		}
	}
	return float64(shared) / float64(len(book)), shared, len(book)
}
