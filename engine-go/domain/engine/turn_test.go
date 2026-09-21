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
	budget := FullTurn()
	budget, err := budget.Spend(ActionStandard)
	if err != nil {
		t.Fatalf("a padrão do turno: %v", err)
	}
	budget, err = budget.Spend(ActionMovement)
	if err != nil {
		t.Fatalf("a de movimento do turno: %v", err)
	}
	if _, err := budget.Spend(ActionStandard); err == nil {
		t.Error("a segunda padrão não existe: o turno tem UMA")
	}

	// A ordem inversa dá no mesmo — "em qualquer ordem".
	other := FullTurn()
	other, _ = other.Spend(ActionMovement)
	if _, err := other.Spend(ActionStandard); err != nil {
		t.Errorf("mover e depois agir é a mesma coisa que agir e depois mover: %v", err)
	}
}

// A TROCA É DE MÃO ÚNICA: a padrão vira movimento, e o inverso não.
func TestTheStandardTradesDownForMovementButNotBack(t *testing.T) {
	budget := FullTurn()
	budget, _ = budget.Spend(ActionMovement) // gasta a de movimento
	budget, err := budget.Spend(ActionMovement)
	if err != nil {
		t.Fatalf("a segunda de movimento sai da padrão (p233): %v", err)
	}
	if _, err := budget.Spend(ActionStandard); err == nil {
		t.Error("a padrão foi trocada: não há terceira ação")
	}

	// O INVERSO NÃO: duas padrão não saem de uma padrão mais um movimento.
	other := FullTurn()
	other, _ = other.Spend(ActionStandard)
	if _, err := other.Spend(ActionStandard); err == nil {
		t.Error("movimento não vira padrão — a troca é de mão única (p233)")
	}
}

// A COMPLETA CUSTA AS DUAS.
func TestTheFullActionGivesUpBoth(t *testing.T) {
	budget, err := FullTurn().Spend(ActionFull)
	if err != nil {
		t.Fatalf("a completa: %v", err)
	}
	if _, err := budget.Spend(ActionMovement); err == nil {
		t.Error("a completa abre mão das DUAS: não sobra movimento")
	}

	// E ela não cabe num turno que já agiu.
	halfway := FullTurn()
	halfway, _ = halfway.Spend(ActionStandard)
	if _, err := halfway.Spend(ActionFull); err == nil {
		t.Error("a completa exige a rodada inteira: não cabe depois da padrão")
	}
}

// LIVRE E REAÇÃO NÃO GASTAM NADA. "Como ações livres, reações tomam tão pouco
// tempo que você pode realizar qualquer quantidade delas" (p233).
func TestFreeActionsAndReactionsCostNothing(t *testing.T) {
	budget := FullTurn()
	budget, _ = budget.Spend(ActionFull) // o turno inteiro já foi
	for _, cost := range []ActionCost{ActionFree, ActionReaction, ActionPassive} {
		after, err := budget.Spend(cost)
		if err != nil {
			t.Errorf("%q não gasta ação e cabe sempre: %v", cost, err)
		}
		if after != budget {
			t.Errorf("%q mudou o turno, e não devia: %+v", cost, after)
		}
	}
}

// AÇÃO DE CUSTO VARIÁVEL não é cobrada aqui — são 27 ativações que dizem
// "varia", e quem decide o custo é a mesa. Cobrar um custo inventado seria pior
// que não cobrar: o poder ficaria indisponível por uma conta que ninguém fez.
func TestAVariableActionIsNotChargedByTheTurn(t *testing.T) {
	budget := FullTurn()
	budget, _ = budget.Spend(ActionFull)
	if _, err := budget.Spend(ActionVaries); err != nil {
		t.Errorf("custo variável não é cobrado pelo turno: %v", err)
	}
}

// PALAVRA DESCONHECIDA RECUSA, e a recusa diz o que sobrou — "sem ação padrão
// neste turno" manda a pessoa esperar o próximo, e "recusado" manda procurar.
func TestTheRefusalSaysWhatIsLeft(t *testing.T) {
	budget := FullTurn()
	budget, _ = budget.Spend(ActionStandard)
	_, err := budget.Spend(ActionStandard)
	if !errors.Is(err, ErrNoActionLeft) {
		t.Fatalf("a recusa tem de ser reconhecível pelo chamador, e veio %v", err)
	}
	if _, err := FullTurn().Spend(ActionCost("dancar")); err == nil {
		t.Error("um custo que o livro não tem precisa recusar")
	}
}

