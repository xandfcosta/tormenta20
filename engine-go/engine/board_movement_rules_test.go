package engine

import "testing"

// Movimento no mapa de batalha — a regra vem do livro e o teste cita a página
// (ALE-124). O que se prova aqui é aritmética de mesa: o mestre e o jogador
// discordarem sobre "cabe ou não cabe" é a discussão que o app existe para
// encerrar.

func path(steps ...Square) []Square { return steps }

// p106: "seu deslocamento é 9 metros (6 quadrados no mapa)". É o exemplo
// trabalhado do próprio livro, e o caso mais comum da mesa inteira.
func TestSixOrthogonalStepsFitInNineMetres(t *testing.T) {
	custo := PathCost(
		path(Square{0, 0}, Square{1, 0}, Square{2, 0}, Square{3, 0}, Square{4, 0}, Square{5, 0}, Square{6, 0}),
		MoveTerrain{}, SquaresForDisplacement(9),
	)

	if custo.Squares != 6 || !custo.Legal {
		t.Errorf("seis passos retos com deslocamento 9m: %+v", custo)
	}
	if custo.StoppedAt != -1 {
		t.Errorf("caminho que coube marcou parada em %d", custo.StoppedAt)
	}
}

// p238: "mover-se na diagonal custa o dobro. Ou seja, andar 1,5m (1 quadrado)
// na diagonal conta como 3m (2 quadrados)". Esta é a asserção que pega quem
// portar por hábito a alternância 1-2-1 de outros d20 — que o livro NÃO tem.
func TestDiagonalStepCostsTwoSquaresAlways(t *testing.T) {
	uma := PathCost(path(Square{0, 0}, Square{1, 1}), MoveTerrain{}, -1)
	if uma.Squares != 2 {
		t.Errorf("uma diagonal custou %d quadrados, esperado 2", uma.Squares)
	}

	// Três diagonais seguidas custam 6, não 4 nem 5: não há desconto na
	// segunda, que é exatamente o que a alternância faria.
	tres := PathCost(path(Square{0, 0}, Square{1, 1}, Square{2, 2}, Square{3, 3}), MoveTerrain{}, 6)
	if tres.Squares != 6 || !tres.Legal {
		t.Errorf("três diagonais com deslocamento 9m: %+v", tres)
	}

	quatro := PathCost(
		path(Square{0, 0}, Square{1, 1}, Square{2, 2}, Square{3, 3}, Square{4, 4}), MoveTerrain{}, 6,
	)
	if quatro.Legal {
		t.Error("a quarta diagonal (8 quadrados) coube num deslocamento de 6")
	}
	// Onde estourou importa: a tela pinta o trecho recusado em vez de recusar
	// o caminho inteiro.
	if quatro.StoppedAt != 4 {
		t.Errorf("estourou no passo %d, esperado o quarto", quatro.StoppedAt)
	}
}

// p238: terreno difícil "gasta 3m de deslocamento por quadrado, em vez de 1,5m".
func TestDifficultTerrainDoublesTheStep(t *testing.T) {
	lama := MoveTerrain{Difficult: map[Square]bool{{X: 1, Y: 0}: true}}

	custo := PathCost(path(Square{0, 0}, Square{1, 0}, Square{2, 0}), lama, -1)

	// Um passo na lama (2) + um no chão limpo (1).
	if custo.Squares != 3 {
		t.Errorf("dois passos, um deles em terreno difícil: %d quadrados, esperado 3", custo.Squares)
	}
	// O custo é de ENTRAR: sair da lama para o chão limpo custa o do chão limpo.
	saindo := PathCost(path(Square{1, 0}, Square{2, 0}), lama, -1)
	if saindo.Squares != 1 {
		t.Errorf("sair do terreno difícil custou %d, esperado 1", saindo.Squares)
	}
}

// O livro dobra a diagonal (p238) e dobra o terreno difícil (p238) em frases
// SEPARADAS, e nunca compõe as duas. A leitura desta casa é multiplicativa —
// decisão de mesa registrada, não texto do livro. Está fixado aqui para que
// mudar de ideia seja uma mudança visível, e não uma descoberta na mesa.
func TestDiagonalIntoDifficultTerrainCostsFour(t *testing.T) {
	mato := MoveTerrain{Difficult: map[Square]bool{{X: 1, Y: 1}: true}}

	custo := PathCost(path(Square{0, 0}, Square{1, 1}), mato, -1)

	if custo.Squares != 4 {
		t.Errorf("diagonal entrando em terreno difícil custou %d quadrados, esperado 4 (6m)", custo.Squares)
	}
}

// Caminho tem de ser quadrado a quadrado: um salto no meio significa que o
// cliente mandou lixo, e aceitar isso mediria uma distância que ninguém andou.
func TestPathMustBeContiguous(t *testing.T) {
	custo := PathCost(path(Square{0, 0}, Square{4, 0}), MoveTerrain{}, 6)

	if custo.Legal {
		t.Error("um salto de quatro quadrados passou como caminho")
	}
	// A mensagem carrega os valores ofendidos — quem recebe precisa saber o que
	// mandou de errado.
	if custo.Reason == "" {
		t.Error("caminho recusado sem dizer por quê")
	}
}

