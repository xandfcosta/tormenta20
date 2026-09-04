package api

import (
	"context"
	"net/http"
	"strings"
	"t20engine/engine"
	"testing"
)

func boardStops(t *testing.T, f pilotoFixture) []engine.Square {
	t.Helper()
	b := f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)
	if b == nil || b.Pending == nil {
		return nil
	}
	return b.Pending.Stops
}

// TestUndoTakesTheLastLegAndRecomputesTheCost.
//
// Reconstruir pelas paradas que sobraram e não cortar o fim do caminho: o número
// de quadrados de um trecho não se deduz das paradas sem redesenhá-lo. O CUSTO é
// a asserção que importa — um desfazer que tirasse os quadrados e deixasse o
// número velho faria a mesa confirmar um movimento por um preço que não é o dele.
func TestUndoTakesTheLastLegAndRecomputesTheCost(t *testing.T) {
	f := novoPiloto(t)
	tokenID := f.onBoard(t)
	base := f.tableUrl() + "/tabuleiro/" + tokenID

	// (0,0) → (2,0) são 2 quadrados; a segunda perna até (2,2) soma mais 2.
	for _, casa := range []string{"/parada/2/0", "/parada/2/2"} {
		if rec := f.pede(t, f.mestre, http.MethodPost, base+casa, ""); rec.Code != http.StatusOK {
			t.Fatalf("a parada %s deu %d", casa, rec.Code)
		}
	}
	antes := f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab).Pending
	if antes == nil || antes.Cost != 4 || len(antes.Stops) != 3 {
		t.Fatalf("as duas pernas ficaram %+v — sem o caso positivo o desfazer não mede nada", antes)
	}

	if rec := f.pede(t, f.mestre, http.MethodPost, base+"/desfazer-parada", ""); rec.Code != http.StatusOK {
		t.Fatalf("desfazer deu %d", rec.Code)
	}
	depois := f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab).Pending
	if depois == nil {
		t.Fatal("desfazer UMA parada cancelou o movimento inteiro")
	}
	if depois.Cost != 2 {
		t.Errorf("o custo ficou %d depois de desfazer, esperado 2", depois.Cost)
	}
	if fim := depois.Path[len(depois.Path)-1]; fim != (engine.Square{X: 2}) {
		t.Errorf("o caminho terminou em %v, esperado a primeira parada (2,0)", fim)
	}
	if paradas := boardStops(t, f); len(paradas) != 2 {
		t.Errorf("sobraram %d paradas, esperado 2 (a origem e a primeira)", len(paradas))
	}
}

// TestUndoingTheLastStopCancelsTheMove.
//
// Uma proposta sem perna nenhuma não é proposta: deixar um provisório de custo
// zero na mesa seria oferecer um "Confirmar" que não move ninguém, e a peça
// ficaria presa num estado que só o Cancelar resolveria.
func TestUndoingTheLastStopCancelsTheMove(t *testing.T) {
	f := novoPiloto(t)
	tokenID := f.onBoard(t)
	base := f.tableUrl() + "/tabuleiro/" + tokenID

	if rec := f.pede(t, f.mestre, http.MethodPost, base+"/parada/2/0", ""); rec.Code != http.StatusOK {
		t.Fatalf("a parada deu %d", rec.Code)
	}
	if f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab).Pending == nil {
		t.Fatal("não havia movimento para desfazer — o caso positivo falhou")
	}

	if rec := f.pede(t, f.mestre, http.MethodPost, base+"/desfazer-parada", ""); rec.Code != http.StatusOK {
		t.Fatalf("desfazer deu %d", rec.Code)
	}
	if p := f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab).Pending; p != nil {
		t.Errorf("sobrou um provisório de custo %d sem perna nenhuma: %+v", p.Cost, p)
	}
}

