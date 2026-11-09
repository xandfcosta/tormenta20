package engine

import "testing"

// A FLAUTA MÍSTICA: "Um bardo que empunhe este item aumenta a CD para resistir às magias lançadas por ele em
// +1. Conta como um instrumento musical" (p158). Os dois portões são da frase: ser bardo, e empunhar.
func TestTheMysticFluteRaisesTheBardsSpellDC(t *testing.T) {
	world := BookRuleset(primeBook(t))
	dcBonus := func(class, wear string) int {
		flute := "flauta-mistica"
		hero := Character{
			Level: 3, Charisma: 3, Intelligence: 3,
			Classes: []CharacterClass{{ClassName: class, Level: 3}},
			Items:   []CharacterItem{{CatalogID: &flute, Name: flute, Improvements: "[]"}},
		}
		if wear != "" {
			hero.Items[0].Equipped = ptrTo(wear)
		}
		if class == "Arcanista" {
			hero.ClassChoices = `{"Arcanista":{"caminho":"mago"}}`
		}
		return world.ComputeSheet(hero, nil).SpellDCBonus.Total
	}
	for _, c := range []struct {
		name, class, wear string
		want              int
	}{
		{"o bardo que a empunha", "Bardo", "wielded", 1},
		{"o bardo com a flauta guardada", "Bardo", "", 0},
		{"o arcanista que a empunha", "Arcanista", "wielded", 0},
	} {
		if got := dcBonus(c.class, c.wear); got != c.want {
			t.Errorf("%s: a CD subiu %+d, e o livro pede %+d", c.name, got, c.want)
		}
	}
}
