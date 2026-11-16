package engine

import "testing"

// O QUE MUDA O GOLPE DIVINO (ALE-652, fatia C).
//
//   - Arma Sagrada (paladino, p82): "Quando usa Golpe Divino para atacar com a arma preferida de sua divindade,
//     o dado de dano que você rola por Golpe Divino aumenta para d12."
//   - Vingador Sagrado (paladino, p84): "seu Golpe Divino tem seu custo reduzido à metade e causa mais dois
//     dados de dano."
//
// A metade arredonda para baixo, as reduções de custo não se acumulam e o custo não desce de 1 PM (p226).

const divineStrike = "class.paladino.golpe-divino-1d8"

// avengerForm é o efeito de cena que o gesto do Vingador Sagrado deixa na ficha, com o que ele faz no Golpe
// Divino. O catálogo é quem o escreve; o teste da ligação gesto → efeito mora em `serve/api`.
const avengerForm = `[{"target":{"k":"boostCostHalved","name":"` + divineStrike + `"},"amount":1,"bonusType":"untyped"},` +
	`{"target":{"k":"boostExtraDice","name":"` + divineStrike + `"},"amount":2,"bonusType":"untyped"}]`

func devotedTo(ch Character, god string) Character {
	ch.God = &god
	return ch
}

func inAvengerForm(ch Character) Character {
	ch.ActiveEffects = append(ch.ActiveEffects, ActiveEffectRow{CatalogID: "class.paladino.vingador-sagrado",
		Scope: "scene", Modifiers: avengerForm})
	return ch
}

// divineCardOn é a carta da arma `weapon` com o Golpe Divino de `steps` degraus a mais.
func divineCardOn(t *testing.T, world *Ruleset, ch Character, weapon string, steps int) WeaponCard {
	t.Helper()
	cards, err := world.BoostedWeaponCards(ch, nil, BoostChoice{PowerID: divineStrike, Steps: steps})
	if err != nil {
		t.Fatalf("o Golpe Divino com %d degraus: %v", steps, err)
	}
	for _, card := range cards {
		if card.CatalogID == weapon {
			return card
		}
	}
	t.Fatalf("nenhuma carta para %q", weapon)
	return WeaponCard{}
}

func divineDiceOf(card WeaponCard) string {
	if len(card.ExtraDamage) != 1 {
		return ""
	}
	return card.ExtraDamage[0].Dice
}

func TestTheSacredWeaponRollsTheDivineStrikeOnD12WithTheFavoredWeapon(t *testing.T) {
	world := BookRuleset(primeBook(t))
	sacred := `["class.paladino.arma-sagrada"]`
	for _, c := range []struct {
		why, powers, god, weapon, dice, buys string
	}{
		{"a espada longa é a arma de Khalmyr", sacred, "khalmyr", "espada-longa", "2d12", "+2d12"},
		{"a espada curta, na outra mão, não é", sacred, "khalmyr", "espada-curta", "2d8", "+2d8"},
		{"sem a Arma Sagrada, a espada de Khalmyr rola d8", `[]`, "khalmyr", "espada-longa", "2d8", "+2d8"},
		{"a arma de Lin-Wu é a katana, e não a espada longa", sacred, "lin-wu", "espada-longa", "2d8", "+2d8"},
	} {
		paladin := devotedTo(wielding(attributeSumHero("Paladino", 5, c.powers), "espada-longa", "espada-curta"), c.god)
		card := divineCardOn(t, world, paladin, c.weapon, 1)
		if divineDiceOf(card) != c.dice || card.BoostBuys != c.buys {
			t.Errorf("%s: o Golpe Divino de 1 degrau rolou %+v e a tira diz %q, want %s e %q",
				c.why, card.ExtraDamage, card.BoostBuys, c.dice, c.buys)
		}
	}
}

func TestTheHolyAvengerHalvesTheDivineStrikeAndAddsTwoDice(t *testing.T) {
	world := BookRuleset(primeBook(t))
	paladin := devotedTo(wielding(attributeSumHero("Paladino", 20, `[]`), "espada-curta"), "khalmyr")
	for _, c := range []struct {
		steps, pm int
		dice      string
	}{
		{0, 1, "3d8"}, // 2 PM pela metade é 1; 1d8 e mais dois dados
		{1, 1, "4d8"}, // 3 PM pela metade, para baixo, é 1
		{4, 3, "7d8"}, // o teto do 20º nível: 6 PM pela metade
	} {
		card := divineCardOn(t, world, inAvengerForm(paladin), "espada-curta", c.steps)
		if card.BoostPM != c.pm || divineDiceOf(card) != c.dice {
			t.Errorf("na forma de vingador, com %d degraus: %d PM e %+v, want %d PM e %s",
				c.steps, card.BoostPM, card.ExtraDamage, c.pm, c.dice)
		}
	}
	plain := divineCardOn(t, world, paladin, "espada-curta", 0)
	if plain.BoostPM != 2 || divineDiceOf(plain) != "1d8" {
		t.Errorf("fora da forma: %d PM e %+v, want 2 PM e 1d8", plain.BoostPM, plain.ExtraDamage)
	}
	// Os dois dados a mais são os do Golpe Divino: com a Arma Sagrada e a arma de Khalmyr, d12.
	sacred := devotedTo(wielding(attributeSumHero("Paladino", 20, `["class.paladino.arma-sagrada"]`), "espada-longa"), "khalmyr")
	if card := divineCardOn(t, world, inAvengerForm(sacred), "espada-longa", 0); divineDiceOf(card) != "3d12" || card.BoostBuys != "+3d12" {
		t.Errorf("o vingador com a Arma Sagrada rolou %+v e a tira diz %q, want 3d12", card.ExtraDamage, card.BoostBuys)
	}
}

// A METADE E O CORTE DA HARMONIZADA NÃO SE ACUMULAM (p226): vale a maior redução. Com 2 degraus o Golpe Divino
// custa 4 — a metade dá 2, a Harmonizada dá 3, e as duas juntas dariam 1.
func TestTheHolyAvengerDoesNotStackWithTheHarmonizedCut(t *testing.T) {
	world := BookRuleset(primeBook(t))
	paladin := inAvengerForm(devotedTo(twoBladesOneHarmonized(attributeSumHero("Paladino", 20, `[]`), divineStrike), "khalmyr"))
	for _, c := range []struct{ steps, pm int }{{2, 2}, {0, 1}} {
		if got := boostPMOn(t, world, paladin, "espada-longa", BoostChoice{PowerID: divineStrike, Steps: c.steps}); got != c.pm {
			t.Errorf("o vingador com a espada harmonizada, %d degraus: %d PM, want %d", c.steps, got, c.pm)
		}
	}
}
