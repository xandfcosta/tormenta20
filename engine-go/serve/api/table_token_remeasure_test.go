package api

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

// A PEÇA ACOMPANHA A FICHA. Ela media o deslocamento e o "ignora terreno difícil" uma vez, ao entrar
// no mapa, e o interruptor de situação muda esse número no meio da cena: a druida que liga "em
// terreno natural" (Caminho dos Ermos, p63) com a peça já no tabuleiro tem de parar de pagar o
// terreno difícil ali, e voltar a pagar ao desligar (decisão do dono, ALE-654).
func TestThePieceFollowsTheSituationSwitchedOnTheSheet(t *testing.T) {
	f := newSceneFixture(t)
	seedClasse(t, f.s, f.charID, "Druida", 2)
	f.scene(t)
	f.seedOpenBoard(t, "stone")
	entry, _ := sceneIds(t, f)
	f.posts(t, f.gm, f.tableUrl()+"/tabuleiro/pecas", `{"map_selection":"`+entry+`"}`)

	ignores := func() bool {
		t.Helper()
		b := boardRead(f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab))
		if len(b.Tokens) != 1 {
			t.Fatalf("o mapa ficou com %d peças, esperado 1", len(b.Tokens))
		}
		return b.Tokens[0].IgnoresDifficult
	}
	if ignores() {
		t.Fatal("a peça nasceu ignorando o terreno difícil — o caso não mede o interruptor")
	}
	if refusal := situationRefusal(t, f, f.charID, "flag:terreno-natural"); refusal != "" {
		t.Fatalf("ligar em terreno natural: %q", refusal)
	}
	if !ignores() {
		t.Error("a druida ligou \"em terreno natural\" e a peça no tabuleiro ainda paga o terreno difícil")
	}
	if refusal := situationRefusal(t, f, f.charID, "flag:terreno-natural"); refusal != "" {
		t.Fatalf("desligar em terreno natural: %q", refusal)
	}
	if ignores() {
		t.Error("a druida desligou \"em terreno natural\" e a peça continua ignorando o terreno difícil")
	}
}

// A CONDIÇÃO POSTA no meio da cena também alcança a peça: o Lento (p394) corta o deslocamento à
// metade, e a peça que mediu a ficha ao entrar no mapa andaria o deslocamento inteiro.
func TestThePieceFollowsTheSlowedConditionPutOnTheSheet(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	f.seedOpenBoard(t, "stone")
	entry, _ := sceneIds(t, f)
	f.posts(t, f.gm, f.tableUrl()+"/tabuleiro/pecas", `{"map_selection":"`+entry+`"}`)
	squares := func() int {
		t.Helper()
		return boardRead(f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)).Tokens[0].SpeedSquares
	}
	before := squares()
	if before < 2 {
		t.Fatalf("a peça nasceu com %d quadrados — o caso não mede a metade", before)
	}
	if rec := f.requests(t, f.player, http.MethodPost, fmt.Sprintf("/personagens/%d/efeitos/condicao/lento", f.charID), ""); rec.Code != http.StatusOK {
		t.Fatalf("aplicar o Lento deu %d: %s", rec.Code, rec.Body.String())
	}
	if after := squares(); after >= before {
		t.Errorf("com Lento, a peça continua com %d quadrados, os mesmos %d de antes", after, before)
	}
}
