package api

import (
	"context"
	"net/http"
	"strings"
	"t20engine/domain/board"
	"testing"
)

// O guarda que mais importa desta superfície.
//
// Esconder a peça é o gesto com que o mestre guarda a emboscada. Um gabarito que
// respondesse "Pega 2 peças: Arwen, Ogro" entregaria a emboscada pela porta dos
// fundos: a peça não está desenhada, mas o NOME dela chega ao HTML do jogador —
// e nesta superfície ele chega até por SINAL, que é mais fácil de ler que o DOM.
//
// A trava é passar pelo mesmo `BoardForRole` do resto da Mesa, e não uma segunda
// decisão sobre quem vê o quê.
func TestThePlayerTemplateDoesNotCountTheHiddenToken(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "crypt")
	if _, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab,
		board.BoardToken{ID: "emboscada", Label: "Ogro emboscado", X: 4, Y: 4, Hidden: true}); err != nil {
		t.Fatalf("pôr a peça escondida: %v", err)
	}

	// Um quadrado de lado 1 exatamente em cima dela.
	path := f.tableUrl() + "/tabuleiro/gabarito"
	square := templateBody("quadrado", "1", 4, 4, 4, 4)
	forGM := f.posta(t, f.gm, path, square)
	if !strings.Contains(forGM, "Ogro emboscado") {
		t.Fatalf("o MESTRE não viu a própria peça: %s\n— sem o caso positivo o resto não mede nada", forGM)
	}
	forPlayer := f.posta(t, f.player, path, square)
	if strings.Contains(forPlayer, "Ogro emboscado") {
		t.Errorf("a emboscada vazou no gabarito do jogador: %s", forPlayer)
	}
	if !strings.Contains(forPlayer, "Ninguém dentro") {
		t.Errorf("o jogador recebeu %q, esperado a área vazia", forPlayer)
	}
}

// A régua não remenda a cena.
//
// A régua não muda a cena, e a resposta dela tem de ser do tamanho disso. Uma
// medição que devolvesse as nove regiões trocaria o mapa debaixo de quem está
// medindo — a peça sob o dedo de quem arrasta some e volta —, que é o mesmo
// defeito que a região `table-populate` já existe para evitar.
//
// Provado VERMELHO trocando o `writeSignals` pelo `respondGm`: a
// resposta passou a trazer `table-board` e este teste acusou.
func TestMeasuringDoesNotPatchTheScene(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")

	// As paradas vêm nos SINAIS: com número variável de pernas, um caminho com
	// as pontas dentro seria uma rota que muda de forma.
	response := f.posta(t, f.gm, f.tableUrl()+"/tabuleiro/regua",
		`{"ruler_points":[[0,0],[3,0]],"ruler_phase":2}`)
	if !strings.Contains(response, "ruler_text") {
		t.Fatalf("a medida não voltou: %s", response)
	}
	if strings.Contains(response, "table-board") || strings.Contains(response, "datastar-patch-elements") {
		t.Errorf("medir remendou a cena inteira:\n%s", response)
	}
}

// O gabarito recusa uma forma que o livro não tem.
//
// O id vem do CLIENTE, e uma forma inventada não pode virar um desenho — nem
// cair calada na esfera, que desenharia uma área que ninguém pediu no lugar de
// dizer que o pedido está errado.
func TestTheTemplateRefusesAShapeTheBookDoesNotHave(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	rec := f.pede(t, f.gm, http.MethodPost, f.tableUrl()+"/tabuleiro/gabarito", templateBody("piramide", "2", 0, 0, 0, 0))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("forma inventada deu %d, esperado 400", rec.Code)
	}
	// A mensagem diz o valor recebido E a lista do que existe: quem lê o erro
	// precisa saber o que digitar em vez do que digitou.
	if body := rec.Body.String(); !strings.Contains(body, "piramide") || !strings.Contains(body, "esfera") {
		t.Errorf("a recusa saiu %q, sem o valor recusado ou sem a lista", body)
	}
}

// Quem não está na mesa não mede a cena dela.
//
// Medir é de todo mundo que joga — "dá para acertar daqui?" é pergunta de quem
// ataca —, e por isso a rota não exige o papel de mestre. A trava que sobra é a
// de sempre, e ela é do SERVIDOR: o gabarito devolve os NOMES das peças, então
// uma rota aberta seria a lista do bestiário da cena para quem tiver a URL.
//
// O 403 é afirmado EXATO, e não como `!= 200`: postando num endereço que já não
// existe, um caso frouxo lê o 404 do chi como recusa e fica verde com a trava de
// autorização INTEIRA removida. O CONTROLE ao lado é o que impede a forma de
// voltar — quem está na mesa mede pelo MESMO endereço e recebe 200.
func TestWhoIsNotAtTheTableDoesNotMeasureItsScene(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	stranger := seedUser(t, f.s, "estranho@t.com")

	path := f.tableUrl() + "/tabuleiro/regua"
	stops := `{"ruler_points":[[0,0],[3,0]],"ruler_phase":2}`

	rec := f.pede(t, stranger, http.MethodPost, path, stops)
	if rec.Code != http.StatusForbidden {
		t.Errorf("quem não está na mesa recebeu %d, quero 403: %s", rec.Code, rec.Body.String())
	}
	// O CONTROLE. Ele responde a pergunta que o `!= 200` não respondia: o canal
	// existe, e o 403 acima é uma RECUSA e não um endereço que não casa.
	if rec := f.pede(t, f.player, http.MethodPost, path, stops); rec.Code != http.StatusOK {
		t.Fatalf("quem ESTÁ na mesa recebeu %d no mesmo endereço — o caso de cima passou a medir nada", rec.Code)
	}
}

// A régua é de quem ATACA, e por isso o jogador ganha trilho: sem ele a
// ferramenta existiria e não teria onde ser ligada.
//
// E o que continua sendo do mestre segue sendo: o pincel de terreno pinta a
// cena, e a trava de verdade é a rota (o `gmBoardCommand`) — isto
// aqui é a cortesia de não oferecer o que seria recusado.
func TestTheRailOffersTheRulerToThePlayer(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")

	body := f.pede(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()
	for _, want := range []string{"Régua", "Gabarito", "Mover a peça"} {
		if !strings.Contains(body, want) {
			t.Errorf("a cena do jogador não ofereceu %q", want)
		}
	}
	if strings.Contains(body, "Borracha") {
		t.Error("o pincel do mestre apareceu na cena do jogador")
	}

	forGM := f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(forGM, "Borracha") {
		t.Error("o mestre perdeu o pincel — sem o caso positivo o de cima não mede nada")
	}
}
