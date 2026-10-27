package book

import "testing"

// ROLAR DE NOVO o teste de perícia (p226, ALE-507): o gasto vem DEPOIS de ver o
// dado, e não entra nas ofertas de antes de rolar.

func TestTheRerollIsOfferedOnlyAfterTheRollAndOnlyWhereItApplies(t *testing.T) {
	owned := spendsOf(t, "class.cavaleiro.etiqueta", "sortudo", "class.cavaleiro.resoluto", "class.bucaneiro.sorte-de-nimb")

	if before := SkillSpendOffers(owned, SkillSpendTarget{Skill: "Diplomacia"}); len(before) != 0 {
		t.Errorf("o rolar de novo foi oferecido ANTES de rolar: %+v", before)
	}
	keys := func(offers []SkillSpendOffer) map[string]SkillSpendOffer {
		out := map[string]SkillSpendOffer{}
		for _, o := range offers {
			out[o.Key] = o
		}
		return out
	}
	diplomacy := keys(RerollOffers(owned, SkillSpendTarget{Skill: "Diplomacia"}))
	if _, ok := diplomacy["etiqueta"]; !ok || diplomacy["etiqueta"].PM != 1 {
		t.Errorf("Etiqueta em Diplomacia (p54, 1 PM): %+v", diplomacy)
	}
	if _, ok := diplomacy["sortudo"]; !ok || diplomacy["sortudo"].PM != 3 {
		t.Errorf("Sortudo em qualquer teste (p131, 3 PM): %+v", diplomacy)
	}
	if _, ok := diplomacy["resoluto"]; ok {
		t.Error("o Resoluto é de teste de resistência, e foi oferecido em Diplomacia")
	}
	will := keys(RerollOffers(owned, SkillSpendTarget{Skill: "Vontade"}))
	if r, ok := will["resoluto"]; !ok || r.Bonus != 5 {
		t.Errorf("Resoluto em Vontade (p55): o segundo teste recebe +5, veio %+v", will)
	}
	if _, ok := will["etiqueta"]; ok {
		t.Error("a Etiqueta foi oferecida em Vontade")
	}
	if n := will["sorte_de_nimb"]; n.NaturalFrom != 11 || n.PM != 5 {
		t.Errorf("Sorte de Nimb (p48): 5 PM e 11+ conta como 20 natural, veio %+v", n)
	}
}
