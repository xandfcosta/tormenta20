package engine

import "testing"

// O QUE MUDA O GOLPE DIVINO (ALE-652, fatia C).
//
//   - Arma Sagrada (paladino, p82): "Quando usa Golpe Divino para atacar com a arma preferida de sua divindade,
//     o dado de dano que você rola por Golpe Divino aumenta para d12."

const divineStrike = "class.paladino.golpe-divino-1d8"

func devotedTo(ch Character, god string) Character {
	ch.God = &god
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
