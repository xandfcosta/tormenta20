package api

import (
	"context"
	"net/http"
	"strings"
	"t20engine/tabuleiro"
	"testing"
)

// TestThePlayerTemplateDoesNotCountTheHiddenToken — o guarda que mais importa.
//
// Esconder a peça é o gesto com que o mestre guarda a emboscada. Um gabarito que
// respondesse "Pega 2 peças: Arwen, Ogro" entregaria a emboscada pela porta dos
// fundos: a peça não está desenhada, mas o NOME dela chega ao HTML do jogador —
// e nesta superfície ele chega até por SINAL, que é mais fácil de ler que o DOM.
//
// A trava é passar pelo mesmo `BoardForRole` do resto da Mesa, e não uma segunda
// decisão sobre quem vê o quê.
func TestThePlayerTemplateDoesNotCountTheHiddenToken(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "cripta")
	if _, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab,
		tabuleiro.BoardToken{ID: "emboscada", Label: "Ogro emboscado", X: 4, Y: 4, Hidden: true}, true); err != nil {
		t.Fatalf("pôr a peça escondida: %v", err)
	}

	// Um quadrado de lado 1 exatamente em cima dela.
	caminho := f.tableUrl() + "/tabuleiro/gabarito/quadrado/1/4/4/4/4"
	doMestre := f.posta(t, f.mestre, caminho, "")
	if !strings.Contains(doMestre, "Ogro emboscado") {
		t.Fatalf("o MESTRE não viu a própria peça: %s\n— sem o caso positivo o resto não mede nada", doMestre)
	}
	doJogador := f.posta(t, f.jogador, caminho, "")
	if strings.Contains(doJogador, "Ogro emboscado") {
		t.Errorf("a emboscada vazou no gabarito do jogador: %s", doJogador)
	}
	if !strings.Contains(doJogador, "Ninguém dentro") {
		t.Errorf("o jogador recebeu %q, esperado a área vazia", doJogador)
	}
}

// TestMeasuringDoesNotPatchTheScene.
//
// A régua não muda a cena, e a resposta dela tem de ser do tamanho disso. Uma
// medição que devolvesse as nove regiões trocaria o mapa debaixo de quem está
// medindo — a peça sob o dedo de quem arrasta some e volta —, que é o mesmo
// defeito que a região `mesa-por-no-mapa` já existe para evitar.
//
// Provado VERMELHO trocando o `writeSignals` pelo `respondGm`: a
// resposta passou a trazer `mesa-tabuleiro` e este teste acusou.
func TestMeasuringDoesNotPatchTheScene(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")

	// As paradas vêm nos SINAIS desde a ALE-203: com número variável de pernas,
	// um caminho com as pontas dentro seria uma rota que muda de forma.
	resposta := f.posta(t, f.mestre, f.tableUrl()+"/tabuleiro/regua",
		`{"reguapontos":[[0,0],[3,0]],"reguafase":2}`)
	if !strings.Contains(resposta, "reguatexto") {
		t.Fatalf("a medida não voltou: %s", resposta)
	}
	if strings.Contains(resposta, "mesa-tabuleiro") || strings.Contains(resposta, "datastar-patch-elements") {
		t.Errorf("medir remendou a cena inteira:\n%s", resposta)
	}
}

// TestTheTemplateRefusesAShapeTheBookDoesNotHave.
//
// O id vem do CLIENTE, e uma forma inventada não pode virar um desenho — nem
// cair calada na esfera, que desenharia uma área que ninguém pediu no lugar de
// dizer que o pedido está errado.
func TestTheTemplateRefusesAShapeTheBookDoesNotHave(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")
	rec := f.pede(t, f.mestre, http.MethodPost, f.tableUrl()+"/tabuleiro/gabarito/piramide/2/0/0/0/0", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("forma inventada deu %d, esperado 400", rec.Code)
	}
	// A mensagem diz o valor recebido E a lista do que existe: quem lê o erro
	// precisa saber o que digitar em vez do que digitou.
	if corpo := rec.Body.String(); !strings.Contains(corpo, "piramide") || !strings.Contains(corpo, "esfera") {
		t.Errorf("a recusa saiu %q, sem o valor recusado ou sem a lista", corpo)
	}
}

// TestWhoIsNotAtTheTableDoesNotMeasureItsScene.
//
// Medir é de todo mundo que joga — "dá para acertar daqui?" é pergunta de quem
// ataca —, e por isso a rota não exige o papel de mestre. A trava que sobra é a
// de sempre, e ela é do SERVIDOR: o gabarito devolve os NOMES das peças, então
// uma rota aberta seria a lista do bestiário da cena para quem tiver a URL.
func TestWhoIsNotAtTheTableDoesNotMeasureItsScene(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")
	estranho := seedUser(t, f.s, "estranho@t.com")

	rec := f.pede(t, estranho, http.MethodPost, f.tableUrl()+"/tabuleiro/regua/0/0/3/0", "")
	if rec.Code == http.StatusOK {
		t.Errorf("quem não está na mesa mediu a cena dela: %s", rec.Body.String())
	}
}

// TestTheRailOffersTheRulerToThePlayer.
//
// Antes desta fatia o trilho inteiro era do mestre, porque só ele tinha modo —
// pintar e marcar. A régua é de quem ataca, e a cena do jogador não desenhava
// trilho nenhum: a ferramenta existiria e não teria onde ser ligada.
//
// E o que continua sendo do mestre segue sendo: o pincel de terreno pinta a
// cena, e a trava de verdade é a rota (o `gmBoardCommand`) — isto
// aqui é a cortesia de não oferecer o que seria recusado.
func TestTheRailOffersTheRulerToThePlayer(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")

	corpo := f.pede(t, f.jogador, http.MethodGet, f.tableUrl(), "").Body.String()
	for _, esperado := range []string{"Régua", "Gabarito", "Mover a peça"} {
		if !strings.Contains(corpo, esperado) {
			t.Errorf("a cena do jogador não ofereceu %q", esperado)
		}
	}
	if strings.Contains(corpo, "Borracha") {
		t.Error("o pincel do mestre apareceu na cena do jogador")
	}

	doMestre := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(doMestre, "Borracha") {
		t.Error("o mestre perdeu o pincel — sem o caso positivo o de cima não mede nada")
	}
}
