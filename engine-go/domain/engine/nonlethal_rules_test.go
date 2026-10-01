package engine

import (
	"path/filepath"
	"testing"
)

// O DANO NÃO LETAL (p236), e ele é a única forma de dano que o motor trata de
// um jeito diferente do resto.
//
//	"Dano não letal conta para determinar quando você cai inconsciente, mas não
//	 para determinar quando você começa a sangrar ou morre. Efeitos de cura
//	 recuperam primeiro pontos de vida perdidos por dano não letal."
//
// São TRÊS regras numa frase, e cada uma puxa um lado diferente do mesmo
// número. O motor hoje não tem o conceito: nove verbetes do catálogo o citam —
// o encanto Piedosa, o Ataque Piedoso de Lena e de Thyatis, a Briga do
// lutador — e os nove viram dano comum.

// O PRIMEIRO: ele DERRUBA. Um personagem com 12 PV que leva 12 de dano não
// letal cai inconsciente, igual a quem levou 12 de dano comum.
func TestNonLethalDamageKnocksYouOut(t *testing.T) {
	add, _ := DyingConditionChangeWith(12, 0, 12, 12)
	if !temCondicao(add, ConditionUnconscious) {
		t.Errorf("12 de dano não letal num herói de 12 PV não o derrubou: %v", add)
	}
}

// O SEGUNDO: ele NÃO FAZ SANGRAR. É a metade que separa esta regra de todas as
// outras — a mesma queda, e um dos dois está morrendo e o outro só apagou.
func TestNonLethalDamageDoesNotStartTheBleeding(t *testing.T) {
	naoLetal, _ := DyingConditionChangeWith(12, 0, 12, 12)
	if temCondicao(naoLetal, ConditionBleeding) {
		t.Errorf("quem apagou de dano não letal começou a sangrar: %v", naoLetal)
	}
	// O CONTROLE é o mesmo golpe sendo LETAL: sem ele, "não sangrou" se
	// explicaria igualmente bem por "o motor parou de fazer alguém sangrar".
	letal, _ := DyingConditionChangeWith(12, 0, 12, 0)
	if !temCondicao(letal, ConditionBleeding) {
		t.Fatalf("o controle já estava errado: 12 de dano letal não fez sangrar: %v", letal)
	}
}

// E MISTURADO: o que sangra é a parte LETAL. Um herói de 12 PV que leva 8 não
// letal e 4 letal está em 0 — inconsciente —, e o PV letal dele ainda é 4.
func TestOnlyTheLethalHalfDecidesTheBleeding(t *testing.T) {
	add, _ := DyingConditionChangeWith(12, 0, 12, 8)
	if !temCondicao(add, ConditionUnconscious) {
		t.Errorf("a queda a 0 PV não derrubou: %v", add)
	}
	if temCondicao(add, ConditionBleeding) {
		t.Errorf("sangrou com 4 de dano letal num herói de 12 PV: %v", add)
	}
}

// O TERCEIRO: ele NÃO MATA. O limiar da p236 é sobre o PV LETAL, e um herói de
// 12 PV morre em –10 (ou metade dos totais, o que for mais baixo).
func TestNonLethalDamageNeverKills(t *testing.T) {
	// 30 de dano, TODO não letal, num herói de 12 PV: ele está em –18, bem
	// abaixo do limiar, e não morreu.
	if got := VitalStateWith(-18, 12, 30, true); got == VitalDead {
		t.Errorf("30 de dano não letal matou um herói de 12 PV, e a p236 diz que " +
			"dano não letal não determina quando se morre")
	}
	// O CONTROLE: o mesmo –18 de dano LETAL mata.
	if got := VitalStateWith(-18, 12, 0, true); got != VitalDead {
		t.Fatalf("o controle já estava errado: –18 de dano letal num herói de 12 PV "+
			"deu %q, e o limiar é –10", got)
	}
}

