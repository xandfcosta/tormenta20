package api

import (
	"testing"

	"t20engine/engine"
)

// Os guardas da SETA em duas cores (ALE-203, item 13).
//
// O que eles prendem é o NÚMERO: em que ponto do plano a cor vira. É regra —
// sai do deslocamento e da diagonal do livro (p238) —, e é a única metade do
// item 13 que um teste de unidade pode segurar; se o fio dourado é dourado
// mesmo, quem responde é a folha de estilo e o olho.

func quadrados(pares ...[2]int) []engine.Square {
	qs := make([]engine.Square, len(pares))
	for i, p := range pares {
		qs[i] = engine.Square{X: p[0], Y: p[1]}
	}
	return qs
}

func brejoEm(pares ...[2]int) engine.MoveTerrain {
	dificil := map[engine.Square]bool{}
	for _, q := range quadrados(pares...) {
		dificil[q] = true
	}
	return engine.MoveTerrain{Difficult: dificil}
}

// A cor vira DENTRO da perna, e não no fim dela.
//
// Nove casas rasas para o leste com deslocamento de 6: dois terços da perna se
// pagam, e em terreno raso essa fração cai no centro de (6,0), que é onde a peça
// também para. O vermelho começa ali e vai até a borda da casa de destino, que é
// onde o `recuoDaSeta` põe a ponta.
func TestOFioViraVermelhoOndeODeslocamentoAcaba(t *testing.T) {
	dourado, alem := osFiosDoMovimento(quadrados([2]int{0, 0}, [2]int{9, 0}), []int{9}, 6)

	if dourado != "M 0.5 0.5 L 6.5 0.5" {
		t.Errorf("o trecho que cabe saiu %q, esperado até o centro de (6,0)", dourado)
	}
	if alem != "M 6.5 0.5 L 9 0.5" {
		t.Errorf("o trecho além saiu %q, esperado do centro de (6,0) até a borda do destino", alem)
	}
}

// A DIAGONAL do livro (p238) entra na conta da cor: quatro diagonais custam 8, e
// um deslocamento de 6 paga três — a virada é no centro de (3,3).
//
// É o mesmo caminho do `TestCaminhoAlemDoDeslocamentoEPropostoERecusadoNoConfirmar`
// lá no `tabuleiro`, medido pela outra ponta: lá o servidor RECUSA no confirmar,
// aqui a tela DESENHA onde a recusa começa. Os dois números têm de casar, senão a
// pessoa encurta o movimento pelo desenho e o servidor recusa mesmo assim.
func TestADiagonalDoLivroDecideOndeAsCoresSeDividem(t *testing.T) {
	dourado, alem := osFiosDoMovimento(quadrados([2]int{0, 0}, [2]int{4, 4}), []int{8}, 6)

	if dourado != "M 0.5 0.5 L 3.5 3.5" {
		t.Errorf("o trecho que cabe saiu %q, esperado até o centro de (3,3)", dourado)
	}
	if alem == "" {
		t.Error("oito quadrados sobre um deslocamento de seis não pintaram nada de vermelho")
	}
}

// O CONTROLE do caso acima, e o que separa "a cor vira no lugar certo" de "a cor
// vira sempre": um caminho que CABE sai inteiro de dourado, sem vermelho nenhum.
func TestOCaminhoQueCabeNaoTemTrechoVermelho(t *testing.T) {
	dourado, alem := osFiosDoMovimento(quadrados([2]int{0, 0}, [2]int{3, 0}), []int{3}, 6)

	if alem != "" {
		t.Errorf("três quadrados sobre um deslocamento de seis pintaram %q de vermelho", alem)
	}
	if dourado == "" {
		t.Error("o caminho que cabe não desenhou fio nenhum")
	}
}

// SEM ORÇAMENTO — o mestre, ou a cena fora de combate — não há vermelho.
//
// Não é que o caminho caiba: é que não há teto contra o que medi-lo, e desenhar
// um seria inventá-lo. O mesmo argumento já governa o `Alcance`, que também não
// aparece para quem não tem deslocamento de turno.
func TestSemOrcamentoASetaEInteiraDourada(t *testing.T) {
	dourado, alem := osFiosDoMovimento(quadrados([2]int{0, 0}, [2]int{40, 0}), []int{40}, -1)

	if alem != "" {
		t.Errorf("um caminho sem orçamento pintou %q de vermelho", alem)
	}
	if dourado != "M 0.5 0.5 L 40 0.5" {
		t.Errorf("a seta sem orçamento saiu %q, esperada inteira", dourado)
	}
}

