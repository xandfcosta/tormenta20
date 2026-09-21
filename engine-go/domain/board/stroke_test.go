package board

import (
	"testing"

	"t20engine/domain/engine"
)

// Os guardas do TRAÇO.
//
// A regra que eles prendem é uma só e é a que o defeito ensinou: **o traço não
// tem buraco**. Um muro de taverna com um quadrado vazio no meio é uma passagem
// que o mestre não desenhou, e ela só aparece na hora em que alguém atravessa.

// Casas VIZINHAS de ponta a ponta, no sentido do rei do xadrez: cada casa do
// traço encosta na anterior. É o invariante, e não uma lista esperada escrita à
// mão — uma lista à mão seria a implementação copiada, e ela passaria verde com
// as duas erradas do mesmo jeito.
func TestTheStrokeHasNoGap(t *testing.T) {
	cases := []struct{ de, ate engine.Square }{
		{engine.Square{}, engine.Square{X: 9, Y: 4}},            // quase horizontal
		{engine.Square{}, engine.Square{X: 4, Y: 9}},            // quase vertical
		{engine.Square{}, engine.Square{X: 6, Y: 6}},            // diagonal exata
		{engine.Square{X: 3, Y: 7}, engine.Square{X: -5, Y: 2}}, // para trás e para o negativo
		{engine.Square{X: 2, Y: 2}, engine.Square{X: 2, Y: 2}},  // parado
	}
	for _, tc := range cases {
		squares := StrokeSquares(tc.de, tc.ate)
		if squares[0] != tc.de {
			t.Errorf("%v→%v: o traço não começa na casa de origem (%v)", tc.de, tc.ate, squares[0])
		}
		if end := squares[len(squares)-1]; end != tc.ate {
			t.Errorf("%v→%v: o traço não chega ao destino (parou em %v)", tc.de, tc.ate, end)
		}
		for i := 1; i < len(squares); i++ {
			dx, dy := abs(squares[i].X-squares[i-1].X), abs(squares[i].Y-squares[i-1].Y)
			if dx > 1 || dy > 1 || dx+dy == 0 {
				t.Errorf("%v→%v: buraco entre %v e %v — o muro tem passagem",
					tc.de, tc.ate, squares[i-1], squares[i])
			}
		}
	}
}

// Num traço quase horizontal, o Bresenham clássico troca de linha ANDANDO NA
// DIAGONAL, e a casa roçada não entra. O guarda acima ACEITA a diagonal, então é
// este que separa os dois algoritmos: nenhum passo mexe nos DOIS eixos ao mesmo
// tempo.
func TestTheStrokeDoesNotGoDiagonalWhenItGrazes(t *testing.T) {
	squares := StrokeSquares(engine.Square{}, engine.Square{X: 2, Y: 1})
	for i := 1; i < len(squares); i++ {
		dx, dy := abs(squares[i].X-squares[i-1].X), abs(squares[i].Y-squares[i-1].Y)
		if dx == 1 && dy == 1 {
			t.Errorf("o traço pulou na diagonal de %v para %v: a casa roçada ficou vazia",
				squares[i-1], squares[i])
		}
	}
	// O CONTROLE: sem isto, um `StrokeSquares` que devolvesse só a origem passaria
	// no laço acima sobre uma lista de um item.
	if len(squares) != 4 {
		t.Errorf("o traço (0,0)→(2,1) tem %d casas, esperado 4 — %v", len(squares), squares)
	}
}

// O teto existe contra o pedido FORJADO, não contra o dedo: num quadro de 16ms
// nenhum gesto atravessa cem casas.
//
// # A FRONTEIRA, e não "absurdo é recusado"
//
// Com só um `9999999` contra o teto de 100, entre nove e dez milhões cabe
// qualquer coisa: **trocar o `strokeFits` para 10 passa verde**. Prender a
// fronteira é o que protege o TETO, e não só o pedido forjado.
//
// O predicado é `max(|dx|,|dy|) < strokeFits`, então o teto é de EXTENSÃO e não de
// contagem de casas: 99 de extensão passa, 100 não. Os números estão escritos à
// mão de propósito; derivá-los de `strokeFits` faria a asserção andar junto com o
// defeito.
func TestAPossessedStrokeIsRefused(t *testing.T) {
	if ValidStroke(engine.Square{}, engine.Square{X: 9999999}) {
		t.Error("um traço de dez milhões de casas foi aceito")
	}
	if !ValidStroke(engine.Square{}, engine.Square{X: 9, Y: 4}) {
		t.Error("um traço de nove casas foi recusado — o teto está mordendo o gesto real")
	}
	// A FRONTEIRA: 99 de extensão é o último que passa.
	if !ValidStroke(engine.Square{}, engine.Square{X: 99}) {
		t.Error("um traço de extensão 99 foi recusado, e o teto é 100")
	}
	if ValidStroke(engine.Square{}, engine.Square{X: 100}) {
		t.Error("um traço de extensão 100 foi aceito, e o teto é 100 — o predicado é `< 100`")
	}
	// E o teto é do MAIOR eixo, não da soma: um traço quase-diagonal de 99 por 99
	// passa, porque o dedo que o fez andou 99 casas e não 198.
	if !ValidStroke(engine.Square{}, engine.Square{X: 99, Y: 99}) {
		t.Error("um traço diagonal de 99×99 foi recusado — o teto mede o maior eixo")
	}
}

// A REGRA: arrastar da direita para a esquerda, ou de baixo para cima, desenha o
// MESMO retângulo — porque é o que o dedo faz. Sem o `min`/`max` um arrasto "para
// trás" devolveria vazio, e a pessoa concluiria que a ferramenta falha às vezes,
// que é a pior forma de falhar.
func TestTheRectangleIsTheSameInAllFourDirections(t *testing.T) {
	a, b := engine.Square{X: 0, Y: 0}, engine.Square{X: 2, Y: 1}
	reference := RectangleSquares(a, b)
	if len(reference) != 6 {
		t.Fatalf("(0,0)→(2,1) deu %d casas, esperado 6 — o guarda mediria o vazio", len(reference))
	}
	for _, par := range [][2]engine.Square{
		{b, a},
		{{X: 2, Y: 0}, {X: 0, Y: 1}},
		{{X: 0, Y: 1}, {X: 2, Y: 0}},
	} {
		if other := RectangleSquares(par[0], par[1]); len(other) != len(reference) {
			t.Errorf("%v→%v deu %d casas, e %v→%v deu %d: a direção do arrasto mudou o retângulo",
				par[0], par[1], len(other), a, b, len(reference))
		}
	}
}
