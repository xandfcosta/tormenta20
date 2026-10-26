package book

import (
	"slices"
	"testing"
)

// O PM NO TESTE DE PERÍCIA (ALE-489), verbete a verbete, contra o livro.

func spendsOf(t *testing.T, ids ...string) []Activation {
	t.Helper()
	out := []Activation{}
	for _, id := range ids {
		a := ActivationOf(id, "")
		if a == nil || a.SkillSpend == nil {
			t.Fatalf("%s não tem gasto de teste no catálogo", id)
		}
		out = append(out, *a)
	}
	return out
}

func offerOf(offers []SkillSpendOffer, id string) (SkillSpendOffer, bool) {
	i := slices.IndexFunc(offers, func(o SkillSpendOffer) bool { return o.ID == id })
	if i < 0 {
		return SkillSpendOffer{}, false
	}
	return offers[i], true
}

var charismaThree = map[string]int{"charisma": 3, "intelligence": 4}

// p47 Audácia: "gastar 2 PM para somar seu Carisma no teste"; p79 Orgulho:
// "uma quantidade de PM a sua escolha (limitado pelo seu Carisma). Para cada PM
// que gastar, recebe +2 no teste." Os dois SOMAM (decisão do dono).
func TestAudacityAndPrideAddUp(t *testing.T) {
	offers := SkillSpendOffers(spendsOf(t, "class.bucaneiro.audacia", "class.nobre.orgulho"),
		SkillSpendTarget{Skill: "Diplomacia", Attributes: charismaThree})
	paid, err := PaySkillSpends(offers, []SkillSpendChoice{
		{Key: "audacia"}, {Key: "orgulho", PM: 2},
	})
	if err != nil {
		t.Fatalf("Audácia + Orgulho 2: %v", err)
	}
	if paid.Bonus != 3+4 || paid.PM != 2+2 {
		t.Errorf("Audácia (Car 3) + Orgulho com 2 PM: +%d por %d PM, want +7 por 4 PM", paid.Bonus, paid.PM)
	}
	if _, err := PaySkillSpends(offers, []SkillSpendChoice{{Key: "orgulho", PM: 4}}); err == nil {
		t.Error("Orgulho com 4 PM e Carisma 3 passou — o limite é o Carisma")
	}
	if offers := SkillSpendOffers(spendsOf(t, "class.nobre.orgulho"),
		SkillSpendTarget{Skill: "Diplomacia", Attributes: map[string]int{"charisma": 0}}); len(offers) != 0 {
		t.Errorf("Orgulho com Carisma 0 foi oferecido: %+v", offers)
	}
}

// p28 kliren: "Caso receba esta habilidade novamente, seu custo é reduzido em
// –1 PM." Um kliren inventor tem as duas Engenhosidades: UMA oferta, por 1 PM.
func TestTheTwiceGivenIngenuityCostsOne(t *testing.T) {
	offers := SkillSpendOffers(spendsOf(t, "race.kliren.engenhosidade", "class.inventor.engenhosidade"),
		SkillSpendTarget{Skill: "Ofício", Attributes: charismaThree})
	if len(offers) != 1 || offers[0].PM != 1 || offers[0].Bonus != 4 {
		t.Errorf("kliren inventor: %+v, want uma Engenhosidade de +4 (Int) por 1 PM", offers)
	}
}

// p74 A Pessoa Certa: "+10 no teste" numa perícia da lista de ladino, por 5 PM.
func TestTheRightPersonOnlyOnTheRogueList(t *testing.T) {
	right := spendsOf(t, "class.ladino.a-pessoa-certa-para-o-trabalho")
	if o, ok := offerOf(SkillSpendOffers(right, SkillSpendTarget{Skill: "Ladinagem"}), right[0].ID); !ok || o.Bonus != 10 || o.PM != 5 {
		t.Errorf("Ladinagem: %+v, want +10 por 5 PM", o)
	}
	if _, ok := offerOf(SkillSpendOffers(right, SkillSpendTarget{Skill: "Religião"}), right[0].ID); ok {
		t.Error("A Pessoa Certa foi oferecida em Religião, que não é da lista de ladino (p72)")
	}
}

// p45 Eclético: "1 PM para receber os benefícios de ser treinado nessa perícia
// para este teste". p90 Cultura Exótica: "1 PM para ignorar esse requerimento".
func TestPaidTrainingUnlocksAndEclecticAddsTheTrainingBonus(t *testing.T) {
	both := spendsOf(t, "class.bardo.ecletico", "origin.forasteiro.cultura-exotica")
	locked := SkillSpendTarget{Skill: "Misticismo", TrainedOnly: true, TrainingBonus: 4}
	eclectic, ok := offerOf(SkillSpendOffers(both, locked), "class.bardo.ecletico")
	if !ok || eclectic.Bonus != 4 || !eclectic.Unlocks || eclectic.PM != 1 {
		t.Errorf("Eclético em Misticismo sem treino, nível 7: %+v, want +4, destrava, 1 PM", eclectic)
	}
	exotic, ok := offerOf(SkillSpendOffers(both, locked), "origin.forasteiro.cultura-exotica")
	if !ok || exotic.Bonus != 0 || !exotic.Unlocks {
		t.Errorf("Cultura Exótica em Misticismo sem treino: %+v, want destrava sem bônus", exotic)
	}
	// Na perícia treinada, nenhum dos dois tem o que fazer.
	if offers := SkillSpendOffers(both, SkillSpendTarget{Skill: "Misticismo", Trained: true, TrainedOnly: true}); len(offers) != 0 {
		t.Errorf("perícia treinada recebeu ofertas de treino: %+v", offers)
	}
	// A Cultura Exótica só serve onde o livro exige treino.
	if _, ok := offerOf(SkillSpendOffers(both, SkillSpendTarget{Skill: "Atletismo"}), "origin.forasteiro.cultura-exotica"); ok {
		t.Error("Cultura Exótica oferecida em Atletismo, que não exige treino")
	}
}

func TestASpendTheSheetDoesNotHaveIsRefused(t *testing.T) {
	offers := SkillSpendOffers(spendsOf(t, "class.bucaneiro.audacia"), SkillSpendTarget{Skill: "Diplomacia", Attributes: charismaThree})
	if _, err := PaySkillSpends(offers, []SkillSpendChoice{{Key: "orgulho", PM: 1}}); err == nil {
		t.Error("o Orgulho de quem não o tem passou")
	}
}
