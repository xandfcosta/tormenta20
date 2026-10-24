// Command grantedpowers confere os PODERES CONCEDIDOS contra a seção deles (p127
// e p132-136).
//
// Os concedidos moram em DOIS catálogos — `DivinePowers` (a lista completa, um
// par por deus) e `GrantedPowers` (o subconjunto que o motor alcança). Que os
// dois concordem entre si não precisa do livro e é do
// `TestEveryGrantedPowerSaysTheSameThingInBothFiles`; aqui fica o que precisa:
// se o que eles dizem é o que a página diz.
//
// O par (deus, poder) é declarado duas vezes — na TABELA da p127 ("Devoto de
// Azgher") e no VERBETE, com o deus num rótulo ao lado do título. O que é
// específico desta seção, e contraria o auditor irmão:
//
//   - o título QUEBRA em duas linhas ("Afinidade com" / "a Tormenta"), e aqui o
//     casamento de duas linhas é obrigatório;
//   - o rótulo do deus tem duas diagramações: linha própria, ou colado na
//     segunda linha do título;
//   - um poder pode ter DOIS deuses, na tabela e no verbete.
//
// Ele PROPÕE: nada é escrito.
//
//	go run ./cmd/audit/grantedpowers
package main

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"t20engine/cmd/internal/pyfmt"

	"t20engine/domain/catalog/transcript"
	"t20engine/infra/bookpdf"
)

const (
	firstPage, lastPage = 138, 142 // PDF; os verbetes, livro p132-136
	tablePage           = 133      // PDF; livro p127
	pdfOffset           = 6
	minCoverage         = 0.5
)

var (
	devoteRe     = regexp.MustCompile(`(?i)^Devoto d[eoa]s?` + bookpdf.PySpace + `+(.+)`)
	godSeparator = regexp.MustCompile(`,| e | ou `)
)

// godsOfPhrase são os deuses que uma frase nomeia, e `false` se ela não é uma
// lista de deuses. "Esta linha não fala de deus" e "fala de zero deuses" têm de
// ser coisas diferentes: tratá-las igual faria toda linha de prosa virar um
// rótulo vazio.
func godsOfPhrase(phrase string, known map[string]bool) (map[string]bool, bool) {
	if m := devoteRe.FindStringSubmatch(bookpdf.Strip(phrase)); m != nil {
		phrase = m[1]
	}
	gods := map[string]bool{}
	for _, part := range godSeparator.Split(phrase, -1) {
		if bookpdf.Strip(part) == "" {
			continue
		}
		k := bookpdf.Key(part)
		if !known[k] {
			return nil, false
		}
		gods[k] = true
	}
	return gods, len(gods) > 0
}

// tableRow é o que a tabela da p127 diz de um poder.
type tableRow struct {
	name string
	gods map[string]bool
}

// readTable é a âncora 1: os deuses de cada poder na tabela da p127.
func readTable(book *bookpdf.Book, known map[string]bool) map[string]tableRow {
	fromTable := map[string]tableRow{}
	for _, table := range book.PowerTables(tablePage) {
		for _, pair := range table.Pairs {
			rows, _ := bookpdf.PairRows(table.Body, pair.PowerX, pair.PrereqX)
			for _, row := range rows {
				if gods, ok := godsOfPhrase(row.Prereq, known); ok {
					fromTable[bookpdf.Key(row.Power)] = tableRow{row.Power, gods}
				}
			}
		}
	}
	return fromTable
}

// entry é o verbete de um poder: a página do livro, os deuses do rótulo (se
// houver) e a regra.
type entry struct {
	page    int
	gods    map[string]bool
	hasGods bool
	rule    string
}

type pageLine struct {
	page int
	text string
}

type titleHit struct {
	index, page int
	key         string
	gods        map[string]bool
	hasGods     bool
}

