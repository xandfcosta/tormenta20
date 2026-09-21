package api

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"t20engine/domain/board"
	"t20engine/domain/engine"
	"testing"
)

func TestTheStrokePaintsTheWholeSegment(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")

	rec := f.pede(t, f.gm, http.MethodPost,
		f.tableUrl()+"/tabuleiro/terreno", stroke("dificil", 2, 2, 8, 5))
	if rec.Code != http.StatusOK {
		t.Fatalf("o traço deu %d", rec.Code)
	}

	b := f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)
	squares := board.SquaresOf(b, "dificil")

	// DEZ, escrito à mão: derivar a contagem de `board.StrokeSquares` — a MESMA
	// função que o handler chama — anda junto com o defeito.
	//
	// O traço é SUPERCOVER e não Bresenham: ele inclui a casa de antes e a de
	// depois em cada degrau, porque um pincel que pula deixa buraco. De (2,2) a
	// (8,5) são as dez de `{2 2} {3 2} {4 2} {4 3} {5 3} {6 3} {6 4} {7 4}
	// {8 4} {8 5}`.
	if len(squares) != 10 {
		t.Errorf("o traço (2,2)→(8,5) pintou %d casas, e o segmento tem 10: %v", len(squares), squares)
	}
	// E o traço não tem buraco na ponta que este lado controla: a primeira e a
	// última casa do segmento estão lá. O meio é problema do `StrokeSquares`, que
	// tem guarda próprio.
	for _, tip := range []engine.Square{{X: 2, Y: 2}, {X: 8, Y: 5}} {
		if !contem(squares, tip) {
			t.Errorf("a casa %v não foi pintada: o traço não chega às pontas", tip)
		}
	}
	// E NADA FORA DA CAIXA do traço, que é a metade que faltava: sabotado para
	// pintar um bloco além do pedido, este caso era VERDE — contar casas e
	// conferir as pontas não diz nada sobre o que foi pintado a mais.
	for _, square := range squares {
		if square.X < 2 || square.X > 8 || square.Y < 2 || square.Y > 5 {
			t.Errorf("o traço (2,2)→(8,5) pintou %v, que está fora da caixa dele", square)
		}
	}
}

// O irmão do de cima, e ele existe porque as duas rotas são caminhos diferentes:
// a da borracha não tem espécie.
func TestTheEraserStrokeClearsTheWholeSegment(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	// A ORIGEM NÃO É (0,0), porque (0,0) é o VALOR-ZERO do struct: um `from` que
	// parasse de ser lido — a tag `json:"from"` trocada, o corpo chegando vazio —
	// decodifica exatamente para (0,0), e o caso passaria sem medir nada.
	if rec := f.pede(t, f.gm, http.MethodPost,
		f.tableUrl()+"/tabuleiro/terreno", stroke("cobertura", 4, 4, 6, 6)); rec.Code != http.StatusOK {
		t.Fatalf("pintar deu %d", rec.Code)
	}
	// A TESTEMUNHA DE FORA, pintada longe do traço da borracha. Ela é a metade
	// que faltava: sabotado para apagar um bloco 10×10 na origem ALÉM do pedido,
	// este caso era verde — "sobrou zero do traço" não diz nada sobre o que foi
	// apagado a mais.
	if rec := f.pede(t, f.gm, http.MethodPost,
		f.tableUrl()+"/tabuleiro/terreno", stroke("dificil", 1, 9, 1, 9)); rec.Code != http.StatusOK {
		t.Fatalf("pintar a testemunha deu %d", rec.Code)
	}
	// O CONTROLE: havia o que apagar. Sem ele, "sobrou zero" é verdade também
	// sobre um tabuleiro em que nada foi pintado.
	b := f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)
	if len(board.SquaresOf(b, "cobertura")) < 5 {
		t.Fatalf("o traço de pintura só fez %d casas — não há o que a borracha apagar",
			len(board.SquaresOf(b, "cobertura")))
	}

	if rec := f.pede(t, f.gm, http.MethodPost,
		f.tableUrl()+"/tabuleiro/terreno/limpar", stroke("", 4, 4, 6, 6)); rec.Code != http.StatusOK {
		t.Fatalf("apagar deu %d", rec.Code)
	}
	b = f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)
	if left := board.SquaresOf(b, "cobertura"); len(left) != 0 {
		t.Errorf("a borracha deixou %v pelo caminho", left)
	}
	if witness := board.SquaresOf(b, "dificil"); len(witness) != 1 {
		t.Errorf("a casa (1,9), que está fora do traço da borracha, virou %v — a borracha apagou além do pedido",
			witness)
	}
}

