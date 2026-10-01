package api

import (
	"context"
	"net/http"
	"testing"

	"t20engine/domain/board"
	"t20engine/domain/live"
)

// O TERRENO QUE O MESTRE PINTA ENTRA NO ATAQUE (ALE-423, p238-239).
//
// Até aqui ele alimentava só o OLHO: as quatro espécies existem desde sempre, o
// `terrain_drawing.go` dá um ícone a cada uma, e o d20 ignorava as três que
// mudam número. O `board_state.go` registrava a lacuna por extenso — "hoje não
// são consumidos por nada".
//
// Este caso é o trajeto inteiro: o mestre pinta, a peça está na casa, e a conta
// do ataque muda.
func TestTheCoverTheMasterPaintsRaisesTheDefense(t *testing.T) {
	f, goblin := attackOnTurn(t)
	f.seedOpenBoard(t, "stone")
	// As duas peças no tabuleiro, e o GOBLIN na casa que vai receber a tinta.
	attackerSquare, targetSquare := 1, 5
	putTokenForEntry(t, f, goblin, targetSquare, 0)
	putTokenForEntry(t, f, entryOnTurn(t, f), attackerSquare, 0)

	nua := proposeAttackOn(t, f, goblin)
	if nua.Defense == 0 {
		t.Fatalf("o controle já estava errado: o ataque não trouxe a Defesa enfrentada")
	}

	// O MESTRE PINTA A COBERTURA na casa do Goblin.
	rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/tabuleiro/terreno",
		stroke(string(board.TerrenoCobertura), targetSquare, 0, targetSquare, 0))
	if rec.Code != http.StatusOK {
		t.Fatalf("pintar a cobertura deu %d", rec.Code)
	}

	coberto := proposeAttackOn(t, f, goblin)
	if coberto.Defense != nua.Defense+5 {
		t.Errorf("o Goblin em cobertura enfrentou Defesa %d e sem ela %d — a p239 dá +5",
			coberto.Defense, nua.Defense)
	}
}

// O ELEVADO é a única espécie que beneficia quem está NELA, e por isso ele é o
// caso que prende o LADO: um medidor que só olhasse a casa do alvo daria o +2
// ao atacante que está embaixo.
func TestTheHigherGroundBelongsToWhoStandsOnIt(t *testing.T) {
	f, goblin := attackOnTurn(t)
	f.seedOpenBoard(t, "stone")
	attackerSquare, targetSquare := 1, 5
	putTokenForEntry(t, f, goblin, targetSquare, 0)
	putTokenForEntry(t, f, entryOnTurn(t, f), attackerSquare, 0)

	nua := proposeAttackOn(t, f, goblin)

	// Pintado na casa do ALVO, o elevado não é do atacante.
	paint(t, f, board.TerrenoElevado, targetSquare)
	doAlvo := proposeAttackOn(t, f, goblin)
	if doAlvo.Total-doAlvo.Roll != nua.Total-nua.Roll {
		t.Errorf("o elevado na casa do ALVO mudou o ataque de quem está embaixo: "+
			"bônus %d contra %d", doAlvo.Total-doAlvo.Roll, nua.Total-nua.Roll)
	}

	// Pintado na casa do ATACANTE, ele vale +2.
	paint(t, f, board.TerrenoElevado, attackerSquare)
	doAtacante := proposeAttackOn(t, f, goblin)
	if got := doAtacante.Total - doAtacante.Roll - (nua.Total - nua.Roll); got != 2 {
		t.Errorf("de posição elevada o atacante somou %d, e a p239 dá +2", got)
	}
}

// ── os ajudantes deste arquivo ───────────────────────────────────────────────

// putTokenForEntry põe no tabuleiro a peça da linha da fila, na casa pedida.
//
// A ligação é o `EntryID`: é por ele que o ataque acha onde a peça está, e sem
// ele o terreno não tem como alcançar o d20.
func putTokenForEntry(t *testing.T, f sceneFixture, entryID string, x, y int) {
	t.Helper()
	if _, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab,
		board.BoardToken{Label: "peça " + entryID, X: x, Y: y, Footprint: 1,
			Kind: "character", EntryID: &entryID},
	); err != nil {
		t.Fatalf("pôr a peça de %q no tabuleiro: %v", entryID, err)
	}
}

// entryOnTurn é a linha de quem está na vez.
func entryOnTurn(t *testing.T, f sceneFixture) string {
	t.Helper()
	state := stateOf(t, f.s.sessions, f.sessionID)
	return state.Initiative[state.TurnIndex].ID
}

// paint pinta UMA casa com a espécie pedida, pela rota do mestre.
func paint(t *testing.T, f sceneFixture, species board.TerrainKind, x int) {
	t.Helper()
	rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/tabuleiro/terreno",
		stroke(string(species), x, 0, x, 0))
	if rec.Code != http.StatusOK {
		t.Fatalf("pintar %s deu %d", species, rec.Code)
	}
}

// proposeAttackOn propõe o ataque e devolve o provisório, CANCELANDO em seguida
// — o caso propõe várias vezes, e um provisório em pé recusa o próximo.
func proposeAttackOn(t *testing.T, f sceneFixture, targetEntry string) live.PendingAttack {
	t.Helper()
	rec := f.requests(t, f.player, http.MethodPost,
		f.tableUrl()+"/iniciativa/"+targetEntry+"/atacar", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("propor o ataque deu %d", rec.Code)
	}
	pending := stateOf(t, f.s.sessions, f.sessionID).PendingAttack
	if pending == nil {
		t.Fatalf("o ataque não virou provisório: %q", tableRefusal(t, rec.Body.String()))
	}
	out := *pending
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/cancelar", ""); rec.Code != http.StatusOK {
		t.Fatalf("cancelar o provisório deu %d", rec.Code)
	}
	return out
}