// readEntries é a âncora 2: o verbete de cada poder, e quantas vezes cada
// título foi achado.
func readEntries(book *bookpdf.Book, names map[string]string, known map[string]bool) (map[string]entry, map[string]int) {
	consumed := book.TableCells(tablePage)
	var lines []pageLine
	for page := firstPage; page <= lastPage; page++ {
		for _, l := range book.Reading(page) {
			if page == tablePage && consumed[bookpdf.Cell{X: bookpdf.Round1(l.X), Y: bookpdf.Round1(l.Y)}] {
				continue
			}
			lines = append(lines, pageLine{page, l.Text})
		}
	}

	var hits []titleHit
	seen := map[string]int{}
	for i, line := range lines {
		// O TÍTULO E O RÓTULO PODEM DIVIDIR A LINHA ("Êxtase da Loucura Aharadak,
		// Nimb" numa só, "Presas" / "Primordiais Kallyadranoch, Megalokk" em
		// duas), e por isso o rótulo se captura no casamento do título: o que
		// sobra depois do nome é ele. Procurá-lo depois, por sufixo das linhas
		// seguintes, leu o deus de dentro do PRÓPRIO NOME — o "Inimigo de
		// Tenebra" é de Azgher, e "Tenebra" passou por rótulo.
		joins := []string{line.text}
		if i+1 < len(lines) {
			joins = append(joins, line.text+" "+lines[i+1].text)
		}
		found, rest := "", ""
		for _, joined := range joins {
			words := strings.Fields(joined)
			for cut := len(words); cut > 0; cut-- {
				k := bookpdf.Key(strings.Join(words[:cut], " "))
				if _, ok := names[k]; ok && seen[k] == 0 {
					found, rest = k, strings.Join(words[cut:], " ")
					break
				}
			}
			if found != "" {
				break
			}
		}
		if found != "" {
			seen[found]++
			gods, ok := godsOfPhrase(rest, known)
			hits = append(hits, titleHit{i, line.page, found, gods, ok})
		}
	}

	entries := map[string]entry{}
	for n, hit := range hits {
		end := len(lines)
		if n+1 < len(hits) {
			end = hits[n+1].index
		}
		var body []string
		for _, l := range lines[hit.index:end] {
			body = append(body, l.text)
		}
		// Sem rótulo colado ao título, ele está numa linha PRÓPRIA logo abaixo —
		// e só ela vale, para o nome do poder nunca entrar na conta.
		gods, ok := hit.gods, hit.hasGods
		if !ok {
			for j := 1; j < len(body) && j < 4; j++ {
				if gods, ok = godsOfPhrase(body[j], known); ok {
					break
				}
			}
		}
		entries[hit.key] = entry{hit.page - pdfOffset, gods, ok, bookpdf.Join(body)}
	}
	return entries, seen
}

