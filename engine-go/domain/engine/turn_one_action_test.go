package engine

import (
	"errors"
	"testing"
)

// ENJOADO. "O personagem só pode realizar uma ação padrão ou de movimento (não
// ambas) por rodada. Ele pode gastar uma ação padrão para fazer uma investida"
// (p394). A primeira ação leva o turno inteiro, e a completa não cabe.
func TestTheSickenedTurnHasOneAction(t *testing.T) {
	sick := TurnBudget{Standard: true, Movement: true, OneAction: true}
	for _, first := range []ActionCost{ActionStandard, ActionMovement, ActionCharge} {
		left, err := sick.Spend(first)
		if err != nil {
			t.Fatalf("enjoado, a primeira ação (%s) tinha de caber, e veio %v", first, err)
		}
		for _, second := range []ActionCost{ActionStandard, ActionMovement} {
			if _, err := left.Spend(second); !errors.Is(err, ErrNoActionLeft) {
				t.Errorf("enjoado, depois de %s, %s tinha de ser recusada, e veio %v", first, second, err)
			}
		}
	}
	if _, err := sick.Spend(ActionFull); !errors.Is(err, ErrNoActionLeft) {
		t.Errorf("enjoado não faz ação completa, que exige as duas, e veio %v", err)
	}
	if _, err := sick.Spend(ActionFree); err != nil {
		t.Errorf("a ação livre não sai do turno, nem do enjoado (p233), e veio %v", err)
	}
}

// A INVESTIDA é ação completa (p235) para quem não está enjoado: o turno inteiro,
// e não cabe depois de andar.
func TestTheChargeIsAFullActionForEveryoneElse(t *testing.T) {
	left, err := FullTurn().Spend(ActionCharge)
	if err != nil || left.Standard || left.Movement {
		t.Errorf("a investida leva o turno inteiro (p235): sobrou %+v, erro %v", left, err)
	}
	moved, _ := FullTurn().Spend(ActionMovement)
	if _, err := moved.Spend(ActionCharge); !errors.Is(err, ErrNoActionLeft) {
		t.Errorf("quem já andou não investe (p235), e veio %v", err)
	}
	if got := StrikeAction(ChargePlain); got != ActionCharge {
		t.Errorf("o golpe da investida custa %q, want %q", got, ActionCharge)
	}
	if err := UsableNow(ActionCharge, ActionMoment{OnTurn: true, CanAct: true, CanReact: true}); err != nil {
		t.Errorf("na vez e de pé, a investida é a hora, e veio %v", err)
	}
}
