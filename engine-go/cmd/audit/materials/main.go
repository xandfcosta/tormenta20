// Command materials confere os MATERIAIS ESPECIAIS do catálogo contra o livro
// (ALE-415).
//
// O `audit-equipment` confere PREÇO por tabela do capítulo 3, e os seis
// materiais não estão nas tabelas que ele lê: a REGRA de um material nunca foi
// medida, e cinco dos seis estavam errados.
//
// As duas âncoras são a Tabela 3-9 (p167), uma MATRIZ de seis materiais por
// cinco tipos de item, e o verbete de cada material (p166-167), um título
// sozinho na linha seguido das metades rotuladas (`Arma.`, `Armadura e
// Escudo.`, `Esotérico.`). A matriz tem DENOMINADOR: trinta células, duas em
// travessão — um leitor que perca uma coluna devolve cinco materiais com nomes
// e preços certos, e trinta é o único número que não fecha.
//
//	go run ./cmd/audit/materials
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

const (
	coverageFloor = 0.90
	category      = "material"
)

func main() {
	book, err := bookpdf.Open()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	code, err := audit(book)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(code)
}

func audit(book *bookpdf.Book) (int, error) {
	matrix, err := readMatrix(book, tablePage)
	if err != nil {
		return 0, err
	}
	entries := readEntries(book, entryPages)
	parts, blanks := 0, 0
	for _, name := range entries.order {
		parts += len(entries.parts[name].order)
	}
	for _, v := range matrix.prices {
		if v == nil {
			blanks++
		}
	}
	fmt.Printf("AS DUAS ÂNCORAS  Tabela 3-9: %d materiais x %d tipos de item = %d células (%d em travessão)\n",
		len(matrix.names), matrix.rows, len(matrix.prices), blanks)
	fmt.Printf("                 verbetes: %d, %d partes rotuladas\n", len(entries.order), parts)

	// AS DUAS SE CONFEREM ANTES DE O CATÁLOGO SER MENCIONADO: foi assim que o
	// boxe "Fabricando Itens Superiores" foi pego roubando o nome do Mitral — a
	// contagem batia em seis e o sexto nome era o do boxe.
	tableKeys, entryKeys := map[string]bool{}, map[string]bool{}
	for _, n := range matrix.names {
		tableKeys[bookpdf.Key(n)] = true
	}
	for _, n := range entries.order {
		entryKeys[bookpdf.Key(n)] = true
	}
	onlyTable, onlyEntry := minus(tableKeys, entryKeys), minus(entryKeys, tableKeys)
	if len(onlyTable) > 0 || len(onlyEntry) > 0 {
		fmt.Printf("  AS DUAS DISCORDAM — só na tabela: %s; só no verbete: %s\n", pyfmt.List(onlyTable), pyfmt.List(onlyEntry))
		fmt.Println("  É defeito de LEITURA, e ele vem antes de qualquer correção de catálogo.")
		return 1, nil
	}
	if len(matrix.prices) != len(matrix.names)*matrix.rows {
		fmt.Printf("  A MATRIZ NÃO FECHA: %d células para %dx%d\n", len(matrix.prices), len(matrix.names), matrix.rows)
		return 1, nil
	}
	fmt.Printf("                 as duas dizem os MESMOS %d nomes\n", len(matrix.names))

	catalog := map[string]transcript.Item{}
	for _, it := range transcript.Items {
		if it.Category == category {
			catalog[bookpdf.Key(it.Name)] = it
		}
	}
	fmt.Printf("\nCATÁLOGO         %d entradas em `%s`\n", len(catalog), category)

	var failures []string
	modeledCount := 0
	type unmodeledPart struct{ name, reason string }
	var unmodeled []unmodeledPart
	lowest, lowestName := 2.0, ""
	for _, name := range matrix.names {
		entry, ok := catalog[bookpdf.Key(name)]
		if !ok {
			failures = append(failures, name+": está na Tabela 3-9 e não está no catálogo")
			continue
		}
		asWritten := entries.byKey(name)
		fromBook := entries.parts[asWritten]
		var texts []string
		for _, p := range fromBook.order {
			texts = append(texts, fromBook.text[p])
		}
		ratio, shared, total := bookpdf.Coverage(strings.Join(texts, " "), entry.Description)
		if ratio < lowest || (ratio == lowest && name < lowestName) {
			lowest, lowestName = ratio, name
		}
		if ratio < coverageFloor {
			failures = append(failures, fmt.Sprintf("%s: a descrição repete %d das %d palavras de conteúdo do verbete (%s), abaixo do piso de %s",
				name, shared, total, pyfmt.Percent(ratio), pyfmt.Percent(coverageFloor)))
		}
		failures = append(failures, priceFailures(entry, name, matrix)...)
		if entry.BookPage != entries.page[asWritten] {
			failures = append(failures, fmt.Sprintf("%s: o catálogo cita a p%s e o verbete está na p%d",
				name, pyfmt.Optional(float64(entry.BookPage)), entries.page[asWritten]))
		}

		// O DENOMINADOR É POR METADE, e não por material: o mitral aplica a
		// margem de ameaça e não o limite de Destreza, e "tem modifiers?" daria
		// verde sobre os dois. As duas listas são PERMITIDOS, e a união delas
		// tem de dar exatamente as partes do verbete — uma parte pode estar nas
		// duas, como o "Armadura e Escudo" do mitral.
		applied := map[string]bool{}
		for _, p := range entry.Modeled {
			applied[p] = true
		}
		silent := map[string]bool{}
		var byPart transcript.OrderedMap[string]
		if entry.Unmodeled != nil {
			byPart = entry.Unmodeled.ByPart
		}
		for _, e := range byPart {
			silent[e.Key] = true
		}
		declared := union(applied, silent)
		bookParts := map[string]bool{}
		for _, p := range fromBook.order {
			bookParts[p] = true
		}
		if !sameSet(declared, bookParts) {
			failures = append(failures, fmt.Sprintf("%s: as partes declaradas não fecham com o verbete — sem declaração: %s; declaradas e inexistentes: %s",
				name, pyfmt.List(minus(bookParts, declared)), pyfmt.List(minus(declared, bookParts))))
		}
		if len(applied) > 0 && len(entry.Modifiers) == 0 {
			failures = append(failures, fmt.Sprintf("%s: diz aplicar %s e não tem modificador nenhum", name, pyfmt.List(sortedKeys(applied))))
		}
		if len(entry.Modifiers) > 0 && len(applied) == 0 {
			failures = append(failures, name+": tem modificador e não diz QUAL metade ele aplica")
		}
		for _, e := range byPart {
			if e.Value == "" {
				failures = append(failures, fmt.Sprintf("%s/%s: está na lista do que não foi modelado sem motivo", name, e.Key))
			}
		}
		modeledCount += len(applied)
		for _, p := range sortedKeys(silent) {
			reason, _ := byPart.Get(p)
			unmodeled = append(unmodeled, unmodeledPart{name + "/" + p, reason})
		}
	}

	fmt.Printf("  metades que o motor APLICA: %d\n", modeledCount)
	fmt.Printf("  metades que ele NÃO aplica: %d, e cada uma diz por quê:\n", len(unmodeled))
	for _, u := range unmodeled {
		reason := []rune(u.reason)
		fmt.Printf("    %-36s %s\n", u.name, string(reason[:min(92, len(reason))]))
	}
	fmt.Printf("\n  cobertura mínima: %s (%s)\n", pyfmt.Percent(lowest), lowestName)
	fmt.Printf("\nFALHAS: %d   (medidos: %d materiais, %d células de preço, %d partes de verbete)\n",
		len(failures), len(matrix.names), len(matrix.prices), parts)
	for _, f := range failures {
		fmt.Printf("  %s\n", f)
	}
	if len(failures) > 0 {
		return 1, nil
	}
	return 0, nil
}

