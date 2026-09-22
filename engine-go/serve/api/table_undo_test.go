package api

import (
	"context"
	"net/http"
	"strings"
	"t20engine/domain/engine"
	"testing"
)

func boardStops(t *testing.T, f sceneFixture) []engine.Square {
	t.Helper()
	b := boardRead(f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab))
	if b == nil || b.Pending == nil {
		return nil
	}
	return b.Pending.Stops
}

// Reconstruir pelas paradas que sobraram e não cortar o fim do caminho: o número
// de quadrados de um trecho não se deduz das paradas sem redesenhá-lo. O CUSTO é
// a asserção que importa — um desfazer que tirasse os quadrados e deixasse o
// número velho faria a mesa confirmar um movimento por um preço que não é o dele.
func TestUndoTakesTheLastLegAndRecomputesTheCost(t *testing.T) {
	f := newSceneFixture(t)
	tokenID := f.onBoard(t)
	base := f.tableUrl() + "/tabuleiro/" + tokenID

	// (0,0) → (2,0) são 2 quadrados; a segunda perna até (2,2) soma mais 2.
	for _, square := range []string{`{"from":{"X":2,"Y":0}}`, `{"from":{"X":2,"Y":2}}`} {
		if rec := f.requests(t, f.gm, http.MethodPost, base+"/parada", square); rec.Code != http.StatusOK {
			t.Fatalf("a parada %s deu %d", square, rec.Code)
		}
	}
	before := boardRead(f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)).Pending
	if before == nil || before.Cost != 4 || len(before.Stops) != 3 {
		t.Fatalf("as duas pernas ficaram %+v — sem o caso positivo o desfazer não mede nada", before)
	}

	if rec := f.requests(t, f.gm, http.MethodPost, base+"/desfazer-parada", ""); rec.Code != http.StatusOK {
		t.Fatalf("desfazer deu %d", rec.Code)
	}
	after := boardRead(f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)).Pending
	if after == nil {
		t.Fatal("desfazer UMA parada cancelou o movimento inteiro")
	}
	if after.Cost != 2 {
		t.Errorf("o custo ficou %d depois de desfazer, esperado 2", after.Cost)
	}
	if end := after.Path[len(after.Path)-1]; end != (engine.Square{X: 2}) {
		t.Errorf("o caminho terminou em %v, esperado a primeira parada (2,0)", end)
	}
	if stops := boardStops(t, f); len(stops) != 2 {
		t.Errorf("sobraram %d paradas, esperado 2 (a origem e a primeira)", len(stops))
	}
}

// Uma proposta sem perna nenhuma não é proposta: deixar um provisório de custo
// zero na mesa seria oferecer um "Confirmar" que não move ninguém, e a peça
// ficaria presa num estado que só o Cancelar resolveria.
func TestUndoingTheLastStopCancelsTheMove(t *testing.T) {
	f := newSceneFixture(t)
	tokenID := f.onBoard(t)
	base := f.tableUrl() + "/tabuleiro/" + tokenID

	if rec := f.requests(t, f.gm, http.MethodPost, base+"/parada", `{"from":{"X":2,"Y":0}}`); rec.Code != http.StatusOK {
		t.Fatalf("a parada deu %d", rec.Code)
	}
	if boardRead(f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)).Pending == nil {
		t.Fatal("não havia movimento para desfazer — o caso positivo falhou")
	}

	if rec := f.requests(t, f.gm, http.MethodPost, base+"/desfazer-parada", ""); rec.Code != http.StatusOK {
		t.Fatalf("desfazer deu %d", rec.Code)
	}
	if p := boardRead(f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)).Pending; p != nil {
		t.Errorf("sobrou um provisório de custo %d sem perna nenhuma: %+v", p.Cost, p)
	}
}

