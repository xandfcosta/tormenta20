package engine

import "testing"

// A INSPIRAÇÃO MARCIAL (bardo, p45): "quando você usa Inspiração, você e seus aliados aplicam o bônus recebido em
// rolagens de dano (além de testes de perícia)". O bônus é o da Inspiração, e só de quem tem a Marcial.
func TestTheMartialInspirationAddsTheInspirationToDamage(t *testing.T) {
	world := BookRuleset(primeBook(t))
	on := map[string]bool{FlagGroupID("inspiracao"): true}
	damage := func(ch Character, active map[string]bool) int {
		return wieldedCards(world.ComputeWeaponCards(ch, active))[0].DamageBonus
	}
	martial := wielding(attributeSumHero("Bardo", 6, `["class.bardo.inspiracao-marcial"]`), "adaga")
	if got := damage(martial, on) - damage(martial, nil); got != 1 {
		t.Errorf("o bardo com a Inspiração Marcial soma o +1 da Inspiração no dano, e somou %+d", got)
	}
	plain := wielding(attributeSumHero("Bardo", 6, `[]`), "adaga")
	if got := damage(plain, on) - damage(plain, nil); got != 0 {
		t.Errorf("o bardo sem a Inspiração Marcial não soma a Inspiração no dano, e somou %+d", got)
	}
}
