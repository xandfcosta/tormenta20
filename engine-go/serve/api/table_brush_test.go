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

	rec := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/terreno", stroke("dificil", 2, 2, 8, 5))
	if rec.Code != http.StatusOK {
		t.Fatalf("o traço deu %d", rec.Code)
	}

	b := f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)
	casas := board.SquaresOf(b, "dificil")

	// DEZ, escrito à mão (ALE-311). Aqui estava
	// `esperadas := board.StrokeSquares(…)` — a MESMA função que o handler chama
	// —, e uma contagem derivada do código sob teste anda junto com o defeito.
	//
	// O traço é SUPERCOVER e não Bresenham: ele inclui a casa de antes e a de
	// depois em cada degrau, porque um pincel que pula deixa buraco. De (2,2) a
	// (8,5) são as dez de `{2 2} {3 2} {4 2} {4 3} {5 3} {6 3} {6 4} {7 4}
	// {8 4} {8 5}`.
	if len(casas) != 10 {
		t.Errorf("o traço (2,2)→(8,5) pintou %d casas, e o segmento tem 10: %v", len(casas), casas)
	}
	// E o traço não tem buraco na ponta que este lado controla: a primeira e a
	// última casa do segmento estão lá. O meio é problema do `StrokeSquares`, que
	// tem guarda próprio.
	for _, ponta := range []engine.Square{{X: 2, Y: 2}, {X: 8, Y: 5}} {
		if !contem(casas, ponta) {
			t.Errorf("a casa %v não foi pintada: o traço não chega às pontas", ponta)
		}
	}
	// E NADA FORA DA CAIXA do traço, que é a metade que faltava: sabotado para
	// pintar um bloco além do pedido, este caso era VERDE — contar casas e
	// conferir as pontas não diz nada sobre o que foi pintado a mais.
	for _, casa := range casas {
		if casa.X < 2 || casa.X > 8 || casa.Y < 2 || casa.Y > 5 {
			t.Errorf("o traço (2,2)→(8,5) pintou %v, que está fora da caixa dele", casa)
		}
	}
}

// TestTheEraserStrokeClearsTheWholeSegment: o irmão do de cima, e ele existe
// porque as duas rotas são caminhos diferentes — a da borracha não tem espécie,
// e foi justamente ela que ficou para trás na primeira versão desta superfície.
func TestTheEraserStrokeClearsTheWholeSegment(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	// A ORIGEM NÃO É (0,0), e isso é o conserto de um defeito do próprio caso
	// (ALE-311): ele apagava de (0,0) a (6,6), e (0,0) é o VALOR-ZERO do struct.
	// Um `from` que parasse de ser lido — a tag `json:"from"` trocada, o corpo
	// chegando vazio — decodifica exatamente para (0,0), e o caso continuaria
	// passando. Ele era estruturalmente incapaz de detectar o que veio medir.
	if rec := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/terreno", stroke("cobertura", 4, 4, 6, 6)); rec.Code != http.StatusOK {
		t.Fatalf("pintar deu %d", rec.Code)
	}
	// A TESTEMUNHA DE FORA, pintada longe do traço da borracha. Ela é a metade
	// que faltava: sabotado para apagar um bloco 10×10 na origem ALÉM do pedido,
	// este caso era verde — "sobrou zero do traço" não diz nada sobre o que foi
	// apagado a mais.
	if rec := f.pede(t, f.mestre, http.MethodPost,
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

	if rec := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/terreno/limpar", stroke("", 4, 4, 6, 6)); rec.Code != http.StatusOK {
		t.Fatalf("apagar deu %d", rec.Code)
	}
	b = f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)
	if sobrou := board.SquaresOf(b, "cobertura"); len(sobrou) != 0 {
		t.Errorf("a borracha deixou %v pelo caminho", sobrou)
	}
	if testemunha := board.SquaresOf(b, "dificil"); len(testemunha) != 1 {
		t.Errorf("a casa (1,9), que está fora do traço da borracha, virou %v — a borracha apagou além do pedido",
			testemunha)
	}
}

// TestAForgedStrokeIsRefused.
//
// O teto é do domínio e a recusa chega como FRASE, não como 500: um traço de dez
// milhões de casas só vem de um pedido montado à mão, e a resposta certa é dizer
// o que houve.
func TestAForgedStrokeIsRefused(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")

	corpo := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/terreno", stroke("dificil", 0, 0, 9999999, 0)).Body.String()
	if !strings.Contains(corpo, "longo demais") {
		t.Errorf("o traço forjado não foi recusado com frase: %q", corpo[max(0, len(corpo)-200):])
	}
	b := f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)
	if casas := board.SquaresOf(b, "dificil"); len(casas) != 0 {
		t.Errorf("o traço recusado pintou %d casas assim mesmo", len(casas))
	}
}

