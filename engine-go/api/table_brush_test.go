package api

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"t20engine/engine"
	"t20engine/tabuleiro"
	"testing"
)

func TestTheStrokePaintsTheWholeSegment(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")

	rec := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/terreno/dificil/2/2/ate/8/5", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("o traço deu %d", rec.Code)
	}

	b := f.s.Boards().Get(context.Background(), f.sessionID, defaultTab)
	casas := tabuleiro.SquaresOf(b, "dificil")
	esperadas := tabuleiro.StrokeSquares(engine.Square{X: 2, Y: 2}, engine.Square{X: 8, Y: 5})
	if len(casas) != len(esperadas) {
		t.Errorf("o traço (2,2)→(8,5) pintou %d casas, esperado as %d do segmento: %v",
			len(casas), len(esperadas), casas)
	}
	// E o traço não tem buraco na ponta que este lado controla: a primeira e a
	// última casa do segmento estão lá. O meio é problema do `StrokeSquares`, que
	// tem guarda próprio.
	for _, ponta := range []engine.Square{{X: 2, Y: 2}, {X: 8, Y: 5}} {
		if !contem(casas, ponta) {
			t.Errorf("a casa %v não foi pintada: o traço não chega às pontas", ponta)
		}
	}
}

// TestTheEraserStrokeClearsTheWholeSegment: o irmão do de cima, e ele existe
// porque as duas rotas são caminhos diferentes — a da borracha não tem espécie,
// e foi justamente ela que ficou para trás na primeira versão desta superfície.
func TestTheEraserStrokeClearsTheWholeSegment(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")
	if rec := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/terreno/cobertura/0/0/ate/6/6", ""); rec.Code != http.StatusOK {
		t.Fatalf("pintar deu %d", rec.Code)
	}
	// O CONTROLE: havia o que apagar. Sem ele, "sobrou zero" é verdade também
	// sobre um tabuleiro em que nada foi pintado.
	b := f.s.Boards().Get(context.Background(), f.sessionID, defaultTab)
	if len(tabuleiro.SquaresOf(b, "cobertura")) < 7 {
		t.Fatalf("o traço de pintura só fez %d casas — não há o que a borracha apagar",
			len(tabuleiro.SquaresOf(b, "cobertura")))
	}

	if rec := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/terreno/limpar/0/0/ate/6/6", ""); rec.Code != http.StatusOK {
		t.Fatalf("apagar deu %d", rec.Code)
	}
	b = f.s.Boards().Get(context.Background(), f.sessionID, defaultTab)
	if sobrou := tabuleiro.SquaresOf(b, "cobertura"); len(sobrou) != 0 {
		t.Errorf("a borracha deixou %v pelo caminho", sobrou)
	}
}

// TestAForgedStrokeIsRefused.
//
// O teto é do domínio e a recusa chega como FRASE, não como 500: um traço de dez
// milhões de casas só vem de um pedido montado à mão, e a resposta certa é dizer
// o que houve.
func TestAForgedStrokeIsRefused(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")

	corpo := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/terreno/dificil/0/0/ate/9999999/0", "").Body.String()
	if !strings.Contains(corpo, "longo demais") {
		t.Errorf("o traço forjado não foi recusado com frase: %q", corpo[max(0, len(corpo)-200):])
	}
	b := f.s.Boards().Get(context.Background(), f.sessionID, defaultTab)
	if casas := tabuleiro.SquaresOf(b, "dificil"); len(casas) != 0 {
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
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")

	corpo := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/terreno/dificil/1/1/ate/1/1", "").Body.String()

	if !strings.Contains(corpo, `id="mesa-tabuleiro"`) {
		t.Error("a resposta do pincel não traz o mapa — a casa pintada não apareceria")
	}
	for _, region := range []string{"mesa-acervo", "mesa-fila", "mesa-grupo", "mesa-npcs"} {
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
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")
	tela := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()

	for _, pedaco := range []string{
		"data-on:pointerdown",
		"data-on:pointermove",
		"data-on:pointerup",
		"data-on:contextmenu",
		"/ate/",
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
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")
	if rec := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/terreno/camuflagem/3/3/ate/3/3", ""); rec.Code != http.StatusOK {
		t.Fatalf("pintar deu %d", rec.Code)
	}
	tela := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()

	// O CANTO vai escrito à mão ("sudeste" é o da camuflagem na tabela do
	// desenho). Ler o valor do `drawing` da cena faria o esperado sair do código
	// sob teste, e os dois andariam juntos com o defeito — é o que o CLAUDE.md
	// proíbe com todas as letras.
	if !strings.Contains(tela, "terreno-canto-sudeste") {
		t.Error("a casa de camuflagem não veste o canto sudeste")
	}
	if !strings.Contains(tela, "terreno-marca") {
		t.Error("a casa pintada não tem a marca da espécie")
	}
	// E o TRILHO usa a mesma tabela: o botão do pincel tinge com a cor dela.
	if !strings.Contains(tela, "tabuleiro-matiz-camuflagem") {
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
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")

	if rec := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/terreno/dificil/retangulo/2/2/4/5", ""); rec.Code != http.StatusOK {
		t.Fatalf("o retângulo deu %d", rec.Code)
	}
	b := f.s.Boards().Get(context.Background(), f.sessionID, defaultTab)
	// 3 colunas × 4 linhas = 12 casas, e as duas pontas incluídas.
	if casas := tabuleiro.SquaresOf(b, "dificil"); len(casas) != 12 {
		t.Errorf("(2,2)→(4,5) pintou %d casas, esperado as 12 do retângulo: %v", len(casas), casas)
	}

	if rec := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/terreno/limpar/retangulo/2/2/4/5", ""); rec.Code != http.StatusOK {
		t.Fatalf("limpar o retângulo deu %d", rec.Code)
	}
	b = f.s.Boards().Get(context.Background(), f.sessionID, defaultTab)
	if sobrou := tabuleiro.SquaresOf(b, "dificil"); len(sobrou) != 0 {
		t.Errorf("a borracha em área deixou %v", sobrou)
	}
}

// TestAForgedRectangleIsRefusedByTheRoute: o teto é do domínio e a recusa chega
// como FRASE. Mil casas são 32×32 — uma sala grande de masmorra.
func TestAForgedRectangleIsRefusedByTheRoute(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")

	corpo := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/terreno/dificil/retangulo/0/0/999/999", "").Body.String()
	if !strings.Contains(corpo, "grande demais") {
		t.Errorf("o retângulo forjado não foi recusado com frase: %q", corpo[max(0, len(corpo)-200):])
	}
	b := f.s.Boards().Get(context.Background(), f.sessionID, defaultTab)
	if casas := tabuleiro.SquaresOf(b, "dificil"); len(casas) != 0 {
		t.Errorf("o retângulo recusado pintou %d casas assim mesmo", len(casas))
	}
}

// TestTheScreenWiresTheRectangleShift.
//
// O `Shift` é o que separa o TRAÇO do RETÂNGULO, e a decisão acontece no
// `pointerdown` para valer o gesto inteiro: soltar a tecla no meio do arrasto não
// pode trocar o que ele está fazendo, porque o dedo já está a caminho de um canto.
func TestTheScreenWiresTheRectangleShift(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")
	tela := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()

	for _, pedaco := range []string{"evt.shiftKey", "/retangulo/", "tabuleiro-laco"} {
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
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")
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
