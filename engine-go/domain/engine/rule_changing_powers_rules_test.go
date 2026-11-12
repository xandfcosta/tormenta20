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

// O Arremesso Potente troca a Destreza pela Força no ataque, e "pode": a ficha fica com a maior (p124). Ele
// também libera o Ataque Poderoso nas armas de arremesso.
func TestPowerfulThrowUsesStrengthAndUnlocksPowerAttack(t *testing.T) {
	world := BookRuleset(primeBook(t))
	strong := world.ComputeWeaponCards(thrower(`["arremesso-potente"]`, 4, 1), nil)[0]
	if strong.Skill != "Pontaria" || strong.Attribute != "strength" {
		t.Errorf("com Força 4 e Destreza 1, ataca com %s e %s", strong.Skill, strong.Attribute)
	}
	nimble := world.ComputeWeaponCards(thrower(`["arremesso-potente"]`, 1, 3), nil)[0]
	if nimble.Attribute != "dexterity" {
		t.Errorf("com Destreza maior, ataca com %s", nimble.Attribute)
	}
	if plain := world.ComputeWeaponCards(thrower(`[]`, 4, 1), nil)[0]; plain.Attribute != "dexterity" {
		t.Errorf("sem o poder, ataca com %s", plain.Attribute)
	}

	on := map[string]bool{FlagGroupID("ataque-poderoso"): true}
	alone := thrower(`["ataque-poderoso"]`, 1, 1)
	if got := world.ComputeWeaponCards(alone, on)[0].DamageBonus - world.ComputeWeaponCards(alone, nil)[0].DamageBonus; got != 0 {
		t.Errorf("sem o Arremesso Potente, o Ataque Poderoso somou %+d na azagaia", got)
	}
	both := thrower(`["ataque-poderoso","arremesso-potente"]`, 1, 1)
	offCard, onCard := world.ComputeWeaponCards(both, nil)[0], world.ComputeWeaponCards(both, on)[0]
	if onCard.DamageBonus-offCard.DamageBonus != 5 || onCard.Attack-offCard.Attack != -2 {
		t.Errorf("com os dois, a azagaia mudou %+d no ataque e %+d no dano",
			onCard.Attack-offCard.Attack, onCard.DamageBonus-offCard.DamageBonus)
	}
}

// O Estilo Desarmado dá 1d6, letal ou não (p128). O desarmado comum é 1d3 não letal (p143).
func TestUnarmedStyleDealsAD6AndCanBeLethal(t *testing.T) {
	world := BookRuleset(primeBook(t))
	unarmed := func(powers string) WeaponCard {
		cards := world.ComputeWeaponCards(switchHero(powers, 0, "espada-longa"), nil)
		return cards[len(cards)-1]
	}
	if plain := unarmed(`[]`); plain.Damage != "1d3" || !plain.NonLethal {
		t.Errorf("o desarmado comum é %s, não letal: %v", plain.Damage, plain.NonLethal)
	}
	if styled := unarmed(`["estilo-desarmado"]`); styled.Damage != "1d6" || styled.NonLethal {
		t.Errorf("com o Estilo Desarmado é %s, não letal: %v", styled.Damage, styled.NonLethal)
	}
}

func caster(powers string, level, wisdom int) Character {
	return Character{
		Level: level, Wisdom: wisdom, ClassPowers: powers,
		Classes: []CharacterClass{{ClassName: "Clérigo", Level: level}},
	}
}

// "Um arcanista de 5º nível com Int 4 e este poder pode gastar até 9 PM em cada magia" (p131). O clérigo é a
// mesma conta, com Sabedoria.
func TestUnlimitedMagicAddsTheKeyAttributeToThePmLimit(t *testing.T) {
	world := BookRuleset(primeBook(t))
	if got := world.ComputeSheet(caster(`[]`, 5, 4), nil).PmLimit.Total; got != 5 {
		t.Errorf("sem o poder, o limite é %d", got)
	}
	hero := caster(`["magia-ilimitada"]`, 5, 4)
	if got := world.ComputeSheet(hero, nil).PmLimit.Total; got != 9 {
		t.Errorf("a ficha diz limite %d, e o livro dá 9", got)
	}
	if got := world.SpellPmLimitFor(hero, nil, []string{"Clérigo"}, SpellAt{Tradition: "divina"}); got != 9 {
		t.Errorf("o conjurar aceita até %d, e o livro dá 9", got)
	}
}

// "Isso dobra seu limite de PM": um de 8º nível lança até 16 (p131). É um interruptor, e fora dele o limite é
// o de sempre.
func TestTheRitualSwitchDoublesThePmLimit(t *testing.T) {
	world := BookRuleset(primeBook(t))
	hero := caster(`["celebrar-ritual"]`, 8, 0)
	on := map[string]bool{FlagGroupID("ritual"): true}
	if got := world.ComputeSheet(hero, nil).PmLimit.Total; got != 8 {
		t.Errorf("desligado, o limite é %d", got)
	}
	sheet := world.ComputeSheet(hero, on)
	if sheet.PmLimit.Total != 16 {
		t.Errorf("como ritual, a ficha diz limite %d, e o livro dá 16", sheet.PmLimit.Total)
	}
	sum := sheet.PmLimit.Base
	for _, part := range sheet.PmLimit.Contributions {
		sum += part.Amount
	}
	if sum != sheet.PmLimit.Total {
		t.Errorf("as parcelas somam %d, e o total é %d", sum, sheet.PmLimit.Total)
	}
	if got := world.SpellPmLimitFor(hero, on, []string{"Clérigo"}, SpellAt{Tradition: "divina"}); got != 16 {
		t.Errorf("como ritual, o conjurar aceita até %d", got)
	}
}
