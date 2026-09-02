package api

import (
	"testing"

	"t20engine/engine"
)

// Os guardas da SETA em TRÊS faixas (ALE-203).
//
// O pedido do dono: *"mostrar a distância possível em amarelo, em azul quando
// ultrapassa o deslocamento e está gastando a ação principal, e vermelho quando
// extrapola ambas"*. O livro dá os dois limiares (T20 p233: "Você pode trocar sua
// ação padrão por uma ação de movimento, para fazer duas ações de movimento, mas
// não pode fazer o inverso"), e não há terceira ação de movimento — então o teto
// é exatamente 2× o deslocamento.
//
// O que estes guardas prendem é o NÚMERO: em que ponto do plano cada cor começa.
// Se o fio azul é azul mesmo, quem responde é a folha de estilo e o olho.

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

// A PRIMEIRA VIRADA: o ouro acaba onde a ação de movimento acaba.
//
// Nove casas rasas para o leste com deslocamento de 6. O ouro cobre dois terços
// da perna e, em terreno raso, essa fração cai no centro de (6,0) — que é onde a
// peça também para. O resto é AZUL e não vermelho: 9 cabe em 12, ou seja, cabe
// na segunda ação de movimento.
func TestTheGoldEndsWhereTheMoveActionEnds(t *testing.T) {
	ouro, azul, vermelho := osFiosDoMovimento(quadrados([2]int{0, 0}, [2]int{9, 0}), []int{9}, 6)

	if ouro != "M 0.5 0.5 L 6.5 0.5" {
		t.Errorf("o ouro saiu %q, esperado até o centro de (6,0)", ouro)
	}
	if azul != "M 6.5 0.5 L 9 0.5" {
		t.Errorf("o azul saiu %q, esperado do centro de (6,0) até a borda do destino", azul)
	}
	if vermelho != "" {
		t.Errorf("nove quadrados cabem em duas ações de movimento e mesmo assim pintaram %q de vermelho", vermelho)
	}
}

// A SEGUNDA VIRADA: o vermelho começa onde a AÇÃO PADRÃO também acaba.
//
// Vinte casas rasas com deslocamento de 6: o ouro vai até 6, o azul de 6 a 12, e
// o que passa de 12 não cabe no turno — não há terceira ação de movimento.
func TestTheRedStartsAfterBothActions(t *testing.T) {
	ouro, azul, vermelho := osFiosDoMovimento(quadrados([2]int{0, 0}, [2]int{20, 0}), []int{20}, 6)

	if ouro != "M 0.5 0.5 L 6.5 0.5" {
		t.Errorf("o ouro saiu %q, esperado até o centro de (6,0)", ouro)
	}
	if azul != "M 6.5 0.5 L 12.5 0.5" {
		t.Errorf("o azul saiu %q, esperado do centro de (6,0) ao de (12,0)", azul)
	}
	if vermelho != "M 12.5 0.5 L 20 0.5" {
		t.Errorf("o vermelho saiu %q, esperado do centro de (12,0) até a borda do destino", vermelho)
	}
}

// O CONTROLE das duas viradas: um caminho que CABE na ação de movimento é ouro
// puro, sem azul nem vermelho.
//
// Sem ele, "achei ouro" não se distingue de "o ouro é o caminho inteiro sempre",
// e as duas tesouras poderiam estar cortando em qualquer lugar.
func TestThePathThatFitsInTheMoveActionIsPureGold(t *testing.T) {
	ouro, azul, vermelho := osFiosDoMovimento(quadrados([2]int{0, 0}, [2]int{3, 0}), []int{3}, 6)

	if azul != "" || vermelho != "" {
		t.Errorf("três quadrados sobre um deslocamento de seis pintaram %q de azul e %q de vermelho", azul, vermelho)
	}
	if ouro == "" {
		t.Error("o caminho que cabe não desenhou fio nenhum")
	}
}

// A DIAGONAL do livro (p238) entra na conta das duas viradas: quatro diagonais
// custam 8, e um deslocamento de 6 paga três — o ouro acaba no centro de (3,3).
//
// Oito cabe em doze, então o resto é azul: este é o caminho que antes era
// RECUSADO pelo servidor, e hoje é um movimento legítimo que custa o turno todo.
func TestTheBookDiagonalDecidesTheFirstTurn(t *testing.T) {
	ouro, azul, vermelho := osFiosDoMovimento(quadrados([2]int{0, 0}, [2]int{4, 4}), []int{8}, 6)

	if ouro != "M 0.5 0.5 L 3.5 3.5" {
		t.Errorf("o ouro saiu %q, esperado até o centro de (3,3)", ouro)
	}
	if azul == "" {
		t.Error("oito quadrados sobre um deslocamento de seis não pintaram nada de azul")
	}
	if vermelho != "" {
		t.Errorf("oito cabe em duas ações de movimento e mesmo assim saiu %q de vermelho", vermelho)
	}
}

