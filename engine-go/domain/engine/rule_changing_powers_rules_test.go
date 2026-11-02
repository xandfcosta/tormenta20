package engine

import "testing"

func thrower(powers string, strength, dexterity int) Character {
	hero := switchHero(powers, dexterity, "azagaia")
	hero.Strength = strength
	return hero
}

// "Para atacar com uma arma de combate à distância, faça um teste de Pontaria", e a arma de arremesso é uma
// delas, somando a Força no dano (p142). A carta a tratava como corpo a corpo.
func TestAThrownWeaponAttacksWithAimAndAddsStrengthToDamage(t *testing.T) {
	world := BookRuleset(primeBook(t))
	card := world.ComputeWeaponCards(thrower(`[]`, 2, 3), nil)[0]
	if card.Skill != "Pontaria" || card.Attribute != "dexterity" || card.IsMelee() {
		t.Errorf("a azagaia ataca com %s e %s (corpo a corpo: %v)", card.Skill, card.Attribute, card.IsMelee())
	}
	if card.StrDamage != 2 {
		t.Errorf("a Força soma %d no dano, e a ficha tem 2", card.StrDamage)
	}
	sword := world.ComputeWeaponCards(switchHero(`[]`, 0, "espada-longa"), nil)[0]
	if sword.Skill != "Luta" || !sword.IsMelee() {
		t.Errorf("a espada passou a atacar com %s", sword.Skill)
	}
	bow := world.ComputeWeaponCards(switchHero(`[]`, 0, "arco-curto"), nil)[0]
	if bow.Skill != "Pontaria" || bow.StrDamage != 0 {
		t.Errorf("o arco ataca com %s e soma %d de Força", bow.Skill, bow.StrDamage)
	}
}

// A Acuidade com Arma põe a Destreza também no dano da arma de arremesso (p124), se ela for maior.
func TestFinesseAddsDexterityToThrownDamage(t *testing.T) {
	world := BookRuleset(primeBook(t))
	card := world.ComputeWeaponCards(thrower(`["acuidade-com-arma"]`, 1, 3), nil)[0]
	if card.StrDamage != 3 || !card.DexDamage {
		t.Errorf("a Acuidade somou %d no dano, e a Destreza é 3", card.StrDamage)
	}
}
