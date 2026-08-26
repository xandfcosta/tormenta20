package api

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"t20engine/engine"
	"t20engine/tabuleiro"
)

// Os guardas da RÉGUA e do GABARITO na Mesa (ALE-269, superfície 8).
//
// A ARITMÉTICA não é medida aqui: `engine.Measure` e `engine.AreaSquares` têm
// guarda de regra próprio, escrito contra a figura da p225 e a tabela da p224.
// Repeti-la deste lado seria a mesma asserção num lugar mais caro.
//
// O que se prende aqui é o que só existe DESTE lado: que a frase diz a faixa
// certa em português, que o segundo clique vira direção com zona morta, que a
// LISTA de quem o gabarito pega obedece à redação por papel, e que medir NÃO
// remenda a cena.

// TestAFraseDaReguaDizAFaixaDoLivro.
//
// A faixa é o que a régua tem de mais útil: "10,5m" obriga o jogador a lembrar
// que curto são 9m, enquanto "alcance médio" já é a resposta. E o "além" não é
// uma faixa com nome — ele é a ausência de uma —, então a frase dele é outra.
func TestAFraseDaReguaDizAFaixaDoLivro(t *testing.T) {
	casos := []struct {
		de, ate  engine.Square
		esperado string
	}{
		// Um quadrado no singular. A frase é lida em voz alta na mesa, e
		// "1 quadrados" apareceu na tela na primeira medição da SPA.
		{engine.Square{}, engine.Square{X: 1}, "1 quadrado (1,5m) · alcance curto"},
		// 6 quadrados são os 9m do alcance curto (p224), e o limite é INCLUSIVO.
		{engine.Square{}, engine.Square{X: 6}, "6 quadrados (9,0m) · alcance curto"},
		{engine.Square{}, engine.Square{X: 7}, "7 quadrados (10,5m) · alcance médio"},
		{engine.Square{}, engine.Square{X: 60}, "60 quadrados (90,0m) · alcance longo"},
		{engine.Square{}, engine.Square{X: 61}, "61 quadrados (91,5m) · além do alcance longo"},
		// A diagonal custa o DOBRO, e a régua do alcance é a mesma do movimento
		// (p238): 3 na diagonal são 6 quadrados, não 3.
		{engine.Square{}, engine.Square{X: 3, Y: 3}, "6 quadrados (9,0m) · alcance curto"},
	}
	for _, c := range casos {
		if lido := leituraDaRegua(engine.Measure(c.de, c.ate)); lido != c.esperado {
			t.Errorf("de %v a %v a régua disse %q, esperado %q", c.de, c.ate, lido, c.esperado)
		}
	}
}

// TestADirecaoTemZonaMorta.
//
// Sem a zona morta, um pixel de diferença no clique trocaria a forma inteira do
// gabarito debaixo do dedo: um clique quase em linha viraria diagonal e o cone
// mudaria de lado enquanto a pessoa tenta acertar a casa.
//
// O livro desenha o cone em DUAS orientações (p225) e não numa terceira, então
// a direção só pode sair ortogonal ou diagonal — nunca um passo com um eixo
// parado que não seja um dos dois.
func TestADirecaoDoGabaritoTemZonaMorta(t *testing.T) {
	origem := engine.Square{X: 5, Y: 5}
	casos := []struct {
		mira     engine.Square
		esperado engine.Square
		porque   string
	}{
		{engine.Square{X: 15, Y: 5}, engine.Square{X: 1}, "reto para a direita"},
		{engine.Square{X: 15, Y: 6}, engine.Square{X: 1}, "quase reto ainda é reto"},
		{engine.Square{X: 10, Y: 10}, engine.Square{X: 1, Y: 1}, "45° é diagonal"},
		{engine.Square{X: 10, Y: 8}, engine.Square{X: 1, Y: 1}, "dentro da zona vira diagonal"},
		{engine.Square{X: 5, Y: 15}, engine.Square{Y: 1}, "reto para baixo"},
		{engine.Square{X: -5, Y: -5}, engine.Square{X: -1, Y: -1}, "o plano não tem bordas"},
		// A mira na PRÓPRIA origem não é direção nenhuma; o caminho que a trata
		// é o do "clique de novo para apontar", e este valor nunca chega ao
		// desenho. Fica preso mesmo assim porque um (0,0) que vazasse faria o
		// `AreaSquares` desenhar um cone sem lado nenhum.
		{origem, engine.Square{X: 1}, "mira parada não é direção"},
	}
	for _, c := range casos {
		if lido := direcaoDoGabarito(origem, c.mira); lido != c.esperado {
			t.Errorf("%s: mira %v deu %v, esperado %v", c.porque, c.mira, lido, c.esperado)
		}
	}
}

