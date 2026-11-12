package engine

import "testing"

// O SÍMBOLO SAGRADO É DE UM DEUS.
//
// "Se você estiver vestindo [...] ou empunhando o símbolo sagrado de um deus do qual é devoto, recebe +1 em
// testes de resistência" (p157). O deus do símbolo é a escolha do item, e o +1 só vale para quem é devoto
// dele. O Símbolo abençoado "conta como um símbolo sagrado" e, "se você for devoto do deus, o custo de
// suas magias divinas diminui em –1 PM (cumulativo com o poder Símbolo Sagrado Energizado)" (p345).

var resistanceTarget = ModifierTarget{K: "resistance"}

func devoteeOf(ch Character, god string) Character {
	ch.God = &god
	return ch
}

func holdingTheSymbol(ch Character, id, slot, god string) Character {
	ch.Items = append(ch.Items, CharacterItem{CatalogID: &id, Name: id, Quantity: 1, Equipped: &slot, Choice: god})
	return ch
}

func TestTheHolySymbolGivesResistanceOnlyToADevoteeOfItsGod(t *testing.T) {
	world := BookRuleset(primeBook(t))
	cleric := devoteeOf(attributeSumHero("Clérigo", 5, `[]`), "wynna")
	for _, c := range []struct {
		why, slot, god string
		want           int
	}{
		{"vestido, do deus da ficha", "vested", "wynna", 1},
		{"empunhado, do deus da ficha", "wielded", "wynna", 1},
		{"de outro deus", "vested", "khalmyr", 0},
		{"sem deus escolhido", "vested", "", 0},
	} {
		hero := holdingTheSymbol(cleric, "simbolo-sagrado", c.slot, c.god)
		if got := StatFor(world.EffectsOf(hero, nil), resistanceTarget).Total; got != c.want {
			t.Errorf("o símbolo sagrado %s deu %+d em resistência, e a p157 dá %+d", c.why, got, c.want)
		}
	}
}

func TestTheBlessedSymbolCountsAsAHolySymbolAndCutsOnlyForADevotee(t *testing.T) {
	world := BookRuleset(primeBook(t))
	cleric := devoteeOf(attributeSumHero("Clérigo", 5, `[]`), "wynna")

	devout := holdingTheSymbol(cleric, "simbolo-abencoado", "vested", "wynna")
	if got := StatFor(world.EffectsOf(devout, nil), resistanceTarget).Total; got != 1 {
		t.Errorf("o abençoado conta como símbolo sagrado, e deu %+d em resistência ao devoto; want +1", got)
	}
	if got := world.ComputeSheet(devout, nil).ForSpell("pmCost", SpellAt{Tradition: "divina"}).Total; got != -1 {
		t.Errorf("o abençoado do deus da ficha corta %d nas divinas; want −1", got)
	}

	other := holdingTheSymbol(cleric, "simbolo-abencoado", "vested", "khalmyr")
	if got := StatFor(world.EffectsOf(other, nil), resistanceTarget).Total; got != 0 {
		t.Errorf("o abençoado de outro deus deu %+d em resistência; want 0", got)
	}
	if got := world.ComputeSheet(other, nil).ForSpell("pmCost", SpellAt{Tradition: "divina"}).Total; got != 0 {
		t.Errorf("o abençoado de outro deus corta %d nas divinas; want 0", got)
	}
}

// "Enquanto você estiver empunhando um símbolo sagrado energizado, o custo em PM para lançar suas magias
// divinas diminui em 1" (p58). O efeito é o que o Usar grava, até o fim da cena; o símbolo tem de estar na
// mão, e tem de ser do deus da ficha ("seu símbolo sagrado", leitura do dono).
func TestTheEnergizedSymbolCutsWhileTheOwnGodsSymbolIsInHand(t *testing.T) {
	world := BookRuleset(primeBook(t))
	energized := devoteeOf(attributeSumHero("Clérigo", 5, `["class.clerigo.simbolo-sagrado-energizado"]`), "wynna")
	energized.ActiveEffects = []ActiveEffectRow{{CatalogID: "class.clerigo.simbolo-sagrado-energizado", Scope: "scene",
		Modifiers: `[{"target":{"k":"pmCost","tradition":"divina"},"amount":-1,"bonusType":"untyped",` +
			`"condition":{"c":"flagRequired","flag":"simbolo-sagrado-empunhado","label":"empunhando um símbolo sagrado do seu deus"}}]`}}
	divine := SpellAt{Tradition: "divina"}

	for _, c := range []struct {
		why, item, slot, god string
		want                 int
	}{
		{"com o símbolo do seu deus na mão", "simbolo-sagrado", "wielded", "wynna", -1},
		{"com o símbolo vestido", "simbolo-sagrado", "vested", "wynna", 0},
		{"com o símbolo de outro deus na mão", "simbolo-sagrado", "wielded", "khalmyr", 0},
		{"com o abençoado do seu deus na mão, que acumula (p345)", "simbolo-abencoado", "wielded", "wynna", -2},
	} {
		hero := holdingTheSymbol(energized, c.item, c.slot, c.god)
		if got := world.ComputeSheet(hero, nil).ForSpell("pmCost", divine).Total; got != c.want {
			t.Errorf("energizado %s, as divinas cortam %d; want %d", c.why, got, c.want)
		}
	}
}
