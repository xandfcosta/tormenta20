package engine

import "testing"

// A ESCARAMUÇA (caçador, p51): "Quando se move 6m ou mais, você recebe +2 na Defesa e Reflexos e +1d8 nas
// rolagens de dano [...] até o início de seu próximo turno. Você não pode usar esta habilidade se estiver vestindo
// armadura pesada."
func TestTheSkirmishObeysTheArmor(t *testing.T) {
	world := BookRuleset(primeBook(t))
	on := map[string]bool{FlagGroupID("escaramuca"): true}
	read := func(ch Character, active map[string]bool) (defense, reflexes int, dice []string) {
		sheet := world.ComputeSheet(ch, active)
		for _, e := range sheet.Expertises {
			if e.Name == "Reflexos" {
				reflexes = e.Total
			}
		}
		for _, card := range wieldedCards(world.ComputeWeaponCards(ch, active)) {
			for _, extra := range card.ExtraDamage {
				dice = append(dice, extra.Dice)
			}
		}
		return sheet.Defense.Total, reflexes, dice
	}
	gain := func(ch Character) (defense, reflexes int, dice []string) {
		d0, r0, _ := read(ch, nil)
		d1, r1, dice := read(ch, on)
		return d1 - d0, r1 - r0, dice
	}

	// REFLEXOS na lista: a ficha só desenha a perícia que o personagem tem.
	withReflexes := func(ch Character) Character {
		ch.Expertises = []CharacterExpertise{{Name: "Reflexos", Attribute: "dexterity"}}
		return ch
	}
	hunter := withReflexes(wielding(attributeSumHero("Caçador", 6, `["class.cacador.escaramuca"]`), "adaga"))
	if d, r, dice := gain(hunter); d != 2 || r != 2 || len(dice) != 1 || dice[0] != "1d8" {
		t.Errorf("a Escaramuça dá +2 na Defesa e em Reflexos e +1d8 no dano, e deu %+d, %+d e %v", d, r, dice)
	}
	if d, r, dice := gain(wearing(hunter, "cota-malha")); d != 0 || r != 0 || len(dice) != 0 {
		t.Errorf("de cota de malha (armadura pesada) a Escaramuça não vale, e deu %+d, %+d e %v", d, r, dice)
	}
}
