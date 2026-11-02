package api

import (
	"context"
	"testing"

	"t20engine/domain/engine"
)

// "Você ganha uma ação de movimento extra no seu primeiro turno de cada combate"
// (Reflexos de Combate, p129).
func movementsOnTurn(t *testing.T, f sceneFixture) int {
	t.Helper()
	paid := 0
	for f.s.sessions.ActionFits(context.Background(), f.sessionID, engine.ActionMovement) == nil {
		if _, err := f.s.sessions.SpendAction(context.Background(), f.sessionID, engine.ActionMovement); err != nil {
			t.Fatalf("cobrar o movimento: %v", err)
		}
		paid++
		if paid > 5 {
			t.Fatal("o turno pagou movimento sem parar")
		}
	}
	return paid
}

// passToTheCharacter gira a vez até chegar no personagem da bancada.
func passToTheCharacter(t *testing.T, f sceneFixture) {
	t.Helper()
	for range 4 {
		if _, err := f.s.sessions.NextTurn(context.Background(), f.sessionID); err != nil {
			t.Fatalf("passar a vez: %v", err)
		}
		state := stateOf(t, f.s.sessions, f.sessionID)
		if who := state.Initiative[state.TurnIndex].CharacterID; who != nil && *who == f.charID {
			return
		}
	}
	t.Fatal("a vez não chegou no personagem")
}

func TestCombatReflexesGiveAMovementOnTheFirstTurnOnly(t *testing.T) {
	f := newSceneFixture(t)
	seedPowerChoices(t, f.s, f.charID, `["reflexos-de-combate"]`, "{}")
	startCombatWithSomeoneElseOnTurn(t, f)

	passToTheCharacter(t, f)
	if got := movementsOnTurn(t, f); got != 3 {
		t.Errorf("o primeiro turno pagou %d movimentos, want 3 (extra, movimento e a padrão trocada)", got)
	}
	passToTheCharacter(t, f)
	if got := movementsOnTurn(t, f); got != 2 {
		t.Errorf("o segundo turno pagou %d movimentos, want 2", got)
	}
}

func TestWithoutCombatReflexesTheFirstTurnIsTheCommonOne(t *testing.T) {
	f := newSceneFixture(t)
	startCombatWithSomeoneElseOnTurn(t, f)
	passToTheCharacter(t, f)
	if got := movementsOnTurn(t, f); got != 2 {
		t.Errorf("sem o poder o turno pagou %d movimentos, want 2", got)
	}
}

// "de cada combate": quem entra no meio da briga tem o primeiro turno dele.
func TestSomeoneJoiningLateStillHasTheirFirstTurn(t *testing.T) {
	f := newSceneFixture(t)
	seedPowerChoices(t, f.s, f.charID, `["reflexos-de-combate"]`, "{}")
	startCombatWithSomeoneElseOnTurn(t, f)
	state := stateOf(t, f.s.sessions, f.sessionID)
	var entryID string
	for _, e := range state.Initiative {
		if e.CharacterID != nil {
			entryID = e.ID
		}
	}
	if _, err := f.s.sessions.RemoveInitiativeEntry(context.Background(), f.sessionID, entryID); err != nil {
		t.Fatalf("tirar da fila: %v", err)
	}
	for range 3 {
		if _, err := f.s.sessions.NextTurn(context.Background(), f.sessionID); err != nil {
			t.Fatalf("girar: %v", err)
		}
	}
	if stateOf(t, f.s.sessions, f.sessionID).Round < 2 {
		t.Fatal("o controle falhou: a briga ainda está na primeira rodada")
	}
	if _, err := f.s.sessions.AddInitiativeEntry(context.Background(), f.sessionID,
		sheetCombatant("Arcanista", 5, f.charID)); err != nil {
		t.Fatalf("entrar na fila: %v", err)
	}
	passToTheCharacter(t, f)
	if got := movementsOnTurn(t, f); got != 3 {
		t.Errorf("quem chegou tarde pagou %d movimentos no primeiro turno, want 3", got)
	}
}

// Voltar a vez não pode tirar o que o turno devolvido já tinha.
func TestRewindingToTheFirstTurnGivesTheMovementBack(t *testing.T) {
	f := newSceneFixture(t)
	seedPowerChoices(t, f.s, f.charID, `["reflexos-de-combate"]`, "{}")
	startCombatWithSomeoneElseOnTurn(t, f)
	passToTheCharacter(t, f)
	if _, err := f.s.sessions.NextTurn(context.Background(), f.sessionID); err != nil {
		t.Fatalf("passar: %v", err)
	}
	if _, err := f.s.sessions.PreviousTurn(context.Background(), f.sessionID); err != nil {
		t.Fatalf("voltar: %v", err)
	}
	state := stateOf(t, f.s.sessions, f.sessionID)
	if who := state.Initiative[state.TurnIndex].CharacterID; who == nil || *who != f.charID {
		t.Fatal("o controle falhou: voltar não devolveu a vez ao personagem")
	}
	if state.Round != 1 {
		t.Fatalf("o controle falhou: a rodada é %d", state.Round)
	}
	if got := movementsOnTurn(t, f); got != 3 {
		t.Errorf("o turno devolvido pagou %d movimentos, want 3", got)
	}
}