// TestOCaminhoDoGabaritoUsaACoordenadaDoPlano.
//
// Com sinal, e é isso que faz o desenho caber num sinal em vez de num remendo: o
// `transform` do grupo — que o servidor redesenha — é quem tira a quina da
// moldura. Se o caminho já viesse relativo à moldura, uma moldura que crescesse
// deslocaria o gabarito sem que nada mudasse na tela.
func TestOCaminhoDoGabaritoUsaACoordenadaDoPlano(t *testing.T) {
	lido := caminhoDasCasas([]engine.Square{{X: -1, Y: 2}, {X: 0, Y: 2}})
	const esperado = "M -1 2 h 1 v 1 h -1 Z M 0 2 h 1 v 1 h -1 Z"
	if lido != esperado {
		t.Errorf("o caminho saiu %q, esperado %q", lido, esperado)
	}
	if caminhoDasCasas(nil) != "" {
		t.Error("área vazia devolveu caminho — o `data-show` do desenho depende do vazio")
	}
}

// TestOGabaritoPegaAPecaGrandePeloCORPO.
//
// Uma Colossal ocupa 6×6 (p107), e exigir que ela caiba inteira na área deixaria
// o dragão de fora do próprio incêndio. Basta UM quadrado do corpo cair dentro.
func TestOGabaritoPegaAPecaGrandePeloCorpo(t *testing.T) {
	b := &tabuleiro.BoardState{Tokens: []tabuleiro.BoardToken{
		{ID: "dragao", Label: "Dragão", X: 10, Y: 10, Footprint: 6},
		{ID: "rato", Label: "Rato", X: 30, Y: 30},
	}}
	// Uma casa só, na quina de baixo do corpo do dragão: a âncora dele é (10,10)
	// e o corpo vai até (15,15).
	dentro := quemOGabaritoPega(b, []engine.Square{{X: 15, Y: 15}})
	if !strings.Contains(dentro, "Dragão") {
		t.Errorf("a área pegou %q — o corpo da peça grande ficou de fora", dentro)
	}
	if strings.Contains(dentro, "Rato") {
		t.Errorf("a área pegou %q — quem está longe entrou", dentro)
	}
	if fora := quemOGabaritoPega(b, []engine.Square{{X: 100, Y: 100}}); fora != "Ninguém dentro." {
		t.Errorf("área sem ninguém disse %q", fora)
	}
	// A frase VAZIA é a dica, e não "0 peças": antes do primeiro clique não há
	// área nenhuma, e dizer que ela não pega ninguém descreveria mal o estado.
	if vazio := quemOGabaritoPega(b, nil); !strings.Contains(vazio, "Clique") {
		t.Errorf("sem gabarito posto a barra disse %q, esperado a dica do clique", vazio)
	}
}

// TestOGabaritoDoJogadorNaoContaAPecaEscondida — o guarda que mais importa.
//
// Esconder a peça é o gesto com que o mestre guarda a emboscada. Um gabarito que
// respondesse "Pega 2 peças: Arwen, Ogro" entregaria a emboscada pela porta dos
// fundos: a peça não está desenhada, mas o NOME dela chega ao HTML do jogador —
// e nesta superfície ele chega até por SINAL, que é mais fácil de ler que o DOM.
//
// A trava é passar pelo mesmo `BoardForRole` do resto da Mesa, e não uma segunda
// decisão sobre quem vê o quê.
func TestOGabaritoDoJogadorNaoContaAPecaEscondida(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "cripta")
	if _, err := f.s.boards.AddToken(context.Background(), f.sessionID,
		tabuleiro.BoardToken{ID: "emboscada", Label: "Ogro emboscado", X: 4, Y: 4, Hidden: true}, true); err != nil {
		t.Fatalf("pôr a peça escondida: %v", err)
	}

	// Um quadrado de lado 1 exatamente em cima dela.
	caminho := f.urlDaMesa() + "/tabuleiro/gabarito/quadrado/1/4/4/4/4"
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

// TestMedirNaoRemendaACena.
//
// A régua não muda a cena, e a resposta dela tem de ser do tamanho disso. Uma
// medição que devolvesse as nove regiões trocaria o mapa debaixo de quem está
// medindo — a peça sob o dedo de quem arrasta some e volta —, que é o mesmo
// defeito que a região `mesa-por-no-mapa` já existe para evitar.
//
// Provado VERMELHO trocando o `escreveSinais` pelo `respondeAoMestre`: a
// resposta passou a trazer `mesa-tabuleiro` e este teste acusou.
func TestMedirNaoRemendaACena(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "pedra")

	resposta := f.posta(t, f.mestre, f.urlDaMesa()+"/tabuleiro/regua/0/0/3/0", "")
	if !strings.Contains(resposta, "reguatexto") {
		t.Fatalf("a medida não voltou: %s", resposta)
	}
	if strings.Contains(resposta, "mesa-tabuleiro") || strings.Contains(resposta, "datastar-patch-elements") {
		t.Errorf("medir remendou a cena inteira:\n%s", resposta)
	}
}