// TestTheBrushDoesNotReturnTheWholeTable — o guarda dos 353 KB.
//
// Medido no navegador antes do conserto: uma casa pintada devolvia **353 KB**,
// porque o `respondGm` repinta TODAS as regiões. Num gesto de clique isso
// era caro; num gesto CONTÍNUO é proibitivo — um traço de vinte casas mandaria
// sete megabytes, e o mestre está arrastando o dedo enquanto isso chega.
//
// A asserção nomeia as duas metades: a região do mapa TEM de vir (senão o traço
// não aparece) e a do acervo NÃO pode (é a maior da Mesa, com 147 lugares, e ela
// não muda quando alguém pinta uma casa).
func TestTheBrushDoesNotReturnTheWholeTable(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")

	corpo := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/terreno", stroke("dificil", 1, 1, 1, 1)).Body.String()

	if !strings.Contains(corpo, `id="table-board"`) {
		t.Error("a resposta do pincel não traz o mapa — a casa pintada não apareceria")
	}
	for _, region := range []string{"table-archive", "table-tracker", "table-party", "table-npcs"} {
		if strings.Contains(corpo, `id="`+region+`"`) {
			t.Errorf("a resposta do pincel repinta a região %q, que não muda ao pintar uma casa", region)
		}
	}
}

// TestTheScreenWiresTheStrokeToTheRightButton.
//
// Uma afirmação sobre a FORMA do que a página serve, e é o único jeito de
// alcançar os três gestos de uma vez: `pointerdown`/`pointermove`/`pointerup` na
// camada de pintura, a rota com `/ate/` (e não a de um ponto só), e o botão 2
// caindo no caminho da borracha.
//
// Se algum deles voltar a ser um `data-on:click`, o traço morre em silêncio — a
// tela continua pintando um quadrado por clique, que é exatamente o estado que o
// dono relatou.
func TestTheScreenWiresTheStrokeToTheRightButton(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	tela := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()

	for _, pedaco := range []string{
		"data-on:pointerdown",
		"data-on:pointermove",
		"data-on:pointerup",
		"data-on:contextmenu",
		// O TRAÇO viaja no corpo desde a ALE-305, então o que a cena mostra é
		// o par de cantos montado como payload, em inglês — não mais um `/ate/` na URL.
		// A chave é MINÚSCULA desde a ALE-313: é a grafia que o `engine.Square`
		// declara (`json:"x"`) e a que está gravada no acervo.
		"to: {x: ",
		"evt.button === 2",
	} {
		if !strings.Contains(tela, pedaco) {
			t.Errorf("a cena não tem %q: o traço do pincel não acontece", pedaco)
		}
	}
	// A CAMADA DE PINTURA não pode ter voltado ao clique de um quadrado só.
	if strings.Contains(tela, `aria-label="Pintar terreno — escolha a casa"`) {
		t.Error("a camada de pintura voltou a ser um clique por casa")
	}
}

// TestThePaintedSquareCarriesTheKindIcon: a ponta que só o HTML servido responde —
// o ícone chega à casa, e o trilho mostra o MESMO.
func TestThePaintedSquareCarriesTheKindIcon(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	if rec := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/terreno", stroke("camuflagem", 3, 3, 3, 3)); rec.Code != http.StatusOK {
		t.Fatalf("pintar deu %d", rec.Code)
	}
	tela := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()

	// O CANTO vai escrito à mão ("sudeste" é o da camuflagem na tabela do
	// desenho). Ler o valor do `drawing` da cena faria o esperado sair do código
	// sob teste, e os dois andariam juntos com o defeito — é o que o CLAUDE.md
	// proíbe com todas as letras.
	if !strings.Contains(tela, "terrain-corner-southeast") {
		t.Error("a casa de camuflagem não veste o canto sudeste")
	}
	if !strings.Contains(tela, "terrain-mark") {
		t.Error("a casa pintada não tem a marca da espécie")
	}
	// E o TRILHO usa a mesma tabela: o botão do pincel tinge com a cor dela.
	if !strings.Contains(tela, "board-hue-concealment") {
		t.Error("o pincel do trilho não veste o matiz da espécie")
	}
}