// O teto é do domínio e a recusa chega como FRASE, não como 500: um traço de dez
// milhões de casas só vem de um pedido montado à mão, e a resposta certa é dizer
// o que houve.
func TestAForgedStrokeIsRefused(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")

	body := f.pede(t, f.gm, http.MethodPost,
		f.tableUrl()+"/tabuleiro/terreno", stroke("dificil", 0, 0, 9999999, 0)).Body.String()
	if !strings.Contains(body, "longo demais") {
		t.Errorf("o traço forjado não foi recusado com frase: %q", body[max(0, len(body)-200):])
	}
	b := f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)
	if squares := board.SquaresOf(b, "dificil"); len(squares) != 0 {
		t.Errorf("o traço recusado pintou %d casas assim mesmo", len(squares))
	}
}

// O guarda dos 353 KB: repintar TODAS as regiões a cada casa é proibitivo num
// gesto CONTÍNUO — um traço de vinte casas mandaria sete megabytes enquanto o
// mestre ainda arrasta o dedo.
//
// A asserção nomeia as duas metades: a região do mapa TEM de vir (senão o traço
// não aparece) e a do acervo NÃO pode (é a maior da Mesa e não muda quando
// alguém pinta uma casa).
func TestTheBrushDoesNotReturnTheWholeTable(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")

	body := f.pede(t, f.gm, http.MethodPost,
		f.tableUrl()+"/tabuleiro/terreno", stroke("dificil", 1, 1, 1, 1)).Body.String()

	if !strings.Contains(body, `id="table-board"`) {
		t.Error("a resposta do pincel não traz o mapa — a casa pintada não apareceria")
	}
	for _, region := range []string{"table-archive", "table-tracker", "table-party", "table-npcs"} {
		if strings.Contains(body, `id="`+region+`"`) {
			t.Errorf("a resposta do pincel repinta a região %q, que não muda ao pintar uma casa", region)
		}
	}
}

// Uma afirmação sobre a FORMA do que a página serve, e é o único jeito de
// alcançar os três gestos de uma vez: `pointerdown`/`pointermove`/`pointerup` na
// camada de pintura e o botão 2 caindo no caminho da borracha.
//
// Se algum deles voltar a ser um `data-on:click`, o traço morre em silêncio — a
// tela continua pintando um quadrado por clique.
func TestTheScreenWiresTheStrokeToTheRightButton(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	screen := f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()

	for _, chunk := range []string{
		"data-on:pointerdown",
		"data-on:pointermove",
		"data-on:pointerup",
		"data-on:contextmenu",
		// O par de cantos viaja no CORPO, não na URL. A chave é minúscula
		// porque é a grafia que o `engine.Square` declara (`json:"x"`).
		"to: {x: ",
		"evt.button === 2",
	} {
		if !strings.Contains(screen, chunk) {
			t.Errorf("a cena não tem %q: o traço do pincel não acontece", chunk)
		}
	}
	// A CAMADA DE PINTURA não pode ter voltado ao clique de um quadrado só.
	if strings.Contains(screen, `aria-label="Pintar terreno — escolha a casa"`) {
		t.Error("a camada de pintura voltou a ser um clique por casa")
	}
}

