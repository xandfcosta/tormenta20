// Command gods confere o Panteão (`transcript.Gods`) contra o Capítulo 1 do
// livro (p96-105). Seção da ALE-391.
//
// São DUAS leituras, que se conferem uma à outra em energia e poderes antes de
// encostar no catálogo: a Tabela 1-20 (p97) e o verbete, com os campos
// rotulados. O verbete acrescenta símbolo, arma, devotos e a página.
//
// O símbolo e a arma do catálogo são prosa REESCRITA (o catálogo tira o
// artigo), e comparar frase com frase dá 6 de 20 com o ruído escondendo o
// sinal. O que se mede é CONTENÇÃO de palavras de conteúdo: o catálogo não pode
// dizer o que o livro não diz. Os devotos são vocabulário FECHADO, e o que o
// auditor não reconhece ele imprime numa lista nomeada em vez de descartar.
//
//	go run ./cmd/audit/gods
package main

import (
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"
	"t20engine/cmd/internal/pyfmt"

	"t20engine/domain/catalog/transcript"
	"t20engine/infra/bookpdf"
)

const noWeapon = "não há"

// emptyWords não distinguem um símbolo de outro. O catálogo tira o artigo por
// estilo, e contá-las transformaria estilo em divergência.
var emptyWords = func() map[string]bool {
	m := map[string]bool{}
	for _, w := range strings.Fields("um uma o a os as de do da dos das e ou com em no na ao para outros outro") {
		m[w] = true
	}
	return m
}()

// acceptedCompression é a ÚNICA compressão editorial aceita, nomeada — é um
// PERMITIDOS, e uma segunda compressão passa a reprovar em vez de se esconder
// num total. Allihanna: "o símbolo corresponde ao respectivo animal" vira
// "animal totêmico".
var acceptedCompression = []struct {
	god, field string
	words      map[string]bool
}{{"Allihanna", "simbolo", map[string]bool{"totemico": true}}}

func contentWords(text string) map[string]bool {
	out := map[string]bool{}
	for _, w := range strings.Fields(bookpdf.NormalizePhrase(text)) {
		if !emptyWords[w] {
			out[w] = true
		}
	}
	return out
}

func minus(a, b map[string]bool) map[string]bool {
	out := map[string]bool{}
	for k := range a {
		if !b[k] {
			out[k] = true
		}
	}
	return out
}

func sameSet(a, b map[string]bool) bool {
	return len(minus(a, b)) == 0 && len(minus(b, a)) == 0
}

// devoteVocabulary são os termos que uma lista de "Devotos" pode usar, com a
// forma canônica. Vocabulário FECHADO: o que não estiver aqui o auditor
// REPROVA — uma lista de proibidos subconta em silêncio, e esta é a inversão.
func devoteVocabulary() map[string]string {
	terms := map[string]string{}
	var order []string
	add := func(k, v string) {
		if _, ok := terms[k]; !ok {
			order = append(order, k)
		}
		terms[k] = v
	}
	for _, r := range transcript.Races {
		add(bookpdf.Key(r.Name), r.Name)
	}
	for _, c := range transcript.Classes {
		add(bookpdf.Key(c.Name), c.Name)
	}
	// Plural: o livro lista "elfos", o catálogo guarda "Elfo". A ordem importa
	// quando dois plurais colidem, e é a de inserção, como no dicionário do Python.
	snapshot := append([]string{}, order...)
	values := map[string]string{}
	for _, k := range snapshot {
		values[k] = terms[k]
	}
	for _, k := range snapshot {
		v := values[k]
		add(k+"s", v)                             // elfo -> elfos
		add(k+"es", v)                            // caçador -> caçadores
		add(strings.TrimSuffix(k, "ao")+"oes", v) // ladrão -> ladrões
	}
	// Os que não são raça nem classe do catálogo: as metades do suraggel, o povo
	// do Thwor, e as duas aberturas ("quaisquer", "aventureiros").
	for _, extra := range []string{"aggelus", "sulfure", "duyshidakk", "quaisquer", "aventureiros",
		"golens", "hynne", "kliren", "trogs", "medusas", "sereias", "tritoes",
		"silfides", "osteon", "dahllan", "qareen"} {
		if _, ok := terms[extra]; !ok {
			terms[extra] = extra
		}
	}
	return terms
}

