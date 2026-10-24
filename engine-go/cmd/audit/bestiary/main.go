// Command bestiary confere o bestiário contra o livro, verbete por verbete
// (ALE-151).
//
// Rodar é ATO DELIBERADO, como o `genoracle`: a ferramenta PROPÕE, o diff se
// revisa contra o PDF, e é a revisão que decide.
//
//	go run ./cmd/audit/bestiary            # relatório
//	go run ./cmd/audit/bestiary --aplicar  # escreve as correções no catálogo
//
// Um bloco só conta quando é COMPLETO — Iniciativa, Defesa com as três
// resistências, Pontos de Vida, e termina em Tesouro — E quando Defesa e PV
// batem com o catálogo. Essas duas âncoras dizem "é esta criatura"; o que
// divergir depois é erro do catálogo.
//
// As armadilhas que já custaram caro: `pdftotext -layout` cola colunas
// vizinhas (quase "corrigiu" a Hidra ao contrário), por isso a leitura é por
// COORDENADA (`infra/bookpdf`); e o livro escreve o nome de quatro jeitos —
// numa linha, junto do ND, quebrado em duas, e quebrado COM o ND no meio
// ("Cavaleiro do Leopardo" / "ND 9" / "Sangrento").
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"t20engine/cmd/internal/pyfmt"

	"t20engine/cmd/internal/transcriptdir"
	"t20engine/domain/catalog/transcript"
	"t20engine/infra/bookpdf"
)

// firstPage e lastPage são o capítulo no PDF; o livro é PDF − 6.
const (
	firstPage = 292
	lastPage  = 322
)

const pdfOffset = 6

func main() {
	apply := flag.Bool("aplicar", false, "escreve as correções no bestiário")
	flag.Parse()

	book, err := bookpdf.Open()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	r := audit(book)
	fmt.Printf("blocos confiáveis: %d/%d\n", len(r.trusted), len(r.trusted)+len(r.suspects))
	for _, s := range r.suspects {
		fmt.Printf("  SUSPEITO %s: %s — confira à mão\n", s.name, s.reason)
	}
	for _, name := range r.truncated {
		fmt.Printf("  TESOURO TRUNCADO %s: a frase passa de %d linhas — confira à mão\n", name, treasureLines)
	}
	for _, s := range r.withoutAttributes {
		fmt.Printf("  SEM AUDITORIA DE ATRIBUTOS %s: %s — os outros campos valem\n", s.name, s.reason)
	}
	fixes := corrections(r)
	fmt.Printf("\ncorreções propostas: %d\n", len(fixes))
	for _, c := range fixes {
		fmt.Printf("  p%3d %-32s %-14s %6s -> %s\n", c.page, c.name, c.field,
			pyIntRepr(c.from), pyIntRepr(c.to))
	}
	for _, field := range []string{"iniciativa", "percepcao", "tesouro"} {
		var missing []string
		for _, t := range r.trusted {
			if !t.read.has(field) {
				missing = append(missing, t.creature.Name)
			}
		}
		if len(missing) > 0 {
			fmt.Printf("\n%s: %d sem valor no bloco — %s\n", field, len(missing), pyfmt.List(missing[:min(6, len(missing))]))
		}
	}

	if *apply {
		dir, err := transcriptdir.Find()
		if err == nil {
			err = applyCorrections(fixes, r, dir)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("\naplicadas em %s\n", filepath.Join(dir, "bestiary*.go"))
	}
}

// suspect é um verbete que não pôde ser auditado, e o motivo.
type suspect struct{ name, reason string }

// trustedBlock é um verbete do catálogo e o bloco do livro que é dele.
type trustedBlock struct {
	creature *transcript.Creature
	read     readBlock
}

// auditResult são os verbetes separados pelo que se pôde conferir, na ordem do
// catálogo.
type auditResult struct {
	trusted           []trustedBlock
	withoutAttributes []suspect
	suspects          []suspect
	truncated         []string
}

func audit(book *bookpdf.Book) auditResult {
	byKey := map[string]*transcript.Creature{}
	var order []string
	for i := range transcript.Bestiary {
		k := bookpdf.Key(transcript.Bestiary[i].Name)
		if _, seen := byKey[k]; !seen {
			order = append(order, k)
		}
		byKey[k] = &transcript.Bestiary[i]
	}
	blocks := blocksByName(bookLines(book), byKey)
	var r auditResult
	for _, k := range order {
		c := byKey[k]
		b, ok := blocks[k]
		if !ok || !completeBlock(b.body) {
			reason := "sem bloco"
			if ok {
				reason = "bloco incompleto"
			}
			r.suspects = append(r.suspects, suspect{c.Name, reason})
			continue
		}
		read := readFields(b)
		// As âncoras: se elas não batem, o bloco é de outra criatura.
		if !read.equals("defesa", c.Defense) || !read.equals("hp", c.HP) {
			r.suspects = append(r.suspects, suspect{c.Name, fmt.Sprintf("âncora não bate (Defesa %s/%d, PV %s/%d)",
				pyIntRepr(read.number("defesa")), c.Defense, pyIntRepr(read.number("hp")), c.HP)})
			continue
		}
		r.trusted = append(r.trusted, trustedBlock{c, read})
		if !hasAttributeLine(bookpdf.Join(b.body)) {
			r.withoutAttributes = append(r.withoutAttributes, suspect{c.Name, "linha de atributos noutra coluna"})
		}
		if b.truncated {
			r.truncated = append(r.truncated, c.Name)
		}
	}
	return r
}

// audited são os campos numéricos conferidos contra o catálogo.
var audited = []string{"hp", "defesa", "fortitude", "reflexos", "vontade",
	"forca", "destreza", "constituicao", "inteligencia", "sabedoria", "carisma"}

// correction é um campo do catálogo que o livro diz ser outro.
type correction struct {
	name     string
	page     int
	field    string
	from, to *int
}

// corrections são as correções propostas, em ordem de página.
func corrections(r auditResult) []correction {
	var out []correction
	for _, t := range r.trusted {
		for _, field := range audited {
			if !t.read.has(field) {
				continue
			}
			from, to := numericField(t.creature, field), t.read.number(field)
			if !sameInt(from, to) {
				out = append(out, correction{t.creature.Name, t.creature.BookPage, field, from, to})
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].page != out[j].page {
			return out[i].page < out[j].page
		}
		if out[i].name != out[j].name {
			return out[i].name < out[j].name
		}
		return out[i].field < out[j].field
	})
	return out
}

func sameInt(a, b *int) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// numericField é o campo conferido do verbete; nil é o `null` do catálogo, que
// só Força e Inteligência admitem (o morto-vivo sem mente, o constructo).
func numericField(c *transcript.Creature, field string) *int {
	switch field {
	case "hp":
		return &c.HP
	case "defesa":
		return &c.Defense
	case "fortitude":
		return &c.Fortitude
	case "reflexos":
		return &c.Reflexes
	case "vontade":
		return &c.Will
	case "forca":
		return c.Strength
	case "destreza":
		return &c.Dexterity
	case "constituicao":
		return &c.Constitution
	case "inteligencia":
		return c.Intelligence
	case "sabedoria":
		return &c.Wisdom
	case "carisma":
		return &c.Charisma
	}
	panic("campo conferido desconhecido: " + field + " (esperado um de " + strings.Join(audited, ", ") + ")")
}
