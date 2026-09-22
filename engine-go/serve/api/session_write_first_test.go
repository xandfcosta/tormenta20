package api

import (
	"context"
	"strings"
	"testing"

	"t20engine/domain/live"
)

// A MUTAÇÃO JÁ ESTÁ GRAVADA QUANDO ELA VOLTA.
//
// Aqui morava o contrário: o store mutava a memória, devolvia o retrato, e a
// gravação saía numa goroutine best-effort. A mesa via a rodada avançar e o
// disco podia recusá-la depois — daí a marca `Dirty` e o aviso na tela
// (ALE-371).
//
// O caso não chama gravação nenhuma de propósito: ele pergunta ao BANCO logo
// depois da mutação. Se for preciso um `Persist` para ele passar, a mutação não
// é a dona da gravação.
func TestAMutationIsAlreadyInTheDatabaseWhenItReturns(t *testing.T) {
	f := newSceneFixture(t)
	ctx := context.Background()

	if _, err := f.s.sessions.AddInitiativeEntry(f.sessionID, live.InitiativeEntry{
		Label: "Ogro cansado", Type: "npc", Initiative: 19,
	}); err != nil {
		t.Fatalf("pôr o ogro na fila: %v", err)
	}

	row, err := f.s.queries.GetSession(ctx, f.sessionID)
	if err != nil {
		t.Fatalf("reler a sessão: %v", err)
	}
	if !strings.Contains(row.Runtimestate, "Ogro cansado") {
		t.Errorf("o banco não tem o ogro logo depois da mutação: %s", row.Runtimestate)
	}
}
