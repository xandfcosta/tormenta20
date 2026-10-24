// Command potions confere as POÇÕES do catálogo contra a Tabela 8-12 (p341,
// ALE-451).
//
// Uma poção é uma MAGIA ENGARRAFADA: a tabela não tem verbete, ela é
// `d% | <nome da MAGIA> (tipo; aprimoramento) | preço`, e a regra da poção É a
// da magia (p341). Por isso este auditor não procura prosa: ele resolve cada
// linha contra o catálogo de magias.
//
// O parêntese carrega três coisas: o TIPO de entrega (*óleo*, *granada*; sem
// parêntese é poção de beber), o APRIMORAMENTO que a linha compra, e o EFEITO
// JÁ CALCULADO quando a magia é de cura — quatro linhas da tabela são a Curar
// Ferimentos em degraus diferentes. O denominador é o das outras tabelas de
// saque: o `d%` tila 1..100.
//
//	go run ./cmd/audit/potions
package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"t20engine/cmd/internal/pyfmt"

	"t20engine/domain/catalog/transcript"
	"t20engine/infra/bookpdf"
)

const (
	tablePage = 347
	bookPage  = 341
	slack     = 3.0
	nameX     = 349.8
)

// A célula do d% é alinhada à DIREITA: "97-100" começa em 314,9 e "80" em
// 322,9. O x é FAIXA, e a borda de baixo custou a última linha da tabela.
var (
	diceX  = [2]float64{313.0, 325.0}
	priceX = [2]float64{490.0, 502.0}
)

// "Curar Ferimentos (4d8+4 PV)" → nome e parêntese, que pode ter duas coisas
// separadas por ponto e vírgula.
var (
	cellRe = regexp.MustCompile(`^(.+?)(?:\s*\((.*)\))?$`)
	cureRe = regexp.MustCompile(`^(\d+d\d+)\+(\d+)\s*PV$`)
)

// cellParts é o que a célula da tabela escreve.
type cellParts struct {
	spell   string
	kind    string
	augment string
	cure    *transcript.RecoveryDice
}

func partsOf(cell string) cellParts {
	m := cellRe.FindStringSubmatch(bookpdf.Strip(cell))
	parts := cellParts{spell: bookpdf.Strip(m[1]), kind: "pocao"}
	for _, piece := range strings.Split(bookpdf.Strip(m[2]), ";") {
		piece = bookpdf.Strip(piece)
		switch {
		case piece == "":
		case piece == "óleo":
			parts.kind = "oleo"
		case piece == "granada":
			parts.kind = "granada"
		case cureRe.MatchString(piece):
			c := cureRe.FindStringSubmatch(piece)
			bonus, _ := strconv.Atoi(c[2])
			parts.cure = &transcript.RecoveryDice{Dice: c[1], Bonus: bonus}
			parts.augment = piece
		default:
			parts.augment = piece
		}
	}
	return parts
}

func main() {
	book, err := bookpdf.Open()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(min(1, audit(book)))
}

func audit(book *bookpdf.Book) int {
	fmt.Printf("══ TABELA 8-12 (poções, p%d) %s\n", bookPage, strings.Repeat("═", 30))
	table := book.LootRows(tablePage, diceX, nameX, priceX, slack)
	if hole := bookpdf.RangesThatDontClose(table); hole != "" {
		fmt.Printf("A TABELA NÃO FECHA: %s\n", hole)
		fmt.Println("Nada do que vem abaixo vale — o auditor leu a página errado.")
		return 1
	}

	spells := map[string]*transcript.Spell{}
	for i := range transcript.Spells {
		spells[bookpdf.Key(transcript.Spells[i].Name)] = &transcript.Spells[i]
	}
	potions := potionsOfCatalog()
	var failures []string
	fmt.Printf("%7s %-28s %-8s %7s  catálogo\n", "d%", "magia", "tipo", "preço")
	matched := 0
	for _, row := range table {
		parts := partsOf(row.Name)
		price := priceText(row.Price)
		span := strconv.Itoa(row.Start)
		if row.End != row.Start {
			span = fmt.Sprintf("%d-%d", row.Start, row.End)
		}
		spell := spells[bookpdf.Key(parts.spell)]
		if spell == nil {
			failures = append(failures, fmt.Sprintf("%s: o catálogo de magias não tem %s", row.Name, pyfmt.Repr(parts.spell)))
			fmt.Printf("%7s %-28s %-8s %7s  — MAGIA AUSENTE\n", span, parts.spell, parts.kind, price)
			continue
		}
		entry := potionOfRow(potions, spell.ID, row.Price)
		label := "— AUSENTE"
		if entry != nil {
			label = entry.ID
			matched++
		}
		fmt.Printf("%7s %-28s %-8s %7s  %s\n", span, spell.ID, parts.kind, price, label)
		if entry != nil {
			failures = append(failures, check(entry, parts)...)
		}
	}
	fmt.Printf("\n%d/%d transcritas\n", matched, len(table))
	for _, f := range failures {
		fmt.Printf("  ✗ %s\n", f)
	}
	return len(failures)
}

