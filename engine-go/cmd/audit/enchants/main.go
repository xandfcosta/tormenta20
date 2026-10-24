// Command enchants confere os ENCANTOS do catálogo contra o livro (ALE-411,
// ALE-439).
//
// São DUAS TABELAS e UM auditor: a 8-8 é de armas (p335-336) e a 8-10 de
// armaduras e escudos (p338-339), com a mesma FORMA — faixa de `d%` + nome +
// efeito na tabela, `Nome. Regra` no verbete. Por isso são PERFIS, e não dois
// comandos: uma segunda cópia divergiria na primeira correção de leitura.
//
// As duas âncoras se cobram antes de o catálogo ser mencionado: nome numa e não
// na outra é defeito de LEITURA. E o denominador é de COBERTURA — o catálogo
// começou vazio, então o relatório diz sempre quantos a tabela tem, quantos
// entraram e quantos ficaram sem modificador, com o MOTIVO. Um encanto com
// modificador não está pronto por isso: a Flamejante é "+1d6 de fogo" E uma
// bola de fogo de 2 PM, e a terceira dimensão é a cobertura de palavras.
//
//	go run ./cmd/audit/enchants            # as duas tabelas
//	go run ./cmd/audit/enchants armadura   # uma só
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

// coverageFloor é alto porque a descrição do catálogo é TRANSCRITA do verbete
// e nasce em 100%: ele prende o dia em que alguém "melhorar" a frase e ela
// passar a dizer outra coisa (ALE-408).
const coverageFloor = 0.95

// profile é uma das duas tabelas de encanto, com onde ela mora na página. Tudo
// foi MEDIDO no PDF; o x do dado é FAIXA porque a coluna é alinhada à direita
// (na 8-8, "01-05" começa em 59,1 e "46" em 65,1).
type profile struct {
	key, table      string
	tablePage       int
	entryPages      []int
	diceX           [2]float64
	nameX, effectX  float64
	proseColumns    map[int][]float64 // pares (corpo, primeira linha recuada 17 pontos)
	tableRegion     map[int][2]float64
	notEnchant      string
	countsDouble    map[string]bool
	entriesEnd      string
	catalogCategory string
}

var profileOrder = []string{"arma", "armadura"}

var profiles = map[string]profile{
	// A Tabela 8-8 é lida SEM o `WithoutJunk`: o padrão de número de página
	// (`^\d{1,3}$`) casa com as faixas de UM valor, e a limpeza comia a
	// Energética (46) e a Lancinante (64) — o que denunciou foi a soma não
	// fechar em 100.
	//
	// E na p336 o x NÃO BASTA: a prosa começa em 51 e o d% cai entre 57 e 68.
	// Quem separa é a ALTURA — a tabela acaba em y=527 e a prosa de baixo
	// começa em y=596. Recortar só por x perdia três verbetes, a Magnífica
	// entre eles.
	"arma": {
		key: "arma", table: "8-8",
		tablePage: 342, entryPages: []int{341, 342},
		diceX: [2]float64{56.0, 67.0}, nameX: 90.8, effectX: 154.3,
		proseColumns: map[int][]float64{341: {71.0, 88.0, 309.0, 326.0}, 342: {51.0, 68.0, 289.0, 306.0}},
		tableRegion:  map[int][2]float64{342: {200.0, 540.0}},
		// 29 linhas e 28 encantos: "91-100 Arma específica" manda rolar em
		// OUTRA tabela. Declarada para o denominador não mudar em silêncio.
		notEnchant: "Arma específica",
		// O asterisco: "*Conta como dois encantos."
		countsDouble: map[string]bool{"Energética": true, "Lancinante": true, "Magnífica": true},
		// O último verbete (Venenosa) não tem sucessor que o pare, e sem isto
		// engolia "Armas Específicas" e o "Arco do Poder".
		entriesEnd:      "Armas Específicas",
		catalogCategory: "weapon-enchant",
	},
	// A 8-10 mora na coluna direita da p339; os verbetes vão da p338 ao
	// Zeloso, a última coisa da página — por isso não há sentinela de fim. A
	// prosa (309/326) e o dado (317) distam oito pontos contra uma folga de
	// três, e este perfil não precisa da região por altura.
	"armadura": {
		key: "armadura", table: "8-10",
		tablePage: 345, entryPages: []int{344, 345},
		diceX: [2]float64{313.0, 319.0}, nameX: 348.7, effectX: 412.2,
		proseColumns:    map[int][]float64{344: {289.1, 306.1}, 345: {70.9, 87.9, 309.0, 326.0}},
		tableRegion:     map[int][2]float64{},
		notEnchant:      "Item específico",
		countsDouble:    map[string]bool{"Guardião": true},
		catalogCategory: "armor-enchant",
	},
}