// SEM ORÇAMENTO — fora de combate — não há faixa nenhuma.
//
// Sem vez não há ação padrão para trocar por movimento, então azul e vermelho
// não querem dizer nada: desenhá-los inventaria um teto que a cena não tem.
func TestOutOfCombatTheArrowIsGoldAllTheWay(t *testing.T) {
	ouro, azul, vermelho := osFiosDoMovimento(quadrados([2]int{0, 0}, [2]int{40, 0}), []int{40}, -1)

	if azul != "" || vermelho != "" {
		t.Errorf("sem orçamento a seta pintou %q de azul e %q de vermelho", azul, vermelho)
	}
	if ouro != "M 0.5 0.5 L 40 0.5" {
		t.Errorf("a seta sem orçamento saiu %q, esperada inteira", ouro)
	}
}

// AS DUAS VIRADAS NA MESMA PERNA, que é o caso em que a aritmética dos índices
// erra sem avisar.
//
// Uma perna só, custando 30 sobre um deslocamento de 6: as duas tesouras caem
// entre as MESMAS duas dobras, e o miolo azul é o trecho entre elas. Uma versão
// que cortasse em cadeia — a segunda tesoura sobre o resto da primeira — teria
// de traduzir o índice e é aqui que ela se perderia, com a linha continuando a
// sair, só que com as cores no lugar errado.
func TestBothTurnsOnTheSameLeg(t *testing.T) {
	ouro, azul, vermelho := osFiosDoMovimento(quadrados([2]int{0, 0}, [2]int{30, 0}), []int{30}, 6)

	if ouro != "M 0.5 0.5 L 6.5 0.5" {
		t.Errorf("o ouro saiu %q", ouro)
	}
	if azul != "M 6.5 0.5 L 12.5 0.5" {
		t.Errorf("o azul saiu %q, esperado o miolo entre as duas viradas", azul)
	}
	if vermelho != "M 12.5 0.5 L 30 0.5" {
		t.Errorf("o vermelho saiu %q", vermelho)
	}
}

