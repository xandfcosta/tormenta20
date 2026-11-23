package engine

import (
	"strings"
	"testing"
)

// CADA ITEM AUMENTA O SEU: a Reforçada da armadura e a do escudo somam, porque cada uma aumenta o bônus do
// próprio item. O mesmo tipo de bônus em dois itens disputaria (p226), e um dos dois +1 sairia da conta.
func TestTheArmorAndTheShieldEachRaiseTheirOwnBonus(t *testing.T) {
	world := BookRuleset(primeBook(t))
	for _, c := range []struct {
		why          string
		improvements string
		enchants     []string
		overlay      string
		want         int
	}{
		{"as duas reforçadas", `["melhoria-reforcada"]`, nil, "Reforçada", 2},
		{"as duas defensoras", `[]`, []string{"encanto-defensor"}, "Defensor", 4},
	} {
		t.Run(c.why, func(t *testing.T) {
			ch := shieldHero(c.improvements, c.enchants)
			vested := "vested"
			armor := "armadura-completa"
			ch.Items = append(ch.Items, CharacterItem{
				CatalogID: &armor, Name: armor, Quantity: 1, Equipped: &vested,
				Improvements: c.improvements, Enchants: c.enchants,
			})
			stat := StatFor(world.EffectsOf(ch, nil), ModifierTarget{K: "defense"})
			got := 0
			for _, contribution := range stat.Contributions {
				if strings.Contains(contribution.Note, c.overlay) {
					got += contribution.Amount
				}
			}
			if got != c.want || len(stat.Dropped) != 0 {
				t.Errorf("com armadura e escudo, %s dão %+d na Defesa (fora da conta: %+v); want %+d",
					c.why, got, stat.Dropped, c.want)
			}
		})
	}
}