func main() {
	chosen := os.Args[1:]
	if len(chosen) == 0 {
		chosen = profileOrder
	}
	var unknown []string
	for _, c := range chosen {
		if _, ok := profiles[c]; !ok {
			unknown = append(unknown, c)
		}
	}
	if len(unknown) > 0 {
		fmt.Printf("não conheço %s; os perfis são %s\n", pyfmt.List(unknown), pyfmt.List(profileOrder))
		os.Exit(1)
	}
	book, err := bookpdf.Open()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	total := 0
	for _, c := range chosen {
		total += audit(book, profiles[c])
	}
	if total > 0 {
		os.Exit(1)
	}
}

// audit audita UMA tabela e devolve quantas falhas achou.
func audit(book *bookpdf.Book, p profile) int {
	fmt.Printf("\n══ TABELA %s (%s) %s\n", p.table, p.key, strings.Repeat("═", 40))
	table := readTable(book, p)
	if gap := rangesThatDontClose(table); gap != "" {
		fmt.Printf("A TABELA %s NÃO FECHA: %s\n", p.table, gap)
		fmt.Println("Nada do que vem abaixo vale — o auditor leu a página errado.")
		return 1
	}

	fromTable := map[string]tableRow{}
	for _, r := range table {
		if r.name != p.notEnchant {
			fromTable[r.name] = r
		}
	}
	fromEntries := readEntries(book, p)
	fmt.Printf("AS DUAS ÂNCORAS  tabela: %d linhas (soma fecha em 100), %d encantos + \"%s\"\n", len(table), len(fromTable), p.notEnchant)
	fmt.Printf("                 verbetes: %d\n", len(fromEntries))
	var onlyTable, onlyEntry []string
	for n := range fromTable {
		if _, ok := fromEntries[n]; !ok {
			onlyTable = append(onlyTable, n)
		}
	}
	for n := range fromEntries {
		if _, ok := fromTable[n]; !ok {
			onlyEntry = append(onlyEntry, n)
		}
	}
	sort.Strings(onlyTable)
	sort.Strings(onlyEntry)
	if len(onlyTable) > 0 || len(onlyEntry) > 0 {
		fmt.Printf("  AS DUAS DISCORDAM — só na tabela: %s; só no verbete: %s\n", pyfmt.List(onlyTable), pyfmt.List(onlyEntry))
		fmt.Println("  É defeito de LEITURA, e ele vem antes de qualquer correção de catálogo.")
		return 1
	}
	fmt.Printf("                 as duas dizem os MESMOS %d nomes\n", len(fromTable))

	catalog := map[string]transcript.Item{}
	for _, it := range transcript.Items {
		if it.Category == p.catalogCategory {
			catalog[it.Name] = it
		}
	}
	fmt.Printf("\nCATÁLOGO         %d entradas em `%s`\n", len(catalog), p.catalogCategory)

	type partial struct {
		name    string
		modeled []string
		reason  string
	}
	type bare struct{ name, reason string }
	whole := 0
	var inPart []partial
	var without []bare
	var missing, failures []string
	lowest, lowestName := 2.0, ""
	names := make([]string, 0, len(fromTable))
	for n := range fromTable {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		row := fromTable[name]
		entry, ok := catalog[name]
		if !ok {
			missing = append(missing, name)
			failures = append(failures, fmt.Sprintf("%s: está na Tabela %s e não está no catálogo", name, p.table))
			continue
		}
		fromBook := fromEntries[name]
		ratio, shared, total := bookpdf.Coverage(fromBook.rule, entry.Description)
		if ratio < lowest || (ratio == lowest && name < lowestName) {
			lowest, lowestName = ratio, name
		}
		if ratio < coverageFloor {
			failures = append(failures, fmt.Sprintf("%s: a descrição do catálogo repete %d das %d palavras de conteúdo do verbete (%s) — abaixo do piso de %s. Ela deixou de dizer a regra do livro.",
				name, shared, total, pyfmt.Percent(ratio), pyfmt.Percent(coverageFloor)))
		}
		if entry.RollMin != row.start || entry.RollMax != row.end {
			failures = append(failures, fmt.Sprintf("%s: o catálogo diz d%% %s-%s e a Tabela %s diz %d-%d",
				name, pyfmt.Optional(float64(entry.RollMin)), pyfmt.Optional(float64(entry.RollMax)), p.table, row.start, row.end))
		}
		expected := 1
		if p.countsDouble[name] {
			expected = 2
		}
		if entry.CountsAs != expected {
			failures = append(failures, fmt.Sprintf("%s: o catálogo conta como %s e o livro diz %d (o asterisco da tabela é \"conta como dois encantos\")",
				name, pyfmt.Optional(float64(entry.CountsAs)), expected))
		}
		if entry.BookPage != fromBook.page {
			failures = append(failures, fmt.Sprintf("%s: o catálogo cita a p%s e o verbete está na p%d", name, pyfmt.Optional(float64(entry.BookPage)), fromBook.page))
		}

		// TRÊS baldes, e não dois. Ter modificador e ter motivo eram
		// exclusivos, pela premissa de que um encanto cabe inteiro no motor ou
		// nada dele — falsa em dez dos vinte e oito, e o efeito era SILÊNCIO:
		// escrever a nota honesta reprovava (ALE-459). Quem aplica EM PARTE diz
		// as duas coisas, e o `modeled` impede a nota de virar desculpa.
		hasModel := len(entry.Modifiers) > 0
		reason := ""
		if entry.Unmodeled != nil {
			reason = entry.Unmodeled.Note
		}
		switch {
		case hasModel && reason != "":
			inPart = append(inPart, partial{name, entry.Modeled, reason})
		case hasModel:
			whole++
		default:
			without = append(without, bare{name, reason})
		}
		if !hasModel && reason == "" {
			failures = append(failures, name+": não tem modificador NEM diz por quê")
		}
		if hasModel && reason != "" && len(entry.Modeled) == 0 {
			failures = append(failures, fmt.Sprintf("%s: aplica alguma coisa, diz o que NÃO entrou (%s) e não diz o que ENTROU — sem o `modeled` a nota parece falar do encanto inteiro",
				name, pyfmt.Repr(reason)))
		}
	}

	fmt.Printf("  no catálogo:     %d de %d\n", len(fromTable)-len(missing), len(fromTable))
	if len(missing) > 0 {
		fmt.Printf("  FALTAM %d: %s\n", len(missing), strings.Join(missing, ", "))
	}
	fmt.Printf("  INTEIROS:        %d\n", whole)
	fmt.Printf("  EM PARTE:        %d, e cada um diz as duas metades:\n", len(inPart))
	for _, e := range inPart {
		fmt.Printf("    %-16s entrou: %s\n", e.name, strings.Join(e.modeled, "; "))
		fmt.Printf("    %-16s   ficou: %s\n", "", e.reason)
	}
	fmt.Printf("  SEM modificador: %d, e cada um diz por quê:\n", len(without))
	for _, e := range without {
		fmt.Printf("    %-16s %s\n", e.name, e.reason)
	}
	if lowestName != "" {
		fmt.Printf("\n  cobertura mínima: %s (%s)\n", pyfmt.Percent(lowest), lowestName)
	}
	fmt.Printf("\nFALHAS: %d   (medidos: %d encantos, %d linhas de tabela, %d verbetes)\n",
		len(failures), len(fromTable), len(table), len(fromEntries))
	for _, f := range failures {
		fmt.Printf("  %s\n", f)
	}
	return len(failures)
}