// priceFailures confere a MATRIZ INTEIRA, célula a célula: o preço de um
// material depende do TIPO DE ITEM, e um catálogo com um número só não tem como
// estar certo para os cinco.
func priceFailures(entry transcript.Item, name string, matrix priceMatrix) []string {
	var failures []string
	families := map[string]bool{}
	for _, k := range itemKinds {
		expected := matrix.prices[cell{name, k.printed}]
		has, present := catalogPrice(entry.PriceByItem, k.priceKey)
		switch {
		case expected == nil && present:
			failures = append(failures, fmt.Sprintf("%s: o catálogo dá preço de %s e a Tabela 3-9 traz travessão — este material não vira isso", name, k.printed))
		case expected != nil && (!present || has != *expected):
			shown := "AUSENTE"
			if present {
				shown = fmt.Sprint(has)
			}
			failures = append(failures, fmt.Sprintf("%s/%s: o catálogo diz %s e a Tabela 3-9 diz %d", name, k.printed, shown, *expected))
		}
		if expected != nil {
			families[k.family] = true
		}
	}
	accepts := map[string]bool{}
	for _, a := range entry.AppliesTo {
		accepts[a] = true
	}
	if !sameSet(accepts, families) {
		failures = append(failures, fmt.Sprintf("%s: o catálogo aceita %s e a Tabela 3-9 dá preço para %s",
			name, pyfmt.List(sortedCopy(entry.AppliesTo)), pyfmt.List(sortedKeys(families))))
	}
	weapon := matrix.prices[cell{name, "Arma"}]
	if (weapon == nil && entry.Price != 0) || (weapon != nil && entry.Price != float64(*weapon)) {
		weaponText := "None"
		if weapon != nil {
			weaponText = fmt.Sprint(*weapon)
		}
		failures = append(failures, fmt.Sprintf("%s: o `price` diz %s e o de ARMA é %s. O campo é um número só e a Tabela 3-9 é uma matriz: ele carrega a coluna de arma, e a verdade inteira mora no `priceByItem`",
			name, pyfmt.Optional(entry.Price), weaponText))
	}
	return failures
}

