// Command races confere as raças (`transcript.Races`) contra o Capítulo 1 do
// livro (p19-31). Seção 1 da ALE-391.
//
// Mede os NÚMEROS da raça — atributos, deslocamento, tamanho e as duas visões —
// e a presença de cada habilidade que o catálogo declara. Não mede a descrição:
// o catálogo reescreve a frase em forma curta de propósito.
//
// A âncora não é o título "Habilidades de Raça" (ele quebra no Elfo e as raças
// extras não o imprimem) nem o `bookPage` (o bloco do Hynne atravessa a página).
// É a LINHA DE ATRIBUTOS, ligada à raça em três regras, nesta ordem:
//
//  1. a linha traz o rótulo da subraça — `(Aggelus)`, `(Sulfure)`. Ela corre
//     PRIMEIRO: a habilidade colada na linha do Suraggel é "Mau Cheiro", do Trog;
//  2. vem colada à primeira habilidade da raça — ou É ela, no Humano;
//  3. fecho da bijeção: sobrando uma raça e uma linha, elas são par. É o que
//     alcança o Trog, cujas habilidades vêm ANTES da linha dele.
//
// Deslocamento, tamanho e visão se leem no TEXTO DAS HABILIDADES DA PRÓPRIA
// RAÇA, achado nome a nome — nunca numa faixa de página. O que não ligou, e a
// habilidade não achada, saem NOMEADOS.
//
//	go run ./cmd/audit/races
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

type abilityKey struct{ race, name string }

// abilityPositions é onde cada habilidade começa no capítulo, na ordem em que
// foi ancorada — a ordem decide em que sequência a região da raça se monta.
type abilityPositions struct {
	order []abilityKey
	at    map[abilityKey]int
}

func (p *abilityPositions) set(k abilityKey, pos int) {
	if _, ok := p.at[k]; !ok {
		p.order = append(p.order, k)
	}
	p.at[k] = pos
}