// A ponta que só o HTML servido responde: o ícone chega à casa, e o trilho
// mostra o MESMO.
func TestThePaintedSquareCarriesTheKindIcon(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	if rec := f.pede(t, f.gm, http.MethodPost,
		f.tableUrl()+"/tabuleiro/terreno", stroke("camuflagem", 3, 3, 3, 3)); rec.Code != http.StatusOK {
		t.Fatalf("pintar deu %d", rec.Code)
	}
	screen := f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()

	// O CANTO vai escrito à mão ("sudeste" é o da camuflagem na tabela do
	// desenho): ler o valor do `drawing` da cena faria o esperado sair do código
	// sob teste, e os dois andariam juntos com o defeito.
	if !strings.Contains(screen, "terrain-corner-southeast") {
		t.Error("a casa de camuflagem não veste o canto sudeste")
	}
	if !strings.Contains(screen, "terrain-mark") {
		t.Error("a casa pintada não tem a marca da espécie")
	}
	// E o TRILHO usa a mesma tabela: o botão do pincel tinge com a cor dela.
	if !strings.Contains(screen, "board-hue-concealment") {
		t.Error("o pincel do trilho não veste o matiz da espécie")
	}
}

// A rota do retângulo é IRMÃ da do traço e chama a mesma gravação — o que muda é
// quais casas o par de cantos nomeia. O guarda mede as duas pontas que só esta
// camada responde: a área inteira pintada, e a borracha usando o caminho SEM
// espécie.
func TestTheRectangleFillsTheWholeArea(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")

	if rec := f.pede(t, f.gm, http.MethodPost,
		f.tableUrl()+"/tabuleiro/terreno/retangulo", stroke("dificil", 2, 2, 4, 5)); rec.Code != http.StatusOK {
		t.Fatalf("o retângulo deu %d", rec.Code)
	}
	b := f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)
	// 3 colunas × 4 linhas = 12 casas, e as duas pontas incluídas.
	if squares := board.SquaresOf(b, "dificil"); len(squares) != 12 {
		t.Errorf("(2,2)→(4,5) pintou %d casas, esperado as 12 do retângulo: %v", len(squares), squares)
	}

	if rec := f.pede(t, f.gm, http.MethodPost,
		f.tableUrl()+"/tabuleiro/terreno/limpar/retangulo", stroke("", 2, 2, 4, 5)); rec.Code != http.StatusOK {
		t.Fatalf("limpar o retângulo deu %d", rec.Code)
	}
	b = f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)
	if left := board.SquaresOf(b, "dificil"); len(left) != 0 {
		t.Errorf("a borracha em área deixou %v", left)
	}
}

// NÃO existe teto de área no retângulo, e isso é decisão do dono (ALE-315): o
// app roda local, e um teto mordia gesto de verdade — no zoom mínimo o tabuleiro
// visível tem 68×29 = 1.972 casas.
//
// Este caso é a INVERSA do guarda de teto: ele prende que o gesto grande chega,
// e chega inteiro. Sem ele, alguém que repusesse um teto qualquer não teria nada
// discordando.
func TestTheWholeViewportFitsInOneRectangle(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")

	// 68×29 é o viewport no zoom MÍNIMO, que é o maior gesto que um dedo alcança.
	rec := f.pede(t, f.gm, http.MethodPost,
		f.tableUrl()+"/tabuleiro/terreno/retangulo", stroke("dificil", 0, 0, 67, 28))
	if body := rec.Body.String(); strings.Contains(body, "grande demais") {
		t.Errorf("o retângulo do viewport inteiro foi recusado: %q", body[max(0, len(body)-200):])
	}
	b := f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)
	if squares := board.SquaresOf(b, "dificil"); len(squares) != 68*29 {
		t.Errorf("o retângulo 68×29 pintou %d casas, e a caixa tem %d", len(squares), 68*29)
	}
}

// O `Shift` é o que separa o TRAÇO do RETÂNGULO, e a decisão acontece no
// `pointerdown` para valer o gesto inteiro: soltar a tecla no meio do arrasto não
// pode trocar o que ele está fazendo, porque o dedo já está a caminho de um canto.
func TestTheScreenWiresTheRectangleShift(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	screen := f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()

	for _, chunk := range []string{"evt.shiftKey", "/terreno/retangulo", "board-lasso"} {
		if !strings.Contains(screen, chunk) {
			t.Errorf("a cena não tem %q: o retângulo do pincel não acontece", chunk)
		}
	}
}

