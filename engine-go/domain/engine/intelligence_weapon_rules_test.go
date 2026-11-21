package engine

import "testing"

// A INTELIGÊNCIA NO LUGAR DO ATRIBUTO DA ARMA (ALE-649). O Armeiro: "quando empunha uma arma corpo a corpo, pode
// usar sua Inteligência em vez de Força nos testes de ataque e rolagens de dano" (p68). A Balística: "quando usa
// uma arma de ataque à distância, pode usar sua Inteligência em vez de Destreza nos testes de ataque" (p69). Os
// dois dizem "pode", e a carta fica com o maior, como na Acuidade.

func inventor(powers string, str, dex, intel int, weapon string) Character {
	ch := wielding(attributeSumHero("Inventor", 5, powers), weapon)
	ch.Strength, ch.Dexterity, ch.Intelligence = str, dex, intel
	ch.Proficiencies = `["armas-simples","armas-marciais"]`
	return ch
}

func TestTheArmorerAttacksAndHitsWithIntelligence(t *testing.T) {
	world := BookRuleset(primeBook(t))
	sword := func(powers string, str int) WeaponCard {
		return cardOf(t, world.ComputeWeaponCards(inventor(powers, str, 0, 4, "espada-longa"), nil), "espada-longa")
	}
	armorer, plain := sword(`["class.inventor.armeiro"]`, 1), sword(`[]`, 1)
	if armorer.Attribute != "intelligence" || armorer.Attack-plain.Attack != 3 {
		t.Errorf("o Armeiro com Int 4 e For 1 ataca com %s, %d acima; want intelligence, +3", armorer.Attribute, armorer.Attack-plain.Attack)
	}
	if armorer.StrDamage != 4 || armorer.IntDamageNote == "" {
		t.Errorf("o Armeiro soma %d no dano (Inteligência: %q); want a Inteligência, 4", armorer.StrDamage, armorer.IntDamageNote)
	}
	// "PODE": com a Força maior, fica a Força.
	if strong := sword(`["class.inventor.armeiro"]`, 5); strong.Attribute != "strength" || strong.StrDamage != 5 {
		t.Errorf("o Armeiro com For 5 e Int 4 ataca com %s e soma %d no dano; want a Força, 5", strong.Attribute, strong.StrDamage)
	}
	// Só corpo a corpo: o arco segue com a Destreza.
	bow := cardOf(t, world.ComputeWeaponCards(inventor(`["class.inventor.armeiro"]`, 1, 0, 4, "arco-longo"), nil), "arco-longo")
	if bow.Attribute != "dexterity" {
		t.Errorf("o Armeiro atira com %s; want a Destreza, ele é corpo a corpo", bow.Attribute)
	}
}

func TestTheBallisticsAimsWithIntelligence(t *testing.T) {
	world := BookRuleset(primeBook(t))
	for _, c := range []struct {
		weapon, want string
		dmg          int
	}{
		{"arco-longo", "intelligence", 0},
		{"azagaia", "intelligence", 1},
		{"espada-longa", "strength", 1},
	} {
		card := thrownCardOf(t, world.ComputeWeaponCards(inventor(`["class.inventor.balistica"]`, 1, 0, 4, c.weapon), nil), c.weapon)
		if c.weapon == "espada-longa" {
			card = cardOf(t, world.ComputeWeaponCards(inventor(`["class.inventor.balistica"]`, 1, 0, 4, c.weapon), nil), c.weapon)
		}
		if card.Attribute != c.want {
			t.Errorf("a Balística com %s ataca com %s; want %s", c.weapon, card.Attribute, c.want)
		}
		// O dano não muda: a Balística troca a Destreza no TESTE, e o arremesso soma a Força no dano (p142).
		if card.StrDamage != c.dmg {
			t.Errorf("a Balística com %s soma %d no dano; want %d", c.weapon, card.StrDamage, c.dmg)
		}
	}
}

// A TROCA É UMA LICENÇA DO CATÁLOGO (ALE-670). O Armeiro e a Balística declaram a troca do atributo da arma — o
// alcance, a conta e a frase —, e a carta leva a frase da licença: o motor não conhece nenhum dos dois pelo nome,
// e a tela não deduz de qual atributo saiu a Inteligência pelo nome do poder.
func TestTheWeaponAttributeSwapCarriesTheLicensePhrase(t *testing.T) {
	world := BookRuleset(primeBook(t))
	sword := cardOf(t, world.ComputeWeaponCards(inventor(`["class.inventor.armeiro"]`, 1, 0, 4, "espada-longa"), nil), "espada-longa")
	if sword.IntNote != "Armeiro: Inteligência no lugar de Força" || sword.IntDamageNote != sword.IntNote {
		t.Errorf("o Armeiro diz %q no ataque e %q no dano; want a frase da licença nos dois", sword.IntNote, sword.IntDamageNote)
	}
	bow := cardOf(t, world.ComputeWeaponCards(inventor(`["class.inventor.balistica"]`, 1, 0, 4, "arco-longo"), nil), "arco-longo")
	if bow.IntNote != "Balística: Inteligência no lugar de Destreza" || bow.IntDamageNote != "" {
		t.Errorf("a Balística diz %q no ataque e %q no dano; want a frase da licença só no ataque", bow.IntNote, bow.IntDamageNote)
	}
	// Sem troca, sem frase: a Força maior fica.
	strong := cardOf(t, world.ComputeWeaponCards(inventor(`["class.inventor.armeiro"]`, 5, 0, 4, "espada-longa"), nil), "espada-longa")
	if strong.IntNote != "" || strong.IntDamageNote != "" {
		t.Errorf("o Armeiro com a Força maior ainda diz %q e %q", strong.IntNote, strong.IntDamageNote)
	}
}
