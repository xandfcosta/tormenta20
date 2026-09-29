package board

import (
	"testing"

	"t20engine/domain/engine"
)

// engineSquare encurta o quadrado do livro nos casos abaixo.
func engineSquare(x, y int) engine.Square { return engine.Square{X: x, Y: y} }

// A REDAÇÃO PRESERVA A ORDEM do que sobrou (ALE-413).
//
// # Por que ela é regra, e não arrumação
//
// A ordem da lista de peças é a ordem em que a mesa as DESENHA: quem vem depois
// fica por cima. E é ela que decide qual peça a tela oferece para mover — o
// `reachAndTarget` pega *"a primeira peça que quem olha pode mover"*.
// Embaralhar a lista troca quem está na frente e troca o alvo do gesto, sem
// nenhum erro no caminho.
//
// # Ela não estava presa, e foi medido
//
// Invertendo a ordem em que a redação visita as peças, a suíte inteira ficava
// VERDE — `domain/board`, `serve/api` e `serve/web/table`. Nenhum caso afirmava
// a ordem porque todos afirmavam CONTEÚDO, e conteúdo é o que sobrevive a um
// embaralhamento.
//
// É a propriedade em volta da qual o `domain/ecs` foi desenhado — o
// armazenamento guarda uma FATIA na ordem de inserção porque `map` em Go não a
// tem —, e ela passou a valer aqui no dia em que a redação virou mundo.
//
// # Quantas peças o caso põe na tela
//
// Cinco, com duas escondidas ENTRE as visíveis. Com uma só, a ordem certa e a
// errada dão o mesmo resultado; com as escondidas nas pontas, uma inversão
// ainda deixaria o miolo plausível.
func TestTheRedactionKeepsTheDrawingOrder(t *testing.T) {
	b := &BoardState{Tokens: []BoardToken{
		{ID: "t1", Label: "Bandido"},
		{ID: "t2", Label: "Assassino na viga", Hidden: true},
		{ID: "t3", Label: "Arwen"},
		{ID: "t4", Label: "Armadilha viva", Hidden: true},
		{ID: "t5", Label: "Ogro"},
	}}

	visto := BoardForRole("player", b)
	quero := []string{"t1", "t3", "t5"}
	if len(visto.Tokens) != len(quero) {
		t.Fatalf("o jogador recebeu %d peças e as visíveis são %d: %+v",
			len(visto.Tokens), len(quero), visto.Tokens)
	}
	for i, id := range quero {
		if visto.Tokens[i].ID != id {
			t.Errorf("a peça %d do jogador é %q e devia ser %q.\n"+
				"A ordem da lista é a ordem de DESENHO — quem vem depois fica por "+
				"cima —, e é ela que decide qual peça a tela oferece para mover.",
				i, visto.Tokens[i].ID, id)
		}
	}

	// O MESTRE VÊ TUDO, e na mesma ordem: sem esta metade, uma redação que
	// devolvesse a lista original embaralhada passaria, porque o jogador só vê
	// três.
	doMestre := BoardForRole("gm", b)
	for i := range b.Tokens {
		if doMestre.Tokens[i].ID != b.Tokens[i].ID {
			t.Errorf("a peça %d do mestre é %q e devia ser %q",
				i, doMestre.Tokens[i].ID, b.Tokens[i].ID)
		}
	}
}

// E A ORDEM DOS MARCADORES também, pela mesma razão: dois marcadores no mesmo
// quadrado desenham um sobre o outro.
func TestTheRedactionKeepsTheMarkerOrder(t *testing.T) {
	b := &BoardState{Markers: []BoardMarker{
		{ID: "m1", Text: "A"},
		{ID: "m2", Text: "armadilha", Hidden: true},
		{ID: "m3", Text: "B"},
		{ID: "m4", Text: "C"},
	}}

	visto := BoardForRole("player", b)
	quero := []string{"m1", "m3", "m4"}
	if len(visto.Markers) != len(quero) {
		t.Fatalf("a mesa recebeu %d marcadores e os visíveis são %d: %+v",
			len(visto.Markers), len(quero), visto.Markers)
	}
	for i, id := range quero {
		if visto.Markers[i].ID != id {
			t.Errorf("o marcador %d é %q e devia ser %q", i, visto.Markers[i].ID, id)
		}
	}
}

// A PEÇA GRAVADA SEM TAMANHO OCUPA UM QUADRADO, e não nenhum (ALE-413).
//
// O `AddToken` põe 1 em quem chega com zero, então nenhuma peça CRIADA pelo app
// tem tamanho zero. Mas o tabuleiro vem do banco como JSON, e linha gravada
// antes de o campo existir chega com o zero do Go — e o corpo dela sairia
// VAZIO, cobrindo quadrado nenhum.
//
// A consequência é silenciosa e visível: a próxima peça nasce em cima dela, e o
// mapa mostra uma sobre a outra sem ninguém ter errado um clique.
//
// Não estava preso: trocando o `side <= 0` por `side < 0`, a suíte inteira
// ficava verde — nenhum caso põe uma peça sem tamanho no mapa, porque todos os
// casos a criam pelo caminho que já a conserta.
func TestATokenSavedWithoutASizeStillTakesItsSquare(t *testing.T) {
	// Direto na fatia, e não pelo `AddToken`: é assim que ela chega do banco, e
	// pelo construtor o defeito é invisível por construção.
	b := &BoardState{Tokens: []BoardToken{{ID: "antiga", X: 4, Y: 4}}}

	plano := occupancyWorldOf(b)
	if !squareIsTaken(plano, 4, 4) {
		t.Fatal("o quadrado (4,4) saiu livre com uma peça em cima dele.\n" +
			"Peça gravada sem `footprint` chega com zero, e o corpo dela tem de " +
			"valer UM quadrado — senão a próxima nasce empilhada.")
	}
	if ids := TokensInRectangle(b, engineSquare(4, 4), engineSquare(4, 4)); len(ids) != 1 {
		t.Errorf("o laço em (4,4) pegou %v, e a peça sem tamanho está ali", ids)
	}
}
