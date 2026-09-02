package engine

import "testing"

/*
A régua e as faixas de alcance (Tormenta 20, p224 e p236).

O que se prova aqui é o TEXTO do livro nas faixas — curto 9m/6 quadrados, médio
30m/20, longo 90m/60 — e a decisão de mesa na diagonal, que o livro não enuncia
para alcance: vale a régua do movimento (p238), a mesma que desenha o losango na
tela.
*/

// As três faixas, nos números que o livro dá em quadrados (p224).
func TestRangeBandsAtTheBookLimits(t *testing.T) {
	casos := []struct {
		squares int
		quer    RangeBand
	}{
		{0, RangeShort},
		{6, RangeShort},   // 9m: o limite do curto
		{7, RangeMedium},  // um quadrado além já é médio
		{20, RangeMedium}, // 30m: o limite do médio
		{21, RangeLong},
		{60, RangeLong}, // 90m: o limite do longo
		{61, RangeBeyond},
	}
	for _, caso := range casos {
		if got := BandFor(caso.squares); got != caso.quer {
			t.Errorf("%d quadrados caiu em %q, esperado %q", caso.squares, got, caso.quer)
		}
	}
}

// A diagonal custa o DOBRO também para a régua (decisão de mesa sobre a p238):
// quatro quadrados na diagonal são oito de distância, e não quatro.
func TestTheDiagonalDoublesOnTheRuler(t *testing.T) {
	diagonal := Measure(Square{X: 0, Y: 0}, Square{X: 4, Y: 4})

	if diagonal.Squares != 8 {
		t.Errorf("quatro passos na diagonal deram %d quadrados, esperado 8", diagonal.Squares)
	}
	// E isso muda a resposta que a mesa quer: o mesmo alvo estaria no alcance
	// curto pela régua de outros jogos, e aqui está no médio.
	if diagonal.Band != RangeMedium {
		t.Errorf("a diagonal de 4 caiu em %q, esperado médio", diagonal.Band)
	}
}

// Metro é a unidade da conversa na mesa, e a conversão sai do motor para a tela
// não ter uma segunda (p236: 1 quadrado = 1,5m).
func TestTheRulerSaysMetresAlongsideSquares(t *testing.T) {
	reta := Measure(Square{X: 0, Y: 0}, Square{X: 6, Y: 0})

	if reta.Squares != 6 || reta.Metres != 9 {
		t.Errorf("seis quadrados em linha reta deram %d quadrados e %.1fm, esperado 6 e 9,0", reta.Squares, reta.Metres)
	}
	if reta.Band != RangeShort {
		t.Errorf("9m caiu em %q, esperado curto (o limite do curto é 9m)", reta.Band)
	}
}

// Medir de A para B é medir de B para A: a régua não tem dono.
func TestTheRulerHasNoDirection(t *testing.T) {
	ida := Measure(Square{X: -3, Y: 2}, Square{X: 5, Y: -4})
	volta := Measure(Square{X: 5, Y: -4}, Square{X: -3, Y: 2})

	if ida != volta {
		t.Errorf("ida %+v e volta %+v discordam", ida, volta)
	}
}
