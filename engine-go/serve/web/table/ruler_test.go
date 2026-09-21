package table

import (
	"strings"
	"testing"

	"t20engine/domain/board"
	"t20engine/domain/engine"
)

// Os guardas da RÉGUA e do GABARITO na Mesa.
//
// A ARITMÉTICA não é medida aqui: `engine.Measure` e `engine.AreaSquares` têm
// guarda de regra próprio, escrito contra a figura da p225 e a tabela da p224.
// Repeti-la deste lado seria a mesma asserção num lugar mais caro.
//
// O que se prende aqui é o que só existe DESTE lado: que a frase diz a faixa
// certa em português, que o segundo clique vira direção com zona morta, que a
// LISTA de quem o gabarito pega obedece à redação por papel, e que medir NÃO
// remenda a cena.

// A faixa é o que a régua tem de mais útil: "10,5m" obriga o jogador a lembrar
// que curto são 9m, enquanto "alcance médio" já é a resposta. E o "além" não é
// uma faixa com nome — ele é a ausência de uma —, então a frase dele é outra.
func TestTheRulerSentenceSaysTheBookRangeBand(t *testing.T) {
	cases := []struct {
		de, ate engine.Square
		want    string
	}{
		// Um quadrado no SINGULAR: a frase é lida em voz alta na mesa.
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
	for _, c := range cases {
		if read := rulerReading(engine.Measure(c.de, c.ate)); read != c.want {
			t.Errorf("de %v a %v a régua disse %q, esperado %q", c.de, c.ate, read, c.want)
		}
	}
}

// Sem a zona morta, um pixel de diferença no clique trocaria a forma inteira do
// gabarito debaixo do dedo: um clique quase em linha viraria diagonal e o cone
// mudaria de lado enquanto a pessoa tenta acertar a casa.
//
// O livro desenha o cone em DUAS orientações (p225) e não numa terceira, então
// a direção só pode sair ortogonal ou diagonal — nunca um passo com um eixo
// parado que não seja um dos dois.
func TestTheTemplateDirectionHasADeadZone(t *testing.T) {
	origin := engine.Square{X: 5, Y: 5}
	cases := []struct {
		mira   engine.Square
		want   engine.Square
		reason string
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
		{origin, engine.Square{X: 1}, "mira parada não é direção"},
	}
	for _, c := range cases {
		if read := templateDirection(origin, c.mira); read != c.want {
			t.Errorf("%s: mira %v deu %v, esperado %v", c.reason, c.mira, read, c.want)
		}
	}
}

// A COORDENADA DO PLANO, com sinal, e é isso que faz o desenho caber num sinal em vez de num remendo: o
// `transform` do grupo — que o servidor redesenha — é quem tira a quina da
// moldura. Se o caminho já viesse relativo à moldura, uma moldura que crescesse
// deslocaria o gabarito sem que nada mudasse na tela.
func TestTheTemplatePathUsesThePlaneCoordinate(t *testing.T) {
	read := squaresPath([]engine.Square{{X: -1, Y: 2}, {X: 0, Y: 2}})
	const want = "M -1 2 h 1 v 1 h -1 Z M 0 2 h 1 v 1 h -1 Z"
	if read != want {
		t.Errorf("o caminho saiu %q, esperado %q", read, want)
	}
	if squaresPath(nil) != "" {
		t.Error("área vazia devolveu caminho — o `data-show` do desenho depende do vazio")
	}
}

// Uma Colossal ocupa 6×6 (p107), e exigir que ela caiba inteira na área deixaria
// o dragão de fora do próprio incêndio. Basta UM quadrado do corpo cair dentro.
func TestTheTemplateCatchesTheLargeTokenByItsBody(t *testing.T) {
	b := &board.BoardState{Tokens: []board.BoardToken{
		{ID: "dragao", Label: "Dragão", X: 10, Y: 10, Footprint: 6},
		{ID: "rato", Label: "Rato", X: 30, Y: 30},
	}}
	// Uma casa só, na quina de baixo do corpo do dragão: a âncora dele é (10,10)
	// e o corpo vai até (15,15).
	inside := takesTemplateWho(b, []engine.Square{{X: 15, Y: 15}})
	if !strings.Contains(inside, "Dragão") {
		t.Errorf("a área pegou %q — o corpo da peça grande ficou de fora", inside)
	}
	if strings.Contains(inside, "Rato") {
		t.Errorf("a área pegou %q — quem está longe entrou", inside)
	}
	if outside := takesTemplateWho(b, []engine.Square{{X: 100, Y: 100}}); outside != "Ninguém dentro." {
		t.Errorf("área sem ninguém disse %q", outside)
	}
	// A frase VAZIA é a dica, e não "0 peças": antes do primeiro clique não há
	// área nenhuma, e dizer que ela não pega ninguém descreveria mal o estado.
	if empty := takesTemplateWho(b, nil); !strings.Contains(empty, "Clique") {
		t.Errorf("sem gabarito posto a barra disse %q, esperado a dica do clique", empty)
	}
}

// O número vem de uma caixa que a pessoa está DIGITANDO, e apagar o conteúdo
// dela passa por zero e por vazio no caminho. Recusar com uma frase acenderia um
// erro no meio da digitação; travar desenha o menor gabarito e segue.
func TestTheTemplateSizeClampsInsteadOfRefusing(t *testing.T) {
	cases := map[string]int{
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
	for raw, want := range cases {
		if read := templateSize(raw); read != want {
			t.Errorf("tamanho %q virou %d, esperado %d", raw, read, want)
		}
	}
}

// Num plano sem bordas, "voltar ao começo" não significa nada — o gesto tem de
// achar o GRUPO. E o corpo entra na conta e não só a âncora: uma Colossal ocupa
// 6×6 (p107), e enquadrar pela quina dela deixaria metade do dragão fora da
// janela justamente na cena em que ele é o motivo de olhar.
func TestTheSceneCenterFramesTheLargeTokenBody(t *testing.T) {
	// Um rato em (0,0) e um dragão cuja âncora é (10,10) e cujo corpo vai até
	// (15,15): o centro pelo corpo é 7, pela âncora seria 5.
	v := BoardView{Tokens: []boardToken{
		{X: 0, Y: 0, Footprint: 1},
		{X: 10, Y: 10, Footprint: 6},
	}}
	if x, y := centerScene(v); x != 7 || y != 7 {
		t.Errorf("o centro saiu (%d,%d), esperado (7,7) — o corpo da peça grande ficou fora da conta", x, y)
	}
	// SEM PEÇA o alvo é a ORIGEM do plano: num plano infinito e vazio, o (0,0) é
	// o único lugar sobre o qual duas pessoas concordam.
	empty := BoardView{}
	if x, y := centerScene(empty); x != 0 || y != 0 {
		t.Errorf("a cena vazia mirou (%d,%d), esperado a origem do plano (0,0)", x, y)
	}
	// E o RÓTULO acompanha: "nas peças" numa cena sem peça nenhuma ensina que o
	// botão está quebrado.
	if CenterTarget(empty) == CenterTarget(v) {
		t.Errorf("o rótulo não distingue cena com peça de cena vazia: %q", CenterTarget(v))
	}
}

// Uma condição por EXCLUSÃO — "qualquer ferramenta que não seja o marcador" — é
// verdade para toda ferramenta que ainda não existe: com a régua ligada, a camada
// de pintar cobriria o mapa e roubaria o clique da medida — um defeito que não dá
// erro em lugar nenhum, só faz a régua não medir.
//
// A lista sai das espécies e nunca de um literal, e é isso que este guarda
// prende: a quinta espécie nasce dentro da condição em vez de fora dela.
func TestThePaintLayerOnlyLightsUpWithABrush(t *testing.T) {
	condition := onIsBrush()
	for _, e := range board.TerrainKinds {
		if !strings.Contains(condition, `"`+string(e.ID)+`"`) {
			t.Errorf("a espécie %q ficou fora da condição da pintura: %s", e.ID, condition)
		}
	}
	for _, outside := range []string{FerramentaDaRegua, FerramentaDoGabarito, MarkTool} {
		if strings.Contains(condition, `"`+outside+`"`) {
			t.Errorf("a ferramenta %q entrou na condição da pintura: %s", outside, condition)
		}
	}
}