// Um botão que não faz nada é pior que nenhum, e com UMA perna desfazer já é
// cancelar — que está ali do lado dizendo isso com a palavra certa.
func TestWithNoLegToUndoTheButtonDoesNotAppear(t *testing.T) {
	f := newSceneFixture(t)
	tokenID := f.onBoard(t)
	base := f.tableUrl() + "/tabuleiro/" + tokenID

	if rec := f.requests(t, f.gm, http.MethodPost, base+"/parada", `{"from":{"X":2,"Y":0}}`); rec.Code != http.StatusOK {
		t.Fatalf("a parada deu %d", rec.Code)
	}
	withOne := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	// O CONTROLE: a faixa do movimento ESTÁ na tela. Sem ele, não achar o botão
	// seria verdade também numa tela sem movimento proposto nenhum — e a
	// asserção de ausência passaria verde sobre nada.
	if !strings.Contains(withOne, "Cancelar") {
		t.Fatal("a faixa do movimento não apareceu")
	}
	if strings.Contains(withOne, "Desfazer parada") {
		t.Error("o botão apareceu com uma perna só, onde desfazer já é cancelar")
	}

	if rec := f.requests(t, f.gm, http.MethodPost, base+"/parada", `{"from":{"X":2,"Y":2}}`); rec.Code != http.StatusOK {
		t.Fatalf("a segunda parada deu %d", rec.Code)
	}
	withTwo := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(withTwo, "Desfazer parada") {
		t.Error("com duas pernas o botão não apareceu")
	}
	// E a JUNTA fica marcada no mapa: sem ela a trilha é uma faixa contínua, e o
	// botão desfaria uma perna que a tela não mostra.
	if !strings.Contains(withTwo, "board-stop") {
		t.Error("a parada intermediária não foi marcada na trilha")
	}
}

// As paradas são estado do SERVIDOR, e não do navegador: recarregar no meio de
// uma proposta não perde nada, e a cena que se abre do zero é a mesma que já
// estava aberta.
func TestTheStopsSurviveAPageReload(t *testing.T) {
	f := newSceneFixture(t)
	tokenID := f.onBoard(t)
	base := f.tableUrl() + "/tabuleiro/" + tokenID
	for _, square := range []string{`{"from":{"X":2,"Y":0}}`, `{"from":{"X":2,"Y":2}}`} {
		if rec := f.requests(t, f.gm, http.MethodPost, base+"/parada", square); rec.Code != http.StatusOK {
			t.Fatalf("a parada %s deu %d", square, rec.Code)
		}
	}

	// Uma carga fria, como quem apertou F5: nada do navegador anterior viaja.
	screen := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(screen, "Desfazer parada") {
		t.Error("a página recarregada perdeu o desfazer — as paradas não sobreviveram")
	}
}

// Duas mãos empilhando pernas no mesmo movimento é o estado que o `ByUserID`
// existe para evitar. Sem a conferência, o clique de um segundo jogador
// continuaria o caminho que o primeiro está montando — e quem confirmasse
// confirmaria um percurso que ninguém inteiro escolheu.
func TestSomeoneElsesProposalDoesNotExtendMine(t *testing.T) {
	f := newSceneFixture(t)
	tokenID := f.onBoard(t)
	base := f.tableUrl() + "/tabuleiro/" + tokenID

	if rec := f.requests(t, f.gm, http.MethodPost, base+"/parada", `{"from":{"X":2,"Y":0}}`); rec.Code != http.StatusOK {
		t.Fatalf("a parada do mestre deu %d", rec.Code)
	}
	// O jogador é dono da peça (ela aponta para a ficha dele) e a cena está fora
	// de combate, então ele PODE propor — o que ele não pode é herdar as paradas
	// de outra pessoa.
	if rec := f.requests(t, f.player, http.MethodPost, base+"/parada", `{"from":{"X":0,"Y":2}}`); rec.Code != http.StatusOK {
		t.Fatalf("a parada do jogador deu %d", rec.Code)
	}
	stops := boardStops(t, f)
	if len(stops) != 2 {
		t.Fatalf("o jogador herdou as paradas do mestre: %v", stops)
	}
	if stops[0] != (engine.Square{}) {
		t.Errorf("o movimento do jogador começou em %v, esperado o lugar da peça (0,0)", stops[0])
	}
}
