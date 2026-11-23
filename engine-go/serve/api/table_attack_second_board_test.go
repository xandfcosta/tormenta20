package api

import (
	"context"
	"testing"

	"t20engine/domain/board"
	"t20engine/domain/engine"
)

// O GOLPE LÊ O TABULEIRO QUE TEM AS DUAS PEÇAS, e não o padrão. Com a taverna aberta primeiro e a cripta depois,
// quem ataca e o alvo na cripta, com o alvo sob cobertura, o golpe trazia a Defesa sem o +5 (p239): a porta lia
// sempre a aba mais antiga, onde as peças não estão.
func TestTheStrikeReadsTheBoardWhereBothPiecesStand(t *testing.T) {
	f, goblin := attackOnTurn(t)
	f.seedOpenBoard(t, "stone")
	crypt, err := f.s.tableHost().Boards().Open(context.Background(), f.sessionID, "Cripta", "stone")
	if err != nil {
		t.Fatalf("abrir a cripta: %v", err)
	}
	attackerSquare, targetSquare := 1, 5
	putTokenOnBoard(t, f, crypt.ID, goblin, targetSquare)
	putTokenOnBoard(t, f, crypt.ID, entryOnTurn(t, f), attackerSquare)

	bare := proposeAttackOn(t, f, goblin)
	if bare.Defense == 0 {
		t.Fatalf("o controle já estava errado: o ataque não trouxe a Defesa enfrentada")
	}
	if _, err := f.s.tableHost().Boards().PaintTerrain(context.Background(), f.sessionID, crypt.ID,
		engine.Square{X: targetSquare, Y: 0}, board.TerrenoCobertura, true); err != nil {
		t.Fatalf("pintar a cobertura na cripta: %v", err)
	}

	covered := proposeAttackOn(t, f, goblin)
	if covered.Defense != bare.Defense+5 {
		t.Errorf("o Goblin em cobertura na cripta enfrentou Defesa %d e sem ela %d; want +5 (p239)",
			covered.Defense, bare.Defense)
	}
}

// putTokenOnBoard põe a peça da linha da fila no tabuleiro dado, na casa (x, 0).
func putTokenOnBoard(t *testing.T, f sceneFixture, boardID, entryID string, x int) {
	t.Helper()
	if _, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, boardID,
		board.BoardToken{Label: "peça " + entryID, X: x, Y: 0, Footprint: 1, Kind: "character", EntryID: &entryID},
	); err != nil {
		t.Fatalf("pôr a peça de %q no tabuleiro %s: %v", entryID, boardID, err)
	}
}
