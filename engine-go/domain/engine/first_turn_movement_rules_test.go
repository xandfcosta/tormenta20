package engine

import (
	"errors"
	"testing"
)

// "Você ganha uma ação de movimento extra no seu primeiro turno de cada combate"
// (Reflexos de Combate, p129).
func TestTheExtraMovementPaysFirstAndLeavesTheFullAction(t *testing.T) {
	first := TurnBudget{Standard: true, Movement: true, ExtraMovement: true}

	moved, err := first.Spend(ActionMovement)
	if err != nil {
		t.Fatalf("mover foi recusado: %v", err)
	}
	if moved.ExtraMovement || !moved.Standard || !moved.Movement {
		t.Fatalf("o movimento devia sair do extra: %+v", moved)
	}
	// Por isso o extra paga primeiro: quem andou com ele ainda faz a ação completa.
	if _, err := moved.Spend(ActionFull); err != nil {
		t.Errorf("depois de usar o extra, a ação completa foi recusada: %v", err)
	}
}

func TestTheFullActionLeavesTheExtraMovementStanding(t *testing.T) {
	left, err := TurnBudget{Standard: true, Movement: true, ExtraMovement: true}.Spend(ActionFull)
	if err != nil {
		t.Fatalf("a ação completa foi recusada: %v", err)
	}
	if _, err := left.Spend(ActionMovement); err != nil {
		t.Errorf("a ação completa levou o movimento extra junto: %v", err)
	}
}

// Extra, movimento e a padrão trocada (p233): três movimentos, e o quarto recusa.
func TestTheFirstTurnPaysThreeMovements(t *testing.T) {
	left := TurnBudget{Standard: true, Movement: true, ExtraMovement: true}
	for i := 1; i <= 3; i++ {
		next, err := left.Spend(ActionMovement)
		if err != nil {
			t.Fatalf("o movimento %d foi recusado: %v", i, err)
		}
		left = next
	}
	if _, err := left.Spend(ActionMovement); !errors.Is(err, ErrNoActionLeft) {
		t.Errorf("o quarto movimento devia ser recusado, veio %v", err)
	}
}
