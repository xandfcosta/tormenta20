package engine

import "testing"

// O ARTISTA COMPLETO (ALE-647, fatia 2). "Enquanto estiver sob efeito de sua Inspiração, suas habilidades de bardo
// (incluindo magias) têm seu custo em PM reduzido pela metade (após aplicar aprimoramentos e quaisquer outros
// efeitos que reduzam custo)" (p45). A metade é FATOR, e vem depois da soma e antes do piso, como a Alta Arcana.

func TestAnAbilityCostHalvesAfterTheSumAndBeforeTheFloor(t *testing.T) {
	half := Ratio{Num: 1, Den: 2}
	for _, c := range []struct {
		why                  string
		base, increase, want int
		cuts                 []int
	}{
		{"6 PM e o Alquebrado: 7, e a metade 3", 6, 1, 3, nil},
		{"o piso de 1 PM vem depois da metade", 1, 0, 1, nil},
		{"4 PM com –2: 2, e a metade 1", 4, 0, 1, []int{-2}},
		{"o que custa zero continua de graça", 0, 1, 0, nil},
	} {
		if got := AbilityPmCostWith(c.base, c.increase, half, c.cuts...); got != c.want {
			t.Errorf("%s: saiu %d; want %d", c.why, got, c.want)
		}
	}
	if got := AbilityPmCostWith(5, 0, Ratio{}); got != 5 {
		t.Errorf("sem fator, 5 PM saiu por %d", got)
	}
}

func fullArtist() Character {
	return Character{Level: 20, Classes: []CharacterClass{{ClassName: "Bardo", Level: 20}}}
}

func TestTheFullArtistHalvesTheBardAbilitiesUnderInspiration(t *testing.T) {
	world := BookRuleset(primeBook(t))
	inspired := map[string]bool{FlagGroupID("inspiracao"): true}
	if got := world.EffectsOf(fullArtist(), inspired).AbilityCostRules().Factor("Bardo"); got != (Ratio{Num: 1, Den: 2}) {
		t.Errorf("o bardo 20 sob a Inspiração corta as habilidades de bardo por %+v; want a metade", got)
	}
	if got := world.EffectsOf(fullArtist(), nil).AbilityCostRules().Factor("Bardo"); got != (Ratio{}) {
		t.Errorf("sem a Inspiração, o fator é %+v; want nenhum", got)
	}
	if got := world.EffectsOf(fullArtist(), inspired).AbilityCostRules().Factor("Guerreiro"); got != (Ratio{}) {
		t.Errorf("a habilidade de outra classe pegou o fator %+v; want nenhum", got)
	}
}

func TestTheFullArtistHalvesTheSpellsCastAsBard(t *testing.T) {
	world := BookRuleset(primeBook(t))
	inspired := map[string]bool{FlagGroupID("inspiracao"): true}
	asBard := SpellAt{Tradition: "arcana", Classes: []string{"Bardo"}}
	if got := world.SpellPmCostFor(fullArtist(), 6, 0, inspired, asBard); got != 3 {
		t.Errorf("a magia de 6 PM lançada como bardo, sob a Inspiração, saiu por %d; want 3", got)
	}
	if got := world.SpellPmCostFor(fullArtist(), 6, 0, nil, asBard); got != 6 {
		t.Errorf("sem a Inspiração, a magia de 6 PM saiu por %d; want 6", got)
	}
	asOther := SpellAt{Tradition: "arcana", Classes: []string{"Arcanista"}}
	if got := world.SpellPmCostFor(fullArtist(), 6, 0, inspired, asOther); got != 6 {
		t.Errorf("a magia lançada por outra classe saiu por %d; want 6, a metade é das de bardo", got)
	}
}
