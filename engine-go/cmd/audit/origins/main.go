// Command origins confere as duas transcrições de origem contra a seção
// "Origens" (p85-95). Seção 3 da ALE-391.
//
// Mede a LISTA DE BENEFÍCIOS de cada origem — perícias, poderes gerais, poder
// único e a categoria do poder "a sua escolha" —, a linha de itens e a página.
// E mede `OriginSources` (o livro, por slug) contra `Origins` (a mesma origem
// achatada em benefícios), que compartilham o `uid`: essa metade não precisa
// do livro e é do `TestEveryOriginSaysTheSameThingInBothFiles`, mas o relatório
// vale mais junto.
//
// Por que são DUAS âncoras, por que a tabela é lida primeiro e por que a
// descrição do poder único se cerca por TRÊS dimensões é do `engine-go/CLAUDE.md`.
//
// Ele PROPÕE: nada é escrito.
//
//	go run ./cmd/audit/origins
package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"t20engine/cmd/internal/pyfmt"

	"t20engine/domain/catalog/transcript"
	"t20engine/infra/bookpdf"
)

// withoutList é a ORIGEM SEM LISTA, e a exceção é NOMEADA: o Amnésico não
// imprime "(perícias); … (poderes)." nenhuma das duas vezes, porque o benefício
// é o mestre. Nomear é o que faz uma origem NOVA com forma nova reprovar em vez
// de nascer sem medição.
const withoutList = "amnesico"

// declaredDisagreements são os lugares onde o LIVRO discorda de si mesmo. O
// catálogo segue a TABELA, como nas outras três origens com especialização.
// Declarar faz uma discordância NOVA reprovar com o nome dela, e uma declarada
// que PARAR de discordar reprova também, senão a lista apodrece sozinha.
var declaredDisagreements = []struct{ slug, reason string }{
	{"fazendeiro", "a Tabela 1-19 grafa \"Ofício (fazendeiro)\" e o verbete grafa \"Ofício\" — o catálogo segue a tabela"},
}

func declaredReason(slug string) (string, bool) {
	for _, d := range declaredDisagreements {
		if d.slug == slug {
			return d.reason, true
		}
	}
	return "", false
}

// nameIndex é o nome de cada origem por slug, e o slug por chave do nome.
type nameIndex struct {
	name  map[string]string
	byKey map[string]string
}

// check são as queixas desta origem, e as linhas em que o LIVRO discorda de si.
func check(slug string, origin transcript.OriginSource, fromTable, fromEntry benefitList, hasTable, hasEntry bool,
	body []string, page int, hasPage bool, expertiseNames []string, flat *transcript.OriginUniquePower, m *measured) ([]string, []string) {
	var out, book []string
	reason, declared := declaredReason(slug)
	quarrel := hasTable && hasEntry && !fromTable.equal(fromEntry)
	if quarrel && !declared {
		book = append(book, fmt.Sprintf("%s: a Tabela 1-19 diz %s e o verbete diz %s — discordância NOVA, não declarada",
			origin.Name, fromTable.repr(), fromEntry.repr()))
	}
	if declared && !quarrel {
		book = append(book, fmt.Sprintf("%s: a discordância declarada (%s) NÃO acontece mais — tire a linha de declaredDisagreements",
			origin.Name, reason))
	}
	// Onde o livro discorda de si, o catálogo segue a declaração que a lista de
	// discordâncias nomeia — aqui, a da Tabela 1-19.
	read, hasRead := fromEntry, hasEntry
	if declared || !hasEntry {
		read, hasRead = fromTable, hasTable
	}
	if hasRead {
		out = append(out, checkList(origin, read, m)...)
		unique, description, found := readUniquePower(body, read.powers)
		if !found {
			out = append(out, "poder único: o verbete não imprime, depois da frase de benefícios, nenhum título que seja um dos poderes listados")
		} else {
			m.unique++
			if bookpdf.Key(unique) != bookpdf.Key(origin.UniquePower) {
				out = append(out, fmt.Sprintf("poderUnico: catálogo %s × livro %s",
					pyfmt.Repr(origin.UniquePower), pyfmt.Repr(unique)))
			}
			out = append(out, checkUniqueRule(description, unique, flat, expertiseNames, m)...)
		}
	}
	if len(body) > 0 {
		out = append(out, checkItems(origin, bookpdf.Join(body), m)...)
	}
	if hasPage {
		m.page++
		if origin.BookPage != page {
			out = append(out, fmt.Sprintf("bookPage: catálogo %d × livro %d (onde o TÍTULO do verbete é impresso)", origin.BookPage, page))
		}
	}
	return out, book
}

func main() {
	book, err := bookpdf.Open()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	os.Exit(run(book))
}