// O TERRENO DIFÍCIL entra na virada pelo CUSTO, e este é o caso que prende a
// escolha — a única em que "repartir o custo" e "achar o quadrado onde a peça
// para" dão números diferentes.
//
// Seis casas para o leste com as três primeiras em brejo: elas custam 2 cada
// (p238) e as outras 1, então a perna custa 9 quadrados, que são 13,5m — e é
// isso que o rótulo escreve por cima dela. Um deslocamento de 6 são 9,0m, dois
// terços de 13,5m, então dois terços da linha saem dourados: a virada cai em 4,5.
//
// A peça de fato para em (3,0), uma casa antes. **Não é essa a pergunta que a
// seta responde**: ela é uma reta entre duas paradas, o caminho no grid não é
// reta, e o que a reta representa é o número escrito nela. Cortá-la onde a peça
// para poria 50% de dourado sob um rótulo que pede 67%, e as duas metades do
// item 13 se desmentiriam na mesma linha. Quem mostra as casas percorridas é a
// TRILHA.
func TestOBrejoEncurtaODouradoPeloCusto(t *testing.T) {
	dourado, _ := osFiosDoMovimento(
		quadrados([2]int{0, 0}, [2]int{6, 0}),
		[]int{9}, 6)

	if dourado != "M 0.5 0.5 L 4.5 0.5" {
		t.Errorf("com a perna custando 9 e o deslocamento pagando 6, o dourado saiu %q, esperado em dois terços da linha", dourado)
	}
}

// O CONTROLE do caso acima: a MESMA geometria sem brejo é dourada inteira.
//
// Seis casas para o leste custam 6 e cabem num deslocamento de 6. Sem ele, "o
// dourado parou em 4,5" não se distingue de "o dourado para sempre em dois
// terços", e a conta do terreno não estaria sendo medida por ninguém.
func TestSemBrejoAMesmaPernaCabeInteira(t *testing.T) {
	dourado, alem := osFiosDoMovimento(quadrados([2]int{0, 0}, [2]int{6, 0}), []int{6}, 6)

	if alem != "" {
		t.Errorf("seis casas rasas sobre um deslocamento de seis pintaram %q de vermelho", alem)
	}
	if dourado != "M 0.5 0.5 L 6 0.5" {
		t.Errorf("o dourado saiu %q, esperado até a borda do destino", dourado)
	}
}

// A virada acontece na perna CERTA de uma polilinha, e o dourado guarda as
// dobras anteriores inteiras.
//
// Três paradas, quatro casas cada perna, deslocamento de 6: a primeira perna
// custa 4 e cabe; a segunda estoura depois de duas casas, em (6,0) — centro 6,5.
func TestNaPolilinhaAViradaCaiNaPernaQueEstoura(t *testing.T) {
	dourado, alem := osFiosDoMovimento(
		quadrados([2]int{0, 0}, [2]int{4, 0}, [2]int{8, 0}), []int{4, 4}, 6)

	if dourado != "M 0.5 0.5 L 4.5 0.5 L 6.5 0.5" {
		t.Errorf("o dourado saiu %q, esperado dobrando em (4,0) e parando no centro de (6,0)", dourado)
	}
	if alem != "M 6.5 0.5 L 8 0.5" {
		t.Errorf("o vermelho saiu %q, esperado do centro de (6,0) até a borda do destino", alem)
	}
}

// O RÓTULO conta o que a perna CUSTA, e não a distância geométrica dela.
//
// Decisão do dono, tomada com este caso na mão: quatro casas de brejo custam 8
// quadrados, que são 12,0m — enquanto a régua, que ignora terreno de propósito
// ("uma flecha não atravessa o brejo mais devagar"), diria 6,0m para a mesma
// linha. A divergência é aceita porque o metro do rótulo precisa ser o mesmo
// metro do deslocamento: é ele que explica onde o vermelho começa.
func TestORotuloDaPernaContaOCustoENaoAGeometria(t *testing.T) {
	dobras := quadrados([2]int{0, 0}, [2]int{4, 0})
	brejo := brejoEm([2]int{1, 0}, [2]int{2, 0}, [2]int{3, 0}, [2]int{4, 0})

	pernas := asPernasDoMovimento(dobras, osCustosDasPernas(dobras, brejo))

	if len(pernas) != 1 {
		t.Fatalf("uma perna virou %d rótulos", len(pernas))
	}
	if pernas[0].Rotulo != "12,0m" {
		t.Errorf("o rótulo saiu %q; quatro casas de brejo custam 8 quadrados, que são 12,0m", pernas[0].Rotulo)
	}
	if pernas[0].MeioX != 2.5 || pernas[0].MeioY != 0.5 {
		t.Errorf("o rótulo pousou em (%v,%v), esperado no meio da perna", pernas[0].MeioX, pernas[0].MeioY)
	}
}

// Uma perna por dobra, e o rótulo de cada uma no meio DELA.
//
// Sem isto, uma polilinha ganharia um número só — e o item 13 pede a distância
// "entre paradas", que é justamente o que uma soma esconde.
func TestCadaPernaGanhaOProprioRotulo(t *testing.T) {
	dobras := quadrados([2]int{0, 0}, [2]int{2, 0}, [2]int{2, 4})

	pernas := asPernasDoMovimento(dobras, osCustosDasPernas(dobras, engine.MoveTerrain{}))

	if len(pernas) != 2 {
		t.Fatalf("duas pernas viraram %d rótulos", len(pernas))
	}
	if pernas[0].Rotulo != "3,0m" || pernas[1].Rotulo != "6,0m" {
		t.Errorf("os rótulos saíram %q e %q, esperados 3,0m e 6,0m", pernas[0].Rotulo, pernas[1].Rotulo)
	}
	if pernas[1].MeioX != 2.5 || pernas[1].MeioY != 2.5 {
		t.Errorf("o segundo rótulo pousou em (%v,%v), esperado no meio da segunda perna", pernas[1].MeioX, pernas[1].MeioY)
	}
}
