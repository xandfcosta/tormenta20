package engine

import (
	"slices"
	"testing"
)

// O DESTRUIDOR (ALE-652, fatia D). "Quando causa dano com uma arma corpo a corpo de duas mãos, você pode rolar
// novamente qualquer resultado 1 ou 2 das rolagens de dano da arma." (bárbaro p41, guerreiro p65)
//
// Da arma, e não os dados a mais (o Golpe Divino, o encanto); de duas mãos e corpo a corpo, e não o arco longo.
// E vale o segundo resultado, mesmo que pior (p226): um dado rolado de novo não se rola outra vez.

func TestTheDestroyerRerollsOnlyOnTheTwoHandedMeleeCard(t *testing.T) {
	world := BookRuleset(primeBook(t))
	for _, c := range []struct {
		why, class, powers, weapon string
		want                       bool
	}{
		{"o montante do guerreiro com o Destruidor", "Guerreiro", `["class.guerreiro.destruidor"]`, "montante", true},
		{"o montante do bárbaro com o Destruidor", "Bárbaro", `["class.barbaro.destruidor"]`, "montante", true},
		{"o machado de batalha é de uma mão", "Guerreiro", `["class.guerreiro.destruidor"]`, "machado-batalha", false},
		{"o arco longo é de duas mãos, mas à distância", "Guerreiro", `["class.guerreiro.destruidor"]`, "arco-longo", false},
		{"o montante sem o Destruidor", "Guerreiro", `[]`, "montante", false},
	} {
		hero := wielding(attributeSumHero(c.class, 6, c.powers), c.weapon)
		hero.Strength = 1
		var card *WeaponCard
		for _, each := range world.ComputeWeaponCards(hero, nil) {
			if each.CatalogID == c.weapon {
				card = &each
			}
		}
		if card == nil {
			t.Fatalf("%s: nenhuma carta para %s", c.why, c.weapon)
		}
		if card.RerollsLowDice != c.want {
			t.Errorf("%s: a carta rola de novo os 1 e 2 = %v, want %v", c.why, card.RerollsLowDice, c.want)
		}
	}
}

func TestTheDestroyerRerollsEveryOneAndTwoOnceAndKeepsTheSecond(t *testing.T) {
	sequence := func(values ...int) func(int) (int, error) {
		return func(int) (int, error) {
			next := values[0]
			values = values[1:]
			return next, nil
		}
	}
	// O 1 e o 2 se rolam de novo; o 5 e o 8 ficam. O 2 que volta 1 FICA 1: vale o segundo, e não há terceiro.
	after, err := RerollLowWeaponDice([]int{1, 5, 2, 8}, 8, sequence(7, 1))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(after, []int{7, 5, 1, 8}) {
		t.Errorf("os dados da arma depois do Destruidor = %v, want [7 5 1 8]", after)
	}
	if got := LowWeaponDice([]int{1, 5, 2, 8}); got != 2 {
		t.Errorf("LowWeaponDice = %d, want 2 (o 1 e o 2)", got)
	}
	if got := LowWeaponDice([]int{3, 6}); got != 0 {
		t.Errorf("sem 1 nem 2, LowWeaponDice = %d, want 0", got)
	}
}