func run(book *bookpdf.Book) int {
	sources := map[string]transcript.OriginSource{}
	names := nameIndex{name: map[string]string{}, byKey: map[string]string{}}
	for _, o := range transcript.OriginSources {
		sources[o.ID] = o
		names.name[o.ID] = o.Name
	}
	for _, o := range transcript.OriginSources {
		names.byKey[bookpdf.Key(names.name[o.ID])] = o.ID
	}
	flat := map[string]transcript.Origin{}
	for _, o := range transcript.Origins {
		flat[o.ID] = o
	}
	var expertiseNames []string
	for _, e := range transcript.Expertises {
		expertiseNames = append(expertiseNames, e.Name)
	}

	betweenCatalogs := checkBothCatalogs(sources, flat)
	fromTable, notes, consumedList := readTable(book, names)
	consumed := map[consumedLine]bool{}
	for _, c := range consumedList {
		consumed[c] = true
	}
	entries, complaints := readEntries(book, names, consumed)

	var m measured
	diverge, inBoth, inOne := 0, 0, 0
	var inNone, bookSelf []string
	for _, origin := range transcript.OriginSources {
		slug := origin.ID
		e, hasPage := entries[slug]
		var table, fromEntry benefitList
		hasTable, hasEntry := false, false
		if raw, ok := fromTable[slug]; ok {
			table, hasTable = readTableList(raw)
		}
		if len(e.body) > 0 {
			fromEntry, hasEntry = readList(bookpdf.Join(e.body))
		}
		switch {
		case slug == withoutList:
			complaints = append(complaints, origin.Name+": a EXCEÇÃO NOMEADA — o livro não imprime "+
				"\"(perícias); … (poderes).\" para ela nas duas declarações, porque os "+
				"benefícios dela são escolhidos pelo mestre. Dela se medem a página e "+
				"os itens; a lista e a regra do poder único se conferem no OLHO")
		case !hasTable && !hasEntry:
			inNone = append(inNone, origin.Name+": não ancorou em nenhuma das duas declarações — a lista de benefícios NÃO foi medida")
		case !hasTable || !hasEntry:
			inOne++
			missing := "verbete"
			if !hasTable {
				missing = "Tabela 1-19"
			}
			complaints = append(complaints, fmt.Sprintf("%s: não ancorou no %s — a lista foi lida numa declaração só, "+
				"sem se conferir contra a outra", origin.Name, missing))
		default:
			inBoth++
		}
		if slug == withoutList {
			hasTable, hasEntry = false, false
		}
		var unique *transcript.OriginUniquePower
		if o, ok := flat[slug]; ok {
			unique = &o.UniquePower
		}
		out, disagree := check(slug, origin, table, fromEntry, hasTable, hasEntry, e.body, e.page, hasPage,
			expertiseNames, unique, &m)
		bookSelf = append(bookSelf, disagree...)
		for _, complaint := range out {
			if strings.HasPrefix(complaint, "  ") {
				fmt.Printf("  %s\n", complaint)
				continue
			}
			fmt.Printf("  %s: %s\n", origin.Name, complaint)
			diverge++
		}
	}

	if len(bookSelf) > 0 {
		fmt.Println("\nO LIVRO DISCORDA DE SI MESMO (achado sobre o LIVRO, não sobre o catálogo):")
		for _, line := range bookSelf {
			fmt.Printf("  %s\n", line)
		}
	}
	for _, d := range declaredDisagreements {
		fmt.Printf("  DISCORDÂNCIA DECLARADA %s: %s\n", sources[d.slug].Name, d.reason)
	}
	if len(betweenCatalogs) > 0 {
		fmt.Println("\nOS DOIS CATÁLOGOS DISCORDAM (defeito sem precisar do livro):")
		for _, line := range betweenCatalogs {
			fmt.Printf("  %s\n", line)
		}
	}
	if len(inNone) > 0 || len(complaints) > 0 {
		fmt.Println()
		for _, line := range append(inNone, complaints...) {
			fmt.Printf("  NÃO MEDIDO %s\n", line)
		}
	}

	fmt.Println("\na Tabela 1-19, medida:")
	for _, note := range notes {
		fmt.Printf("  %s\n", note)
	}
	fmt.Printf("\norigens: %d | ancoraram nas DUAS declarações: %d | em UMA só: %d | em NENHUMA (≠ corretas): %d "+
		"| exceção nomeada: %s\n", len(sources), inBoth, inOne, len(inNone), withoutList)
	fmt.Printf("verbetes achados: %d/%d | linhas da tabela: %d/%d\n", len(entries), len(sources), len(fromTable), len(sources))
	fmt.Printf("BENEFÍCIOS COMPARADOS: pericias %d, poderes %d, unico %d, regra_do_unico %d, pagina %d, itens %d\n",
		m.expertises, m.powers, m.unique, m.uniqueRule, m.page, m.items)
	// A cobertura de TODAS sai impressa, e não só a das reprovadas: uma lista de
	// reprovados vazia e um medidor que não mediu se parecem no terminal.
	fmt.Printf("cobertura da regra do poder único, da pior para a melhor (%d medidas, piso %s):\n",
		len(m.coverage), pyfmt.Percent(minCoverage))
	sort.Slice(m.coverage, func(i, j int) bool {
		if m.coverage[i].ratio != m.coverage[j].ratio {
			return m.coverage[i].ratio < m.coverage[j].ratio
		}
		return m.coverage[i].name < m.coverage[j].name
	})
	for _, c := range m.coverage {
		fmt.Printf("  %5s %s\n", pyfmt.Percent(c.ratio), c.name)
	}
	fmt.Printf("divergem do livro: %d | o livro discorda de si: %d | os dois catálogos discordam: %d\n",
		diverge, len(bookSelf), len(betweenCatalogs))
	if diverge > 0 || len(bookSelf) > 0 || len(betweenCatalogs) > 0 || len(inNone) > 0 {
		return 1
	}
	return 0
}
