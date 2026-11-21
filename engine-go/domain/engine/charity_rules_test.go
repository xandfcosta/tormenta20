package engine

import "testing"

// A CARIDADE (ALE-647, fatia 3). "O custo de suas habilidades de paladino que tenham um aliado como alvo é reduzido
// em –1 PM" (p83). O corte é da CLASSE e só vale no uso cujo alvo é um aliado — quem diz o alvo é o gesto de usar;
// aqui se prende só que a regra existe, e de quem ela é.
func TestTheCharityCutsOneOffThePaladinAbilitiesOnAnAlly(t *testing.T) {
	world := BookRuleset(primeBook(t))
	paladin := Character{Level: 5, Classes: []CharacterClass{{ClassName: "Paladino", Level: 5}},
		ClassPowers: `["class.paladino.virtude-paladinesca-caridade"]`}
	rules := world.EffectsOf(paladin, nil).AbilityCostRules()
	if got := rules.AllyCut("Paladino"); got != -1 {
		t.Errorf("a Caridade corta %d nas habilidades de paladino com aliado como alvo; want -1", got)
	}
	if got := rules.AllyCut("Bardo"); got != 0 {
		t.Errorf("a Caridade cortou %d numa habilidade de bardo; want 0", got)
	}
	plain := paladin
	plain.ClassPowers = `[]`
	if got := world.EffectsOf(plain, nil).AbilityCostRules().AllyCut("Paladino"); got != 0 {
		t.Errorf("o paladino sem a Caridade corta %d; want 0", got)
	}
}