// catalogPrice é a célula do `priceByItem` e se ela EXISTE no JSON servido:
// as colunas de armadura somem quando zeradas, as outras três não.
func catalogPrice(p *transcript.PriceByItem, key string) (int, bool) {
	if p == nil {
		return 0, false
	}
	switch key {
	case "weapon":
		return p.Weapon, true
	case "armorLight":
		return p.ArmorLight, p.ArmorLight != 0
	case "armorHeavy":
		return p.ArmorHeavy, p.ArmorHeavy != 0
	case "shield":
		return p.Shield, true
	case "esoteric":
		return p.Esoteric, true
	}
	return 0, false
}

// byKey é o nome como o VERBETE o escreve: a tabela e o verbete divergem na
// caixa ("Gelo Eterno" contra "Gelo eterno"), e as duas grafias são do livro.
func (e bookEntries) byKey(name string) string {
	for _, n := range e.order {
		if bookpdf.Key(n) == bookpdf.Key(name) {
			return n
		}
	}
	return ""
}

func minus(a, b map[string]bool) []string {
	var out []string
	for k := range a {
		if !b[k] {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func union(a, b map[string]bool) map[string]bool {
	out := map[string]bool{}
	for k := range a {
		out[k] = true
	}
	for k := range b {
		out[k] = true
	}
	return out
}

func sameSet(a, b map[string]bool) bool {
	return len(minus(a, b)) == 0 && len(minus(b, a)) == 0
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedCopy(xs []string) []string {
	out := slices.Clone(xs)
	sort.Strings(out)
	return out
}