// TestTheRectangleFillsTheWholeArea (ALE-203, item 10).
//
// A rota do retângulo é IRMÃ da do traço e chama a mesma gravação — o que muda é
// quais casas o par de cantos nomeia. O guarda mede as duas pontas que só esta
// camada responde: a área inteira pintada, e a borracha usando o caminho SEM
// espécie (o conserto da fatia 1, que não pode se perder numa rota nova).
func TestTheRectangleFillsTheWholeArea(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")

	if rec := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/terreno/retangulo", stroke("dificil", 2, 2, 4, 5)); rec.Code != http.StatusOK {
		t.Fatalf("o retângulo deu %d", rec.Code)
	}
	b := f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)
	// 3 colunas × 4 linhas = 12 casas, e as duas pontas incluídas.
	if casas := board.SquaresOf(b, "dificil"); len(casas) != 12 {
		t.Errorf("(2,2)→(4,5) pintou %d casas, esperado as 12 do retângulo: %v", len(casas), casas)
	}

	if rec := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/terreno/limpar/retangulo", stroke("", 2, 2, 4, 5)); rec.Code != http.StatusOK {
		t.Fatalf("limpar o retângulo deu %d", rec.Code)
	}
	b = f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)
	if sobrou := board.SquaresOf(b, "dificil"); len(sobrou) != 0 {
		t.Errorf("a borracha em área deixou %v", sobrou)
	}
}

// TestTheWholeViewportFitsInOneRectangle — o SUBSTITUTO do caso do teto.
//
// Aqui morava o `TestAForgedRectangleIsRefusedByTheRoute`, que provava que um
// retângulo de mil casas para cima era recusado. **O teto saiu por decisão do
// dono (ALE-315)**: o app roda local, e ele mordia gesto de verdade — no zoom
// mínimo o tabuleiro visível tem 68×29 = 1.972 casas, e "pinte tudo o que estou
// vendo" não passava.
//
// O caso não some, ele INVERTE: o que se prende agora é que o gesto grande
// CHEGA, e chega inteiro. Sem ele, alguém que devolvesse um teto qualquer não
// teria nada discordando.
func TestTheWholeViewportFitsInOneRectangle(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")

	// 68×29 é o viewport no zoom MÍNIMO, que é o maior gesto que um dedo alcança.
	rec := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/terreno/retangulo", stroke("dificil", 0, 0, 67, 28))
	if corpo := rec.Body.String(); strings.Contains(corpo, "grande demais") {
		t.Errorf("o retângulo do viewport inteiro foi recusado: %q", corpo[max(0, len(corpo)-200):])
	}
	b := f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)
	if casas := board.SquaresOf(b, "dificil"); len(casas) != 68*29 {
		t.Errorf("o retângulo 68×29 pintou %d casas, e a caixa tem %d", len(casas), 68*29)
	}
}

// TestTheScreenWiresTheRectangleShift.
//
// O `Shift` é o que separa o TRAÇO do RETÂNGULO, e a decisão acontece no
// `pointerdown` para valer o gesto inteiro: soltar a tecla no meio do arrasto não
// pode trocar o que ele está fazendo, porque o dedo já está a caminho de um canto.
func TestTheScreenWiresTheRectangleShift(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	tela := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()

	for _, pedaco := range []string{"evt.shiftKey", "/terreno/retangulo", "board-lasso"} {
		if !strings.Contains(tela, pedaco) {
			t.Errorf("a cena não tem %q: o retângulo do pincel não acontece", pedaco)
		}
	}
}

// TestNoNodeHasDataShowAndDataAttrStyleTogether — o guarda de um defeito que
// CONGELA A ABA, e que não deixa erro nenhum para trás.
//
// Os dois escrevem no MESMO lugar: o `data-show` põe `el.style.display` e o
// `data-attr:style` reescreve o atributo `style` inteiro, apagando o `display`
// que o outro acabou de pôr — que faz o outro pôr de novo. O renderizador entra
// em laço.
//
// Medido na bancada, e o sintoma é o pior possível: a aba para de responder a
// TUDO. Sem console, sem exceção, sem sequer conseguir navegar para fora — a
// própria ferramenta de medir some junto, e o que sobra é "o navegador travou",
// que não aponta para lugar nenhum.
//
// O conserto é sempre o mesmo: quem ESCONDE é um nó, quem POSICIONA é outro.
func TestNoNodeHasDataShowAndDataAttrStyleTogether(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	tela := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()

	// O CONTROLE: as duas diretivas existem na cena, em nós diferentes. Sem ele,
	// não achar a combinação seria verdade também sobre uma página vazia.
	for _, diretiva := range []string{"data-show=", "data-attr:style="} {
		if !strings.Contains(tela, diretiva) {
			t.Fatalf("a cena não usa %q — o guarda mediria o vazio", diretiva)
		}
	}

	// Cada tag aberta é uma lista de atributos até o `>`. Um `<` dentro de valor
	// de atributo não acontece no HTML servido (o templ escapa), então o corte
	// simples basta.
	for _, tag := range regexp.MustCompile(`<[a-zA-Z][^>]*>`).FindAllString(tela, -1) {
		if strings.Contains(tag, "data-show=") && strings.Contains(tag, "data-attr:style=") {
			t.Errorf("um nó tem `data-show` e `data-attr:style` juntos e vai CONGELAR a aba "+
				"em laço de escrita: %s", primeirosAtributos(tag))
		}
	}
}

