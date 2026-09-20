package engine

import (
	"errors"
	"testing"
)

// A ECONOMIA DE AÇÃO DO TURNO (T20 p233), palavra por palavra:
//
//	"No seu turno, você pode fazer uma ação padrão e uma ação de movimento, em
//	qualquer ordem. Você pode trocar sua ação padrão por uma ação de movimento,
//	para fazer duas ações de movimento, mas não pode fazer o inverso. Você
//	também pode abrir mão das duas para fazer uma ação completa."

// UMA PADRÃO E UMA DE MOVIMENTO, em qualquer ordem.
func TestATurnHasOneStandardAndOneMovement(t *testing.T) {
	turno := FullTurn()
	turno, err := turno.Spend(ActionStandard)
	if err != nil {
		t.Fatalf("a padrão do turno: %v", err)
	}
	turno, err = turno.Spend(ActionMovement)
	if err != nil {
		t.Fatalf("a de movimento do turno: %v", err)
	}
	if _, err := turno.Spend(ActionStandard); err == nil {
		t.Error("a segunda padrão não existe: o turno tem UMA")
	}

	// A ordem inversa dá no mesmo — "em qualquer ordem".
	outro := FullTurn()
	outro, _ = outro.Spend(ActionMovement)
	if _, err := outro.Spend(ActionStandard); err != nil {
		t.Errorf("mover e depois agir é a mesma coisa que agir e depois mover: %v", err)
	}
}

// A TROCA É DE MÃO ÚNICA: a padrão vira movimento, e o inverso não.
func TestTheStandardTradesDownForMovementButNotBack(t *testing.T) {
	turno := FullTurn()
	turno, _ = turno.Spend(ActionMovement) // gasta a de movimento
	turno, err := turno.Spend(ActionMovement)
	if err != nil {
		t.Fatalf("a segunda de movimento sai da padrão (p233): %v", err)
	}
	if _, err := turno.Spend(ActionStandard); err == nil {
		t.Error("a padrão foi trocada: não há terceira ação")
	}

	// O INVERSO NÃO: duas padrão não saem de uma padrão mais um movimento.
	outro := FullTurn()
	outro, _ = outro.Spend(ActionStandard)
	if _, err := outro.Spend(ActionStandard); err == nil {
		t.Error("movimento não vira padrão — a troca é de mão única (p233)")
	}
}

// A COMPLETA CUSTA AS DUAS.
func TestTheFullActionGivesUpBoth(t *testing.T) {
	turno, err := FullTurn().Spend(ActionFull)
	if err != nil {
		t.Fatalf("a completa: %v", err)
	}
	if _, err := turno.Spend(ActionMovement); err == nil {
		t.Error("a completa abre mão das DUAS: não sobra movimento")
	}

	// E ela não cabe num turno que já agiu.
	meio := FullTurn()
	meio, _ = meio.Spend(ActionStandard)
	if _, err := meio.Spend(ActionFull); err == nil {
		t.Error("a completa exige a rodada inteira: não cabe depois da padrão")
	}
}

// LIVRE E REAÇÃO NÃO GASTAM NADA. "Como ações livres, reações tomam tão pouco
// tempo que você pode realizar qualquer quantidade delas" (p233).
func TestFreeActionsAndReactionsCostNothing(t *testing.T) {
	turno := FullTurn()
	turno, _ = turno.Spend(ActionFull) // o turno inteiro já foi
	for _, custo := range []ActionCost{ActionFree, ActionReaction, ActionPassive} {
		depois, err := turno.Spend(custo)
		if err != nil {
			t.Errorf("%q não gasta ação e cabe sempre: %v", custo, err)
		}
		if depois != turno {
			t.Errorf("%q mudou o turno, e não devia: %+v", custo, depois)
		}
	}
}

// AÇÃO DE CUSTO VARIÁVEL não é cobrada aqui — são 27 ativações que dizem
// "varia", e quem decide o custo é a mesa. Cobrar um custo inventado seria pior
// que não cobrar: o poder ficaria indisponível por uma conta que ninguém fez.
func TestAVariableActionIsNotChargedByTheTurn(t *testing.T) {
	turno := FullTurn()
	turno, _ = turno.Spend(ActionFull)
	if _, err := turno.Spend(ActionVaries); err != nil {
		t.Errorf("custo variável não é cobrado pelo turno: %v", err)
	}
}

// PALAVRA DESCONHECIDA RECUSA, e a recusa diz o que sobrou — "sem ação padrão
// neste turno" manda a pessoa esperar o próximo, e "recusado" manda procurar.
func TestTheRefusalSaysWhatIsLeft(t *testing.T) {
	turno := FullTurn()
	turno, _ = turno.Spend(ActionStandard)
	_, err := turno.Spend(ActionStandard)
	if !errors.Is(err, ErrNoActionLeft) {
		t.Fatalf("a recusa tem de ser reconhecível pelo chamador, e veio %v", err)
	}
	if _, err := FullTurn().Spend(ActionCost("dancar")); err == nil {
		t.Error("um custo que o livro não tem precisa recusar")
	}
}
