package engine

import "testing"

// A ARMA FEITA POR QUEM A USA (ALE-649). O Ferreiro: "quando usa uma arma corpo a corpo feita por você mesmo, o
// dano dela aumenta em um passo"; o Cano Raiado: "quando usa uma arma de disparo feita por você mesmo, ela recebe
// +1 na margem de ameaça" (p69). Quem diz que a arma é dela é a marca "feita por mim" da folha do item (decisão
// do dono).

func TestTheSmithAndTheRifledBarrelReadTheMark(t *testing.T) {
	world := BookRuleset(primeBook(t))
	card := func(powers, weapon string, made bool) WeaponCard {
		ch := attributeSumHero("Inventor", 7, powers)
		ch.Proficiencies = `["armas-simples","armas-marciais"]`
		slot := "wielded"
		ch.Items = append(ch.Items, CharacterItem{ID: 1, CatalogID: &weapon, Name: weapon, Quantity: 1, Equipped: &slot, MadeByOwner: made})
		return cardOf(t, world.ComputeWeaponCards(ch, nil), weapon)
	}
	smith := `["class.inventor.armeiro","class.inventor.ferreiro"]`
	if got, was := card(smith, "espada-longa", true).Damage, card(smith, "espada-longa", false).Damage; got != DamageStep(was, 1) {
		t.Errorf("a espada feita pelo Ferreiro causa %s; want um passo acima de %s", got, was)
	}
	if got, was := card(`[]`, "espada-longa", true).Damage, card(`[]`, "espada-longa", false).Damage; got != was {
		t.Errorf("a espada marcada sem o Ferreiro causa %s; want %s, a marca sozinha não dá nada", got, was)
	}
	barrel := `["class.inventor.balistica","class.inventor.cano-raiado"]`
	if got, was := card(barrel, "arco-longo", true).CritRange, card(barrel, "arco-longo", false).CritRange; got != was-1 {
		t.Errorf("o arco feito pelo dono do Cano Raiado ameaça a partir de %d; want %d", got, was-1)
	}
	if got, was := card(barrel, "espada-longa", true).CritRange, card(barrel, "espada-longa", false).CritRange; got != was {
		t.Errorf("o Cano Raiado mexeu na margem da espada (%d, era %d): ele é da arma de disparo", got, was)
	}
}
