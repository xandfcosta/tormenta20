package engine

import "testing"

// O ALQUEBRADO NO GOLPE (ALE-647). "O custo em pontos de mana das habilidades do personagem aumenta em +1"
// (p394), e cada habilidade paga o seu: o impulso, o acréscimo engatilhado nele (o Romper Resistências, p66 — a
// p224 chama de habilidade a que se ativa por outro evento), o ataque pago e a manobra em seguida. A ordem é a
// da conta única (`AbilityPmCost`): o +1 entra antes do piso.
func broken(ch Character) Character {
	ch.ActiveConditions = `["alquebrado"]`
	return ch
}

func TestTheBrokenConditionRaisesTheBoostPrice(t *testing.T) {
	world := BookRuleset(primeBook(t))
	const divine = "class.paladino.golpe-divino-1d8"
	paladin := wielding(attributeSumHero("Paladino", 5, `[]`), "espada-longa")
	plain := boostPMOn(t, world, paladin, "espada-longa", BoostChoice{PowerID: divine})
	if got := boostPMOn(t, world, broken(paladin), "espada-longa", BoostChoice{PowerID: divine}); got != plain+1 {
		t.Errorf("Alquebrado, o Golpe Divino custou %d PM; want %d + 1", got, plain)
	}

	// O EXEMPLO DA DECISÃO: o Ataque Especial de 1 PM na espada harmonizada para ele (–1) e o Alquebrado (+1)
	// custam 1 PM — o +1 entra antes do piso.
	const special = "class.guerreiro.ataque-especial-4"
	fighter := twoBladesOneHarmonized(attributeSumHero("Guerreiro", 5, `[]`), special)
	if got := boostPMOn(t, world, broken(fighter), "espada-longa", BoostChoice{PowerID: special, Split: BoostToAttack}); got != 1 {
		t.Errorf("o Ataque Especial na espada harmonizada, Alquebrado, custou %d PM; want 1", got)
	}
}

func TestTheBrokenConditionRaisesTheAddOnToo(t *testing.T) {
	world := BookRuleset(primeBook(t))
	fighter := wielding(attributeSumHero("Guerreiro", 6, `["class.guerreiro.romper-resistencias"]`), "espada-longa")
	choice := BoostChoice{PowerID: "class.guerreiro.ataque-especial-4", Split: BoostToAttack, AddOn: "class.guerreiro.romper-resistencias"}
	_, plain, err := boostedCard(t, world, fighter, "espada-longa", choice)
	if err != nil {
		t.Fatal(err)
	}
	if _, got, err := boostedCard(t, world, broken(fighter), "espada-longa", choice); err != nil || got != plain+2 {
		t.Errorf("Alquebrado, o Ataque Especial com o Romper Resistências custou %d PM (erro %v); want %d + 2, um por habilidade",
			got, err, plain)
	}

	for _, offer := range world.BoostOffers(broken(fighter)) {
		if offer.PowerID != "class.guerreiro.ataque-especial-4" {
			continue
		}
		if offer.BasePM != 2 || len(offer.AddOns) != 1 || offer.AddOns[0].PM != 2 {
			t.Errorf("Alquebrado, a oferta do Ataque Especial diz %d PM e o acréscimo %+v; want 2 PM e o acréscimo por 2", offer.BasePM, offer.AddOns)
		}
	}
}

func TestTheBrokenConditionRaisesThePaidExtraAndTheFollowUp(t *testing.T) {
	world := BookRuleset(primeBook(t))
	fighter := wielding(attributeSumHero("Guerreiro", 6, `[]`), "espada-longa")
	card := cardOf(t, world.ComputeWeaponCards(broken(fighter), nil), "espada-longa")
	if len(card.PaidExtras) != 1 || card.PaidExtras[0].PM != 3 {
		t.Errorf("Alquebrado, o Ataque Extra (2 PM, p66) custa %+v; want 3 PM", card.PaidExtras)
	}

	grappler := attributeSumHero("Lutador", 5, `["class.lutador.lutador-de-chao"]`)
	follow := world.FollowUpManeuvers(broken(grappler), nil)
	if len(follow) != 1 || follow[0].PM != 2 {
		t.Errorf("Alquebrado, a derrubada do Lutador de Chão (1 PM, p77) custa %+v; want 2 PM", follow)
	}
}
