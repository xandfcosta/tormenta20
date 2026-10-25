package engine

import (
	"testing"
)

// O TAMANHO DA RAÇA SOMA EM FURTIVIDADE E NAS MANOBRAS (p106–107, Tab. 1-21).
//
//	"Criaturas menores recebem bônus em Furtividade e penalidade em manobras de
//	 combate." — Minúsculo +5/–5, Pequeno +2/–2, Médio 0.
//
//	"Criaturas Minúsculas usam armas reduzidas, que causam um passo a menos de
//	 dano."
//
// O `races.json` já guardava o tamanho (goblin e hynne Pequenos, sílfide
// Minúscula), e o motor só o lia no tabuleiro.
//
// # O que se prende
//
// O PREDICADO é a categoria: o humano, Médio, é o controle que não muda. A
// manobra vale para os DOIS lados do teste oposto (p234) — o goblin pequeno
// derruba pior E resiste pior a ser derrubado.
func TestTheRaceSizeShowsInStealthManeuversAndWeaponDamage(t *testing.T) {
	world := BookRuleset(primeBook(t))

	espada, mao := "espada-curta", "wielded"
	of := func(race string) Character {
		return Character{
			Level: 1, Dexterity: 2,
			Races:      []CharacterRace{{Race: race}},
			Expertises: []CharacterExpertise{{Name: "Furtividade", Attribute: "dexterity"}},
			Items:      []CharacterItem{{CatalogID: &espada, Name: "Espada curta", Quantity: 1, Equipped: &mao}},
		}
	}
	furtividade := func(ch Character) int {
		for _, e := range world.ComputeSheet(ch, nil).Expertises {
			if e.Name == "Furtividade" {
				return e.Total
			}
		}
		t.Fatal("a ficha não decompôs Furtividade")
		return 0
	}
	manobra := func(ch Character, lado ManeuverRole) int {
		return ManeuverBonus(ComputeItemEffects(world.ActiveItemsFor(ch)), "derrubar", lado)
	}
	dado := func(ch Character) string {
		cards := wieldedCards(world.ComputeWeaponCards(ch, nil))
		if len(cards) != 1 {
			t.Fatalf("esperava a carta da espada curta, e vieram %d", len(cards))
		}
		return cards[0].Damage
	}

	cases := []struct {
		race           string
		stealth, maneu int
		damage         string
	}{
		// Destreza 2 mais a da raça (goblin +2, sílfide +1, p23 e p30) em
		// Furtividade; a espada curta é 1d6 (p143).
		{"humano", 2, 0, "1d6"},
		{"goblin", 2 + 2 + 2, -2, "1d6"},
		{"silfide", 2 + 1 + 5, -5, "1d4"},
	}
	for _, c := range cases {
		ch := of(c.race)
		if got := furtividade(ch); got != c.stealth {
			t.Errorf("%s: Furtividade %d, e a Tab. 1-21 (p107) dá %d", c.race, got, c.stealth)
		}
		for _, lado := range []ManeuverRole{ManeuverOffense, ManeuverDefense} {
			if got := manobra(ch, lado); got != c.maneu {
				t.Errorf("%s: derrubar (lado %q) %+d, e a Tab. 1-21 (p107) dá %+d",
					c.race, lado, got, c.maneu)
			}
		}
		if got := dado(ch); got != c.damage {
			t.Errorf("%s: a espada curta rola %s, e o esperado é %s (p106: o Minúsculo "+
				"usa arma reduzida, um passo a menos)", c.race, got, c.damage)
		}
	}
}
