package engine

import "testing"

// p137 "Membros Extras. […] Se possuir Ambidestria ou Estilo de Duas Armas, pode empunhar armas leves em
// suas patas insetoides (mas ainda precisa pagar 2 PM para atacar com elas e sofre a penalidade de –2 em
// todos os ataques)."

func inALeg(ch Character, id string) Character {
	slot := LegSlot
	ch.Items = append(ch.Items, CharacterItem{CatalogID: &id, Name: id, Quantity: 1, Equipped: &slot})
	return ch
}

func cardNamed(cards []WeaponCard, name string) *WeaponCard {
	for i := range cards {
		if cards[i].Name == name {
			return &cards[i]
		}
	}
	return nil
}

func TestTheArmedLegAttacksWithItsWeaponAndCostsTwoOnEveryAttack(t *testing.T) {
	world := BookRuleset(primeBook(t))
	powers := `["membros-extras","estilo-de-duas-armas"]`
	bare := wielding(attributeSumHero("Guerreiro", 5, powers), "adaga")
	armed := inALeg(bare, "espada-curta")

	before := world.ComputeWeaponCards(bare, nil)
	after := world.ComputeWeaponCards(armed, nil)

	sword := cardNamed(after, "espada-curta (pata)")
	if sword == nil {
		t.Fatalf("a espada curta na pata devia ter carta: %d cartas", len(after))
	}
	if sword.Natural == nil || sword.Natural.SourceID != "membros-extras" || sword.Natural.ExtraPM != 2 {
		t.Errorf("a pata armada ataca pelo extra das patas, com os mesmos 2 PM: %+v", sword.Natural)
	}
	if sword.Damage != "1d6" {
		t.Errorf("a pata armada usa o dano da arma: %s, want 1d6", sword.Damage)
	}
	if cardNamed(after, "Patas insetoides") == nil {
		t.Error("com uma pata livre, a pata natural continua atacando")
	}
	for _, name := range []string{"adaga", "Ataque desarmado", "Patas insetoides"} {
		was, is := cardNamed(before, name), cardNamed(after, name)
		if was == nil || is == nil {
			t.Fatalf("faltou a carta %q: antes %v, depois %v", name, was != nil, is != nil)
		}
		if is.Attack != was.Attack-2 {
			t.Errorf("%s: ataque %+d com a pata armada, want %+d (–2 em todos os ataques)", name, is.Attack, was.Attack-2)
		}
	}

	both := inALeg(armed, "adaga")
	if cardNamed(world.ComputeWeaponCards(both, nil), "Patas insetoides") != nil {
		t.Error("com as duas patas armadas, não sobra pata natural para atacar")
	}
}

func TestTheLegHoldsNothingWithoutTheStyleOrAHeavyWeapon(t *testing.T) {
	world := BookRuleset(primeBook(t))

	noStyle := inALeg(attributeSumHero("Guerreiro", 5, `["membros-extras"]`), "espada-curta")
	if cardNamed(world.ComputeWeaponCards(noStyle, nil), "espada-curta (pata)") != nil {
		t.Error("sem Ambidestria nem Estilo de Duas Armas, a pata não empunha arma")
	}

	heavy := inALeg(attributeSumHero("Guerreiro", 5, `["membros-extras","estilo-de-duas-armas"]`), "espada-longa")
	cards := world.ComputeWeaponCards(heavy, nil)
	if cardNamed(cards, "espada-longa (pata)") != nil {
		t.Error("a pata empunha só arma leve")
	}
	if patas := cardNamed(cards, "Patas insetoides"); patas == nil {
		t.Error("a pata com arma que não serve continua natural")
	}
}

func TestTheWarriorAmbidexterityAlsoArmsTheLegs(t *testing.T) {
	world := BookRuleset(primeBook(t))
	ch := inALeg(attributeSumHero("Guerreiro", 5, `["membros-extras","class.guerreiro.ambidestria"]`), "espada-curta")
	if cardNamed(world.ComputeWeaponCards(ch, nil), "espada-curta (pata)") == nil {
		t.Error("a Ambidestria de classe também deixa empunhar na pata (p137)")
	}
}
