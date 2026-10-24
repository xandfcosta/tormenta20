package main

import (
	"fmt"
	"t20engine/cmd/internal/pyfmt"

	"t20engine/infra/bookpdf"
)

type coverageOf struct {
	ratio float64
	name  string
}

// measured é o DENOMINADOR: quantos pré-requisitos, páginas e regras foram de
// fato comparados, e a cobertura de cada regra.
type measured struct {
	prereqs, pages, rules int
	coverage              []coverageOf
}

// check devolve as queixas deste poder, onde o LIVRO discorda de si, e as notas
// a reler. `nil` num conjunto é "não ancorou ali".
func check(p power, fromTable, fromEntry stringSet, page *int, rule string, expertiseNames []string,
	m *measured) (found, bookSelf, notes []string) {
	if fromTable != nil && fromEntry != nil {
		switch {
		case !shapeOnly(fromTable).equal(shapeOnly(fromEntry)):
			if !isDeclared(selfDisagreement, p.name) {
				bookSelf = append(bookSelf, fmt.Sprintf("%s: a tabela diz %s e o verbete diz %s — discordância NOVA",
					p.name, pyfmt.List(shapeOnly(fromTable).sorted()), pyfmt.List(shapeOnly(fromEntry).sorted())))
			}
		case isDeclared(selfDisagreement, p.name):
			bookSelf = append(bookSelf, p.name+": a discordância declarada NÃO acontece mais — tire a linha de `selfDisagreement` (cmd/audit/powers)")
		case !fromTable.equal(fromEntry):
			notes = append(notes, fmt.Sprintf("%s: tabela %s × verbete %s", p.name,
				pyfmt.List(fromTable.minus(fromEntry).sorted()), pyfmt.List(fromEntry.minus(fromTable).sorted())))
		}
	}
	read := fromTable
	if len(fromEntry) > 0 {
		read = fromEntry
	}
	if read != nil {
		m.prereqs += max(len(shapeOnly(read)), 1)
		switch {
		case !shapeOnly(read).equal(shapeOnly(p.catalogPrereqs)):
			found = append(found, fmt.Sprintf("pré-requisitos: catálogo %s × livro %s",
				pyfmt.List(shapeOnly(p.catalogPrereqs).sorted()), pyfmt.List(shapeOnly(read).sorted())))
		case !read.equal(p.catalogPrereqs):
			notes = append(notes, fmt.Sprintf("%s: catálogo %s × livro %s", p.name,
				pyfmt.List(p.catalogPrereqs.minus(read).sorted()), pyfmt.List(read.minus(p.catalogPrereqs).sorted())))
		}
	}
	if page != nil {
		m.pages++
		if p.bookPage != *page {
			found = append(found, fmt.Sprintf("bookPage: catálogo %d × livro %d", p.bookPage, *page))
		}
	}
	if rule != "" {
		found = append(found, checkRule(p, rule, expertiseNames, m)...)
	}
	return found, bookSelf, notes
}

// checkRule mede três dimensões sobre a descrição: números, perícias nomeadas,
// e quanto das palavras de conteúdo do catálogo o livro tem. A prosa não se
// compara — o catálogo encurta de propósito —, e a terceira existe porque as
// duas primeiras deixaram passar seis regras trocadas na ALE-405.
//
// AS TRÊS MEDEM NA MESMA DIREÇÃO: o que o CATÁLOGO afirma tem de estar na
// página. Exigir que o catálogo repita tudo acusaria quase todos; o que não se
// admite é número ou perícia que o catálogo afirma e a página não tem.
func checkRule(p power, fromBook string, expertiseNames []string, m *measured) []string {
	fromCatalog := p.description
	m.rules++
	var out []string
	if invented := numbers(fromCatalog).minus(numbers(fromBook)); len(invented) > 0 {
		out = append(out, "números que o catálogo afirma e a página não tem: "+pyfmt.List(invented.sorted()))
	}
	outside := citedExpertises(fromCatalog, expertiseNames).minus(citedExpertises(fromBook, expertiseNames))
	if len(outside) > 0 {
		out = append(out, "perícias que o catálogo cita e a página não: "+pyfmt.List(outside.sorted()))
	}
	ratio, shared, total := bookpdf.Coverage(fromCatalog, fromBook)
	m.coverage = append(m.coverage, coverageOf{ratio, p.name})
	if ratio < minimumCoverage {
		out = append(out, fmt.Sprintf("%d das %d palavras de conteúdo do CATÁLOGO estão no livro (%s, piso %s) — "+
			"um resumo fiel só usa palavras que a página tem", shared, total, pyfmt.Percent(ratio), pyfmt.Percent(minimumCoverage)))
	}
	if len(out) > 0 {
		out = append(out, "  NO LIVRO:    "+pyfmt.Repr(pyfmt.FirstRunes(fromBook, 260)))
		out = append(out, "  NO CATÁLOGO: "+pyfmt.Repr(pyfmt.FirstRunes(fromCatalog, 260)))
	}
	return out
}
