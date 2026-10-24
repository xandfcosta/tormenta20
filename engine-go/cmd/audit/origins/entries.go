package main

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"t20engine/cmd/internal/pyfmt"

	"t20engine/infra/bookpdf"
)

// closesSection é o título que fecha a seção. O último verbete — o Trabalhador
// — não tem sucessor, e sem esta fronteira a descrição do poder único dele
// engole a barra lateral "Sua Própria Origem" inteira: 2.696 caracteres de
// prosa entram na comparação como se fossem a regra.
const closesSection = "Sua Própria Origem"

var (
	// "um poder de combate a sua escolha" não é nome de poder: o catálogo o
	// guarda como CATEGORIA de escolha.
	choiceRe = regexp.MustCompile(`um poder (?:de |da )?(` + bookpdf.PyWord + `+) a sua escolha`)
	// `(poder)` no SINGULAR existe: o Amigo dos Animais lista um poder só, e um
	// regex que só conhecesse o plural deixaria o verbete dele sem ancorar — em
	// silêncio, porque "não casou" e "não existe" se parecem.
	listRe = regexp.MustCompile(`Benefícios\.` + bookpdf.PySpace + `*(.*?)` + bookpdf.PySpace + `*\(perícias\);` +
		bookpdf.PySpace + `*(.*?)` + bookpdf.PySpace + `*\(poder(?:es)?\)\.`)
)

// entry é o verbete de uma origem: a página do livro e as linhas.
type entry struct {
	page int
	body []string
}

type boundary struct {
	index, page int
	slug        string
	isOrigin    bool
}

// readEntries é a âncora 2: o verbete de cada origem — o título com
// `Benefícios.` abaixo. A segunda condição separa verbete de linha de tabela:
// "Batedor" é impresso na p87 antes da p88, e casar só por título acusa 29 das
// 35 origens com `bookPage` errado.
func readEntries(book *bookpdf.Book, names nameIndex, consumed map[consumedLine]bool) (map[string]entry, []string) {
	type pageLine struct {
		page int
		text string
	}
	var lines []pageLine
	for page := firstPage; page <= lastPage; page++ {
		for _, l := range book.Reading(page) {
			if consumed[consumedLine{page, l.X, l.Y}] {
				continue
			}
			lines = append(lines, pageLine{page, l.Text})
		}
	}

	// Um título é uma linha que É o nome da origem, e ele pode quebrar em duas
	// ("Assistente" / "de Laboratório").
	var bounds []boundary
	for i, l := range lines {
		joined := l.text
		if i+1 < len(lines) {
			joined = l.text + " " + lines[i+1].text
		}
		if slug, ok := names.byKey[bookpdf.Key(l.text)]; ok {
			bounds = append(bounds, boundary{i, l.page, slug, true})
		} else if slug, ok := names.byKey[bookpdf.Key(joined)]; ok {
			bounds = append(bounds, boundary{i, l.page, slug, true})
		} else if l.text == closesSection {
			bounds = append(bounds, boundary{i, l.page, "", false})
		}
	}

	var complaints []string
	if !slices.ContainsFunc(bounds, func(b boundary) bool { return !b.isOrigin }) {
		complaints = append(complaints, pyfmt.Repr(closesSection)+" não foi achado — o último verbete não tem "+
			"fronteira de fim, e a descrição do poder único dele engole a prosa que vem depois")
	}
	entries := map[string]entry{}
	seen := map[string]int{}
	for n, b := range bounds {
		if !b.isOrigin {
			continue
		}
		end := len(lines)
		if n+1 < len(bounds) {
			end = bounds[n+1].index
		}
		var body []string
		for _, l := range lines[b.index:end] {
			body = append(body, l.text)
		}
		if !strings.Contains(bookpdf.Join(body), "Benefícios.") {
			continue // é menção na prosa, não verbete
		}
		seen[b.slug]++
		if _, again := entries[b.slug]; again {
			complaints = append(complaints, fmt.Sprintf("%s: %d trechos com o título e `Benefícios.` — o primeiro foi usado",
				names.name[b.slug], seen[b.slug]))
			continue
		}
		entries[b.slug] = entry{b.page - pdfOffset, body}
	}
	return entries, complaints
}

// benefitList é a frase de benefícios lida: as perícias, os poderes e a
// categoria do poder "a sua escolha".
type benefitList struct {
	expertises, powers []string
	category           string
	hasCategory        bool
}

func (b benefitList) equal(o benefitList) bool {
	return slices.Equal(b.expertises, o.expertises) && slices.Equal(b.powers, o.powers) &&
		b.hasCategory == o.hasCategory && b.category == o.category
}

// repr é a tupla do Python: `(['A'], ['B'], None)`.
func (b benefitList) repr() string {
	return "(" + pyfmt.List(b.expertises) + ", " + pyfmt.List(b.powers) + ", " + pyOptional(b.category, b.hasCategory) + ")"
}

func pyOptional(s string, ok bool) string {
	if !ok {
		return "None"
	}
	return pyfmt.Repr(s)
}

// readList lê `Benefícios. A, B (perícias); C, D (poderes).`.
func readList(body string) (benefitList, bool) {
	m := listRe.FindStringSubmatch(body)
	if m == nil {
		return benefitList{}, false
	}
	out := benefitList{expertises: []string{}, powers: []string{}}
	for _, p := range strings.Split(m[1], ",") {
		if p = bookpdf.Strip(p); p != "" {
			out.expertises = append(out.expertises, p)
		}
	}
	for _, raw := range strings.Split(m[2], ",") {
		item := bookpdf.Strip(raw)
		if item == "" {
			continue
		}
		if choice := choiceRe.FindStringSubmatch(item); choice != nil {
			out.category, out.hasCategory = strings.ToLower(choice[1]), true
			continue
		}
		out.powers = append(out.powers, item)
	}
	return out, true
}

// readTableList lê a célula da tabela, que não rotula as duas metades: o `;` é
// que as separa.
func readTableList(raw string) (benefitList, bool) {
	if !strings.Contains(raw, ";") {
		return benefitList{}, false
	}
	return readList("Benefícios. " + strings.ReplaceAll(raw, ";", " (perícias);") + " (poderes).")
}

// readUniquePower é o (título, descrição) do poder único, procurados LINHA a
// LINHA.
//
// O título é uma linha que É um dos poderes que a própria frase de benefícios
// lista, e a busca começa DEPOIS dessa frase, porque ela já os nomeia todos. A
// comparação é por `Key`: o livro imprime o nome em versalete e o `pdftotext`
// devolve 'Quebra-galho' onde o catálogo escreve 'Quebra-Galho'.
//
// A descrição se corta no ÚLTIMO ponto, que descarta a legenda da ilustração —
// 'Aventureiros podem ter as origens mais surpreendentes' não termina em ponto.
func readUniquePower(body []string, powers []string) (string, string, bool) {
	closes := slices.IndexFunc(body, func(l string) bool { return strings.Contains(l, "(poder") })
	if closes < 0 {
		return "", "", false
	}
	byKey := map[string]string{}
	for _, p := range powers {
		byKey[bookpdf.Key(p)] = p
	}
	for i := closes + 1; i < len(body); i++ {
		name, ok := byKey[bookpdf.Key(body[i])]
		if !ok {
			continue
		}
		description := bookpdf.Join(body[i+1:])
		return name, description[:strings.LastIndex(description, ".")+1], true
	}
	return "", "", false
}