// E A CURA PAGA O NÃO LETAL PRIMEIRO. Quatro de cura sobre 8 não letal + 4
// letal deixa o herói com 4 de dano LETAL e nenhum não letal.
func TestHealingPaysTheNonLethalFirst(t *testing.T) {
	naoLetal, letal := HealingSpends(4, 8, 4)
	if naoLetal != 4 || letal != 4 {
		t.Errorf("4 de cura sobre 8 não letal e 4 letal deixaram %d não letal e %d letal, "+
			"e a p236 manda pagar o não letal primeiro: 4 e 4", naoLetal, letal)
	}
	// E O QUE SOBRA desce no letal: 10 de cura paga os 8 e ainda tira 2 do letal.
	if naoLetal, letal := HealingSpends(10, 8, 4); naoLetal != 0 || letal != 2 {
		t.Errorf("10 de cura deixaram %d não letal e %d letal, e o esperado é 0 e 2",
			naoLetal, letal)
	}
	// E ela não cria PV do nada: 50 de cura zera os dois e não passa disso.
	if naoLetal, letal := HealingSpends(50, 8, 4); naoLetal != 0 || letal != 0 {
		t.Errorf("50 de cura deixaram %d não letal e %d letal, e os dois têm piso em zero",
			naoLetal, letal)
	}
}

func temCondicao(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

// A PIEDOSA FAZ TODO O DANO DA ARMA SER NÃO LETAL (p336).
//
//	"A arma causa +1d8 de dano, mas todo o dano causado é não letal."
//
// São DUAS metades, e o catálogo só tinha a primeira: o +1d8 entrou na ALE-411
// e a outra não tinha como ser escrita — o `Modifier` não sabia dizer "todo o
// dano desta arma é não letal".
func TestThePiedosaEnchantMakesTheWholeDamageNonLethal(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	catalogs := primeFromDump(t, dir)
	ptr := func(s string) *string { return &s }
	var oracle struct {
		Char Character `json:"char"`
	}
	readJSON(t, filepath.Join(dir, "bardo-versatil-nv7.json"), &oracle)

	card := func(enchants ...string) WeaponCard {
		ch := oracle.Char
		ch.Items = []CharacterItem{{
			CatalogID: ptr("espada-longa"), Name: "Espada longa",
			Equipped: ptr("wielded"), Improvements: "[]", Enchants: enchants,
		}}
		return BookRuleset(catalogs).ComputeWeaponCards(ch, map[string]bool{})[0]
	}

	// O CONTROLE: a espada nua causa dano letal, como toda arma.
	if card().NonLethal {
		t.Fatal("o controle já estava errado: uma espada longa sem encanto causa dano letal")
	}
	piedosa := card("encanto-piedosa")
	if !piedosa.NonLethal {
		t.Error("a espada Piedosa causa dano letal, e a p336 diz que TODO o dano dela " +
			"é não letal")
	}
	// E A OUTRA METADE continua lá: o +1d8 não pode ter sumido no caminho.
	if len(piedosa.ExtraDamage) != 1 || piedosa.ExtraDamage[0].Dice != "1d8" {
		t.Errorf("a espada Piedosa veio com as parcelas %+v, e a p336 diz +1d8",
			piedosa.ExtraDamage)
	}
}

// E O ATAQUE DIZ QUANTO FOI NÃO LETAL.
//
// TUDO, quando a arma é Piedosa: o livro diz "todo o dano causado", e isso
// inclui o bônus de Força e as parcelas dos outros encantos — a arma é que é
// piedosa, não um dos dados dela.
func TestTheAttackReportsHowMuchDamageWasNonLethal(t *testing.T) {
	weapon := aWeapon("1d8", 3, 20, 2, 5)
	weapon.NonLethal = true

	out, err := ResolveAttack(weapon, AttackTarget{Defense: 10}, 15, fixedDice(t, 6))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if out.Damage != 9 {
		t.Fatalf("o controle já estava errado: 1d8 deu 6 mais 3 de bônus é 9, e veio %d",
			out.Damage)
	}
	if out.NonLethal != 9 {
		t.Errorf("o ataque com arma Piedosa causou %d de dano e disse que %d foi não "+
			"letal — a p336 diz que TODO o dano dela é", out.Damage, out.NonLethal)
	}

	// E A ARMA COMUM não reporta nada: o campo existe para o caso raro, e um
	// número onde não há regra faria a mesa procurar de onde ele saiu.
	comum, err := ResolveAttack(aWeapon("1d8", 3, 20, 2, 5), AttackTarget{Defense: 10}, 15, fixedDice(t, 6))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if comum.NonLethal != 0 {
		t.Errorf("uma espada comum reportou %d de dano não letal", comum.NonLethal)
	}
}
