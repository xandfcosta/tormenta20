package book

import "testing"

// A SORTE SALVADORA (hynne, p28): "Quando faz um teste de resistência, você pode gastar 1 PM para rolar este teste
// novamente." É o rolar de novo do teste, a forma do Resoluto, em Fortitude, Reflexos e Vontade — e só nelas.
func TestTheSavingLuckRerollsTheResistanceTestForOneMana(t *testing.T) {
	owned := spendsOf(t, "race.hynne.sorte-salvadora")
	for _, skill := range []string{"Fortitude", "Reflexos", "Vontade"} {
		offers := RerollOffers(owned, SkillSpendTarget{Skill: skill})
		if len(offers) != 1 || offers[0].PM != 1 || offers[0].Bonus != 0 {
			t.Errorf("a Sorte Salvadora em %s é rolar de novo por 1 PM, sem bônus, e veio %+v", skill, offers)
		}
	}
	if offers := RerollOffers(owned, SkillSpendTarget{Skill: "Atletismo"}); len(offers) != 0 {
		t.Errorf("a Sorte Salvadora é de teste de resistência, e foi oferecida em Atletismo: %+v", offers)
	}
}
