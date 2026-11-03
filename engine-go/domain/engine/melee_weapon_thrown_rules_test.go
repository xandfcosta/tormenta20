package engine

import "testing"

// "Armas com alcance podem ser usadas para ataques à distância" (p143), e a Tabela 3-3 (p144) põe Adaga, Lança,
// Machadinha e Tridente em Corpo a Corpo com alcance Curto: são dois usos da mesma arma, e cada um é uma carta.
func throwableHero(powers, weapon string) Character {
	hero := switchHero(powers, 2, weapon)
	hero.Strength = 4
	hero.Proficiencies = `["armas-simples","armas-marciais"]`
	return hero
}

func TestAMeleeWeaponWithRangeHasOneCardForEachUse(t *testing.T) {
	world := BookRuleset(primeBook(t))
	for _, weapon := range []string{"adaga", "lanca", "machadinha", "tridente"} {
		t.Run(weapon, func(t *testing.T) {
			cards := world.ComputeWeaponCards(throwableHero(`[]`, weapon), nil)
			if len(cards) != 3 {
				t.Fatalf("a mão tem %d cartas, e a arma pede corpo a corpo, arremesso e o desarmado: %+v", len(cards), cards)
			}
			melee, thrown := cards[0], cards[1]
			// Nível 5 dá +2. Luta com Força 4 dá +6, e Pontaria com Destreza 2 dá +4.
			if melee.Skill != "Luta" || melee.Attribute != "strength" || melee.Attack != 6 || !melee.IsMelee() {
				t.Errorf("corpo a corpo: %s, %s, %+d, corpo a corpo %v", melee.Skill, melee.Attribute, melee.Attack, melee.IsMelee())
			}
			if thrown.Skill != "Pontaria" || thrown.Attribute != "dexterity" || thrown.Attack != 4 || thrown.IsMelee() {
				t.Errorf("arremesso: %s, %s, %+d, corpo a corpo %v", thrown.Skill, thrown.Attribute, thrown.Attack, thrown.IsMelee())
			}
			// "Quando você ataca com uma arma de arremesso, soma sua Força às rolagens de dano" (p142).
			if melee.StrDamage != 4 || thrown.StrDamage != 4 {
				t.Errorf("a Força soma %+d e %+d, e a ficha tem 4", melee.StrDamage, thrown.StrDamage)
			}
			if thrown.Name != weapon+" (arremesso)" || thrown.CatalogID != weapon {
				t.Errorf("a carta de arremesso se chama %q e aponta para %q", thrown.Name, thrown.CatalogID)
			}
		})
	}
}

func TestOnlyTheMeleeWeaponWithRangeHasTwoUses(t *testing.T) {
	world := BookRuleset(primeBook(t))
	for _, c := range []struct {
		weapon, skill string
	}{
		{"azagaia", "Pontaria"},    // Ataque à Distância na Tabela 3-3 (p144).
		{"espada-longa", "Luta"},   // Corpo a corpo, sem alcance.
		{"arco-curto", "Pontaria"}, // Disparo não se arremessa.
	} {
		cards := world.ComputeWeaponCards(throwableHero(`[]`, c.weapon), nil)
		if len(cards) != 2 || cards[0].Skill != c.skill {
			t.Errorf("%s: %d cartas, a primeira com %s; o livro dá uma arma com %s e o desarmado", c.weapon, len(cards), cards[0].Skill, c.skill)
		}
	}
}

// Cada uso recebe os poderes do uso dele: Ataque Poderoso é corpo a corpo (p124), Estilo de Arremesso é de arremesso (p125).
func TestEachUseOfTheWeaponTakesThePowersOfThatUse(t *testing.T) {
	world := BookRuleset(primeBook(t))
	for _, c := range []struct {
		power         string
		melee, thrown int
	}{
		{"ataque-poderoso", 5, 0},
		{"estilo-de-arremesso", 0, 2},
	} {
		hero := throwableHero(`["`+c.power+`"]`, "lanca")
		off := world.ComputeWeaponCards(hero, nil)
		on := world.ComputeWeaponCards(hero, map[string]bool{FlagGroupID(c.power): true})
		if got := on[0].DamageBonus - off[0].DamageBonus; got != c.melee {
			t.Errorf("%s no corpo a corpo: dano %+d, e o livro dá %+d", c.power, got, c.melee)
		}
		if got := on[1].DamageBonus - off[1].DamageBonus; got != c.thrown {
			t.Errorf("%s no arremesso: dano %+d, e o livro dá %+d", c.power, got, c.thrown)
		}
	}
}

// A melhoria e a falta de proficiência são da ARMA, e não de um dos usos dela (p142, p164).
func TestWhatBelongsToTheWeaponReachesBothUses(t *testing.T) {
	world := BookRuleset(primeBook(t))
	hero := throwableHero(`[]`, "lanca")
	hero.Items[0].Improvements = `["melhoria-certeira"]`
	sharp := world.ComputeWeaponCards(hero, nil)
	if sharp[0].Attack != 7 || sharp[1].Attack != 5 {
		t.Errorf("a Certeira dá +1: corpo a corpo %+d, want +7; arremesso %+d, want +5", sharp[0].Attack, sharp[1].Attack)
	}

	hero = throwableHero(`[]`, "lanca")
	hero.Proficiencies = `[]`
	hero.Classes = nil
	clumsy := world.ComputeWeaponCards(hero, nil)
	if clumsy[0].Attack != 1 || clumsy[1].Attack != -1 {
		t.Errorf("sem proficiência é −5: corpo a corpo %+d, want +1; arremesso %+d, want -1", clumsy[0].Attack, clumsy[1].Attack)
	}
}

