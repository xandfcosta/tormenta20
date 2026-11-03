package engine

import "testing"

// A arma é uma linha só, e o outro jeito de usá-la mora dentro dela. "Armas com alcance podem ser usadas para ataques
// à distância", e "armas sem alcance podem ser arremessadas em alcance curto com –5" (p143). A azagaia é o caso
// inverso: "pode ser usada como arma corpo a corpo, mas você sofre uma penalidade de –5" (p146).
func otherUseHero(powers, weapon string) Character {
	hero := switchHero(powers, 2, weapon)
	hero.Strength = 4
	hero.Proficiencies = `["armas-simples","armas-marciais"]`
	return hero
}

func TestAWeaponKeepsItsOtherUseInsideItsOwnCard(t *testing.T) {
	world := BookRuleset(primeBook(t))
	// Nível 5 dá +2. Luta com Força 4 dá +6, e Pontaria com Destreza 2 dá +4.
	for _, c := range []struct {
		weapon, skill, otherSkill, otherName string
		attack, otherAttack, penalty         int
	}{
		{"adaga", "Luta", "Pontaria", "adaga (arremesso)", 6, 4, 0},
		{"lanca", "Luta", "Pontaria", "lanca (arremesso)", 6, 4, 0},
		{"machadinha", "Luta", "Pontaria", "machadinha (arremesso)", 6, 4, 0},
		{"tridente", "Luta", "Pontaria", "tridente (arremesso)", 6, 4, 0},
		{"espada-longa", "Luta", "Pontaria", "espada-longa (arremesso)", 6, -1, -5},
		{"azagaia", "Pontaria", "Luta", "azagaia (corpo a corpo)", 4, 1, -5},
	} {
		t.Run(c.weapon, func(t *testing.T) {
			cards := world.ComputeWeaponCards(otherUseHero(`[]`, c.weapon), nil)
			if len(cards) != 2 {
				t.Fatalf("a mão tem %d cartas, e a arma é uma mais o desarmado: %+v", len(cards), cards)
			}
			main, other := cards[0], cards[0].OtherUse
			if main.Skill != c.skill || main.Attack != c.attack {
				t.Errorf("a arma ataca com %s e %+d, want %s e %+d", main.Skill, main.Attack, c.skill, c.attack)
			}
			if other == nil {
				t.Fatal("a arma não tem o outro uso")
			}
			if other.Skill != c.otherSkill || other.Attack != c.otherAttack || other.Name != c.otherName {
				t.Errorf("o outro uso é %q, com %s e %+d; want %q, com %s e %+d",
					other.Name, other.Skill, other.Attack, c.otherName, c.otherSkill, c.otherAttack)
			}
			if other.CatalogID != c.weapon {
				t.Errorf("o outro uso aponta para %q", other.CatalogID)
			}
			// "Quando você ataca com uma arma de arremesso, soma sua Força às rolagens de dano" (p142).
			if main.StrDamage != 4 || other.StrDamage != 4 {
				t.Errorf("a Força soma %+d e %+d, e a ficha tem 4", main.StrDamage, other.StrDamage)
			}
			// A penalidade tem nome na conta, para quem abre saber de onde veio.
			if c.penalty != 0 && (other.AttackOwn.Total != c.penalty || len(other.AttackOwn.Contributions) != 1) {
				t.Errorf("a conta do outro uso não nomeia o %+d: %+v", c.penalty, other.AttackOwn)
			}
		})
	}
}

func TestSomeCardsHaveNoOtherUse(t *testing.T) {
	world := BookRuleset(primeBook(t))
	// O arco dispara um projétil, e a rede é só de arremesso: a p146 só dá o corpo a corpo à azagaia.
	for _, weapon := range []string{"arco-curto", "rede"} {
		cards := world.ComputeWeaponCards(otherUseHero(`[]`, weapon), nil)
		if cards[0].OtherUse != nil {
			t.Errorf("%s ganhou outro uso: %+v", weapon, cards[0].OtherUse)
		}
	}
	cards := world.ComputeWeaponCards(otherUseHero(`[]`, "lanca"), nil)
	if unarmed := cards[len(cards)-1]; unarmed.Name != UnarmedName || unarmed.OtherUse != nil {
		t.Errorf("o desarmado tem outro uso: %+v", unarmed)
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
		hero := otherUseHero(`["`+c.power+`"]`, "lanca")
		off := world.ComputeWeaponCards(hero, nil)[0]
		on := world.ComputeWeaponCards(hero, map[string]bool{FlagGroupID(c.power): true})[0]
		if got := on.DamageBonus - off.DamageBonus; got != c.melee {
			t.Errorf("%s no corpo a corpo: dano %+d, e o livro dá %+d", c.power, got, c.melee)
		}
		if got := on.OtherUse.DamageBonus - off.OtherUse.DamageBonus; got != c.thrown {
			t.Errorf("%s no arremesso: dano %+d, e o livro dá %+d", c.power, got, c.thrown)
		}
	}
}