// findAbilities acha cada habilidade do catálogo no livro.
func findAbilities(c chapter, races []transcript.Race) (abilityPositions, []string) {
	where := abilityPositions{at: map[abilityKey]int{}}
	var missing []string
	type ambiguous struct {
		race       transcript.Race
		name       string
		candidates []int
	}
	var ambiguities []ambiguous
	for _, r := range races {
		for _, a := range r.Abilities {
			var found []int
			for _, m := range regexp.MustCompile(regexp.QuoteMeta(a.Name+".")).FindAllStringIndex(c.text, -1) {
				found = append(found, c.index[m[0]])
			}
			switch len(found) {
			case 0:
				missing = append(missing, r.Name+" / "+a.Name+": não achada no livro")
			case 1:
				where.set(abilityKey{r.ID, a.Name}, found[0])
			default:
				ambiguities = append(ambiguities, ambiguous{r, a.Name, found})
			}
		}
	}
	// Nem todo nome é único: "+1 em Três Atributos Diferentes" é habilidade do
	// Humano e linha de atributos da Sereia/Tritão. Desempata a distância às
	// habilidades da MESMA raça que já ancoraram.
	for _, amb := range ambiguities {
		var anchors []int
		for _, k := range where.order {
			if k.race == amb.race.ID {
				anchors = append(anchors, where.at[k])
			}
		}
		if len(anchors) == 0 {
			missing = append(missing, fmt.Sprintf("%s / %s: achada %dx e sem âncora na mesma raça para desempatar",
				amb.race.Name, amb.name, len(amb.candidates)))
			continue
		}
		best, bestDistance := 0, -1
		for _, p := range amb.candidates {
			d := -1
			for _, a := range anchors {
				if dist := abs(p - a); d < 0 || dist < d {
					d = dist
				}
			}
			if bestDistance < 0 || d < bestDistance {
				best, bestDistance = p, d
			}
		}
		where.set(abilityKey{amb.race.ID, amb.name}, best)
	}
	return where, missing
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// link liga cada linha de atributos à raça, pelas três regras do cabeçalho.
// A ordem é a regra: deixar a colagem correr antes do rótulo dá a linha do
// Suraggel ao Trog, com a bijeção fechando por cima e nada reclamando.
func link(lines []attributeLine, where abilityPositions, races []transcript.Race) (map[string]attributeLine, []string) {
	byRace := map[string]attributeLine{}
	taken := map[int]bool{}
	for _, l := range lines {
		labels, _ := readAttributeLine(l.text).get("variants")
		if len(labels.dict) == 0 {
			continue
		}
		for _, r := range races {
			if subset(labels.dict, r.AttributeRule.Variants) {
				if _, ok := byRace[r.ID]; !ok {
					byRace[r.ID] = l
				}
				taken[l.start] = true
			}
		}
	}
	for _, l := range lines {
		if taken[l.start] {
			continue
		}
		owners := map[string]bool{}
		var first string
		for _, k := range where.order {
			if pos := where.at[k]; pos == l.end+1 || pos == l.start {
				if len(owners) == 0 {
					first = k.race
				}
				owners[k.race] = true
			}
		}
		if _, ok := byRace[first]; len(owners) == 1 && !ok {
			byRace[first] = l
			taken[l.start] = true
		}
	}
	var leftRaces []string
	for _, r := range races {
		if _, ok := byRace[r.ID]; !ok {
			leftRaces = append(leftRaces, r.ID)
		}
	}
	var leftLines []attributeLine
	for _, l := range lines {
		if !taken[l.start] {
			leftLines = append(leftLines, l)
		}
	}
	if len(leftRaces) == 1 && len(leftLines) == 1 {
		byRace[leftRaces[0]] = leftLines[0]
		leftRaces, leftLines = nil, nil
	}
	names := map[string]string{}
	for _, r := range races {
		names[r.ID] = r.Name
	}
	var complaints []string
	for _, id := range leftRaces {
		complaints = append(complaints, names[id]+": nenhuma linha de atributos ligou")
	}
	for _, l := range leftLines {
		complaints = append(complaints, "linha sem dona: "+pyfmt.Repr(l.text))
	}
	return byRace, complaints
}

func subset(labels pyDict, declared transcript.OrderedMap[transcript.OrderedMap[int]]) bool {
	for _, e := range labels {
		if _, ok := declared.Get(e.key); !ok {
			return false
		}
	}
	return true
}

// region é o texto das habilidades DESTA raça, cada uma até a próxima de
// QUALQUER raça: o Suraggel e o Trog imprimem as suas alternadas.
func region(c chapter, where abilityPositions, cuts []int, raceID string) string {
	var pieces []string
	for _, k := range where.order {
		if k.race != raceID {
			continue
		}
		pos := where.at[k]
		end := pos + regionFallback
		for _, cut := range cuts {
			if cut > pos {
				end = cut
				break
			}
		}
		pieces = append(pieces, c.slice(pos, end))
	}
	return strings.Join(pieces, " ")
}

// printed conta quantas vezes o livro DISSE cada coisa, em vez de o auditor ter
// caído no padrão: sem ela, um regex de deslocamento que parasse de casar
// devolveria 9m para as dezessete e metade da tabela passaria por medição.
type printed struct{ displacement, size, darkvision, lowLight int }

func check(r transcript.Race, read pyDict, body string, page int, havePage bool,
	extras int, haveExtras bool, seen *printed) []string {
	var out []string
	if catalog := ruleDict(r.AttributeRule); !catalog.equal(read) {
		out = append(out, fmt.Sprintf("atributos: catálogo %s × livro %s", catalog.repr(), read.repr()))
	}

	text, found := displacement(body)
	moves := defaultDisplacement
	if found {
		moves = num(text)
		seen.displacement++
	}
	if r.Displacement != moves {
		where := "padrão do livro (nenhuma habilidade o muda)"
		if found {
			where = "impresso"
		}
		out = append(out, fmt.Sprintf("deslocamento: catálogo %dm × livro %dm (%s)", r.Displacement, moves, where))
	}

	size := defaultSize
	if m := sizeRe.FindStringSubmatch(body); m != nil {
		size = m[1]
		seen.size++
	}
	if r.Size != size {
		out = append(out, fmt.Sprintf("tamanho: catálogo %s × livro %s", r.Size, size))
	}

	if havePage {
		if r.BookPage != page {
			out = append(out, fmt.Sprintf("bookPage: catálogo %d × livro %d", r.BookPage, page))
		}
		if haveExtras {
			tier := "comum"
			if page >= extras {
				tier = "extra"
			}
			if r.Tier != tier {
				out = append(out, fmt.Sprintf("tier: catálogo %s × livro %s (\"%s\" abre na p%d)", r.Tier, tier, extrasSection, extras))
			}
		}
	}

	for _, v := range []struct {
		field, phrase string
		value         bool
		count         *int
	}{
		{"visaoNoEscuro", "visão no escuro", r.Darkvision, &seen.darkvision},
		{"visaoNaPenumbra", "visão na penumbra", r.LowLightVision, &seen.lowLight},
	} {
		inBook := strings.Contains(body, v.phrase)
		if inBook {
			*v.count++
		}
		if v.value != inBook {
			out = append(out, fmt.Sprintf("%s: catálogo %s × livro %s", v.field, pyfmt.Bool(v.value), pyfmt.Bool(inBook)))
		}
	}
	return out
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
	races := transcript.Races
	c := readChapter(book)
	lines := findAttributeLines(c)
	where, missing := findAbilities(c, races)
	byRace, complaints := link(lines, where, races)
	pages, extras, haveExtras := titlePages(book, races)
	for _, r := range races {
		if _, ok := pages[r.ID]; !ok {
			complaints = append(complaints, r.Name+": título não achado no capítulo — bookPage e tier não medidos")
		}
	}
	cutSet := map[int]bool{}
	for _, pos := range where.at {
		cutSet[pos] = true
	}
	for _, l := range lines {
		cutSet[l.start] = true
	}
	var cuts []int
	for pos := range cutSet {
		cuts = append(cuts, pos)
	}
	sort.Ints(cuts)

	diverge := 0
	var seen printed
	for _, r := range races {
		line, ok := byRace[r.ID]
		if !ok {
			continue
		}
		page, havePage := pages[r.ID]
		for _, complaint := range check(r, readAttributeLine(line.text), region(c, where, cuts, r.ID),
			page, havePage, extras, haveExtras, &seen) {
			fmt.Printf("  %s: %s\n", r.Name, complaint)
			diverge++
		}
	}

	for _, complaint := range append(complaints, missing...) {
		fmt.Printf("  NÃO MEDIDO %s\n", complaint)
	}

	fmt.Printf("\nraças: %d | ligadas a uma linha de atributos: %d | habilidades achadas no livro: %d "+
		"| divergem do livro: %d\n", len(races), len(byRace), len(where.at), diverge)
	fmt.Printf("títulos de raça achados no capítulo (bookPage e tier medidos): %d/%d\n", len(pages), len(races))
	fmt.Printf("o livro IMPRIMIU (o resto caiu no padrão): deslocamento %dx, tamanho %dx, visaoNoEscuro %dx, visaoNaPenumbra %dx\n",
		seen.displacement, seen.size, seen.darkvision, seen.lowLight)
	if diverge > 0 || len(complaints) > 0 || len(missing) > 0 {
		return 1
	}
	return 0
}