// AS VIRADAS EM PERNAS DIFERENTES de uma polilinha, e o dourado guardando as
// dobras anteriores inteiras.
//
// Três paradas de oito casas cada, deslocamento de 6: o ouro estoura no meio da
// primeira perna, o azul atravessa a dobra em (8,0) e o vermelho fecha o resto.
// É o caso que prova que `pontos[i1:i2]` traz as dobras do MIOLO — cortar só nas
// pontas perderia a dobra e a seta azul viraria uma reta por cima do mapa.
func TestOnThePolylineTheBlueCrossesTheBend(t *testing.T) {
	ouro, azul, vermelho := osFiosDoMovimento(
		quadrados([2]int{0, 0}, [2]int{8, 0}, [2]int{8, 8}), []int{8, 8}, 6)

	if ouro != "M 0.5 0.5 L 6.5 0.5" {
		t.Errorf("o ouro saiu %q, esperado até o centro de (6,0)", ouro)
	}
	if azul != "M 6.5 0.5 L 8.5 0.5 L 8.5 4.5" {
		t.Errorf("o azul saiu %q, esperado dobrando em (8,0) e parando no centro de (8,4)", azul)
	}
	if vermelho != "M 8.5 4.5 L 8.5 8" {
		t.Errorf("o vermelho saiu %q, esperado do centro de (8,4) até a borda do destino", vermelho)
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
// para poria 50% de dourado sob um rótulo que pede 67%, e as duas metades se
// desmentiriam na mesma linha. Quem mostra as casas percorridas é a TRILHA.
func TestTheMarshShortensTheGoldByCost(t *testing.T) {
	ouro, _, _ := osFiosDoMovimento(quadrados([2]int{0, 0}, [2]int{6, 0}), []int{9}, 6)

	if ouro != "M 0.5 0.5 L 4.5 0.5" {
		t.Errorf("com a perna custando 9 e a ação de movimento pagando 6, o ouro saiu %q, esperado em dois terços da linha", ouro)
	}
}

// O CONTROLE do caso acima: a MESMA geometria sem brejo é dourada inteira.
//
// Seis casas para o leste custam 6 e cabem num deslocamento de 6. Sem ele, "o
// ouro parou em 4,5" não se distingue de "o ouro para sempre em dois terços".
func TestWithoutTheMarshTheSameLegFitsWhole(t *testing.T) {
	ouro, azul, _ := osFiosDoMovimento(quadrados([2]int{0, 0}, [2]int{6, 0}), []int{6}, 6)

	if azul != "" {
		t.Errorf("seis casas rasas sobre um deslocamento de seis pintaram %q de azul", azul)
	}
	if ouro != "M 0.5 0.5 L 6 0.5" {
		t.Errorf("o ouro saiu %q, esperado até a borda do destino", ouro)
	}
}

// O RÓTULO conta o que a perna CUSTA, e não a distância geométrica dela.
//
// Decisão do dono, tomada com este caso na mão: quatro casas de brejo custam 8
// quadrados, que são 12,0m — enquanto a régua, que ignora terreno de propósito
// ("uma flecha não atravessa o brejo mais devagar"), diria 6,0m para a mesma
// linha. A divergência é aceita porque o metro do rótulo precisa ser o mesmo
// metro do deslocamento: é ele que explica onde cada cor começa.
func TestTheLegLabelCountsTheCostAndNotTheGeometry(t *testing.T) {
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
// Sem isto, uma polilinha ganharia um número só — e o pedido é a distância
// "entre paradas", que é justamente o que uma soma esconde.
func TestEachLegGetsItsOwnLabel(t *testing.T) {
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

// A FRASE DO RODAPÉ nomeia as ações, que é a pergunta que o dono pôs primeiro:
// "se eles precisam gastar a ação de movimento e a ação principal".
//
// Ela é a mesma leitura das cores, em palavras — quem não distingue azul de
// vermelho no mapa lê aqui.
func TestTheFooterNamesTheActionsSpent(t *testing.T) {
	casos := []struct {
		custo, orcamento int
		frase            string
	}{
		{custo: 4, orcamento: 6, frase: "ação de movimento"},
		{custo: 6, orcamento: 6, frase: "ação de movimento"},
		{custo: 7, orcamento: 6, frase: "ação de movimento + ação principal"},
		{custo: 12, orcamento: 6, frase: "ação de movimento + ação principal"},
		{custo: 13, orcamento: 6, frase: "não cabe no turno"},
	}
	for _, c := range casos {
		if got := asAcoesGastas(&movimentoView{Custo: c.custo, Orcamento: c.orcamento}); got != c.frase {
			t.Errorf("custo %d sobre deslocamento %d disse %q, esperado %q", c.custo, c.orcamento, got, c.frase)
		}
	}
}

// A LEGENDA mostra as TRÊS faixas e acende a da vez.
//
// O dono: *"não tem um modo do usuário saber o que as cores indicam"*. Mostrar
// só a faixa ativa ensinaria a cor do momento e esconderia a escala — quem nunca
// passou do deslocamento não descobriria o azul, que é a informação que muda a
// decisão. Por isso o guarda prende as duas metades: que as três SAEM, e que
// exatamente uma está acesa.
func TestTheLegendShowsTheThreeBandsAndLightsTheCurrentOne(t *testing.T) {
	casos := []struct {
		custo, orcamento, acesa int
	}{
		{custo: 4, orcamento: 6, acesa: 0},
		{custo: 7, orcamento: 6, acesa: 1},
		{custo: 13, orcamento: 6, acesa: 2},
	}
	for _, c := range casos {
		legenda := aLegendaDoMovimento(&movimentoView{Custo: c.custo, Orcamento: c.orcamento})
		if len(legenda) != 3 {
			t.Fatalf("custo %d: a legenda saiu com %d faixas, e a escala tem três", c.custo, len(legenda))
		}
		acesas := 0
		for i, f := range legenda {
			if !f.Ativa {
				continue
			}
			acesas++
			if i != c.acesa {
				t.Errorf("custo %d sobre deslocamento %d acendeu a faixa %d, esperada a %d", c.custo, c.orcamento, i, c.acesa)
			}
		}
		if acesas != 1 {
			t.Errorf("custo %d acendeu %d faixas, e a leitura só faz sentido com uma", c.custo, acesas)
		}
	}
}

// A LEGENDA e a FRASE dizem a mesma coisa, porque saem da mesma lista.
//
// Sem isto, as duas envelhecem separadas — e o pior sintoma possível é a tela
// dizendo "gasta a ação principal" ao lado de uma bolinha que diz outra coisa.
func TestTheLitLegendAndTheSentenceAreTheSameText(t *testing.T) {
	m := &movimentoView{Custo: 8, Orcamento: 6}

	for _, f := range aLegendaDoMovimento(m) {
		if f.Ativa && f.Texto != asAcoesGastas(m) {
			t.Errorf("a legenda acesa diz %q e a frase diz %q", f.Texto, asAcoesGastas(m))
		}
	}
}
