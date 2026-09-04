package table

import (
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

// TestTheRulerSentenceSaysTheBookRangeBand.
//
// A faixa é o que a régua tem de mais útil: "10,5m" obriga o jogador a lembrar
// que curto são 9m, enquanto "alcance médio" já é a resposta. E o "além" não é
// uma faixa com nome — ele é a ausência de uma —, então a frase dele é outra.
func TestTheRulerSentenceSaysTheBookRangeBand(t *testing.T) {
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
		if lido := rulerReading(engine.Measure(c.de, c.ate)); lido != c.esperado {
			t.Errorf("de %v a %v a régua disse %q, esperado %q", c.de, c.ate, lido, c.esperado)
		}
	}
}

// TestTheTemplateDirectionHasADeadZone.
//
// Sem a zona morta, um pixel de diferença no clique trocaria a forma inteira do
// gabarito debaixo do dedo: um clique quase em linha viraria diagonal e o cone
// mudaria de lado enquanto a pessoa tenta acertar a casa.
//
// O livro desenha o cone em DUAS orientações (p225) e não numa terceira, então
// a direção só pode sair ortogonal ou diagonal — nunca um passo com um eixo
// parado que não seja um dos dois.
func TestTheTemplateDirectionHasADeadZone(t *testing.T) {
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
		if lido := templateDirection(origem, c.mira); lido != c.esperado {
			t.Errorf("%s: mira %v deu %v, esperado %v", c.porque, c.mira, lido, c.esperado)
		}
	}
}

// TestTheTemplatePathUsesThePlaneCoordinate.
//
// Com sinal, e é isso que faz o desenho caber num sinal em vez de num remendo: o
// `transform` do grupo — que o servidor redesenha — é quem tira a quina da
// moldura. Se o caminho já viesse relativo à moldura, uma moldura que crescesse
// deslocaria o gabarito sem que nada mudasse na tela.
func TestTheTemplatePathUsesThePlaneCoordinate(t *testing.T) {
	lido := squaresPath([]engine.Square{{X: -1, Y: 2}, {X: 0, Y: 2}})
	const esperado = "M -1 2 h 1 v 1 h -1 Z M 0 2 h 1 v 1 h -1 Z"
	if lido != esperado {
		t.Errorf("o caminho saiu %q, esperado %q", lido, esperado)
	}
	if squaresPath(nil) != "" {
		t.Error("área vazia devolveu caminho — o `data-show` do desenho depende do vazio")
	}
}

// TestTheTemplateCatchesTheLargeTokenByItsBody.
//
// Uma Colossal ocupa 6×6 (p107), e exigir que ela caiba inteira na área deixaria
// o dragão de fora do próprio incêndio. Basta UM quadrado do corpo cair dentro.
func TestTheTemplateCatchesTheLargeTokenByItsBody(t *testing.T) {
	b := &tabuleiro.BoardState{Tokens: []tabuleiro.BoardToken{
		{ID: "dragao", Label: "Dragão", X: 10, Y: 10, Footprint: 6},
		{ID: "rato", Label: "Rato", X: 30, Y: 30},
	}}
	// Uma casa só, na quina de baixo do corpo do dragão: a âncora dele é (10,10)
	// e o corpo vai até (15,15).
	dentro := takesTemplateWho(b, []engine.Square{{X: 15, Y: 15}})
	if !strings.Contains(dentro, "Dragão") {
		t.Errorf("a área pegou %q — o corpo da peça grande ficou de fora", dentro)
	}
	if strings.Contains(dentro, "Rato") {
		t.Errorf("a área pegou %q — quem está longe entrou", dentro)
	}
	if fora := takesTemplateWho(b, []engine.Square{{X: 100, Y: 100}}); fora != "Ninguém dentro." {
		t.Errorf("área sem ninguém disse %q", fora)
	}
	// A frase VAZIA é a dica, e não "0 peças": antes do primeiro clique não há
	// área nenhuma, e dizer que ela não pega ninguém descreveria mal o estado.
	if vazio := takesTemplateWho(b, nil); !strings.Contains(vazio, "Clique") {
		t.Errorf("sem gabarito posto a barra disse %q, esperado a dica do clique", vazio)
	}
}

