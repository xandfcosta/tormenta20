package api

import (
	"t20engine/domain/live"
	"testing"
)

// actionSceneWithTheGoblinFirst põe a cena de ação no ar com um Goblin do
// bestiário (iniciativa 20) à frente da ficha `charID`, que entra na fila com
// o rótulo `label`, e gira a vez `turns` vezes: uma deixa a vez com o Goblin,
// duas a passam para a ficha. Devolve o id da entrada do Goblin, que é o alvo
// de quem ataca.
//
// O Goblin vem do BESTIÁRIO porque é ele que traz a Defesa que o ataque mede.
// E ele vai à FRENTE para que a vez de outro possa acontecer: com um
// combatente só, "não é a sua vez" não tem como acontecer, e o caso mediria a
// metade em que o defeito é invisível.
//
// Exemplo: goblin := f.actionSceneWithTheGoblinFirst(t, "Arcanista", f.charID, 2)
func (f sceneFixture) actionSceneWithTheGoblinFirst(t *testing.T, label string, charID int64, turns int) string {
	t.Helper()
	if turns != 1 && turns != 2 {
		t.Fatalf("girar %d vezes não cabe numa fila de dois: 1 deixa a vez com o Goblin, 2 a passa para %s", turns, label)
	}
	store := f.s.sessions
	ctx := t.Context()
	if _, err := store.State(ctx, f.sessionID); err != nil {
		t.Fatalf("carregar a sessão %d: %v", f.sessionID, err)
	}
	if _, err := store.StartScene(ctx, f.sessionID, live.SceneAction); err != nil {
		t.Fatalf("começar a cena de ação: %v", err)
	}
	goblin := "goblin-salteador"
	if _, err := store.AddInitiativeEntry(ctx, f.sessionID, live.InitiativeEntry{
		Label: "Goblin", Initiative: 20, Type: "npc", MonsterID: &goblin,
	}); err != nil {
		t.Fatalf("pôr o Goblin na fila: %v", err)
	}
	if _, err := store.AddInitiativeEntry(ctx, f.sessionID, sheetCombatant(label, 5, charID)); err != nil {
		t.Fatalf("pôr %s na fila: %v", label, err)
	}
	for range turns {
		if _, err := store.NextTurn(ctx, f.sessionID); err != nil {
			t.Fatalf("girar a vez: %v", err)
		}
	}

	// O CONTROLE: o caso que pediu a vez da ficha e recebeu a do Goblin (ou o
	// contrário) mediria a outra metade da regra, e passaria verde nela.
	state := stateOf(t, store, f.sessionID)
	if state.TurnIndex < 0 || state.TurnIndex >= len(state.Initiative) {
		t.Fatalf("depois de girar %d vezes a vez está no índice %d, e a fila tem %d", turns, state.TurnIndex, len(state.Initiative))
	}
	onTurn := state.Initiative[state.TurnIndex]
	sheetOnTurn := onTurn.CharacterID != nil && *onTurn.CharacterID == charID
	if sheetOnTurn != (turns == 2) {
		t.Fatalf("depois de girar %d vezes a vez é de %q", turns, onTurn.Label)
	}
	return entryLabeled(t, state, "Goblin")
}
