package engine

import (
	"path/filepath"
	"strings"
	"testing"
)

// A ORDEM DAS QUATRO REGRAS DO ITEM, PRESA NUM CASO SÓ (ALE-385).
//
// # Por que este teste existe: o oráculo NÃO prende isto
//
// Medido desligando um sistema de cada vez, e depois TROCANDO pares de lugar
// nas 18 fichas. Cada sistema é exercitado por alguma ficha — nenhum é código
// morto —, mas dos quatro pares ADJACENTES só UM muda o resultado quando
// invertido:
//
//	penalizeUnproficient ↔ mirrorWeaponAttack          → 0 fichas reprovam
//	mirrorWeaponAttack   ↔ grantEquilibradaHomebrew    → 1 ficha reprova
//	grantEquilibradaHomebrew ↔ grantVestedEsotericHomebrew → 0
//	penalizeUnproficient ↔ grantVestedEsotericHomebrew → 0
//
// A razão é que nenhuma das 18 fichas tem um item que dispare DUAS das regras
// ao mesmo tempo, fora de um par. Sabotagem que não reprova não é guarda cego —
// é o oráculo não visitando o caso, que é a família que o `CLAUDE.md` cataloga
// em "um guarda só mede o que ele VISITA".
//
// # O caso que dispara TRÊS regras de uma vez
//
// O `machado-taurico` é `weapon-exotic` (então quem não tem `armas-exoticas`
// leva a penalidade da p142), é arma (então o ataque espelha na perícia) e tem
// o traço `desbalanceada` (então a melhoria equilibrada o anula). Três das
// quatro num item só.
//
// **A quarta não cabe aqui, e isso é do catálogo e não do teste:** a única
// entrada que a regra do esotérico vestido alcança é o `medalhao-de-prata`, que
// não é arma e não exige proficiência. Com os itens de hoje, a posição daquele
// sistema é INOBSERVÁVEL — e dizer isso é mais honesto que fingir um caso.
func TestTheFourItemRulesLandInOrder(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	catalogs := primeFromDump(t, dir)

	wielded := "wielded"
	axe := "machado-taurico"
	ch := Character{Items: []CharacterItem{{
		Name:         "Machado táurico",
		CatalogID:    &axe,
		Equipped:     &wielded,
		Improvements: `["melhoria-equilibrada"]`,
	}}}

	items := BookRuleset(catalogs).ActiveItemsFor(ch)
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
		return m.Target.K == "expertise" && strings.Contains(m.Note, "desbalanceada: -2")
	})
	fixed := at("a melhoria equilibrada", func(m Modifier) bool {
		return m.Note == "anula desbalanceada"
	})

	for _, passo := range []struct {
		antes, depois int
		frase         string
	}{
		{own, penalty, "os mods próprios vêm antes da penalidade"},
		{penalty, mirror, "a penalidade vem antes do espelho"},
		{mirror, fixed, "o espelho vem antes da melhoria equilibrada"},
	} {
		if passo.antes >= passo.depois {
			t.Fatalf("a ordem quebrou: %s (%d contra %d).\n"+
				"A ordem dos modificadores É a ordem da lista em `itemSystems` — "+
				"trocar duas linhas lá quebra este caso.", passo.frase, passo.antes, passo.depois)
		}
	}
}

// Aqui morava o `TestBothPathsAgreeOnTheItemThatFiresThreeRules`, que conferia
// este machado contra o coletor legado — o oráculo não tem uma ficha com ele, e
// a paridade cruzada das 18 fichas não passava por aqui.
//
// Ele saiu com o coletor legado (ALE-378): com uma coleta só, ele comparava uma
// função consigo mesma. O que ele protegia continua preso pelo caso ACIMA, que
// afirma os quatro estágios do equipamento pela assinatura de cada um.
