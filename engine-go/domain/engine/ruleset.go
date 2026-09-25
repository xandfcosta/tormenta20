package engine

import (
	"encoding/json"
	"fmt"
)

// O MUNDO DE UMA CAMPANHA (ALE-387).
//
// Cada campanha carrega a própria versão de Arton: o livro, mais o que o mestre
// mudou. É o que justifica a ficha jogada ser um CLONE — dois mundos não se
// afetam —, e é por isso que o `Ruleset` existe como tipo em vez de o remendo
// viajar no personagem.
//
// # Por que um TIPO, e não um campo no `Character`
//
// Com as emendas viajando no personagem, campanha-inteira e personagem-
// específico chegam ao motor já misturados pelo carregamento: o ESCOPO deixa de
// ser representável e a procedência morre no caminho — e é ela que a ficha usa
// para dizer de onde veio cada termo.
//
// Separado, o compilador vira o guarda: computar uma ficha pede um `*Ruleset`,
// e quem só tem o livro na mão NÃO COMPILA. Era um `TestEvery…` varrendo as
// portas; hoje é o tipo, que é a versão que não se esquece de rodar.
//
// # `Catalogs` é O LIVRO, e ele continua imutável
//
// Ele é primado uma vez por processo, de `go:embed`, e é compartilhado por
// todas as mesas. O `Ruleset` é uma VISTA dele: struct rasa, mapas do livro
// compartilhados, uma alocação por campanha. Nada aqui escreve no livro.

// Ruleset é o livro sob as emendas de UMA mesa.
//
// @example engine.BookRuleset(catalogs).ComputeSheet(ch, nil) // fora de campanha
type Ruleset struct {
	book *Catalogs
	mesa Amendments
}

// Amendments é o que uma mesa MUDA no livro.
//
// Um mapa por espécie de mudança, e não uma lista de "regras" com um `kind`: as
// espécies respondem a perguntas diferentes e cada uma tem a forma dela. O que
// as une é a campanha, não o formato.
type Amendments struct {
	// Entries é a emenda de VERBETE: o id do livro para o que a mesa ACRESCENTA
	// àquele verbete. Ela muda o que uma COISA é, e por isso vale para a
	// campanha inteira — um item que significa uma coisa para você e outra para
	// o vizinho de mesa não é um mundo, são dois.
	Entries map[string][]Modifier
}

// AmendmentsFrom monta as emendas a partir do que a mesa GRAVOU: o id do
// verbete do livro e o JSON dos modificadores que ela acrescenta.
//
// @example engine.AmendmentsFrom(map[string]string{"medalhao-de-prata": `[{"target":{"k":"expertise","name":"Luta"},"amount":1}]`})
func AmendmentsFrom(addsByItem map[string]string) (Amendments, error) {
	if len(addsByItem) == 0 {
		return Amendments{}, nil
	}
	out := Amendments{Entries: make(map[string][]Modifier, len(addsByItem))}
	for id, raw := range addsByItem {
		var mods []Modifier
		if err := json.Unmarshal([]byte(raw), &mods); err != nil {
			return Amendments{}, fmt.Errorf(
				"a emenda da campanha para %q não é uma lista de modificadores (%s): %w", id, raw, err)
		}
		out.Entries[id] = mods
	}
	return out, nil
}

// BookRuleset é o mundo SEM mesa nenhuma: só o livro.
//
// Ele é explícito de propósito. Quem computa uma ficha fora de campanha — o
// molde do elenco, o oráculo, um fixture — está tomando uma DECISÃO, e ela
// aparece na linha em vez de ser o que acontece quando ninguém disse nada.
//
// @example engine.BookRuleset(catalogs) // o molde do elenco vê o livro puro
func BookRuleset(book *Catalogs) *Ruleset { return &Ruleset{book: book} }

// RulesetOf é o mundo de uma mesa.
//
// @example engine.RulesetOf(catalogs, engine.Amendments{Entries: …})
func RulesetOf(book *Catalogs, mesa Amendments) *Ruleset {
	return &Ruleset{book: book, mesa: mesa}
}

// Book devolve o livro cru, para quem pergunta o que o LIVRO diz — os
// validadores de mutação, a tela que lista o catálogo. Quem pergunta o que vale
// NESTA MESA usa o `Ruleset`.
func (r *Ruleset) Book() *Catalogs { return r.book }

// itemOf é a ÚNICA leitura de verbete de item do mundo.
//
// Sem emenda ela devolve o ponteiro do livro, intocado. Com emenda ela devolve
// uma CÓPIA com os modificadores da mesa no fim — no fim porque a ordem dos
// modificadores é o que o oráculo compara, e o livro vem primeiro.
//
// Ela nunca escreve no mapa do livro: duas mesas leem o mesmo `Catalogs`
// primado, e sujá-lo vazaria a regra de uma para a outra e para o molde.
func (r *Ruleset) itemOf(id string) *CatalogItem {
	book := r.book.itemsByID[id]
	adds := r.mesa.Entries[id]
	if book == nil || len(adds) == 0 {
		return book
	}
	patched := *book
	patched.Modifiers = append(append([]Modifier{}, book.Modifiers...), adds...)
	return &patched
}

// ─── O que o mundo pergunta ao LIVRO sem mudar a resposta ────────────────────
//
// Um repasse de uma linha por consulta, e não o `*Catalogs` EMBUTIDO, que seria
// a forma curta. Embutir promove os métodos do livro para o mundo — inclusive o
// `getCatalogItem` —, e aí a coleta leria o verbete SEM a emenda sempre que o
// caminho passasse por um método promovido. Não é erro de compilação, não é
// erro em tempo de execução: é a mesa perdendo a regra dela em silêncio, que é
// a família de defeito que este tipo existe para fechar.
//
// A lista é o preço, e ela é barata de manter: consulta nova do livro que a
// coleta precise ganha uma linha aqui, e o compilador cobra.

func (r *Ruleset) getCatalogItem(id string) *CatalogItem { return r.itemOf(id) }

func (r *Ruleset) getRace(id string) *RaceDefinition { return r.book.getRace(id) }

func (r *Ruleset) getGeneralPower(id string) *GeneralPower { return r.book.getGeneralPower(id) }

func (r *Ruleset) grantedPowerByName(name string) *GrantedPower {
	return r.book.grantedPowerByName(name)
}

func (r *Ruleset) getOrigin(id string) *OriginDefinition { return r.book.getOrigin(id) }

func (r *Ruleset) getOriginBenefit(benefitID string) *OriginBenefit {
	return r.book.getOriginBenefit(benefitID)
}

func (r *Ruleset) raceEntryByName(name string) *RaceAttributeEntry {
	return r.book.raceEntryByName(name)
}

func (r *Ruleset) raceWithDeformidade(names ...string) string {
	return r.book.raceWithDeformidade(names...)
}

func (r *Ruleset) isTormentaPower(id string) bool { return r.book.isTormentaPower(id) }

func (r *Ruleset) ownedClassPowers(
	className string, classLevel int, chosen map[string]bool, choice ClassChoiceSelections,
) []*ClassPower {
	return r.book.ownedClassPowers(className, classLevel, chosen, choice)
}
