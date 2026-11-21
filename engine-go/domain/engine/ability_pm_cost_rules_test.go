package engine

import "testing"

// O CUSTO DE UMA HABILIDADE (ALE-647, decisão do dono, livro conferido). A ordem é a da magia: o custo, mais os
// aumentos (o Alquebrado, p394), mais o MAIOR corte — "reduções no custo de PM não são cumulativas" (p226) —, e o
// piso de 1 PM por último: "uma habilidade nunca pode ter seu custo reduzido para menos de 1 PM" (p226). O que
// custa zero continua de graça.
func TestAnAbilityCostSumsTheIncreaseAndTheBiggestCutBeforeTheFloor(t *testing.T) {
	for _, c := range []struct {
		why                  string
		base, increase, want int
		cuts                 []int
	}{
		{"o exemplo da decisão: 1 PM, o Alquebrado e o –1 da Harmonizada", 1, 1, 1, []int{-1}},
		{"o Alquebrado sozinho", 3, 1, 4, nil},
		{"sem nada", 3, 0, 3, nil},
		{"os cortes não se acumulam: vale o maior", 5, 0, 2, []int{-1, -3}},
		{"o piso de 1 PM", 2, 0, 1, []int{-5}},
		{"o Alquebrado não deixa de graça o que custa zero... nem cobra por ele", 0, 1, 0, nil},
	} {
		if got := AbilityPmCost(c.base, c.increase, c.cuts...); got != c.want {
			t.Errorf("%s: %d PM + %d com os cortes %v saiu por %d; want %d", c.why, c.base, c.increase, c.cuts, got, c.want)
		}
	}
}

// O ALQUEBRADO ALCANÇA O CUSTO DE TODA HABILIDADE, e não só o das magias: "o custo em pontos de mana das
// habilidades do personagem aumenta em +1" (p394).
func TestTheBrokenConditionRaisesEveryAbilityCost(t *testing.T) {
	world := BookRuleset(primeBook(t))
	hero := Character{Level: 5, Classes: []CharacterClass{{ClassName: "Guerreiro", Level: 5}}}
	if got := world.EffectsOf(hero, nil).AbilityCostIncrease(); got != 0 {
		t.Errorf("sem condição, o custo das habilidades sobe %d; want 0", got)
	}
	hero.ActiveConditions = `["alquebrado"]`
	if got := world.EffectsOf(hero, nil).AbilityCostIncrease(); got != 1 {
		t.Errorf("Alquebrado, o custo das habilidades sobe %d; want +1 (p394)", got)
	}
}
