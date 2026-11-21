package engine

import "testing"

// AS LINHAS DO DANO DESARMADO (ALE-655, desenho do dono). A Arma Improvisada (p76): atacar com arma improvisada
// "conta como fazer um ataque desarmado, mas seu dano aumenta em um passo". A Chave (p76): agarrando e fazendo o
// teste de manobra para causar dano, "o dano desarmado aumenta em um passo". Nenhum interruptor: a carta desarmada
// leva uma linha por situação, com o dado já um passo acima, e a mesa escolhe a linha no passo do dano.
func TestTheUnarmedCardOffersADamageLinePerSituation(t *testing.T) {
	world := BookRuleset(primeBook(t))

	card := unarmedCard(t, world, attributeSumHero("Lutador", 5, `["class.lutador.arma-improvisada","class.lutador.chave"]`))
	if card.Damage != "1d8" {
		t.Fatalf("o desarmado do lutador de 5º nível é %s; want 1d8 (p75)", card.Damage)
	}
	want := []DamageVariant{
		{ID: "class.lutador.arma-improvisada", Label: "com arma improvisada", Source: "Arma Improvisada", BookPage: 76, Damage: "1d10"},
		{ID: "class.lutador.chave", Label: "esmagando quem agarra", Source: "Chave", BookPage: 76, Damage: "1d10"},
	}
	if len(card.DamageVariants) != len(want) {
		t.Fatalf("a carta desarmada tem as linhas %+v; want %+v", card.DamageVariants, want)
	}
	for i := range want {
		if card.DamageVariants[i] != want[i] {
			t.Errorf("a linha %d é %+v; want %+v", i, card.DamageVariants[i], want[i])
		}
	}

	// Só quem tem o poder, e só na carta desarmada.
	if plain := unarmedCard(t, world, attributeSumHero("Lutador", 5, `[]`)); len(plain.DamageVariants) != 0 {
		t.Errorf("o lutador sem os poderes tem as linhas %+v", plain.DamageVariants)
	}
	sword := cardOf(t, world.ComputeWeaponCards(wielding(attributeSumHero("Lutador", 5, `["class.lutador.chave"]`), "espada-longa"), nil), "espada-longa")
	if len(sword.DamageVariants) != 0 {
		t.Errorf("a espada longa ganhou as linhas do desarmado: %+v", sword.DamageVariants)
	}
}

// A LINHA ESCOLHIDA É O GOLPE QUE SE ROLA: a carta com a variante tem o dado um passo acima, e o resto da conta
// (o bônus, o crítico) é o mesmo da carta.
func TestTheChosenDamageLineStepsTheCardDice(t *testing.T) {
	world := BookRuleset(primeBook(t))
	card := unarmedCard(t, world, attributeSumHero("Lutador", 5, `["class.lutador.chave"]`))

	held, err := WithDamageVariant(card, "class.lutador.chave")
	if err != nil {
		t.Fatal(err)
	}
	if held.Damage != "1d10" || held.DamageBonus != card.DamageBonus || held.CritRange != card.CritRange {
		t.Errorf("a carta esmagando tem %s+%d (crítico %d); want 1d10+%d (crítico %d)",
			held.Damage, held.DamageBonus, held.CritRange, card.DamageBonus, card.CritRange)
	}
	if same, err := WithDamageVariant(card, ""); err != nil || same.Damage != card.Damage {
		t.Errorf("sem linha escolhida a carta mudou para %s (erro %v)", same.Damage, err)
	}
	if _, err := WithDamageVariant(card, "class.lutador.arma-improvisada"); err == nil {
		t.Error("a carta aceitou a linha de um poder que a ficha não tem")
	}
}