// TestWithNoLegToUndoTheButtonDoesNotAppear.
//
// Um botão que não faz nada é pior que nenhum, e com UMA perna desfazer já é
// cancelar — que está ali do lado dizendo isso com a palavra certa.
func TestWithNoLegToUndoTheButtonDoesNotAppear(t *testing.T) {
	f := novoPiloto(t)
	tokenID := f.onBoard(t)
	base := f.tableUrl() + "/tabuleiro/" + tokenID

	if rec := f.pede(t, f.mestre, http.MethodPost, base+"/parada/2/0", ""); rec.Code != http.StatusOK {
		t.Fatalf("a parada deu %d", rec.Code)
	}
	comUma := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()
	// O CONTROLE: a faixa do movimento ESTÁ na tela. Sem ele, não achar o botão
	// seria verdade também numa tela sem movimento proposto nenhum — e a
	// asserção de ausência passaria verde sobre nada.
	if !strings.Contains(comUma, "Cancelar") {
		t.Fatal("a faixa do movimento não apareceu")
	}
	if strings.Contains(comUma, "Desfazer parada") {
		t.Error("o botão apareceu com uma perna só, onde desfazer já é cancelar")
	}

	if rec := f.pede(t, f.mestre, http.MethodPost, base+"/parada/2/2", ""); rec.Code != http.StatusOK {
		t.Fatalf("a segunda parada deu %d", rec.Code)
	}
	comDuas := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(comDuas, "Desfazer parada") {
		t.Error("com duas pernas o botão não apareceu")
	}
	// E a JUNTA fica marcada no mapa: sem ela a trilha é uma faixa contínua, e o
	// botão desfaria uma perna que a tela não mostra.
	if !strings.Contains(comDuas, "tabuleiro-parada") {
		t.Error("a parada intermediária não foi marcada na trilha")
	}
}

// TestTheStopsSurviveAPageReload.
//
// É a divergência DELIBERADA em relação à SPA, e vale registrar: lá a lista mora
// no navegador e o `board-region` documenta a consequência aceita — quem recarrega
// no meio de uma proposta perde o desfazer de UMA. Aqui o estado é do servidor,
// então recarregar não perde nada. A cena que se abre do zero é a mesma que já
// estava aberta.
func TestTheStopsSurviveAPageReload(t *testing.T) {
	f := novoPiloto(t)
	tokenID := f.onBoard(t)
	base := f.tableUrl() + "/tabuleiro/" + tokenID
	for _, casa := range []string{"/parada/2/0", "/parada/2/2"} {
		if rec := f.pede(t, f.mestre, http.MethodPost, base+casa, ""); rec.Code != http.StatusOK {
			t.Fatalf("a parada %s deu %d", casa, rec.Code)
		}
	}

	// Uma carga fria, como quem apertou F5: nada do navegador anterior viaja.
	tela := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(tela, "Desfazer parada") {
		t.Error("a página recarregada perdeu o desfazer — as paradas não sobreviveram")
	}
}

// TestSomeoneElsesProposalDoesNotExtendMine.
//
// Duas mãos empilhando pernas no mesmo movimento é o estado que o `ByUserID`
// existe para evitar. Sem a conferência, o clique de um segundo jogador
// continuaria o caminho que o primeiro está montando — e quem confirmasse
// confirmaria um percurso que ninguém inteiro escolheu.
func TestSomeoneElsesProposalDoesNotExtendMine(t *testing.T) {
	f := novoPiloto(t)
	tokenID := f.onBoard(t)
	base := f.tableUrl() + "/tabuleiro/" + tokenID

	if rec := f.pede(t, f.mestre, http.MethodPost, base+"/parada/2/0", ""); rec.Code != http.StatusOK {
		t.Fatalf("a parada do mestre deu %d", rec.Code)
	}
	// O jogador é dono da peça (ela aponta para a ficha dele) e a cena está fora
	// de combate, então ele PODE propor — o que ele não pode é herdar as paradas
	// de outra pessoa.
	if rec := f.pede(t, f.jogador, http.MethodPost, base+"/parada/0/2", ""); rec.Code != http.StatusOK {
		t.Fatalf("a parada do jogador deu %d", rec.Code)
	}
	paradas := boardStops(t, f)
	if len(paradas) != 2 {
		t.Fatalf("o jogador herdou as paradas do mestre: %v", paradas)
	}
	if paradas[0] != (engine.Square{}) {
		t.Errorf("o movimento do jogador começou em %v, esperado o lugar da peça (0,0)", paradas[0])
	}
}
