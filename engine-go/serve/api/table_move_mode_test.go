package api

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"t20engine/domain/board"
	"t20engine/infra/db/sqlcgen"
)

// O MODO DE DESLOCAMENTO NO TABULEIRO (decisão do dono, mockup aprovado): a peça do tritão nasce com a
// natação medida da ficha (12m, p30, são 8 quadrados), o cartão do movimento oferece "Anda por", e
// tocar na natação troca o modo do movimento proposto.
func TestTheMermanPieceSwimsWhenTheCardSaysSo(t *testing.T) {
	f := newSceneFixture(t)
	if err := f.s.queries.CreateRace(context.Background(), sqlcgen.CreateRaceParams{Characterid: f.charID, Race: "sereia-tritao"}); err != nil {
		t.Fatalf("pôr a raça: %v", err)
	}
	f.scene(t)
	f.seedOpenBoard(t, "stone")
	entry, _ := sceneIds(t, f)
	f.posts(t, f.gm, f.tableUrl()+"/tabuleiro/pecas", `{"map_selection":"`+entry+`"}`)
	// EM COMBATE, na vez dele: sem vez não há orçamento, e sem orçamento o "Anda por" não aparece.
	for _, step := range []string{"/cena/iniciar/acao", "/iniciativa/proxima-vez"} {
		if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+step, ""); rec.Code != http.StatusOK {
			t.Fatalf("%s deu %d", step, rec.Code)
		}
	}

	piece := boardRead(f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)).Tokens[0]
	if len(piece.OtherModes) != 1 || piece.OtherModes[0] != (board.MoveMode{Kind: board.MoveModeSwim, Squares: 8}) {
		t.Fatalf("a peça do tritão nasceu com os modos %+v; a natação de 12m são 8 quadrados", piece.OtherModes)
	}

	base := f.tableUrl() + "/tabuleiro/" + piece.ID
	if rec := f.requests(t, f.gm, http.MethodPost, base+"/parada", `{"from":{"X":2,"Y":0}}`); rec.Code != http.StatusOK {
		t.Fatalf("propor deu %d", rec.Code)
	}
	page := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(page, "Anda por") || !strings.Contains(page, "Natação 12m") {
		t.Error("o cartão do movimento não oferece \"Anda por\" com a natação do tritão")
	}

	if rec := f.requests(t, f.gm, http.MethodPost, base+"/modo/natacao", ""); rec.Code != http.StatusOK {
		t.Fatalf("trocar para a natação deu %d: %s", rec.Code, rec.Body.String())
	}
	pending := boardRead(f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)).Pending
	if pending == nil || pending.Mode != board.MoveModeSwim || pending.Budget != 8 {
		t.Fatalf("o movimento proposto ficou %+v; o mestre tocou na natação, que são 8 quadrados", pending)
	}
}

// A peça de quem só anda não ganha "Anda por": o cartão de hoje fica como está.
func TestTheWalkingPieceHasNoMoveModeChoice(t *testing.T) {
	f := newSceneFixture(t)
	tokenID := f.onBoard(t)
	for _, step := range []string{"/cena/iniciar/acao", "/iniciativa/proxima-vez"} {
		if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+step, ""); rec.Code != http.StatusOK {
			t.Fatalf("%s deu %d", step, rec.Code)
		}
	}
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/tabuleiro/"+tokenID+"/parada", `{"from":{"X":2,"Y":0}}`); rec.Code != http.StatusOK {
		t.Fatalf("propor deu %d", rec.Code)
	}
	if strings.Contains(f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String(), "Anda por") {
		t.Error("a peça que só anda ganhou \"Anda por\" no cartão do movimento")
	}
}