// A melhoria e a falta de proficiência são da ARMA, e não de um dos usos dela (p142, p164).
func TestWhatBelongsToTheWeaponReachesBothUses(t *testing.T) {
	world := BookRuleset(primeBook(t))
	hero := otherUseHero(`[]`, "lanca")
	hero.Items[0].Improvements = `["melhoria-certeira"]`
	sharp := world.ComputeWeaponCards(hero, nil)[0]
	if sharp.Attack != 7 || sharp.OtherUse.Attack != 5 {
		t.Errorf("a Certeira dá +1: corpo a corpo %+d, want +7; arremesso %+d, want +5", sharp.Attack, sharp.OtherUse.Attack)
	}

	hero = otherUseHero(`[]`, "lanca")
	hero.Proficiencies = `[]`
	hero.Classes = nil
	clumsy := world.ComputeWeaponCards(hero, nil)[0]
	if clumsy.Attack != 1 || clumsy.OtherUse.Attack != -1 {
		t.Errorf("sem proficiência é −5: corpo a corpo %+d, want +1; arremesso %+d, want -1", clumsy.Attack, clumsy.OtherUse.Attack)
	}
}

// As cartas seguem as mãos: o índice do pedido é o da arma, mais o desarmado no fim.
func TestTwoWeaponsInHandAreTwoCards(t *testing.T) {
	world := BookRuleset(primeBook(t))
	hero := wielding(otherUseHero(`[]`, "lanca"), "adaga")
	hero.Items[1].Equipped = ptrTo("wielded2")
	cards := world.ComputeWeaponCards(hero, nil)
	if len(cards) != 3 || cards[0].Name != "lanca" || cards[1].Name != "adaga" || cards[2].Name != UnarmedName {
		t.Fatalf("as cartas são %+v", cards)
	}
}

// O −5 é "com uma arma" que não se sabe usar (p142), e a melhoria é "da arma": nenhum dos dois sai da carta dela.
func TestWhatBelongsToAWeaponStaysOffTheOtherCards(t *testing.T) {
	world := BookRuleset(primeBook(t))
	for _, c := range []struct {
		name                         string
		first, second                string
		improvements, proficiencies  string
		firstAt, secondAt, unarmedAt int
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
			hero := wielding(otherUseHero(`[]`, c.first), c.second)
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

// O arremesso segue os poderes do arremesso (p124), e o que anda o dado anda nos dois usos (p143).
func TestTheThrowFollowsThePowersOfTheThrow(t *testing.T) {
	world := BookRuleset(primeBook(t))
	usesOf := func(hero Character) (melee, thrown WeaponCard) {
		card := world.ComputeWeaponCards(hero, nil)[0]
		return card, *card.OtherUse
	}

	// Acuidade com Arma: "arma corpo a corpo leve ou uma arma de arremesso". A lança não é leve: em corpo a corpo
	// fica com a Força, e arremessada troca ataque e dano pela Destreza.
	finesse := otherUseHero(`["acuidade-com-arma"]`, "lanca")
	finesse.Strength, finesse.Dexterity = 1, 5
	melee, thrown := usesOf(finesse)
	if melee.Attribute != "strength" || melee.StrDamage != 1 {
		t.Errorf("Acuidade na lança corpo a corpo: %s e %+d no dano, want Força e +1", melee.Attribute, melee.StrDamage)
	}
	if thrown.Attribute != "dexterity" || thrown.StrDamage != 5 || !thrown.DexDamage {
		t.Errorf("Acuidade na lança arremessada: %s e %+d no dano, want Destreza e +5", thrown.Attribute, thrown.StrDamage)
	}

	// Arremesso Potente: a Força no teste de ataque, só no arremesso.
	melee, thrown = usesOf(otherUseHero(`["arremesso-potente"]`, "lanca"))
	if thrown.Attribute != "strength" || thrown.Skill != "Pontaria" || thrown.Attack != 6 {
		t.Errorf("Arremesso Potente: %s com %s e %+d, want Pontaria com Força e +6", thrown.Skill, thrown.Attribute, thrown.Attack)
	}
	if melee.Skill != "Luta" {
		t.Errorf("Arremesso Potente mudou o corpo a corpo: %s", melee.Skill)
	}

	// Categoria maior: um passo no dado e −5 no ataque, nos dois usos (p143, p106).
	big := otherUseHero(`[]`, "lanca")
	big.Items[0].Oversized = true
	melee, thrown = usesOf(big)
	if melee.Damage != "1d8" || thrown.Damage != "1d8" || melee.Attack != 1 || thrown.Attack != -1 {
		t.Errorf("categoria maior: %s %+d e %s %+d, want 1d8 +1 e 1d8 -1", melee.Damage, melee.Attack, thrown.Damage, thrown.Attack)
	}
}

// O encanto Arremesso diz que a arma "pode ser arremessada em alcance curto" (p335). Decisão do dono: isso tira o −5 que
// a p143 cobra da arma sem alcance, ou o encanto não daria ao ataque nada que toda arma já não tenha.
func TestTheThrowingEnchantRemovesThePenaltyOfTheThrow(t *testing.T) {
	world := BookRuleset(primeBook(t))
	hero := otherUseHero(`[]`, "espada-longa")
	hero.Items[0].Enchants = []string{"encanto-arremesso"}
	thrown := world.ComputeWeaponCards(hero, nil)[0].OtherUse
	// Nível 5 dá +2, e Pontaria soma a Destreza 2: +4, sem o −5.
	if thrown.Attack != 4 || thrown.AttackOwn.Total != 0 {
		t.Errorf("a espada encantada arremessa com %+d e %+d de penalidade, want +4 e 0", thrown.Attack, thrown.AttackOwn.Total)
	}
}
