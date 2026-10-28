package sheetui

import (
	"testing"

	"t20engine/domain/engine"
	"t20engine/serve/web/ds"
)

// A CONTA FECHA NO TOTAL. As parcelas são as do livro mais uma linha por item,
// e o que os itens somam e não explicam vira "Outros": sem essa linha, quem
// soma o que vê chega a um número que não é o escrito.
func TestTheExpertiseSumAddsUpToTheTotal(t *testing.T) {
	got := expertiseSumTerms(engine.ExpertiseBreakdown{
		Attribute: "dexterity", AttrValue: 3, HalfLevel: 2, Training: 4, Trained: true,
		TrainedBy: []string{"Origem: Soldado"},
		ItemBonus: 0, Total: 9,
		ItemContributions: []engine.BreakdownContribution{
			{Source: "Manto élfico", Amount: 2},
			{Source: "Armadura", Amount: -3, Note: "penalidade"},
		},
	})
	want := []ds.SumTerm{
		{Label: "Destreza", Value: "+3"},
		{Label: "Metade do nível", Value: "+2"},
		{Label: "Treinada", Note: "Origem: Soldado", Value: "+4"},
		{Label: "Manto élfico", Value: "+2"},
		{Label: "Armadura", Note: "penalidade", Value: "-3", Penalty: true},
		{Label: "Outros", Value: "+1"},
	}
	if len(got) != len(want) {
		t.Fatalf("parcelas: %+v\nesperado: %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("parcela %d: %+v, esperado %+v", i, got[i], want[i])
		}
	}
}