// ── O INSTANTE, que é outra pergunta que o custo (T20 p233) ─────────────────
//
//	"Uma reação acontece em resposta a outra coisa. Como ações livres, reações
//	tomam tão pouco tempo que você pode realizar qualquer quantidade delas. A
//	diferença é que uma ação livre é uma escolha consciente, feita no seu
//	turno. Já uma reação é uma resposta automática, que pode ocorrer mesmo fora
//	do seu turno. Você pode reagir mesmo se não puder realizar ações, como por
//	estar atordoado."

// A REAÇÃO ATRAVESSA OS DOIS PORTÕES, e é o único custo que atravessa.
//
// Ela é a razão de o instante existir separado do custo: o `Spend` já dizia que
// reação não gasta nada, e "não gasta nada" não responde se PODE agora.
func TestAReactionHappensOutOfTurnAndWithoutBeingAbleToAct(t *testing.T) {
	outside := ActionMoment{OnTurn: false, CanAct: false}
	if err := UsableNow(ActionReaction, outside); err != nil {
		t.Errorf("a reação ocorre fora do seu turno e mesmo sem poder agir (p233): %v", err)
	}
}

// A LIVRE É ESCOLHA CONSCIENTE, e é por aí que ela difere da reação: mesma
// conta no turno, instantes diferentes.
func TestAFreeActionIsAConsciousChoiceMadeOnYourTurn(t *testing.T) {
	if err := UsableNow(ActionFree, ActionMoment{OnTurn: true, CanAct: true}); err != nil {
		t.Errorf("a livre cabe na sua vez: %v", err)
	}
	if err := UsableNow(ActionFree, ActionMoment{OnTurn: false, CanAct: true}); !errors.Is(err, ErrNotYourTurn) {
		t.Errorf("a livre é feita NO SEU TURNO (p233), e a recusa veio %v", err)
	}
	// É a manutenção da sustentada: manter é ação livre, e a 0 PV "você cai
	// inconsciente" (p236).
	if err := UsableNow(ActionFree, ActionMoment{OnTurn: true, CanAct: false}); !errors.Is(err, ErrCannotAct) {
		t.Errorf("quem não pode agir não faz ação livre, e a recusa veio %v", err)
	}
}

// AS TRÊS QUE GASTAM TURNO só acontecem na sua vez, e a recusa por INSTANTE é
// reconhecível separada da recusa por CUSTO: "espere a sua vez" e "não sobrou
// ação" mandam a pessoa fazer coisas diferentes.
func TestTheActionsThatSpendTheTurnOnlyHappenOnYourOwn(t *testing.T) {
	for _, cost := range []ActionCost{ActionStandard, ActionMovement, ActionFull} {
		if err := UsableNow(cost, ActionMoment{OnTurn: true, CanAct: true}); err != nil {
			t.Errorf("%q cabe na sua vez: %v", cost, err)
		}
		if err := UsableNow(cost, ActionMoment{OnTurn: false, CanAct: true}); !errors.Is(err, ErrNotYourTurn) {
			t.Errorf("%q fora da vez tinha de recusar por INSTANTE, e veio %v", cost, err)
		}
	}
}

// A PASSIVA E A VARIÁVEL não têm instante para conferir — a primeira não é
// acionada (238 das 411 ativações) e a segunda é negociada com a mesa. É a
// mesma isenção que o `Spend` lhes dá, e pela mesma razão.
func TestThePassiveAndTheVariableHaveNoMomentToCheck(t *testing.T) {
	none := ActionMoment{OnTurn: false, CanAct: false}
	for _, cost := range []ActionCost{ActionPassive, ActionVaries} {
		if err := UsableNow(cost, none); err != nil {
			t.Errorf("%q não tem instante para conferir: %v", cost, err)
		}
	}
}

// PALAVRA DESCONHECIDA RECUSA no instante como recusa no custo — um `action`
// com erro de digitação no catálogo não pode virar "pode sempre".
func TestAnUnknownCostHasNoMomentEither(t *testing.T) {
	if err := UsableNow(ActionCost("dancar"), ActionMoment{OnTurn: true, CanAct: true}); err == nil {
		t.Error("um custo que o livro não tem precisa recusar também no instante")
	}
}