// TestTheTemplateSizeClampsInsteadOfRefusing.
//
// O número vem de uma caixa que a pessoa está DIGITANDO, e apagar o conteúdo
// dela passa por zero e por vazio no caminho. Recusar com uma frase acenderia um
// erro no meio da digitação; travar desenha o menor gabarito e segue.
func TestTheTemplateSizeClampsInsteadOfRefusing(t *testing.T) {
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
		if lido := templateSize(bruto); lido != esperado {
			t.Errorf("tamanho %q virou %d, esperado %d", bruto, lido, esperado)
		}
	}
}

// TestTheSceneCenterFramesTheLargeTokenBody (ALE-269, item 9).
//
// Num plano sem bordas, "voltar ao começo" não significa nada — o gesto tem de
// achar o GRUPO. E o corpo entra na conta e não só a âncora: uma Colossal ocupa
// 6×6 (p107), e enquadrar pela quina dela deixaria metade do dragão fora da
// janela justamente na cena em que ele é o motivo de olhar.
func TestTheSceneCenterFramesTheLargeTokenBody(t *testing.T) {
	// Um rato em (0,0) e um dragão cuja âncora é (10,10) e cujo corpo vai até
	// (15,15): o centro pelo corpo é 7, pela âncora seria 5.
	v := BoardView{Pecas: []boardToken{
		{X: 0, Y: 0, Pegada: 1},
		{X: 10, Y: 10, Pegada: 6},
	}}
	if x, y := centerScene(v); x != 7 || y != 7 {
		t.Errorf("o centro saiu (%d,%d), esperado (7,7) — o corpo da peça grande ficou fora da conta", x, y)
	}
	// SEM PEÇA o alvo é a ORIGEM do plano (ALE-203). Era o meio da MOLDURA, e a
	// moldura saiu: num plano infinito e vazio, o (0,0) é o único lugar sobre o
	// qual duas pessoas concordam.
	vazia := BoardView{}
	if x, y := centerScene(vazia); x != 0 || y != 0 {
		t.Errorf("a cena vazia mirou (%d,%d), esperado a origem do plano (0,0)", x, y)
	}
	// E o RÓTULO acompanha: "nas peças" numa cena sem peça nenhuma ensina que o
	// botão está quebrado.
	if CenterTarget(vazia) == CenterTarget(v) {
		t.Errorf("o rótulo não distingue cena com peça de cena vazia: %q", CenterTarget(v))
	}
}

// TestThePaintLayerOnlyLightsUpWithABrush.
//
// A pergunta antiga era `$ferramenta != ” && != 'marcador'`, e ela era VERDADE
// para toda ferramenta que ainda não existia: com a régua ligada, a camada de
// pintar cobriria o mapa e roubaria o clique da medida — um defeito que não dá
// erro em lugar nenhum, só faz a régua não medir.
//
// A lista sai das espécies e nunca de um literal, e é isso que este guarda
// prende: a quinta espécie nasce dentro da condição em vez de fora dela.
func TestThePaintLayerOnlyLightsUpWithABrush(t *testing.T) {
	condicao := onIsBrush()
	for _, e := range tabuleiro.TerrainKinds {
		if !strings.Contains(condicao, `"`+string(e.ID)+`"`) {
			t.Errorf("a espécie %q ficou fora da condição da pintura: %s", e.ID, condicao)
		}
	}
	for _, fora := range []string{FerramentaDaRegua, FerramentaDoGabarito, MarkTool} {
		if strings.Contains(condicao, `"`+fora+`"`) {
			t.Errorf("a ferramenta %q entrou na condição da pintura: %s", fora, condicao)
		}
	}
}
