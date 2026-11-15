package engine

import "testing"

// A HARMONIZADA (ARMA) CORTA A HABILIDADE ESCOLHIDA, NAQUELA ARMA.
//
// "Escolha uma habilidade ativada ao se fazer um ataque ou usar a ação agredir e que custe pontos de mana.
// Esta habilidade tem seu custo em PM reduzido em –1 se utilizada com esta arma" (p165). A habilidade é a
// escolha da melhoria, gravada no item; o corte vale só nela e só no golpe com aquela arma, e o custo
// não desce de 1 PM.

const harmonizedWeapon = "melhoria-harmonizada-arma"

// twoBladesOneHarmonized empunha duas armas de corpo a corpo, e a primeira leva a Harmonizada.
func twoBladesOneHarmonized(ch Character, chosen string) Character {
	main, off := "espada-longa", "espada-curta"
	wielded, second := "wielded", "wielded2"
	sword := CharacterItem{ID: 1, CatalogID: &main, Name: "Espada longa", Quantity: 1, Equipped: &wielded,
		Improvements: `["` + harmonizedWeapon + `"]`}
	if chosen != "" {
		sword.ImprovementChoices = map[string]string{harmonizedWeapon: chosen}
	}
	short := CharacterItem{ID: 2, CatalogID: &off, Name: "Espada curta", Quantity: 1, Equipped: &second}
	ch.Items = append(ch.Items, sword, short)
	return ch
}

func boostPMOn(t *testing.T, world *Ruleset, ch Character, weapon string, choice BoostChoice) int {
	t.Helper()
	cards, err := world.BoostedWeaponCards(ch, nil, choice)
	if err != nil {
		t.Fatalf("pedir %s: %v", choice.PowerID, err)
	}
	for _, card := range cards {
		if card.CatalogID == weapon {
			return card.BoostPM
		}
	}
	t.Fatalf("nenhuma carta para %q", weapon)
	return 0
}

func TestTheHarmonizedWeaponCutsTheChosenBoostOnIt(t *testing.T) {
	world := BookRuleset(primeBook(t))
	const divine = "class.paladino.golpe-divino-1d8"
	paladin := attributeSumHero("Paladino", 5, `[]`)

	for _, c := range []struct {
		why, chosen, weapon string
		steps, want         int
	}{
		{"o Golpe Divino com a espada harmonizada para ele", divine, "espada-longa", 0, 1},
		{"com um degrau a mais, na mesma espada", divine, "espada-longa", 1, 2},
		{"com a outra arma, que não é harmonizada", divine, "espada-curta", 0, 2},
		{"com a espada harmonizada sem habilidade escolhida", "", "espada-longa", 0, 2},
		{"com a espada harmonizada para outra habilidade", "class.guerreiro.ataque-especial-4", "espada-longa", 0, 2},
	} {
		hero := twoBladesOneHarmonized(paladin, c.chosen)
		if got := boostPMOn(t, world, hero, c.weapon, BoostChoice{PowerID: divine, Steps: c.steps}); got != c.want {
			t.Errorf("%s: o Golpe Divino custou %d PM, e são %d (p165)", c.why, got, c.want)
		}
	}
}

// O PISO é 1 PM: o Ataque Especial custa 1, e a Harmonizada não o deixa de graça.
func TestTheHarmonizedWeaponDoesNotTakeABoostBelowOnePM(t *testing.T) {
	world := BookRuleset(primeBook(t))
	const special = "class.guerreiro.ataque-especial-4"
	fighter := twoBladesOneHarmonized(attributeSumHero("Guerreiro", 5, `[]`), special)
	if got := boostPMOn(t, world, fighter, "espada-longa", BoostChoice{PowerID: special, Split: BoostToAttack}); got != 1 {
		t.Errorf("o Ataque Especial com a espada harmonizada custou %d PM, e o piso é 1", got)
	}
}