// potionsOfCatalog são os itens que engarrafam uma magia, um por id e na ordem
// em que o id aparece primeiro — a do dicionário do auditor original.
func potionsOfCatalog() []*transcript.Item {
	var out []*transcript.Item
	at := map[string]int{}
	for i := range transcript.Items {
		it := &transcript.Items[i]
		if it.Consumable == nil || it.Consumable.Spell == "" {
			continue
		}
		if j, seen := at[it.ID]; seen {
			out[j] = it
			continue
		}
		at[it.ID] = len(out)
		out = append(out, it)
	}
	return out
}

// potionOfRow é a entrada do catálogo que é ESTA linha da tabela. A magia não
// basta para achar: a Curar Ferimentos tem QUATRO linhas e a Arma Mágica tem
// duas. O que as separa é o preço, que é o degrau de aprimoramento.
func potionOfRow(potions []*transcript.Item, spellID string, price *int) *transcript.Item {
	for _, it := range potions {
		// Preço ausente e preço zero são o mesmo no dado tipado.
		want := 0.0
		if price != nil {
			want = float64(*price)
		}
		if it.Consumable.Spell == spellID && it.Price == want {
			return it
		}
	}
	return nil
}

func check(entry *transcript.Item, parts cellParts) []string {
	var failures []string
	c := entry.Consumable
	if c.Kind != parts.kind {
		failures = append(failures, fmt.Sprintf("%s: o catálogo diz tipo %s e a tabela diz %s",
			entry.ID, optionalRepr(c.Kind), pyfmt.Repr(parts.kind)))
	}
	if entry.BookPage != bookPage {
		failures = append(failures, fmt.Sprintf("%s: cita p%s e a tabela está na p%d",
			entry.ID, optionalInt(entry.BookPage), bookPage))
	}
	if parts.augment != "" && c.Augment == "" {
		failures = append(failures, fmt.Sprintf(`%s: a tabela escreve "%s" e o catálogo não diz aprimoramento nenhum`,
			entry.ID, parts.augment))
	}
	if parts.augment == "" && c.Augment != "" {
		failures = append(failures, fmt.Sprintf("%s: o catálogo diz o aprimoramento %s e a tabela não escreve nenhum",
			entry.ID, pyfmt.Repr(c.Augment)))
	}
	// A CURA JÁ CALCULADA tem de bater dado a dado: é o único número que a
	// tabela imprime, e é o que a ficha vai somar no PV de alguém.
	if parts.cure != nil {
		var hp *transcript.RecoveryDice
		if c.Instant != nil {
			hp = c.Instant.HP
		}
		if hp == nil || hp.Dice != parts.cure.Dice || hp.Bonus != parts.cure.Bonus {
			failures = append(failures, fmt.Sprintf("%s: a tabela diz %s+%d PV e o catálogo diz %s",
				entry.ID, parts.cure.Dice, parts.cure.Bonus, recoveryRepr(hp)))
		}
	}
	return failures
}

// recoveryRepr é o `repr()` do dicionário `hp` como o Python o via: `{}` quando
// não há, e sem `bonus` quando ele é zero (o JSON o omite).
func recoveryRepr(hp *transcript.RecoveryDice) string {
	if hp == nil {
		return "{}"
	}
	if hp.Bonus == 0 {
		return fmt.Sprintf("{'dice': %s}", pyfmt.Repr(hp.Dice))
	}
	return fmt.Sprintf("{'dice': %s, 'bonus': %d}", pyfmt.Repr(hp.Dice), hp.Bonus)
}

func priceText(price *int) string {
	if price == nil {
		return "None"
	}
	return strconv.Itoa(*price)
}
