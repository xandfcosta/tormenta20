package api

import (
	"context"
	"net/http"
	"testing"

	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// O PRIMEIRO GESTO DEPOIS DE UM REINÍCIO. O combate está no banco e a memória
// do servidor está vazia; a aba do mestre sobreviveu ao deploy e ele clica em
// "Próximo" antes de alguém abrir a Mesa. O comando tem de agir sobre o combate
// GRAVADO — agir sobre uma fila vazia e gravá-la por cima é perder a sessão em
// silêncio (ALE-369).
func TestACommandBeforeTheFirstPageActsOnTheStoredCombat(t *testing.T) {
	f := newSceneFixture(t)
	ctx := context.Background()
	stored := `{"initiative":[` +
		`{"id":"a","label":"Ogro","initiative":19,"type":"npc"},` +
		`{"id":"b","label":"Goblin","initiative":12,"type":"npc"}],` +
		`"round":1,"turnIndex":0,` +
		`"scene":{"kind":"acao","number":1,"standardLeft":true,"movementLeft":true},"scenesSoFar":1}`
	if err := f.s.queries.ResetSessionTracker(ctx, sqlcgen.ResetSessionTrackerParams{
		RuntimeState: stored, UpdatedAt: dbvalue.NowISO(), ID: f.sessionID,
	}); err != nil {
		t.Fatalf("gravar o combate: %v", err)
	}

	if rec := f.pede(t, f.gm, http.MethodPost, f.tableUrl()+"/iniciativa/proxima-vez", ""); rec.Code != http.StatusOK {
		t.Fatalf("passar a vez deu %d", rec.Code)
	}

	state := f.s.sessions.GetState(f.sessionID)
	if len(state.Initiative) != 2 {
		t.Fatalf("o comando agiu sobre uma fila de %d linhas, e o banco tinha 2", len(state.Initiative))
	}
	if state.TurnIndex != 1 || state.Initiative[state.TurnIndex].Label != "Goblin" {
		t.Errorf("a vez passou do Ogro para o Goblin? turnIndex=%d", state.TurnIndex)
	}
	if !state.InScene() {
		t.Error("o combate gravado estava em cena, e o comando o tirou dela")
	}
}

// A LEITURA TAMBÉM. Quem pergunta pelo estado antes do primeiro GET — o
// ataque pergunta de quem é a vez — recebe o combate gravado, e não um
// "fora de combate" que parece verdade.
func TestAReadBeforeTheFirstPageSeesTheStoredCombat(t *testing.T) {
	f := newSceneFixture(t)
	ctx := context.Background()
	stored := `{"initiative":[{"id":"a","label":"Ogro","initiative":19,"type":"npc"}],"round":1,"turnIndex":0,` +
		`"scene":{"kind":"acao","number":1,"standardLeft":true,"movementLeft":true},"scenesSoFar":1}`
	if err := f.s.queries.ResetSessionTracker(ctx, sqlcgen.ResetSessionTrackerParams{
		RuntimeState: stored, UpdatedAt: dbvalue.NowISO(), ID: f.sessionID,
	}); err != nil {
		t.Fatalf("gravar o combate: %v", err)
	}
	if state := f.s.sessions.GetState(f.sessionID); len(state.Initiative) != 1 || state.TurnIndex != 0 {
		t.Errorf("a leitura antes do primeiro GET viu %d linhas e turnIndex %d; o banco tem 1 e 0",
			len(state.Initiative), state.TurnIndex)
	}
}