func sameSet(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

type coverageRow struct {
	ratio float64
	name  string
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
	granted := transcript.GrantedPowers
	known := map[string]bool{}
	for _, g := range transcript.Gods {
		known[bookpdf.Key(g.Name)] = true
	}
	names := map[string]string{}
	byKey := map[string]transcript.GrantedPower{}
	var order []string
	for _, p := range granted {
		k := bookpdf.Key(p.Name)
		if _, again := byKey[k]; !again {
			order = append(order, k)
		}
		names[k] = p.Name
		byKey[k] = p
	}

	fromTable := readTable(book, known)
	entries, seen := readEntries(book, names, known)

	measured := map[string]int{}
	diverge, inBoth, inOne := 0, 0, 0
	var inNone, bookSelf []string
	var coverages []coverageRow
	for _, k := range order {
		power := byKey[k]
		fromCatalog := map[string]bool{}
		for _, g := range power.Gods {
			fromCatalog[bookpdf.Key(g)] = true
		}
		row, hasRow := fromTable[k]
		e, hasEntry := entries[k]
		tableSays := hasRow
		entrySays := hasEntry && e.hasGods
		switch {
		case !tableSays && !entrySays:
			inNone = append(inNone, power.Name+": não ancorou em nenhuma das duas "+
				"declarações — os deuses dele NÃO foram medidos")
		case !tableSays || !entrySays:
			inOne++
		default:
			inBoth++
			if !sameSet(row.gods, e.gods) {
				bookSelf = append(bookSelf, fmt.Sprintf("%s: a tabela diz %s e o verbete diz %s",
					power.Name, pyfmt.Sorted(row.gods), pyfmt.Sorted(e.gods)))
			}
		}
		var read map[string]bool
		switch {
		case entrySays:
			read = e.gods
		case tableSays:
			read = row.gods
		}
		if read != nil {
			measured["deuses"] += len(read)
			if !sameSet(read, fromCatalog) {
				fmt.Printf("  %s: deuses catálogo %s × livro %s\n", power.Name, pyfmt.Sorted(fromCatalog), pyfmt.Sorted(read))
				diverge++
			}
		}
		if !hasEntry {
			continue
		}
		measured["pagina"]++
		if power.BookPage != e.page {
			fmt.Printf("  %s: bookPage catálogo %d × livro %d\n", power.Name, power.BookPage, e.page)
			diverge++
		}
		if e.rule != "" && power.Effect != "" {
			measured["regra"]++
			// O denominador é o CATÁLOGO: a pergunta é "o que ele diz está no livro?".
			ratio, shared, total := bookpdf.Coverage(power.Effect, e.rule)
			coverages = append(coverages, coverageRow{ratio, power.Name})
			if ratio < minCoverage {
				fmt.Printf("  %s: %d das %d palavras do CATÁLOGO estão no livro (%s, piso %s)\n",
					power.Name, shared, total, pyfmt.Percent(ratio), pyfmt.Percent(minCoverage))
				diverge++
			}
		}
	}

	if len(bookSelf) > 0 {
		fmt.Println("\nO LIVRO DISCORDA DE SI MESMO (achado sobre o LIVRO):")
		for _, line := range bookSelf {
			fmt.Printf("  %s\n", line)
		}
	}
	for _, line := range inNone {
		fmt.Printf("  NÃO MEDIDO %s\n", line)
	}
	repeated := map[string]bool{}
	for k, n := range seen {
		if n > 1 {
			repeated[k] = true
		}
	}
	if len(repeated) > 0 {
		fmt.Printf("  NÃO MEDIDO títulos achados mais de uma vez: %s\n", pyfmt.Sorted(repeated))
	}

	fmt.Printf("\npoderes concedidos: %d | pares no divine: %d | ancoraram nas DUAS declarações: %d "+
		"| em UMA só: %d | em NENHUMA (≠ corretos): %d\n",
		len(granted), len(transcript.DivinePowers), inBoth, inOne, len(inNone))
	fmt.Printf("lidos na tabela da p127: %d | verbetes achados: %d/%d\n", len(fromTable), len(entries), len(granted))
	fmt.Printf("COMPARADOS: deuses %d, pagina %d, regra %d\n", measured["deuses"], measured["pagina"], measured["regra"])
	fmt.Printf("cobertura da regra, da pior para a melhor (piso %s):\n", pyfmt.Percent(minCoverage))
	sort.Slice(coverages, func(i, j int) bool {
		if coverages[i].ratio != coverages[j].ratio {
			return coverages[i].ratio < coverages[j].ratio
		}
		return coverages[i].name < coverages[j].name
	})
	for i, c := range coverages {
		if i == 8 {
			break
		}
		fmt.Printf("  %5s %s\n", pyfmt.Percent(c.ratio), c.name)
	}
	fmt.Printf("divergem do livro: %d | o livro discorda de si: %d\n", diverge, len(bookSelf))
	if diverge > 0 || len(bookSelf) > 0 || len(inNone) > 0 {
		return 1
	}
	return 0
}