// O guarda de um defeito que CONGELA A ABA e não deixa erro nenhum para trás.
//
// Os dois escrevem no MESMO lugar: o `data-show` põe `el.style.display` e o
// `data-attr:style` reescreve o atributo `style` inteiro, apagando o `display`
// que o outro acabou de pôr — que faz o outro pôr de novo. O renderizador entra
// em laço, e a aba para de responder a TUDO: sem console, sem exceção, sem
// conseguir navegar para fora. A ferramenta de medir some junto.
//
// O conserto é sempre o mesmo: quem ESCONDE é um nó, quem POSICIONA é outro.
func TestNoNodeHasDataShowAndDataAttrStyleTogether(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	screen := f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()

	// O CONTROLE: as duas diretivas existem na cena, em nós diferentes. Sem ele,
	// não achar a combinação seria verdade também sobre uma página vazia.
	for _, directive := range []string{"data-show=", "data-attr:style="} {
		if !strings.Contains(screen, directive) {
			t.Fatalf("a cena não usa %q — o guarda mediria o vazio", directive)
		}
	}

	// Cada tag aberta é uma lista de atributos até o `>`. Um `<` dentro de valor
	// de atributo não acontece no HTML servido (o templ escapa), então o corte
	// simples basta.
	for _, tag := range regexp.MustCompile(`<[a-zA-Z][^>]*>`).FindAllString(screen, -1) {
		if strings.Contains(tag, "data-show=") && strings.Contains(tag, "data-attr:style=") {
			t.Errorf("um nó tem `data-show` e `data-attr:style` juntos e vai CONGELAR a aba "+
				"em laço de escrita: %s", primeirosAtributos(tag))
		}
	}
}

// A COORDENADA NEGATIVA ATRAVESSA O CORPO: o plano não tem bordas, então
// (−3,−5) é lugar legítimo. Se a travessia perdesse o sinal, o pincel pintaria
// no quadrante errado e o mestre veria a casa não acender onde ele olhou.
func TestAStrokeInTheNegativeQuadrantPaintsThere(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")

	if rec := f.pede(t, f.gm, http.MethodPost,
		f.tableUrl()+"/tabuleiro/terreno", stroke("dificil", -3, -5, -1, -5)); rec.Code != http.StatusOK {
		t.Fatalf("o traço negativo deu %d", rec.Code)
	}

	b := f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)
	squares := board.SquaresOf(b, "dificil")
	expected := board.StrokeSquares(engine.Square{X: -3, Y: -5}, engine.Square{X: -1, Y: -5})
	if len(squares) != len(expected) {
		t.Fatalf("o traço (−3,−5)→(−1,−5) pintou %d casas, esperado %d: %v",
			len(squares), len(expected), squares)
	}
	// E elas estão MESMO no quadrante negativo: contar as casas certas não
	// distingue "pintou lá" de "pintou o espelho em (3,5)".
	for _, q := range squares {
		if q.X >= 0 || q.Y >= 0 {
			t.Errorf("a casa %v não está no quadrante negativo — o sinal se perdeu na travessia", q)
		}
	}
}

