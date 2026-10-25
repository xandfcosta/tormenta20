package engine

import "testing"

// O IMPULSO DE PM NO ATAQUE (cap. 1, ALE-487).
//
// O jogador escolhe, ao atacar, gastar PM num poder que vende bônus por degrau:
// o PM de base compra o bônus e cada DEGRAU a mais custa +1 PM — a mesma conta
// das posturas (`Steps` são os degraus além do primeiro) —, e o teto de degraus
// cresce com o nível NA CLASSE: um no 5º, dois no 9º, até quatro no 17º. O motor confere
// o pedido contra o livro e devolve a carta com o bônus e o PM que ele custa;
// quem cobra é a confirmação do mestre.

func boostedCard(t *testing.T, world *Ruleset, ch Character, weapon string, choice BoostChoice) (WeaponCard, int, error) {
	t.Helper()
	cards, cost, err := world.BoostedWeaponCards(ch, nil, choice)
	if err != nil {
		return WeaponCard{}, 0, err
	}
	for _, c := range cards {
		if c.Name == weapon {
			return c, cost, nil
		}
	}
	t.Fatalf("nenhuma carta para %q", weapon)
	return WeaponCard{}, 0, nil
}

// p82 "Quando faz um ataque corpo a corpo, você pode gastar 2 PM para desferir
// um golpe destruidor. Você soma seu Carisma no teste de ataque e +1d8 na
// rolagem de dano. A cada quatro níveis, pode gastar +1 PM para aumentar o dano
// em +1d8."
func TestTheDivineStrikeAddsCharismaAndOneD8PerDegree(t *testing.T) {
	world := BookRuleset(primeBook(t))
	paladin := wielding(attributeSumHero("Paladino", 5, `[]`), "espada-longa", "arco-curto")
	paladin.Charisma = 3
	plain, _, _ := boostedCard(t, world, paladin, "espada-longa", BoostChoice{})

	one, cost, err := boostedCard(t, world, paladin, "espada-longa", BoostChoice{PowerID: "class.paladino.golpe-divino-1d8"})
	if err != nil {
		t.Fatalf("Golpe Divino sem degrau: %v", err)
	}
	if cost != 2 || one.Attack-plain.Attack != 3 || len(one.ExtraDamage) != 1 || one.ExtraDamage[0].Dice != "1d8" {
		t.Errorf("sem degrau: custo %d, ataque +%d, extra %+v; want 2 PM, +3 (Car), 1d8",
			cost, one.Attack-plain.Attack, one.ExtraDamage)
	}

	two, cost, err := boostedCard(t, world, paladin, "espada-longa", BoostChoice{PowerID: "class.paladino.golpe-divino-1d8", Steps: 1})
	if err != nil {
		t.Fatalf("Golpe Divino com 1 degrau no 5º nível: %v", err)
	}
	if cost != 3 || two.Attack-plain.Attack != 3 || len(two.ExtraDamage) != 1 || two.ExtraDamage[0].Dice != "2d8" {
		t.Errorf("1 degrau: custo %d, ataque +%d, extra %+v; want 3 PM, +3 (o Carisma não cresce), 2d8",
			cost, two.Attack-plain.Attack, two.ExtraDamage)
	}

	if _, _, err := world.BoostedWeaponCards(paladin, nil, BoostChoice{PowerID: "class.paladino.golpe-divino-1d8", Steps: 2}); err == nil {
		t.Errorf("2 degraus no 5º nível passaram — o teto é 1 até o 9º")
	}
	low := wielding(attributeSumHero("Paladino", 4, `[]`), "espada-longa")
	if _, _, err := world.BoostedWeaponCards(low, nil, BoostChoice{PowerID: "class.paladino.golpe-divino-1d8", Steps: 1}); err == nil {
		t.Errorf("1 degrau no 4º nível passou — o primeiro é do 5º")
	}
	// "Quando faz um ataque CORPO A CORPO": o arco fica sem o golpe.
	bow, _, err := boostedCard(t, world, paladin, "arco-curto", BoostChoice{PowerID: "class.paladino.golpe-divino-1d8"})
	if err != nil {
		t.Fatalf("o pedido inteiro caiu por causa do arco: %v", err)
	}
	if len(bow.ExtraDamage) != 0 {
		t.Errorf("o arco ganhou o Golpe Divino: %+v — o livro diz corpo a corpo", bow.ExtraDamage)
	}
	// Quem não é paladino não tem o poder.
	fighter := wielding(attributeSumHero("Guerreiro", 5, `[]`), "espada-longa")
	if _, _, err := world.BoostedWeaponCards(fighter, nil, BoostChoice{PowerID: "class.paladino.golpe-divino-1d8"}); err == nil {
		t.Errorf("um guerreiro usou o Golpe Divino")
	}
}

// p65 "Quando faz um ataque, você pode gastar 1 PM para receber +4 no teste de
// ataque ou na rolagem de dano. A cada quatro níveis, pode gastar +1 PM para
// aumentar o bônus em +4. Você pode dividir os bônus igualmente. Por exemplo,
// no 17º nível, pode gastar 5 PM para receber +20 no ataque, +20 no dano ou +10
// no ataque e +10 no dano."
func TestTheSpecialAttackGoesToAttackDamageOrHalfEach(t *testing.T) {
	world := BookRuleset(primeBook(t))
	fighter := wielding(attributeSumHero("Guerreiro", 17, `[]`), "espada-longa")
	plain, _, _ := boostedCard(t, world, fighter, "espada-longa", BoostChoice{})
	special := func(split string) (int, int, int) {
		card, cost, err := boostedCard(t, world, fighter, "espada-longa",
			BoostChoice{PowerID: "class.guerreiro.ataque-especial-4", Steps: 4, Split: split})
		if err != nil {
			t.Fatalf("Ataque Especial %s: %v", split, err)
		}
		return card.Attack - plain.Attack, card.DamageBonus - plain.DamageBonus, cost
	}
	if atk, dmg, cost := special(BoostToAttack); atk != 20 || dmg != 0 || cost != 5 {
		t.Errorf("no ataque: +%d/+%d por %d PM, want +20/+0 por 5 PM (o exemplo da p65)", atk, dmg, cost)
	}
	if atk, dmg, _ := special(BoostToDamage); atk != 0 || dmg != 20 {
		t.Errorf("no dano: +%d/+%d, want +0/+20", atk, dmg)
	}
	if atk, dmg, _ := special(BoostSplitEvenly); atk != 10 || dmg != 10 {
		t.Errorf("dividido: +%d/+%d, want +10/+10", atk, dmg)
	}
	if _, _, err := world.BoostedWeaponCards(fighter, nil,
		BoostChoice{PowerID: "class.guerreiro.ataque-especial-4"}); err == nil {
		t.Errorf("o Ataque Especial sem dizer onde vai o bônus passou")
	}
}
