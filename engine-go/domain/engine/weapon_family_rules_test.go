package engine

import "testing"

// O BÔNUS QUE VALE NUMA FAMÍLIA DE ARMAS (ALE-649): o corpo a corpo, o arremesso, as armas de fogo, ou uma lista
// de armas que o livro nomeia. A carta soma o que alcança os escopos DELA, e o item específico também é a arma
// base dele: a Azagaia dos relâmpagos é uma azagaia (decisão do dono).

func cardOf(t *testing.T, cards []WeaponCard, catalogID string) WeaponCard {
	t.Helper()
	for _, c := range cards {
		if c.CatalogID == catalogID {
			return c
		}
	}
	t.Fatalf("nenhuma carta para %q", catalogID)
	return WeaponCard{}
}

// thrownCardOf é a carta da arma usada à distância: a própria, se ela é de arremesso, ou o outro uso dela.
func thrownCardOf(t *testing.T, cards []WeaponCard, catalogID string) WeaponCard {
	t.Helper()
	card := cardOf(t, cards, catalogID)
	if card.IsMelee() && card.OtherUse != nil {
		return *card.OtherUse
	}
	return card
}

func barbarian(powers string, ids ...string) Character {
	ch := wielding(attributeSumHero("Bárbaro", 10, powers), ids...)
	ch.Strength = 3
	return ch
}

// A FÚRIA É CORPO A CORPO: "+2 em testes de ataque e rolagens de dano corpo a corpo" (p40). Com a Fúria da
// Savana, "você aplica o bônus em ataque e dano também a armas de arremesso" (p42) — e só a elas.
func TestTheFuryIsMeleeAndTheSavannaFuryReachesThrowing(t *testing.T) {
	world := BookRuleset(primeBook(t))
	rage := map[string]bool{FlagGroupID("furia"): true}
	for _, c := range []struct {
		why, powers, weapon string
		thrown              bool
		want                int
	}{
		{"o machado corpo a corpo", `[]`, "machado-batalha", false, 2},
		{"a azagaia arremessada, sem a Savana", `[]`, "azagaia", true, 0},
		{"o arco, sem a Savana", `[]`, "arco-curto", false, 0},
		{"a azagaia arremessada, com a Savana", `["class.barbaro.furia-da-savana"]`, "azagaia", true, 2},
		{"o arco, com a Savana", `["class.barbaro.furia-da-savana"]`, "arco-curto", false, 0},
	} {
		ch := barbarian(c.powers, c.weapon)
		pick := cardOf
		if c.thrown {
			pick = thrownCardOf
		}
		calm := pick(t, world.ComputeWeaponCards(ch, nil), c.weapon)
		furious := pick(t, world.ComputeWeaponCards(ch, rage), c.weapon)
		if got := furious.Attack - calm.Attack; got != c.want {
			t.Errorf("%s: a Fúria subiu o ataque em %d; want %d", c.why, got, c.want)
		}
		if got := furious.DamageBonus - calm.DamageBonus; got != c.want {
			t.Errorf("%s: a Fúria subiu o dano em %d; want %d", c.why, got, c.want)
		}
	}
}

// O CRÍTICO BRUTAL: "seu multiplicador de crítico com armas corpo a corpo e de arremesso aumenta em +1" (p41).
func TestTheBrutalCriticalReachesMeleeAndThrowing(t *testing.T) {
	world := BookRuleset(primeBook(t))
	ch := barbarian(`["class.barbaro.critico-brutal"]`, "machado-batalha", "azagaia")
	plain := world.ComputeWeaponCards(barbarian(`[]`, "machado-batalha", "azagaia"), nil)
	brutal := world.ComputeWeaponCards(ch, nil)
	if got := cardOf(t, brutal, "machado-batalha").CritMult - cardOf(t, plain, "machado-batalha").CritMult; got != 1 {
		t.Errorf("o Crítico Brutal subiu o multiplicador do machado em %d; want 1 (x3 vira x4)", got)
	}
	if got := thrownCardOf(t, brutal, "azagaia").CritMult - thrownCardOf(t, plain, "azagaia").CritMult; got != 1 {
		t.Errorf("o Crítico Brutal subiu o multiplicador da azagaia arremessada em %d; want 1", got)
	}
	bow := world.ComputeWeaponCards(barbarian(`["class.barbaro.critico-brutal"]`, "arco-curto"), nil)
	if got := cardOf(t, bow, "arco-curto").CritMult - cardOf(t, world.ComputeWeaponCards(barbarian(`[]`, "arco-curto"), nil), "arco-curto").CritMult; got != 0 {
		t.Errorf("o Crítico Brutal subiu o multiplicador do arco em %d; o arco não é corpo a corpo nem arremesso", got)
	}
}

func ofRace(ch Character, race string) Character {
	ch.Races = []CharacterRace{{Race: race}}
	return ch
}