// Guarda de varredura sobre o caminho de RECUSA. Provado vermelho: apagado o
// `if err != nil` do `pointsFromBody` — que serve quatorze rotas —, a suíte
// inteira ficava verde e as frases de recusa não tinham leitor nenhum.
//
// O corpo quebrado é o caso REAL desta borda: o `payload` do `@post` é calculado
// no instante do gesto, e um sinal indefinido no meio da expressão manda
// `undefined` — que não é JSON. O que não pode acontecer é o servidor decidir
// sozinho que o gesto foi na origem.
//
// # Ele afirma a FRASE, e não o status
//
// As rotas se dividem em duas famílias com contratos diferentes:
//
//   - as que respondem SÓ SINAIS (`marcar-area`, `gabarito`, `regua`) recusam
//     em 400, porque não há cena para redesenhar;
//   - as que são COMANDO (terreno, retângulo, peça avulsa, grupo) recusam em
//     200 com a frase no `$command_error` — o Datastar DESCARTA o remendo de
//     toda resposta não-2xx, então uma recusa em 4xx não apareceria na tela.
//
// O que as duas têm em comum é a única coisa que importa para quem está na mesa:
// a frase CHEGA.
//
// # O CONTROLE
//
// Um corpo BEM FORMADO na mesma rota não pode trazer a frase. Sem ele, um
// servidor que gritasse "não entendi" em todo gesto passaria com louvor.
//
// # O que ele NÃO cobra, e por quê
//
// Corpo VAZIO não é recusa, e isso é desenho: `{}` decodifica para (0,0) em
// silêncio, porque `from` ausente e `from` em (0,0) são indistinguíveis num
// struct de inteiros. Prender isso exigiria ponteiro em todo campo de
// coordenada.
func TestEveryGestureThatReadsPointsRefusesABrokenBody(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")

	cases := []struct{ route, sentence, goodBody string }{
		{"/tabuleiro/terreno", "não entendi o gesto enviado", stroke("dificil", 2, 3, 4, 3)},
		{"/tabuleiro/terreno/limpar", "não entendi o gesto enviado", stroke("", 2, 3, 4, 3)},
		{"/tabuleiro/terreno/retangulo", "não entendi o gesto enviado", stroke("dificil", 2, 3, 4, 5)},
		{"/tabuleiro/terreno/limpar/retangulo", "não entendi o gesto enviado", stroke("", 2, 3, 4, 5)},
		{"/tabuleiro/marcar-area", "os cantos do laço precisam ser dois pares de números",
			`{"from":{"X":2,"Y":2},"to":{"X":5,"Y":5}}`},
		{"/tabuleiro/gabarito", "a origem e a mira do gabarito precisam ser dois pares de números",
			templateBody("quadrado", "1", 4, 4, 4, 4)},
		{"/tabuleiro/pecas/nova", "não entendi a peça",
			`{"from":{"X":4,"Y":3},"new_token_name":"Porta","new_token_size":1,"new_token_look":"object"}`},
		{"/tabuleiro/regua", "as paradas da régua não vieram",
			`{"ruler_points":[[0,0],[3,0]],"ruler_phase":2}`},
		{"/tabuleiro/grupo/mover", "as peças marcadas não vieram",
			`{"delta":{"X":1,"Y":1},"marked_tokens":""}`},
	}
	// `undefined` é o que uma expressão do Datastar manda quando um sinal do
	// meio dela não existe — o corpo quebrado que acontece de verdade.
	const corpoQuebrado = `{"from":{"X":undefined}}`

	measured := 0
	for _, tc := range cases {
		t.Run(tc.route, func(t *testing.T) {
			measured++
			broken := f.pede(t, f.gm, http.MethodPost, f.tableUrl()+tc.route, corpoQuebrado).Body.String()
			if !strings.Contains(broken, tc.sentence) {
				t.Errorf("corpo quebrado em %s não trouxe %q — o servidor decidiu sozinho onde foi o gesto:\n%s",
					tc.route, tc.sentence, firstChunk(broken))
			}
			// O CONTROLE, na mesma rota e no mesmo gesto.
			good := f.pede(t, f.gm, http.MethodPost, f.tableUrl()+tc.route, tc.goodBody).Body.String()
			if strings.Contains(good, tc.sentence) {
				t.Errorf("%s recusou um corpo BEM FORMADO com %q — a frase acima não prova nada",
					tc.route, tc.sentence)
			}
		})
	}
	// O DENOMINADOR. Uma tabela que encolhesse sem ninguém notar deixaria rotas
	// sem o caminho de recusa medido, que é exatamente o estado de antes.
	if measured < 9 {
		t.Errorf("a varredura mediu %d rotas, e são pelo menos nove", measured)
	}
}

// firstChunk corta a resposta para a mensagem de falha caber na tela: a cena
// redesenhada tem dezenas de milhares de bytes, e o que interessa é o começo.
func firstChunk(s string) string {
	if len(s) > 300 {
		return s[:300] + "…"
	}
	return s
}