// A COORDENADA NEGATIVA ATRAVESSA O CORPO (ALE-305).
//
// Ela é a razão ESCRITA para as pontas do traço não virarem sinal da página — o
// plano não tem bordas, então (−3,−5) é lugar legítimo — e nunca teve teste: o
// caminho a carregava por acaso, porque `/-3/-5` é segmento válido e ninguém
// tinha medido.
//
// O corte da ALE-305 tirou as pontas do caminho e pôs no corpo. Se a travessia
// não preservasse o sinal negativo, o pincel pintaria no quadrante errado e o
// mapa pareceria vazio — o mestre pinta e nada acontece onde ele olhou.
func TestAStrokeInTheNegativeQuadrantPaintsThere(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")

	if rec := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/terreno", stroke("dificil", -3, -5, -1, -5)); rec.Code != http.StatusOK {
		t.Fatalf("o traço negativo deu %d", rec.Code)
	}

	b := f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)
	casas := board.SquaresOf(b, "dificil")
	esperadas := board.StrokeSquares(engine.Square{X: -3, Y: -5}, engine.Square{X: -1, Y: -5})
	if len(casas) != len(esperadas) {
		t.Fatalf("o traço (−3,−5)→(−1,−5) pintou %d casas, esperado %d: %v",
			len(casas), len(esperadas), casas)
	}
	// E elas estão MESMO no quadrante negativo: contar as casas certas não
	// distingue "pintou lá" de "pintou o espelho em (3,5)".
	for _, q := range casas {
		if q.X >= 0 || q.Y >= 0 {
			t.Errorf("a casa %v não está no quadrante negativo — o sinal se perdeu na travessia", q)
		}
	}
}

// TestEveryGestureThatReadsPointsRefusesABrokenBody é guarda de varredura sobre
// o caminho de RECUSA (ALE-311).
//
// Nenhuma das vinte rotas convertidas na ALE-305/306/307 tinha caso aqui.
// Medido: apagado o `if err != nil` do `pointsFromBody` — que serve QUATORZE
// delas —, a suíte inteira ficava verde, e as sete frases de recusa tinham ZERO
// ocorrências em `api/*_test.go`, `web/table/*_test.go` e `e2e/`.
//
// O corpo quebrado é o caso REAL desta borda: o `payload` do `@post` é calculado
// no instante do gesto, e um sinal indefinido no meio da expressão manda
// `undefined` — que não é JSON. O que não pode acontecer é o servidor decidir
// sozinho que o gesto foi na origem.
//
// # O que ele afirma é a FRASE, e não o status
//
// As nove rotas se dividem em duas famílias com contratos diferentes, e a
// primeira versão deste caso reprovou quatro delas por medir o contrato errado:
//
//   - as que respondem SÓ SINAIS (`marcar-area`, `gabarito`, `regua`) recusam
//     em 400, porque não há cena para redesenhar;
//   - as que são COMANDO (terreno, retângulo, peça avulsa, grupo) recusam em
//     200 com a frase no `$command_error`, que é o padrão da casa — o Datastar
//     DESCARTA o remendo de toda resposta não-2xx, então uma recusa em 4xx aqui
//     não apareceria na tela.
//
// O que as duas famílias têm em comum é a única coisa que importa para quem
// está na mesa: a frase CHEGA. É por isso que o guarda prende a frase.
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
// coordenada, e a decisão fica registrada aqui em vez de virar um caso que
// afirma o contrário do produto.
func TestEveryGestureThatReadsPointsRefusesABrokenBody(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")

	casos := []struct{ rota, frase, corpoBom string }{
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

	medidos := 0
	for _, caso := range casos {
		t.Run(caso.rota, func(t *testing.T) {
			medidos++
			quebrado := f.pede(t, f.mestre, http.MethodPost, f.tableUrl()+caso.rota, corpoQuebrado).Body.String()
			if !strings.Contains(quebrado, caso.frase) {
				t.Errorf("corpo quebrado em %s não trouxe %q — o servidor decidiu sozinho onde foi o gesto:\n%s",
					caso.rota, caso.frase, firstChunk(quebrado))
			}
			// O CONTROLE, na mesma rota e no mesmo gesto.
			bom := f.pede(t, f.mestre, http.MethodPost, f.tableUrl()+caso.rota, caso.corpoBom).Body.String()
			if strings.Contains(bom, caso.frase) {
				t.Errorf("%s recusou um corpo BEM FORMADO com %q — a frase acima não prova nada",
					caso.rota, caso.frase)
			}
		})
	}
	// O DENOMINADOR. Uma tabela que encolhesse sem ninguém notar deixaria rotas
	// sem o caminho de recusa medido, que é exatamente o estado de antes.
	if medidos < 9 {
		t.Errorf("a varredura mediu %d rotas, e são pelo menos nove", medidos)
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
