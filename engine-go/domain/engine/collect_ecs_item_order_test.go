package engine

import (
	"strings"
	"testing"
)

// A ORDEM DAS REGRAS DO ITEM, PRESA NUM CASO SÓ (ALE-385).
//
// # Por que este teste existe: o oráculo NÃO prende isto
//
// Medido trocando pares de lugar nas fichas do oráculo: a penalidade e o
// espelho invertidos não reprovam ficha nenhuma, porque nenhuma tem um item que
// dispare as duas regras ao mesmo tempo. Sabotagem que não reprova não é guarda
// cego — é o oráculo não visitando o caso, que é a família que o
// `docs/testing.md` cataloga em "um guarda só mede o que ele VISITA".
//
// # O caso que dispara as duas de uma vez
//
// O `machado-taurico` é `weapon-exotic` (então quem não tem `armas-exoticas`
// leva a penalidade da p142) e é arma (então o ataque espelha na perícia).
//
// Havia mais duas regras na lista, a Equilibrada e o medalhão vestido. As duas
// eram homebrew e viraram regra da mesa: quem as escreve é o mestre, no editor.
func TestTheItemRulesLandInOrder(t *testing.T) {
	catalogs := primeBook(t)

	wielded := "wielded"
	axe := "machado-taurico"
	ch := Character{Items: []CharacterItem{{
		Name:         "Machado táurico",
		CatalogID:    &axe,
		Equipped:     &wielded,
		Improvements: `["melhoria-equilibrada"]`,
	}}}

	items := withoutTheBody(BookRuleset(catalogs).ActiveItemsFor(ch))
	if len(items) != 1 {
		t.Fatalf("colhi %d fontes, esperava 1", len(items))
	}
	mods := items[0].Modifiers

	// Cada estágio é achado pela ASSINATURA do que ele produz, e não por
	// índice: índice cravado quebraria ao acrescentar um modificador ao verbete
	// do machado, que é mudança de catálogo e não de ordem.
	at := func(nome string, casa func(Modifier) bool) int {
		for i, m := range mods {
			if casa(m) {
				return i
			}
		}
		t.Fatalf("o estágio %q não aparece entre os %d modificadores — ele não rodou.\n%+v", nome, len(mods), mods)
		return -1
	}

	own := at("os mods PRÓPRIOS", func(m Modifier) bool {
		return m.Target.K == "attack" && strings.Contains(m.Note, "desbalanceada")
	})
	penalty := at("a penalidade da p142", func(m Modifier) bool {
		return strings.Contains(m.Note, "sem proficiência")
	})
	mirror := at("o espelho de ataque", func(m Modifier) bool {
		return m.Target.Scope == WeaponDamageScope("machado-taurico") && strings.Contains(m.Note, "desbalanceada: -2")
	})

	for _, passo := range []struct {
		antes, depois int
		frase         string
	}{
		{own, penalty, "os mods próprios vêm antes da penalidade"},
		{penalty, mirror, "a penalidade vem antes do espelho"},
	} {
		if passo.antes >= passo.depois {
			t.Fatalf("a ordem quebrou: %s (%d contra %d).\n"+
				"A ordem dos modificadores É a ordem da lista em `itemSystems` — "+
				"trocar duas linhas lá quebra este caso.", passo.frase, passo.antes, passo.depois)
		}
	}
}

// Aqui morava o `TestBothPathsAgreeOnTheItemThatFiresThreeRules`, que conferia
// este machado contra o coletor legado — o oráculo não tem uma ficha com ele.
//
// Ele saiu com o coletor legado (ALE-378): com uma coleta só, ele comparava uma
// função consigo mesma. O que ele protegia continua preso pelo caso ACIMA, que
// afirma os estágios do equipamento pela assinatura de cada um.
