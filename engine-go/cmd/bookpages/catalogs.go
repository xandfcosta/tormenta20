package main

import (
	"fmt"
	"sort"

	"t20engine/domain/catalog/transcript"
)

// entryRef é uma entrada de catálogo vista pela escada: o nome, o id e o
// endereço do `BookPage` dela, que o `--gravar` acerta no lugar.
type entryRef struct {
	name, id string
	page     *int
}

func (e entryRef) label() string { return lookup{name: e.name, id: e.id}.label() }

// catalog é um catálogo e o que ele diz ao índice.
//
// `qualifiers` são os do próprio livro, e existem onde o nome colide: "grifo
// (criatura)" contra "grifo (parceiro)". Vazio na lista aceita a entrada sem
// parênteses, que é o caso da maioria.
//
// `signature` é como o LIVRO imprime o começo daquele verbete, e é a prova mais
// forte que existe aqui — mais forte que o índice e que o `bookPage` que já
// estava. Ela nasceu de uma imprecisão medida: a p289 contém
// "lobos-das-cavernas" no texto corrido, e a conferência por substring aprovava
// a página errada enquanto o bloco do Lobo abre na 290. `{nome}` é o nome
// dobrado. Onde não há assinatura confiável (item, poder, magia), a escada
// continua sendo índice → título na faixa: assinatura inventada passaria a
// decidir com a autoridade de uma prova.
type catalog struct {
	label      string
	qualifiers []string
	signature  string
	entries    func() []entryRef
}

func refs[T any](list []T, ref func(*T) entryRef) func() []entryRef {
	return func() []entryRef {
		out := make([]entryRef, len(list))
		for i := range list {
			out[i] = ref(&list[i])
		}
		return out
	}
}

// catalogs na ordem em que o relatório os imprime.
var catalogs = []catalog{
	{"condições", []string{"condição", ""}, pySpaceSignature(`{nome}\.\s+(o personagem|voce|a criatura)`),
		refs(transcript.Conditions, func(v *transcript.Condition) entryRef { return entryRef{v.Name, v.ID, &v.BookPage} })},
	{"itens", []string{""}, "",
		refs(transcript.Items, func(v *transcript.Item) entryRef { return entryRef{v.Name, v.ID, &v.BookPage} })},
	{"magias", []string{"magia", ""}, "",
		refs(transcript.Spells, func(v *transcript.Spell) entryRef { return entryRef{v.Name, v.ID, &v.BookPage} })},
	{"criaturas", []string{"criatura", ""}, pySpaceSignature(`{nome}\s+nd\s+[0-9]`),
		refs(transcript.Bestiary, func(v *transcript.Creature) entryRef { return entryRef{v.Name, v.ID, &v.BookPage} })},
	{"raças", []string{"raça", ""}, "",
		refs(transcript.Races, func(v *transcript.Race) entryRef { return entryRef{v.Name, v.ID, &v.BookPage} })},
	{"deuses", []string{""}, "",
		refs(transcript.Gods, func(v *transcript.God) entryRef { return entryRef{v.Name, v.ID, &v.BookPage} })},
	{"poderes gerais", []string{""}, "",
		refs(transcript.GeneralPowers, func(v *transcript.GeneralPower) entryRef { return entryRef{v.Name, v.ID, &v.BookPage} })},
	{"poderes de classe", []string{""}, "",
		refs(transcript.ClassPowers, func(v *transcript.ClassPower) entryRef { return entryRef{v.Name, v.ID, &v.BookPage} })},
	{"poderes concedidos", []string{""}, "",
		refs(transcript.GrantedPowers, func(v *transcript.GrantedPower) entryRef { return entryRef{v.Name, v.ID, &v.BookPage} })},
	{"poderes da Tormenta", []string{""}, "",
		refs(transcript.TormentaPowers, func(v *transcript.TormentaPower) entryRef { return entryRef{v.Name, v.ID, &v.BookPage} })},
	{"poderes divinos", []string{""}, "",
		refs(transcript.DivinePowers, func(v *transcript.DivinePower) entryRef { return entryRef{v.Name, "", &v.BookPage} })},
	{"origens", []string{"origem", ""}, "",
		refs(transcript.OriginSources, func(v *transcript.OriginSource) entryRef { return entryRef{v.Name, v.ID, &v.BookPage} })},
}

// pySpaceSignature troca o `\s` do Python pela classe Unicode dele.
func pySpaceSignature(signature string) string {
	out := ""
	for i := 0; i < len(signature); i++ {
		if signature[i] == '\\' && i+1 < len(signature) && signature[i+1] == 's' {
			out += pySpace
			i++
			continue
		}
		out += string(signature[i])
	}
	return out
}

// pyPage é a página como o Python a imprimia: `None` para nenhuma.
func pyPage(page int) string {
	if page == 0 {
		return "None"
	}
	return fmt.Sprint(page)
}

// process passa as duas voltas da escada por um catálogo, imprime o relatório
// dele e acerta o `BookPage` no lugar. Devolve quantas mudaram e quantas ficaram
// sem página.
func (b *book) process(c catalog) (int, int) {
	entries := c.entries()

	// Primeira volta: só as fontes fortes, para o envelope nascer de página
	// conferida e não de chute.
	type answer struct {
		page   int
		source string
	}
	first := make([]answer, len(entries))
	var confirmed []int
	for i, e := range entries {
		page, source := b.resolve(lookup{e.name, e.id, *e.page}, c.qualifiers, nil, c.signature)
		first[i] = answer{page, source}
		if page != 0 {
			confirmed = append(confirmed, page)
		}
	}
	span := spanOfConfirmed(confirmed)

	counts := map[string]int{}
	var order []string
	var changed, missing []string
	nChanged, nMissing := 0, 0
	for i, e := range entries {
		page, source := first[i].page, first[i].source
		if span != nil {
			// A segunda volta roda para TODOS e não só para quem ficou sem
			// página: é nela que a faixa existe, e é a faixa que habilita a
			// assinatura — sem ela o Lobo continuaria em 289, resolvido pelo
			// "já estava" da primeira volta.
			page, source = b.resolve(lookup{e.name, e.id, *e.page}, c.qualifiers, span, c.signature)
		}
		key := source
		if page == 0 {
			key = "SEM PÁGINA"
		}
		if _, seen := counts[key]; !seen {
			order = append(order, key)
		}
		counts[key]++
		if page == 0 {
			nMissing++
			if len(missing) < 8 {
				missing = append(missing, fmt.Sprintf("   ✗ %s (tinha %s): %s", e.label(), pyPage(*e.page), source))
			}
			*e.page = 0
			continue
		}
		if *e.page != page {
			nChanged++
			if len(changed) < 12 {
				changed = append(changed, fmt.Sprintf("   ~ %s: %s → %d (%s)", e.label(), pyPage(*e.page), page, source))
			}
		}
		*e.page = page
	}

	fmt.Printf("\n── %s (%d) · faixa %s\n", c.label, len(entries), span)
	// O `Counter.most_common()`: por contagem, e o empate na ordem de chegada.
	sort.SliceStable(order, func(i, j int) bool { return counts[order[i]] > counts[order[j]] })
	for _, source := range order {
		fmt.Printf("   %4d  %s\n", counts[source], source)
	}
	for _, line := range changed {
		fmt.Println(line)
	}
	if nChanged > 12 {
		fmt.Printf("   ~ … e mais %d\n", nChanged-12)
	}
	for _, line := range missing {
		fmt.Println(line)
	}
	if nMissing > 8 {
		fmt.Printf("   ✗ … e mais %d sem página\n", nMissing-8)
	}
	return nChanged, nMissing
}
