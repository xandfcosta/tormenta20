package engine

import "testing"

func TestTheInitiativeBonusIsTheExpertiseTotal(t *testing.T) {
	ficha := ComputedSheet{Expertises: []ExpertiseBreakdown{
		{Name: "Atletismo", Total: 12},
		{Name: "Iniciativa", Total: 8},
		{Name: "Percepção", Total: 5},
	}}
	if bonus := InitiativeTotal(ficha); bonus != 8 {
		t.Errorf("o bônus deu %d, e a perícia Iniciativa desta ficha soma 8", bonus)
	}
}

// ZERO é resposta, e não falha: ficha sem classe não tem perícia computada, e
// recusar deixaria o jogador fora da fila por causa de uma ficha incompleta.
func TestASheetWithoutTheExpertiseRollsTheBareD20(t *testing.T) {
	if bonus := InitiativeTotal(ComputedSheet{}); bonus != 0 {
		t.Errorf("a ficha sem perícia nenhuma deu %d, e o d20 pelado vale", bonus)
	}
	semIniciativa := ComputedSheet{Expertises: []ExpertiseBreakdown{{Name: "Atletismo", Total: 12}}}
	if bonus := InitiativeTotal(semIniciativa); bonus != 0 {
		t.Errorf("a ficha sem Iniciativa deu %d", bonus)
	}
}

// A CAIXA conta: o catálogo escreve "Iniciativa", e uma busca que casasse sem
// diferenciar aceitaria duas grafias para o mesmo conceito.
func TestTheLookupIsExactOnTheCatalogSpelling(t *testing.T) {
	ficha := ComputedSheet{Expertises: []ExpertiseBreakdown{{Name: "iniciativa", Total: 8}}}
	if bonus := InitiativeTotal(ficha); bonus != 0 {
		t.Errorf("a grafia minúscula casou e deu %d; o catálogo escreve \"Iniciativa\"", bonus)
	}
}
