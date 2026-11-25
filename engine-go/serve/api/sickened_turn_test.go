package api

import (
	"testing"

	"t20engine/domain/engine"
	"t20engine/domain/live"
	"t20engine/domain/sheet"
)

// ENJOADO (p394): "só pode realizar uma ação padrão ou de movimento (não ambas)
// por rodada", e "pode gastar uma ação padrão para fazer uma investida". A
// condição é lida ao ENTRAR na vez, da ficha.
func TestTheSickenedCharacterHasOneActionOnTheTurn(t *testing.T) {
	f := newSceneFixture(t)
	if err := sheet.SaveConditions(t.Context(), f.s.queries, f.charID, []string{"enjoado"}); err != nil {
		t.Fatalf("marcar enjoado: %v", err)
	}
	startCombatWithSomeoneElseOnTurn(t, f)
	passToTheCharacter(t, f)
	if got := movementsOnTurn(t, f); got != 1 {
		t.Errorf("enjoado, o turno pagou %d movimentos, want 1", got)
	}

	passToTheCharacter(t, f)
	ctx := t.Context()
	if err := f.s.sessions.ActionFits(ctx, f.sessionID, engine.ActionFull); err == nil {
		t.Error("enjoado não faz ação completa, e ela coube")
	}
	if err := f.s.sessions.CharacterActionFits(ctx, f.charID, engine.StrikeAction(engine.ChargePlain)); err != nil {
		t.Errorf("enjoado investe com a padrão (p394), e a investida foi recusada: %v", err)
	}
	if _, err := f.s.sessions.SpendAction(ctx, f.sessionID, engine.ActionStandard); err != nil {
		t.Fatalf("a padrão do enjoado tinha de caber: %v", err)
	}
	if err := f.s.sessions.ActionFits(ctx, f.sessionID, engine.ActionMovement); err == nil {
		t.Error("enjoado, depois da padrão, ainda coube um movimento")
	}
}

// O NPC não tem ficha: a condição dele mora na linha da fila, e vale do mesmo jeito.
func TestTheSickenedCreatureHasOneActionOnTheTurn(t *testing.T) {
	f := newSceneFixture(t)
	startCombatWithSomeoneElseOnTurn(t, f)
	state := stateOf(t, f.s.sessions, f.sessionID)
	goblin := state.Initiative[state.TurnIndex]
	if goblin.CharacterID != nil {
		t.Fatal("o controle falhou: a vez devia estar com o Goblin")
	}
	sick := []string{"enjoado"}
	if _, err := f.s.sessions.UpdateInitiativeEntry(t.Context(), f.sessionID, goblin.ID, live.EntryPatch{Conditions: &sick}); err != nil {
		t.Fatalf("marcar o Goblin enjoado: %v", err)
	}
	passToTheCharacter(t, f)
	if _, err := f.s.sessions.NextTurn(t.Context(), f.sessionID); err != nil {
		t.Fatalf("passar a vez ao Goblin: %v", err)
	}
	if got := movementsOnTurn(t, f); got != 1 {
		t.Errorf("o Goblin enjoado pagou %d movimentos, want 1", got)
	}
}

// Voltar a vez devolve ao enjoado o turno de UMA ação, e não um turno cheio.
func TestRewindingToTheSickenedTurnKeepsOneAction(t *testing.T) {
	f := newSceneFixture(t)
	if err := sheet.SaveConditions(t.Context(), f.s.queries, f.charID, []string{"enjoado"}); err != nil {
		t.Fatalf("marcar enjoado: %v", err)
	}
	startCombatWithSomeoneElseOnTurn(t, f)
	passToTheCharacter(t, f)
	if _, err := f.s.sessions.NextTurn(t.Context(), f.sessionID); err != nil {
		t.Fatalf("passar: %v", err)
	}
	if _, err := f.s.sessions.PreviousTurn(t.Context(), f.sessionID); err != nil {
		t.Fatalf("voltar: %v", err)
	}
	if got := movementsOnTurn(t, f); got != 1 {
		t.Errorf("o turno devolvido ao enjoado pagou %d movimentos, want 1", got)
	}
}
