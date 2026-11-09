package engine

import "testing"

// A ARMA ADAPTÁVEL: "uma arma de uma mão com esta habilidade pode ser usada com as duas mãos para
// aumentar seu dano em um passo" (p143). Quem decide é a MÃO, e não o catálogo: em duas mãos ela é
// arma de duas mãos para o dado e para os estilos.
func TestAnAdaptableWeaponIsWhatTheGripSays(t *testing.T) {
	world := BookRuleset(primeBook(t))
	for _, c := range []struct {
		name, weapon, grip, damage string
	}{
		{"bastarda em uma mão", "espada-bastarda", "wielded", "1d10"},
		{"bastarda em duas mãos", "espada-bastarda", "wielded2", "1d12"},
		{"katana em uma mão", "katana", "wielded", "1d8"},
		{"katana em duas mãos", "katana", "wielded2", "1d10"},
	} {
		t.Run(c.name, func(t *testing.T) {
			grip := c.grip
			hero := func(powers string) Character {
				base := switchHero(powers, 0, c.weapon)
				base.Items[0].Equipped = &grip
				return base
			}
			card := world.ComputeWeaponCards(hero(`[]`), nil)[0]
			if card.CatalogID != c.weapon {
				t.Fatalf("a primeira carta não é %s: %+v", c.weapon, card)
			}
			if card.Damage != c.damage {
				t.Errorf("o dado é %q, e o livro dá %q", card.Damage, c.damage)
			}
			inTwoHands := grip == "wielded2"

			styled := func(power string) WeaponCard {
				return world.ComputeWeaponCards(hero(`["`+power+`"]`), map[string]bool{FlagGroupID(power): true})[0]
			}
			twoHanded := styled("estilo-de-duas-maos").DamageBonus - card.DamageBonus
			if (twoHanded == 5) != inTwoHands {
				t.Errorf("Estilo de Duas Mãos somou %+d no dano com a pegada %s", twoHanded, grip)
			}
			oneHanded := styled("estilo-de-uma-arma").Attack - card.Attack
			if (oneHanded == 2) == inTwoHands {
				t.Errorf("Estilo de Uma Arma somou %+d no ataque com a pegada %s", oneHanded, grip)
			}
		})
	}
}

// "É muito grande para ser usada com uma só mão sem treinamento especial; por isso, é uma arma exótica.
// Ela pode ser usada como uma arma marcial de duas mãos" (p147).
func TestAnAdaptableExoticWeaponIsMartialInTwoHands(t *testing.T) {
	world := BookRuleset(primeBook(t))
	for _, weapon := range []string{"espada-bastarda", "katana"} {
		attack := func(grip, proficiencies string) int {
			hero := switchHero(`[]`, 0, weapon)
			hero.Proficiencies = proficiencies
			hero.Items[0].Equipped = &grip
			return world.ComputeWeaponCards(hero, nil)[0].Attack
		}
		const exotic, martial = `["armas-exoticas"]`, `["armas-marciais"]`
		// O CONTROLE: com a exótica, as duas pegadas somam igual.
		if attack("wielded", exotic) != attack("wielded2", exotic) {
			t.Fatalf("%s: com a proficiência exótica, a pegada mudou o ataque", weapon)
		}
		if got, want := attack("wielded", martial), attack("wielded", exotic)-5; got != want {
			t.Errorf("%s em uma mão, só com a marcial: %+d, e a falta de proficiência dá %+d", weapon, got, want)
		}
		if got, want := attack("wielded2", martial), attack("wielded2", exotic); got != want {
			t.Errorf("%s em duas mãos, com a marcial: %+d, e a p147 não cobra nada: %+d", weapon, got, want)
		}
	}
}