func TestSquaresForDisplacementTruncates(t *testing.T) {
	// p106: 9m = 6 quadrados. É a linha trabalhada do livro.
	if got := SquaresForDisplacement(9); got != 6 {
		t.Errorf("9m viraram %d quadrados, esperado 6", got)
	}
	// 10m não compra um sétimo quadrado: o sétimo custa 1,5m inteiros.
	if got := SquaresForDisplacement(10); got != 6 {
		t.Errorf("10m viraram %d quadrados, esperado 6", got)
	}
	// Sobrecarga tira 3m (p238): 9 − 3 = 6m = 4 quadrados.
	if got := SquaresForDisplacement(6); got != 4 {
		t.Errorf("6m viraram %d quadrados, esperado 4", got)
	}
	if got := SquaresForDisplacement(0); got != 0 {
		t.Errorf("quem não anda recebeu %d quadrados", got)
	}
}

// p107, Tab. 1-21. Fixa só o que é EXCEÇÃO à regra "ocupa um quadrado" — os três
// tamanhos que ocupam 1×1 são o default, e repeti-los seria transcrever tabela.
func TestFootprintFollowsTheSizeTable(t *testing.T) {
	casos := map[string]int{
		"Grande":   2,
		"Enorme":   3,
		"Colossal": 6,
		// As duas grafias do projeto convergem: o bestiário guarda "medio",
		// a ficha guarda "Médio", e uma peça de tamanho errado no tabuleiro
		// erraria alcance e ocupação de quadrado.
		"medio":  1,
		"Médio":  1,
		"grande": 2,
	}
	for size, esperado := range casos {
		if got := FootprintForSize(size); got != esperado {
			t.Errorf("%q ocupou %d quadrados de lado, esperado %d", size, got, esperado)
		}
	}
	// Tamanho desconhecido cai em 1: uma peça sem tamanho é Média, nunca some.
	if got := FootprintForSize("inventado"); got != 1 {
		t.Errorf("tamanho desconhecido virou %d", got)
	}
}

// O alcance é um LOSANGO, não um quadrado — e essa é a consequência VISÍVEL da
// diagonal dobrada (T20 p238): com 6 quadrados de deslocamento (9m, p106)
// dá para andar 6 em linha reta e só 3 na diagonal.
func TestAlcanceEUmLosangoPorCausaDaDiagonal(t *testing.T) {
	reach := ReachableSquares(Square{X: 0, Y: 0}, 6, MoveTerrain{})

	dentro := map[Square]bool{}
	for _, s := range reach {
		dentro[s] = true
	}

	if !dentro[Square{X: 6, Y: 0}] {
		t.Error("seis quadrados em linha reta não alcançaram: o deslocamento de 9m anda 6 (p106)")
	}
	if !dentro[Square{X: 3, Y: 3}] {
		t.Error("três diagonais custam 6 e deveriam caber no orçamento de 6")
	}
	if dentro[Square{X: 4, Y: 4}] {
		t.Error("quatro diagonais custam 8 e passaram por um orçamento de 6: a diagonal não está dobrando")
	}
	if dentro[Square{X: 7, Y: 0}] {
		t.Error("sete quadrados em linha reta passaram por um orçamento de 6")
	}
	// A origem não é destino: acender a casa onde a peça já está seria oferecer
	// um movimento que não move.
	if dentro[Square{X: 0, Y: 0}] {
		t.Error("a origem entrou na lista de casas alcançáveis")
	}
}

// Sem orçamento não há casas acesas: fora de combate a régua mede, mas não há
// limite para desenhar, e uma busca sem teto não termina num plano infinito.
func TestSemOrcamentoNaoHaAlcanceParaAcender(t *testing.T) {
	if got := ReachableSquares(Square{X: 2, Y: 2}, -1, MoveTerrain{}); len(got) != 0 {
		t.Errorf("orçamento negativo devolveu %d casas, esperava nenhuma", len(got))
	}
}

// Terreno difícil encolhe o alcance pela mesma conta do passo (p238): 3m por
// quadrado, ou seja, o dobro.
func TestTerrenoDificilEncolheOAlcance(t *testing.T) {
	lama := MoveTerrain{Difficult: map[Square]bool{{X: 1, Y: 0}: true, {X: 2, Y: 0}: true}}

	reach := ReachableSquares(Square{X: 0, Y: 0}, 4, MoveTerrain{})
	naLama := ReachableSquares(Square{X: 0, Y: 0}, 4, lama)

	if len(naLama) >= len(reach) {
		t.Errorf("a lama não encolheu o alcance: %d contra %d", len(naLama), len(reach))
	}
	// (3,0) custa 5 por qualquer caminho: pela linha da lama, 1+2+2; pelo
	// contorno diagonal, 2 (diagonal) + 1 + 2 (diagonal). Os dois passam de 4.
	for _, s := range naLama {
		if s == (Square{X: 3, Y: 0}) {
			t.Error("(3,0) alcançado com 4 quadrados atravessando dois de terreno difícil")
		}
	}
}