// As duas mãos continuam sendo duas armas, por mais cartas que cada uma tenha.
func TestTwoThrowableWeaponsInHandKeepBothUses(t *testing.T) {
	world := BookRuleset(primeBook(t))
	hero := wielding(throwableHero(`[]`, "lanca"), "adaga")
	hero.Items[1].Equipped = ptrTo("wielded2")
	cards := world.ComputeWeaponCards(hero, nil)
	want := []string{"lanca", "lanca (arremesso)", "adaga", "adaga (arremesso)", ""}
	if len(cards) != len(want) {
		t.Fatalf("%d cartas, want %d: %+v", len(cards), len(want), cards)
	}
	for i, name := range want[:4] {
		if cards[i].Name != name {
			t.Errorf("carta %d: %q, want %q", i, cards[i].Name, name)
		}
	}
}

// O −5 é "com uma arma" que não se sabe usar (p142), e a melhoria é "da arma": nenhum dos dois sai da carta dela.
func TestWhatBelongsToAWeaponStaysOffTheOtherCards(t *testing.T) {
	world := BookRuleset(primeBook(t))
	for _, c := range []struct {
		name              string
		first, second     string
		improvements      string
		proficiencies     string
		firstAt, secondAt int
		unarmedAt         int
	}{
		// Nível 5 dá +2, e Força 4 leva Luta a +6. O arco usa Pontaria, com Destreza 2: +4.
		{name: "a Certeira da espada não sobe a clava", first: "espada-longa", second: "clava",
			improvements: `["melhoria-certeira"]`, proficiencies: `["armas-simples","armas-marciais"]`,
			firstAt: 7, secondAt: 6, unarmedAt: 6},
		{name: "a Certeira da adaga não sobe o arco", first: "adaga", second: "arco-curto",
			improvements: `["melhoria-certeira"]`, proficiencies: `["armas-simples","armas-marciais"]`,
			firstAt: 7, secondAt: 4, unarmedAt: 6},
		{name: "o −5 da espada sem proficiência não derruba a clava", first: "espada-longa", second: "clava",
			improvements: `[]`, proficiencies: `["armas-simples"]`,
			firstAt: 1, secondAt: 6, unarmedAt: 6},
	} {
		t.Run(c.name, func(t *testing.T) {
			hero := wielding(throwableHero(`[]`, c.first), c.second)
			hero.Items[1].Equipped = ptrTo("wielded2")
			hero.Items[0].Improvements = c.improvements
			hero.Proficiencies = c.proficiencies
			got := map[string]int{}
			for _, card := range world.ComputeWeaponCards(hero, nil) {
				got[card.Name] = card.Attack
			}
			if got[c.first] != c.firstAt || got[c.second] != c.secondAt || got[UnarmedName] != c.unarmedAt {
				t.Errorf("%s %+d (want %+d), %s %+d (want %+d), desarmado %+d (want %+d)",
					c.first, got[c.first], c.firstAt, c.second, got[c.second], c.secondAt, got[UnarmedName], c.unarmedAt)
			}
		})
	}
}

// A carta de arremesso segue os poderes do arremesso (p124), e o que anda o dado anda nos dois usos (p143).
func TestTheThrownCardFollowsThePowersOfTheThrow(t *testing.T) {
	world := BookRuleset(primeBook(t))
	thrownOf := func(hero Character) (melee, thrown WeaponCard) {
		cards := world.ComputeWeaponCards(hero, nil)
		return cards[0], cards[1]
	}

	// Acuidade com Arma: "arma corpo a corpo leve ou uma arma de arremesso". A lança só é leve para ninguém: em
	// corpo a corpo fica com a Força, e arremessada troca ataque e dano pela Destreza.
	finesse := throwableHero(`["acuidade-com-arma"]`, "lanca")
	finesse.Strength, finesse.Dexterity = 1, 5
	melee, thrown := thrownOf(finesse)
	if melee.Attribute != "strength" || melee.StrDamage != 1 {
		t.Errorf("Acuidade na lança corpo a corpo: %s e %+d no dano, want Força e +1", melee.Attribute, melee.StrDamage)
	}
	if thrown.Attribute != "dexterity" || thrown.StrDamage != 5 || !thrown.DexDamage {
		t.Errorf("Acuidade na lança arremessada: %s e %+d no dano, want Destreza e +5", thrown.Attribute, thrown.StrDamage)
	}

	// Arremesso Potente: a Força no teste de ataque, só no arremesso.
	potent := throwableHero(`["arremesso-potente"]`, "lanca")
	melee, thrown = thrownOf(potent)
	if thrown.Attribute != "strength" || thrown.Skill != "Pontaria" || thrown.Attack != 6 {
		t.Errorf("Arremesso Potente: %s com %s e %+d, want Pontaria com Força e +6", thrown.Skill, thrown.Attribute, thrown.Attack)
	}
	if melee.Skill != "Luta" {
		t.Errorf("Arremesso Potente mudou o corpo a corpo: %s", melee.Skill)
	}

	// Categoria maior: um passo no dado e −5 no ataque, nos dois usos (p143, p106).
	big := throwableHero(`[]`, "lanca")
	big.Items[0].Oversized = true
	melee, thrown = thrownOf(big)
	if melee.Damage != "1d8" || thrown.Damage != "1d8" || melee.Attack != 1 || thrown.Attack != -1 {
		t.Errorf("categoria maior: %s %+d e %s %+d, want 1d8 +1 e 1d8 -1", melee.Damage, melee.Attack, thrown.Damage, thrown.Attack)
	}
}
