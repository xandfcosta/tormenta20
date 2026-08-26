package tabuleiro

import (
	"testing"

	"t20engine/engine"
)

// Os guardas do TRAÇO (ALE-203).
//
// A regra que eles prendem é uma só e é a que o defeito ensinou: **o traço não
// tem buraco**. Um muro de taverna com um quadrado vazio no meio é uma passagem
// que o mestre não desenhou, e ela só aparece na hora em que alguém atravessa.

// TestOTracoNaoTemBuraco.
//
// Casas VIZINHAS de ponta a ponta, no sentido do rei do xadrez: cada casa do
// traço encosta na anterior. É o invariante, e não uma lista esperada escrita à
// mão — uma lista à mão seria a implementação copiada, e ela passaria verde com
// as duas erradas do mesmo jeito.
func TestOTracoNaoTemBuraco(t *testing.T) {
	casos := []struct{ de, ate engine.Square }{
		{engine.Square{}, engine.Square{X: 9, Y: 4}},            // quase horizontal
		{engine.Square{}, engine.Square{X: 4, Y: 9}},            // quase vertical
		{engine.Square{}, engine.Square{X: 6, Y: 6}},            // diagonal exata
		{engine.Square{X: 3, Y: 7}, engine.Square{X: -5, Y: 2}}, // para trás e para o negativo
		{engine.Square{X: 2, Y: 2}, engine.Square{X: 2, Y: 2}},  // parado
	}
	for _, caso := range casos {
		casas := CasasDoTraco(caso.de, caso.ate)
		if casas[0] != caso.de {
			t.Errorf("%v→%v: o traço não começa na casa de origem (%v)", caso.de, caso.ate, casas[0])
		}
		if fim := casas[len(casas)-1]; fim != caso.ate {
			t.Errorf("%v→%v: o traço não chega ao destino (parou em %v)", caso.de, caso.ate, fim)
		}
		for i := 1; i < len(casas); i++ {
			dx, dy := abs(casas[i].X-casas[i-1].X), abs(casas[i].Y-casas[i-1].Y)
			if dx > 1 || dy > 1 || dx+dy == 0 {
				t.Errorf("%v→%v: buraco entre %v e %v — o muro tem passagem",
					caso.de, caso.ate, casas[i-1], casas[i])
			}
		}
	}
}

// TestOTracoNaoAndaNaDiagonalQuandoRoCA.
//
// O CASO que motivou a supercobertura: num traço quase horizontal, o Bresenham
// clássico troca de linha ANDANDO NA DIAGONAL, e a casa roçada não entra. O
// guarda acima já recusaria o pulo diagonal (ele exige `dx+dy` de exatamente um
// passo em UM eixo? não — ele aceita a diagonal), então este aqui é o que separa
// os dois algoritmos: nenhum passo do traço mexe nos DOIS eixos ao mesmo tempo.
func TestOTracoNaoAndaNaDiagonalQuandoRoca(t *testing.T) {
	casas := CasasDoTraco(engine.Square{}, engine.Square{X: 2, Y: 1})
	for i := 1; i < len(casas); i++ {
		dx, dy := abs(casas[i].X-casas[i-1].X), abs(casas[i].Y-casas[i-1].Y)
		if dx == 1 && dy == 1 {
			t.Errorf("o traço pulou na diagonal de %v para %v: a casa roçada ficou vazia",
				casas[i-1], casas[i])
		}
	}
	// O CONTROLE: sem isto, um `CasasDoTraco` que devolvesse só a origem passaria
	// no laço acima sobre uma lista de um item.
	if len(casas) != 4 {
		t.Errorf("o traço (0,0)→(2,1) tem %d casas, esperado 4 — %v", len(casas), casas)
	}
}

// TestOTracoPossuidoERecusado: o teto existe contra o pedido forjado, não contra
// o dedo. Num quadro de 16ms nenhum gesto atravessa cem casas.
func TestOTracoPossuidoERecusado(t *testing.T) {
	if TracoValido(engine.Square{}, engine.Square{X: 9999999}) {
		t.Error("um traço de dez milhões de casas foi aceito")
	}
	if !TracoValido(engine.Square{}, engine.Square{X: 9, Y: 4}) {
		t.Error("um traço de nove casas foi recusado — o teto está mordendo o gesto real")
	}
}