// termsOf são os termos reconhecidos e as palavras que não são termo conhecido.
func termsOf(text string, vocabulary map[string]string) (map[string]bool, map[string]bool) {
	known, strange := map[string]bool{}, map[string]bool{}
	for _, w := range strings.Fields(bookpdf.NormalizePhrase(text)) {
		if v, ok := vocabulary[w]; ok {
			known[v] = true
		} else if !emptyWords[w] {
			strange[w] = true
		}
	}
	return known, strange
}

func check(god transcript.God, e map[string]fieldValue, page int, vocabulary map[string]string,
	strange map[string]bool, byClass map[string]map[string]bool) []string {
	var out []string
	energy := strings.ToLower(e["Canalizar Energia"].text)
	if god.Energy != energy {
		out = append(out, fmt.Sprintf("energia: catálogo %s × livro %s", pyfmt.Repr(god.Energy), pyfmt.Repr(energy)))
	}

	powers := splitPowers(e["Poderes Concedidos"].text, true)
	if !slices.Equal(god.GrantedPowers, powers) {
		out = append(out, fmt.Sprintf("poderes: catálogo %s × livro %s", pyfmt.List(god.GrantedPowers), pyfmt.List(powers)))
	}

	if god.BookPage != page {
		out = append(out, fmt.Sprintf("bookPage: catálogo %d × livro %d", god.BookPage, page))
	}

	weapon := ""
	if god.FavoredWeapon != nil {
		weapon = *god.FavoredWeapon
	}
	inBook := e["Arma Preferida"].text
	if strings.HasPrefix(strings.ToLower(inBook), noWeapon) {
		if weapon != "" {
			out = append(out, fmt.Sprintf("armaPreferida: catálogo %s × livro \"não há\" (o deus não tem arma preferida)", pyfmt.Repr(weapon)))
		}
	} else if extra := minus(contentWords(weapon), contentWords(inBook)); len(extra) > 0 {
		out = append(out, fmt.Sprintf("armaPreferida: o catálogo diz %s, que o livro não diz (%s)", pyfmt.Sorted(extra), pyfmt.Repr(inBook)))
	}

	if extra := minus(contentWords(god.Symbol), contentWords(e["Símbolo Sagrado"].text)); len(extra) > 0 {
		accepted := false
		for _, c := range acceptedCompression {
			if c.god == god.Name && c.field == "simbolo" && sameSet(extra, c.words) {
				accepted = true
			}
		}
		if !accepted {
			out = append(out, fmt.Sprintf("simbolo: o catálogo diz %s, que o livro não diz (%s)", pyfmt.Sorted(extra), e["Símbolo Sagrado"].repr()))
		}
	}

	fromBook, unknown := termsOf(e["Devotos"].text, vocabulary)
	for w := range unknown {
		strange[w] = true
	}
	fromCatalog, unknownCatalog := termsOf(strings.Join(god.Devotees, " "), vocabulary)
	for w := range unknownCatalog {
		strange[w] = true
	}
	if !sameSet(fromBook, fromCatalog) {
		out = append(out, fmt.Sprintf("devotos: só no catálogo %s, só no livro %s",
			pyfmt.Sorted(minus(fromCatalog, fromBook)), pyfmt.Sorted(minus(fromBook, fromCatalog))))
	}

	for _, c := range []struct {
		field, who string
		value      bool
	}{{"paladinoEligible", "paladinos", god.PaladinEligible}, {"druidaEligible", "druidas", god.DruidEligible}} {
		allowed, ok := byClass[c.who]
		if !ok {
			continue // a frase da classe não foi achada; já saiu como NÃO MEDIDO
		}
		if c.value != allowed[god.Name] {
			out = append(out, fmt.Sprintf("%s: catálogo %s × livro %s (a classe lista %s)",
				c.field, pyfmt.Bool(c.value), pyfmt.Bool(allowed[god.Name]), pyfmt.Sorted(allowed)))
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
	gods := transcript.Gods
	names := map[string]string{}
	byName := map[string]transcript.God{}
	for _, g := range gods {
		names[bookpdf.Key(g.Name)] = g.Name
		byName[g.Name] = g
	}
	classPages := map[string]int{}
	for _, c := range transcript.Classes {
		classPages[c.Name] = c.BookPage
	}
	vocabulary := devoteVocabulary()

	table := readTable(book)
	blocks, complaints := chapterBlocks(book, names)
	byClass, classComplaints := godsByClass(book, classPages)
	complaints = append(complaints, classComplaints...)

	// CONTROLE da ligação: um deus por bloco, e a ordem tem de sair alfabética.
	var linked []string
	distinct := map[string]bool{}
	for _, b := range blocks {
		linked = append(linked, b.god)
		distinct[b.god] = true
	}
	if len(distinct) != len(linked) {
		complaints = append(complaints, "algum deus ficou com dois blocos: "+pyfmt.List(linked))
	}
	if !sort.StringsAreSorted(linked) {
		complaints = append(complaints, "a ordem dos blocos não saiu alfabética — a geometria errou: "+pyfmt.List(linked))
	}

	quarrels := 0
	for _, b := range blocks {
		e := readEntry(b.body)
		row, ok := table[bookpdf.Key(b.god)]
		if !ok {
			complaints = append(complaints, b.god+": sem linha na Tabela 1-20")
			continue
		}
		if row.energy != strings.ToLower(e["Canalizar Energia"].text) {
			fmt.Printf("  INSTRUMENTOS DISCORDAM %s.energia: tabela %s × verbete %s\n",
				b.god, pyfmt.Repr(row.energy), e["Canalizar Energia"].repr())
			quarrels++
		}
		fromEntry := splitPowers(e["Poderes Concedidos"].text, false)
		if !slices.Equal(row.powers, fromEntry) {
			fmt.Printf("  INSTRUMENTOS DISCORDAM %s.poderes: tabela %s × verbete %s\n",
				b.god, pyfmt.List(row.powers), pyfmt.List(fromEntry))
			quarrels++
		}
	}

	diverge, strange := 0, map[string]bool{}
	for _, b := range blocks {
		for _, complaint := range check(byName[b.god], readEntry(b.body), b.page, vocabulary, strange, byClass) {
			fmt.Printf("  %s: %s\n", b.god, complaint)
			diverge++
		}
	}

	for _, c := range complaints {
		fmt.Printf("  NÃO MEDIDO %s\n", c)
	}
	for _, c := range acceptedCompression {
		fmt.Printf("  COMPRESSÃO ACEITA %s.%s: %s resume o que o livro diz por extenso\n", c.god, c.field, pyfmt.Sorted(c.words))
	}
	fmt.Println("  NÃO MEDIDO major: é `true` nos vinte, e o livro não separa maior de menor")
	fmt.Println("  NÃO MEDIDO portfolio: é resumo de três palavras sobre o parágrafo " +
		"\"Crenças e Objetivos\" — prosa reescrita, não transcrição")

	fmt.Printf("\ndeuses: %d | lidos na Tabela 1-20: %d | verbetes ligados: %d | instrumentos discordam: %d "+
		"| divergem do livro: %d\n", len(gods), len(table), len(blocks), quarrels, diverge)
	if len(strange) > 0 {
		fmt.Printf("palavras da lista de Devotos que não são termo conhecido "+
			"(%d, para conferir que nenhuma é raça ou classe): %s\n", len(strange), pyfmt.Sorted(strange))
	}
	if diverge > 0 || quarrels > 0 || len(complaints) > 0 {
		return 1
	}
	return 0
}