// TestOGabaritoRecusaFormaQueOLivroNaoTem.
//
// O id vem do CLIENTE, e uma forma inventada não pode virar um desenho — nem
// cair calada na esfera, que desenharia uma área que ninguém pediu no lugar de
// dizer que o pedido está errado.
func TestOGabaritoRecusaFormaQueOLivroNaoTem(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "pedra")
	rec := f.pede(t, f.mestre, http.MethodPost, f.urlDaMesa()+"/tabuleiro/gabarito/piramide/2/0/0/0/0", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("forma inventada deu %d, esperado 400", rec.Code)
	}
	// A mensagem diz o valor recebido E a lista do que existe: quem lê o erro
	// precisa saber o que digitar em vez do que digitou.
	if corpo := rec.Body.String(); !strings.Contains(corpo, "piramide") || !strings.Contains(corpo, "esfera") {
		t.Errorf("a recusa saiu %q, sem o valor recusado ou sem a lista", corpo)
	}
}

// TestQuemNaoEstaNaMesaNaoMedeACenaDela.
//
// Medir é de todo mundo que joga — "dá para acertar daqui?" é pergunta de quem
// ataca —, e por isso a rota não exige o papel de mestre. A trava que sobra é a
// de sempre, e ela é do SERVIDOR: o gabarito devolve os NOMES das peças, então
// uma rota aberta seria a lista do bestiário da cena para quem tiver a URL.
func TestQuemNaoEstaNaMesaNaoMedeACenaDela(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "pedra")
	estranho := seedUser(t, f.s, "estranho@t.com")

	rec := f.pede(t, estranho, http.MethodPost, f.urlDaMesa()+"/tabuleiro/regua/0/0/3/0", "")
	if rec.Code == http.StatusOK {
		t.Errorf("quem não está na mesa mediu a cena dela: %s", rec.Body.String())
	}
}

// TestOTamanhoDoGabaritoTravaEmVezDeRecusar.
//
// O número vem de uma caixa que a pessoa está DIGITANDO, e apagar o conteúdo
// dela passa por zero e por vazio no caminho. Recusar com uma frase acenderia um
// erro no meio da digitação; travar desenha o menor gabarito e segue.
func TestOTamanhoDoGabaritoTravaEmVezDeRecusar(t *testing.T) {
	casos := map[string]int{
		"":     1,
		"0":    1,
		"-4":   1,
		"3":    3,
		"999":  engine.LongRangeSquares,
		"abc":  1,
		"60":   engine.LongRangeSquares,
		"61":   engine.LongRangeSquares,
		"2.5":  2, // o `Sscanf` lê o inteiro e larga o resto: 2 é gabarito, não erro.
		"  7 ": 7,
	}
	for bruto, esperado := range casos {
		if lido := tamanhoDoGabarito(bruto); lido != esperado {
			t.Errorf("tamanho %q virou %d, esperado %d", bruto, lido, esperado)
		}
	}
}

// TestOTrilhoOfereceAReguaAoJOGADOR.
//
// Antes desta fatia o trilho inteiro era do mestre, porque só ele tinha modo —
// pintar e marcar. A régua é de quem ataca, e a cena do jogador não desenhava
// trilho nenhum: a ferramenta existiria e não teria onde ser ligada.
//
// E o que continua sendo do mestre segue sendo: o pincel de terreno pinta a
// cena, e a trava de verdade é a rota (o `comandoDoMestreNoTabuleiro`) — isto
// aqui é a cortesia de não oferecer o que seria recusado.
func TestOTrilhoOfereceAReguaAoJogador(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "pedra")

	corpo := f.pede(t, f.jogador, http.MethodGet, f.urlDaMesa(), "").Body.String()
	for _, esperado := range []string{"Régua", "Gabarito", "Mover a peça"} {
		if !strings.Contains(corpo, esperado) {
			t.Errorf("a cena do jogador não ofereceu %q", esperado)
		}
	}
	if strings.Contains(corpo, "Borracha") {
		t.Error("o pincel do mestre apareceu na cena do jogador")
	}

	doMestre := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()
	if !strings.Contains(doMestre, "Borracha") {
		t.Error("o mestre perdeu o pincel — sem o caso positivo o de cima não mede nada")
	}
}

// TestACamadaDePinturaSoAcendeComPincel.
//
// A pergunta antiga era `$ferramenta != ” && != 'marcador'`, e ela era VERDADE
// para toda ferramenta que ainda não existia: com a régua ligada, a camada de
// pintar cobriria o mapa e roubaria o clique da medida — um defeito que não dá
// erro em lugar nenhum, só faz a régua não medir.
//
// A lista sai das espécies e nunca de um literal, e é isso que este guarda
// prende: a quinta espécie nasce dentro da condição em vez de fora dela.
func TestACamadaDePinturaSoAcendeComPincel(t *testing.T) {
	condicao := oPincelEstaLigado()
	for _, e := range tabuleiro.EspeciesDeTerreno {
		if !strings.Contains(condicao, `"`+string(e.ID)+`"`) {
			t.Errorf("a espécie %q ficou fora da condição da pintura: %s", e.ID, condicao)
		}
	}
	for _, fora := range []string{FerramentaDaRegua, FerramentaDoGabarito, FerramentaDeMarcar} {
		if strings.Contains(condicao, `"`+fora+`"`) {
			t.Errorf("a ferramenta %q entrou na condição da pintura: %s", fora, condicao)
		}
	}
}
