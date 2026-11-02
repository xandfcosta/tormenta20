package engine

import "testing"

// "Uma criatura pode usar uma arma feita para até uma categoria de tamanho maior
// [...], mas sofre –5 nos testes de ataque" (p106), e "armas aumentadas [...] causam
// um passo a mais de dano" (p143). A Empunhadura Poderosa diminui a penalidade para
// –2 (p125).
func TestAnOversizedWeaponCostsAttackAndGivesADamageStep(t *testing.T) {
	world := BookRuleset(primeBook(t))
	card := func(powers string, oversized bool) WeaponCard {
		t.Helper()
		hero := switchHero(powers, 0, "espada-longa")
		hero.Items[0].Oversized = oversized
		return world.ComputeWeaponCards(hero, nil)[0]
	}

	plain := card(`[]`, false)
	if plain.Damage != "1d8" {
		t.Fatalf("o controle falhou: a espada longa rola %q, want 1d8", plain.Damage)
	}
	marked := card(`[]`, true)
	if got := marked.Attack - plain.Attack; got != -5 {
		t.Errorf("a arma de categoria maior mudou o ataque em %+d, want -5", got)
	}
	// Tabela 3-2 (p143): um passo acima de 1d8 é 1d10.
	if marked.Damage != "1d10" {
		t.Errorf("a arma de categoria maior rola %q, want 1d10", marked.Damage)
	}

	strong := card(`["empunhadura-poderosa"]`, true)
	if got := strong.Attack - plain.Attack; got != -2 {
		t.Errorf("com Empunhadura Poderosa o ataque mudou %+d, want -2", got)
	}
	if strong.Damage != "1d10" {
		t.Errorf("com Empunhadura Poderosa a arma rola %q, want 1d10", strong.Damage)
	}

	// O poder só diminui a penalidade: com a arma comum, ele não soma nada.
	if got := card(`["empunhadura-poderosa"]`, false).Attack - plain.Attack; got != 0 {
		t.Errorf("com a arma comum, Empunhadura Poderosa mudou o ataque em %+d", got)
	}
}

// A marca é do ITEM: duas espadas iguais não dividem a penalidade.
func TestTheOversizedMarkStaysOnItsOwnWeapon(t *testing.T) {
	world := BookRuleset(primeBook(t))
	hero := switchHero(`[]`, 0, "espada-longa")
	second := hero.Items[0]
	second.Oversized = true
	hero.Items = append(hero.Items, second)

	cards := world.ComputeWeaponCards(hero, nil)
	if got := cards[1].Attack - cards[0].Attack; got != -5 {
		t.Errorf("a diferença entre as duas espadas é %+d, want -5", got)
	}
	if cards[0].Damage != "1d8" || cards[1].Damage != "1d10" {
		t.Errorf("as espadas rolam %q e %q, want 1d8 e 1d10", cards[0].Damage, cards[1].Damage)
	}
}
