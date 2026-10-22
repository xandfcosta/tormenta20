package board

import (
	"testing"

	"t20engine/domain/engine"
)

// QUEM IGNORA O TERRENO DIFÍCIL ANDA NUM MAPA SEM ELE (p129, ALE-431).
//
//	"Acrobático. […] terreno difícil não reduz seu deslocamento nem o impede
//	 de realizar investidas."
//
// O `PathCost` não conhece personagem nenhum, e isso é desenho e não limitação:
// ele recebe o TERRENO, e "não reduz seu deslocamento" é, para ele, um terreno
// sem casa difícil. Dar personagem ao cálculo do caminho faria do tabuleiro um
// resolvedor tático, que é a linha que a casa não cruza (ALE-423).
//
// A propriedade viaja na PEÇA, como o `SpeedSquares` já viaja: o motor a
// computa da ficha e o tabuleiro a guarda, re-sincronizada junto com o
// deslocamento.
func TestWhoIgnoresDifficultTerrainWalksOnAMapWithoutIt(t *testing.T) {
	hard := engine.Square{X: 1, Y: 0}
	b := &BoardState{Difficult: []engine.Square{hard}}

	// O CONTROLE: o mapa TEM terreno difícil, e quem não é isento o paga. Sem
	// ele, um tabuleiro que perdesse a pintura passaria verde no caso de baixo.
	comum := &BoardToken{ID: "comum"}
	if terrain := moveTerrainFor(b, comum); !terrain.Difficult[hard] {
		t.Fatalf("o controle já estava errado: a casa (1,0) está pintada e o terreno "+
			"de quem anda não a cobra — %+v", terrain)
	}

	acrobatico := &BoardToken{ID: "acrobatico", IgnoresDifficult: true}
	if terrain := moveTerrainFor(b, acrobatico); len(terrain.Difficult) != 0 {
		t.Errorf("quem ignora terreno difícil recebeu %d casas difíceis, e a p129 diz "+
			"que ele não paga nenhuma", len(terrain.Difficult))
	}

	// E O CUSTO É O QUE MUDA, que é a razão de tudo isto existir: um passo em
	// casa difícil custa DOIS quadrados (p238) e para ele custa um.
	path := []engine.Square{{X: 0, Y: 0}, hard}
	pago := engine.PathCost(path, moveTerrainFor(b, comum), -1)
	isento := engine.PathCost(path, moveTerrainFor(b, acrobatico), -1)
	if pago.Squares != 2 {
		t.Errorf("o passo em terreno difícil custou %d quadrados a quem o paga, e a "+
			"p238 cobra 2", pago.Squares)
	}
	if isento.Squares != 1 {
		t.Errorf("o mesmo passo custou %d quadrados ao Acrobático, e ele anda como se "+
			"a casa fosse limpa: 1", isento.Squares)
	}
	if isento.Difficult != 0 {
		t.Errorf("a conta do Acrobático contou %d casas difíceis — ela é o que a tela "+
			"escreve ao lado da seta, e dizer 'uma difícil' num caminho que não "+
			"custou nada é a prévia contradizendo o custo", isento.Difficult)
	}
}

// E A PEÇA SEM MOVEDOR CONHECIDO PAGA, que é o recuo seguro.
//
// `nil` acontece em leitura — a tela que desenha o tabuleiro sem ninguém
// movendo —, e ali o terreno é o do MAPA. Isentar por omissão faria o desenho
// prometer um caminho barato a quem vai pagar caro.
func TestAnUnknownMoverPaysTheDifficultTerrain(t *testing.T) {
	hard := engine.Square{X: 2, Y: 2}
	b := &BoardState{Difficult: []engine.Square{hard}}
	if terrain := moveTerrainFor(b, nil); !terrain.Difficult[hard] {
		t.Error("sem movedor conhecido o terreno saiu limpo: o recuo tem de ser o " +
			"mapa, e não a isenção")
	}
}
