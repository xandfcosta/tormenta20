package engine

import "testing"

// IGNORAR A REDUÇÃO DE DANO (ALE-652, fatia B).
//
//   - Punhos de Adamante (lutador, p77): "Seus ataques desarmados ignoram 10 pontos de redução de dano do alvo, se
//     houver."
//   - Romper Resistências (guerreiro, p66): "Quando faz um Ataque Especial, você pode gastar 1 PM adicional para
//     ignorar 10 pontos de redução de dano."
//
// "Ignorar 10 pontos" é a RD do alvo valer 10 a menos: a RD 12 vira 2, e a RD 5 não absorve nada — não há RD
// negativa que some dano.

func TestTheAdamantFistsIgnoreTenOnTheUnarmedStrike(t *testing.T) {
	world := BookRuleset(primeBook(t))
	fighter := wielding(attributeSumHero("Lutador", 8, `["class.lutador.punhos-de-adamante"]`), "adaga")
	cards := world.ComputeWeaponCards(fighter, nil)
	for _, card := range cards {
		want := 0
		if card.Name == UnarmedName {
			want = 10
		}
		if card.IgnoresDamageReduction != want {
			t.Errorf("%s ignora %d de RD com o Punhos de Adamante, want %d", card.Name, card.IgnoresDamageReduction, want)
		}
	}
}

// O golpe MENOR que a RD é o caso que separa as duas leituras: a RD 12 ignorando 10 é RD 2, e absorve 2 de um
// golpe de 7. Tirar os 10 do que a RD 12 já tinha absorvido (os 7 inteiros) não deixaria absorção nenhuma.
func TestTheIgnoredReductionAbsorbsTenLess(t *testing.T) {
	ones := func(int) (int, error) { return 1, nil }
	for _, c := range []struct {
		damage, general, impact, absorbed int
	}{
		{20, 12, 0, 2},
		{7, 12, 0, 2},
		{20, 5, 0, 0},
		{20, 0, 0, 0},
		// O que a geral não tem para ignorar sai da RD do tipo: 4 da geral, e 6 da de impacto 10.
		{20, 4, 10, 4},
	} {
		fist := WeaponCard{Name: UnarmedName, Skill: "Luta", Damage: "1d1", DamageType: "impacto", CritRange: 20,
			CritMult: 2, DamageBonus: c.damage - 1, IgnoresDamageReduction: 10}
		target := AttackTarget{Defense: 10, DamageReduction: c.general}
		if c.impact > 0 {
			target.DamageReductionByType = map[string]int{"impacto": c.impact}
		}
		out, err := RollAttackDamage(fist, target, nil, AttackDie{Rolled: 15, Natural: 15}, RulingHit, ones)
		if err != nil {
			t.Fatal(err)
		}
		if out.Absorbed != c.absorbed || out.Damage != c.damage-c.absorbed {
			t.Errorf("golpe de %d contra RD %d (impacto %d) ignorando 10: absorveu %d e ficou %d de dano, want %d e %d",
				c.damage, c.general, c.impact, out.Absorbed, out.Damage, c.absorbed, c.damage-c.absorbed)
		}
	}
}

func TestTheBreakResistancesIsAPaidAddOnOfTheSpecialAttack(t *testing.T) {
	world := BookRuleset(primeBook(t))
	fighter := wielding(attributeSumHero("Guerreiro", 6, `["class.guerreiro.romper-resistencias"]`), "espada-longa")
	special := BoostChoice{PowerID: "class.guerreiro.ataque-especial-4", Split: BoostToAttack}
	plain, plainCost, err := boostedCard(t, world, fighter, "espada-longa", special)
	if err != nil {
		t.Fatal(err)
	}
	if plain.IgnoresDamageReduction != 0 {
		t.Errorf("o Ataque Especial sem o acréscimo ignorou %d de RD", plain.IgnoresDamageReduction)
	}
	special.AddOn = "class.guerreiro.romper-resistencias"
	broken, brokenCost, err := boostedCard(t, world, fighter, "espada-longa", special)
	if err != nil {
		t.Fatalf("o Romper Resistências no Ataque Especial foi recusado: %v", err)
	}
	if broken.IgnoresDamageReduction != 10 || brokenCost != plainCost+1 {
		t.Errorf("o Romper Resistências ignora 10 de RD por +1 PM, e deu %d de RD por %d PM (o Ataque Especial custa %d)",
			broken.IgnoresDamageReduction, brokenCost, plainCost)
	}

	without := wielding(attributeSumHero("Guerreiro", 6, `[]`), "espada-longa")
	if _, err := world.BoostedWeaponCards(without, nil, special); err == nil {
		t.Error("o acréscimo foi aceito de quem não tem o Romper Resistências")
	}

	var offer *BoostOffer
	for _, o := range world.BoostOffers(fighter) {
		if o.PowerID == "class.guerreiro.ataque-especial-4" {
			offer = &o
		}
	}
	if offer == nil || len(offer.AddOns) != 1 || offer.AddOns[0].PowerID != "class.guerreiro.romper-resistencias" || offer.AddOns[0].PM != 1 {
		t.Errorf("a oferta do Ataque Especial não traz o acréscimo do Romper Resistências por 1 PM: %+v", offer)
	}
}
