package engine

import "testing"

// A MELHORIA E O ENCANTO DE ARMADURA VALEM NO ESCUDO, que se empunha e não se veste.
//
//	"Se for um escudo, possui uma chapa mais espessa. O bônus na Defesa e a penalidade de armadura do item
//	 aumentam em 1." (Reforçada, p165)
//	"O bônus na Defesa do item aumenta em +2." (Defensor, p337)
//	"O bônus na Defesa do item aumenta em +4. Pré-requisito: defensor." (Guardião, p339)
func TestTheArmorOverlaysCountOnAWieldedShield(t *testing.T) {
	world := BookRuleset(primeBook(t))
	for _, c := range []struct {
		why          string
		improvements string
		enchants     []string
		defense      int
		penalty      int
	}{
		{"o escudo pesado, o controle", `[]`, nil, 2, -2},
		{"o escudo pesado reforçado", `["melhoria-reforcada"]`, nil, 3, -3},
		{"o escudo pesado defensor", `[]`, []string{"encanto-defensor"}, 4, -2},
		// O Guardião e o Defensor aumentam o MESMO item: vale o maior, como encanto com encanto (p333).
		{"o escudo pesado defensor e guardião", `[]`, []string{"encanto-defensor", "encanto-guardiao"}, 6, -2},
	} {
		t.Run(c.why, func(t *testing.T) {
			ch := shieldHero(c.improvements, c.enchants)
			effects := world.EffectsOf(ch, nil)
			if got := fromItem(effects, ModifierTarget{K: "defense"}, "escudo-pesado"); got != c.defense {
				t.Errorf("%s dá %+d na Defesa; want %+d", c.why, got, c.defense)
			}
			if got := fromItem(effects, ModifierTarget{K: "armorPenalty"}, "escudo-pesado"); got != c.penalty {
				t.Errorf("%s dá %+d na penalidade de armadura; want %+d", c.why, got, c.penalty)
			}
		})
	}
}

// A SOLIDEZ LÊ O ESCUDO INTEIRO, com a melhoria posta nele: o escudo pesado reforçado dá +3 (p55, p165).
func TestTheSolidityReadsTheReinforcedShield(t *testing.T) {
	world := BookRuleset(primeBook(t))
	ch := saving("Cavaleiro", `["class.cavaleiro.solidez"]`)
	ch.Items = shieldHero(`["melhoria-reforcada"]`, nil).Items
	if got := expertiseFrom(t, world, ch, "Fortitude", "Solidez"); got != 3 {
		t.Errorf("a Solidez pôs %+d em Fortitude com o escudo pesado reforçado; want +3", got)
	}
}

// shieldHero é o guerreiro com o escudo pesado empunhado, com as melhorias e os encantos dados.
func shieldHero(improvements string, enchants []string) Character {
	ch := wielding(attributeSumHero("Guerreiro", 6, `[]`), "escudo-pesado")
	ch.Items[len(ch.Items)-1].Improvements = improvements
	ch.Items[len(ch.Items)-1].Enchants = enchants
	return ch
}

// fromItem é o que as parcelas do item `id` põem no alvo, depois do empilhamento.
func fromItem(effects ItemEffects, target ModifierTarget, id string) int {
	total := 0
	for _, c := range StatFor(effects, target).Contributions {
		if c.SourceID == id {
			total += c.Amount
		}
	}
	return total
}