// O ARREMESSADOR: "quando faz um ataque à distância com uma funda ou uma arma de arremesso, seu dano aumenta em
// um passo" (p28). A funda 1d4 vira 1d6; a adaga corpo a corpo não muda.
func TestTheThrowerStepsUpTheSlingAndTheThrownWeapon(t *testing.T) {
	world := BookRuleset(primeBook(t))
	hynne := world.ComputeWeaponCards(ofRace(barbarian(`[]`, "funda", "azagaia"), "hynne"), nil)
	human := world.ComputeWeaponCards(ofRace(barbarian(`[]`, "funda", "azagaia"), "humano"), nil)
	for _, weapon := range []string{"funda", "azagaia"} {
		if got, was := thrownCardOf(t, hynne, weapon).Damage, thrownCardOf(t, human, weapon).Damage; got != DamageStep(was, 1) {
			t.Errorf("a %s do hynne causa %s; want um passo acima de %s", weapon, got, was)
		}
	}
	dagger := world.ComputeWeaponCards(ofRace(barbarian(`[]`, "adaga"), "hynne"), nil)
	if got, was := cardOf(t, dagger, "adaga").Damage, cardOf(t, world.ComputeWeaponCards(ofRace(barbarian(`[]`, "adaga"), "humano"), nil), "adaga").Damage; got != was {
		t.Errorf("a adaga corpo a corpo do hynne causa %s, e o Arremessador não vale corpo a corpo (%s)", got, was)
	}
}

// A ARMA NOMEADA PELA RAÇA, e o item específico dela: o Mestre do Tridente dá +2 no dano com azagaias, lanças e
// tridentes (p30), e a Tradição de Heredrimm +2 no ataque com machados, martelos, marretas e picaretas (p20).
func TestTheRaceWeaponsReachTheSpecificItemToo(t *testing.T) {
	world := BookRuleset(primeBook(t))
	for _, c := range []struct {
		race, weapon string
		attack, dmg  int
	}{
		{"sereia-tritao", "tridente", 0, 2},
		{"sereia-tritao", "azagaia-dos-relampagos", 0, 2},
		{"sereia-tritao", "espada-longa", 0, 0},
		{"anao", "machado-batalha", 2, 0},
		{"anao", "martelo-guerra", 2, 0},
		{"anao", "espada-longa", 0, 0},
	} {
		with := cardOf(t, world.ComputeWeaponCards(ofRace(barbarian(`[]`, c.weapon), c.race), nil), c.weapon)
		without := cardOf(t, world.ComputeWeaponCards(ofRace(barbarian(`[]`, c.weapon), "humano"), nil), c.weapon)
		if got := with.Attack - without.Attack; got != c.attack {
			t.Errorf("%s com %s: o ataque subiu %d; want %d", c.race, c.weapon, got, c.attack)
		}
		if got := with.DamageBonus - without.DamageBonus; got != c.dmg {
			t.Errorf("%s com %s: o dano subiu %d; want %d", c.race, c.weapon, got, c.dmg)
		}
	}
}

// O PISTOLEIRO: "+2 nas rolagens de dano com essas armas" — as armas de fogo (p48).
func TestThePistoleerAddsDamageToFirearms(t *testing.T) {
	world := BookRuleset(primeBook(t))
	gunner := func(powers string, weapon string) WeaponCard {
		ch := wielding(attributeSumHero("Bucaneiro", 5, powers), weapon)
		return cardOf(t, world.ComputeWeaponCards(ch, nil), weapon)
	}
	if got := gunner(`["class.bucaneiro.pistoleiro"]`, "pistola").DamageBonus - gunner(`[]`, "pistola").DamageBonus; got != 2 {
		t.Errorf("o Pistoleiro subiu o dano da pistola em %d; want 2", got)
	}
	if got := gunner(`["class.bucaneiro.pistoleiro"]`, "arco-curto").DamageBonus - gunner(`[]`, "arco-curto").DamageBonus; got != 0 {
		t.Errorf("o Pistoleiro subiu o dano do arco em %d; o arco não é arma de fogo", got)
	}
}

// O ATAQUE CORPO A CORPO DA FICHA soma a Fúria, e o à distância não: é o número que a aba Combate e as manobras
// das Ações mostram fora de uma arma, e a Fúria é "corpo a corpo" (p40).
func TestTheSheetMeleeAttackCarriesTheFuryAndTheRangedDoesNot(t *testing.T) {
	world := BookRuleset(primeBook(t))
	ch := barbarian(`[]`, "machado-batalha")
	calm := world.ComputeSheet(ch, nil)
	furious := world.ComputeSheet(ch, map[string]bool{FlagGroupID("furia"): true})
	if got := furious.AttackMelee.Total - calm.AttackMelee.Total; got != 2 {
		t.Errorf("a Fúria subiu o ataque corpo a corpo da ficha em %d; want 2", got)
	}
	if got := furious.AttackAll.Total - calm.AttackAll.Total; got != 0 {
		t.Errorf("a Fúria subiu o ataque de TODA arma em %d, e ela é só corpo a corpo", got)
	}
}
