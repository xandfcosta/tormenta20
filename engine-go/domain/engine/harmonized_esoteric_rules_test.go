package engine

import "testing"

// O HARMONIZADO É DE UMA MAGIA.
//
// "Escolha uma magia. Seu custo diminui em –1 PM" (Harmonizado, p165). A magia é a escolha da melhoria, gravada
// no item, e o corte só vale para ela, com o esotérico na mão. Sem magia escolhida, o Harmonizado não corta nada.

const harmonized = "melhoria-harmonizado-esoterico"

func holdingTheHarmonizedWand(ch Character, slot, spell string) Character {
	id := "varinha-arcana"
	item := CharacterItem{CatalogID: &id, Name: "Varinha arcana", Quantity: 1, Equipped: &slot,
		Improvements: `["` + harmonized + `"]`}
	if spell != "" {
		item.ImprovementChoices = map[string]string{harmonized: spell}
	}
	ch.Items = append(ch.Items, item)
	return ch
}

func TestTheHarmonizedEsotericCutsOnlyTheChosenSpell(t *testing.T) {
	world := BookRuleset(primeBook(t))
	mage := attributeSumHero("Arcanista", 5, `[]`)
	fireball := SpellAt{Tradition: "arcana", School: "evocacao", Spell: "bola-de-fogo"}
	light := SpellAt{Tradition: "arcana", School: "evocacao", Spell: "luz"}

	for _, c := range []struct {
		why, slot, spell string
		at               SpellAt
		want             int
	}{
		{"na magia escolhida, com o esotérico na mão", "wielded", "bola-de-fogo", fireball, -1},
		{"em outra magia", "wielded", "bola-de-fogo", light, 0},
		{"sem magia escolhida", "wielded", "", fireball, 0},
		{"com o esotérico guardado no corpo", "vested", "bola-de-fogo", fireball, 0},
	} {
		hero := holdingTheHarmonizedWand(mage, c.slot, c.spell)
		if got := world.ComputeSheet(hero, nil).ForSpell("pmCost", c.at).Total; got != c.want {
			t.Errorf("o Harmonizado %s cortou %d no custo; a p165 corta %d", c.why, got, c.want)
		}
	}
}
