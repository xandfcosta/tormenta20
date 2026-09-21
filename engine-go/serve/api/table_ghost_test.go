package api

import (
	"context"
	"net/http"
	"strings"
	"t20engine/domain/board"
	"testing"

	"golang.org/x/net/html"
)

func (f sceneFixture) onBoardAt(t *testing.T, x, y int) string {
	t.Helper()
	f.seedOpenBoard(t, "stone")
	entryID := f.tracker(t)
	placed, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab,
		board.BoardToken{Label: "Arcanista", X: x, Y: y, EntryID: &entryID, CharacterID: &f.charID})
	if err != nil {
		t.Fatalf("pôr a peça em %d,%d: %v", x, y, err)
	}
	return placed.Tokens[len(placed.Tokens)-1].ID
}

// A peça é DESENHADA onde foi solta, em vez de voltar para o início do
// movimento aos olhos de quem arrastou.
//
// O CONTROLE vem antes: sem ele, "achei a peça em 3,1" não distingue "a peça
// andou" de "eu procurei a coisa errada e casei com outro nó".
func TestTheTokenIsDrawnWhereItWasDropped(t *testing.T) {
	f := newSceneFixture(t)
	tokenID := f.onBoardAt(t, 4, 2)
	f.turnPlayer(t)

	before := f.pede(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()
	if token := element(t, before, "aria-label", "Arcanista em 4, 2"); token == nil {
		t.Fatal("a peça não está em 4,2 ANTES da proposta: o canal não está aberto e o que vem abaixo não é evidência")
	} else if !strings.Contains(token["style"], "--col:4; --lin:2;") {
		t.Fatalf("a peça parada está desenhada em %q", token["style"])
	}

	// O JOGADOR desenha, e é pelos olhos DELE que se lê: para quem pede, o
	// destino é o fato — a peça sólida vai para lá. Para o mestre é o contrário,
	// e o guarda disso é o `TestForTheGmTheTokenStaysAndTheGhostGoes`.
	if rec := f.pede(t, f.player, http.MethodPost,
		f.tableUrl()+"/tabuleiro/"+tokenID+"/parada", `{"from":{"X":7,"Y":3}}`); rec.Code != http.StatusOK {
		t.Fatalf("propor a parada deu %d", rec.Code)
	}
	after := f.pede(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()

	piece := element(t, after, "aria-label", "Arcanista em 7, 3")
	if piece == nil {
		t.Fatal("a peça não é desenhada onde foi solta: ela voltou para a origem, que é o defeito do dono")
	}
	if !strings.Contains(piece["style"], "--col:7; --lin:3;") {
		t.Errorf("a peça solta em 7,3 está desenhada em %q", piece["style"])
	}
	// O ARRASTO conta do lugar DESENHADO, senão a próxima parada cai longe do
	// dedo — é a regra do `nextStepOrigin`, que antes morava no losango.
	if !strings.Contains(piece["data-on:pointerup__window"], "x: 7 + dx") {
		t.Errorf("o arrasto da peça proposta conta da origem: %q", piece["data-on:pointerup__window"])
	}

	// E a METADE QUE NÃO PODE TER MUDADO: a peça continua GRAVADA em 4,2. Sem
	// esta asserção o guarda acima passaria verde sobre uma peça que ANDOU sem
	// confirmação, que é pior que o defeito que ele conserta.
	saved := board.FindToken(f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab), tokenID)
	if saved.X != 4 || saved.Y != 2 {
		t.Errorf("a peça ANDOU na proposta, para %d,%d — o desenho virou gravação", saved.X, saved.Y)
	}
}

// O fantasma é a PEÇA e não um disco genérico: com três zumbis em campo, uma sombra
// anônima na casa não responde qual deles está a caminho.
func TestTheGhostMarksTheOriginWithTheTokenMonogram(t *testing.T) {
	f := newSceneFixture(t)
	tokenID := f.onBoardAt(t, 4, 2)

	noMovement := f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if strings.Contains(noMovement, "board-token-ghost") {
		t.Fatal("há fantasma SEM movimento proposto: ele estaria marcando um começo que não existe")
	}

	if rec := f.pede(t, f.gm, http.MethodPost,
		f.tableUrl()+"/tabuleiro/"+tokenID+"/parada", `{"from":{"X":7,"Y":3}}`); rec.Code != http.StatusOK {
		t.Fatalf("propor a parada deu %d", rec.Code)
	}
	screen := f.pede(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()

	ghost := element(t, screen, "class", "board-token-ghost")
	if ghost == nil {
		t.Fatal("a origem do movimento não tem fantasma: o começo do caminho não está marcado em lugar nenhum")
	}
	if !strings.Contains(ghost["style"], "--col:4; --lin:2;") {
		t.Errorf("o fantasma está em %q, e a peça saiu de 4,2", ghost["style"])
	}
	// Ele veste a PEÇA, e é o `--matiz` que prova: sem ele o disco sairia cinza,
	// e a cor da espécie é metade de quem ele diz que é.
	if !strings.Contains(ghost["style"], "--matiz:") {
		t.Errorf("o fantasma saiu sem a cor da espécie: %q", ghost["style"])
	}
	// E o leitor de tela não perde de onde ela saiu: quem conta é o nome da PEÇA,
	// porque o fantasma é `aria-hidden` para não anunciar o mesmo combatente duas
	// vezes na mesma cena.
	token := element(t, screen, "aria-label", "Arcanista em 7, 3")
	if token == nil || !strings.Contains(token["aria-label"], "saiu de 4, 2") {
		t.Fatalf("o nome da peça não diz de onde ela saiu: %+v", token)
	}
}

// PARA O MESTRE É O CONTRÁRIO: a peça SÓLIDA fica onde ela realmente está, e o
// FANTASMA vai para o fim do caminho (decisão do dono).
//
// A inversão diz DE QUEM É A DECISÃO. O jogador está pedindo — para ele o
// destino é o fato, e é lá que a peça dele aparece. O mestre está olhando uma
// cena que ele ainda não mudou: para ele o fato é onde a peça está, e o que é
// hipótese é o destino. Quem confirma vê o mundo como ele é; quem pede vê o
// mundo como ele quer.
//
// O guarda é o PAR: a mesma proposta lida pelos dois papéis, senão "achei a peça
// em 4,2" não se distingue de "a proposta não chegou".
func TestForTheGmTheTokenStaysAndTheGhostGoes(t *testing.T) {
	f := newSceneFixture(t)
	tokenID := f.onBoardAt(t, 4, 2)

	if rec := f.pede(t, f.gm, http.MethodPost,
		f.tableUrl()+"/tabuleiro/"+tokenID+"/parada", `{"from":{"X":7,"Y":3}}`); rec.Code != http.StatusOK {
		t.Fatalf("propor a parada deu %d", rec.Code)
	}
	screen := f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()

	if token := element(t, screen, "aria-label", "Arcanista em 4, 2"); token == nil {
		t.Error("a peça do mestre saiu da casa dela numa proposta que ele ainda não confirmou")
	}
	ghost := element(t, screen, "class", "board-token-ghost")
	if ghost == nil {
		t.Fatal("o mestre não vê fantasma nenhum: o destino proposto não está marcado")
	}
	if !strings.Contains(ghost["style"], "--col:7; --lin:3;") {
		t.Errorf("o fantasma do mestre está em %q, e o destino proposto é 7,3", ghost["style"])
	}
}

// Duas afirmações num caso só porque elas são a MESMA decisão vista de dois
// lados: a seta é o GESTO (dobra onde a pessoa clicou) e não a trilha (que dobra
// em cada casa), e ela para antes do centro para apontar a peça em vez de riscá-la.
func TestTheArrowBendsAtTheStopsAndEndsAtTheDestinationEdge(t *testing.T) {
	f := newSceneFixture(t)
	tokenID := f.onBoardAt(t, 0, 0)
	base := f.tableUrl() + "/tabuleiro/" + tokenID

	// Três paradas: (0,0) de onde ela saiu, depois (3,0) e (3,4).
	for _, square := range []string{`{"from":{"X":3,"Y":0}}`, `{"from":{"X":3,"Y":4}}`} {
		if rec := f.pede(t, f.gm, http.MethodPost, base+"/parada", square); rec.Code != http.StatusOK {
			t.Fatalf("a parada %s deu %d", square, rec.Code)
		}
	}
	screen := f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()

	wire := element(t, screen, "class", "board-move-arrow")
	if wire == nil {
		t.Fatal("o movimento não tem seta")
	}
	// Três paradas, três pontos: (0,0) → (3,0) → (3,4). A ÚLTIMA perna desce 4
	// quadrados e a ponta recua meio, então ela termina em y=4, não em 4,5.
	//
	// A trilha desta proposta tem NOVE casas; uma seta que dobrasse nelas teria
	// nove pontos, e é essa a confusão que o caso separa.
	if wire["d"] != "M 0.5 0.5 L 3.5 0.5 L 3.5 4" {
		t.Errorf("a seta saiu %q, esperado \"M 0.5 0.5 L 3.5 0.5 L 3.5 4\"", wire["d"])
	}
	if !strings.Contains(wire["marker-end"], "board-tip-move") {
		t.Errorf("a seta não tem ponta: %q — sem ela o desenho é uma régua, que não tem sentido", wire["marker-end"])
	}
}

// O guarda da FAMÍLIA de um defeito que o CSS não denuncia.
//
// `posicaoNoPlano` escreve `--col`/`--lin`/`--pegada` no `style`, e quem os
// transforma em pixels é UMA regra da folha, com a lista das classes que a
// recebem. Quem escreve a tinta de uma classe nova não volta a essa lista — e o
// resultado é um `<div>` estático de 0×0 com o `background` certo: `position` é
// `static`, a caixa tem área zero, e uma tinta de área zero simplesmente não
// desenha. Nada no DOM diz "isto está sem caixa".
//
// Já aconteceu com a TRILHA do movimento e com o ALCANCE ao mesmo tempo: o
// caminho proposto e as casas alcançáveis não apareciam para ninguém. Copiar a
// geometria para dentro de UMA classe fecha um buraco e deixa a família aberta.
//
// Ele cruza as duas pontas: o HTML SERVIDO diz quem é posicionado por `--col`, e
// a FOLHA COMPILADA diz quem recebe caixa. É por AMOSTRAGEM: quem escrever a
// classe nova amanhã cai aqui sem acrescentar uma linha, porque a pergunta é
// sobre o `--col` e não sobre um nome.
func TestEveryClassPositionedByColAndRowHasABox(t *testing.T) {
	f := newSceneFixture(t)
	tokenID := f.onBoardAt(t, 4, 2)
	// A cena precisa ter as três famílias no ar, senão o guarda mede o que
	// sobrou: terreno pintado, movimento proposto (trilha e paradas) e alcance.
	if rec := f.pede(t, f.gm, http.MethodPost,
		f.tableUrl()+"/tabuleiro/terreno", stroke("dificil", 5, 2, 5, 2)); rec.Code != http.StatusOK {
		t.Fatalf("pintar terreno deu %d", rec.Code)
	}
	for _, stop := range []string{`{"from":{"X":7,"Y":3}}`, `{"from":{"X":7,"Y":6}}`} {
		if rec := f.pede(t, f.gm, http.MethodPost,
			f.tableUrl()+"/tabuleiro/"+tokenID+"/parada", stop); rec.Code != http.StatusOK {
			t.Fatalf("a parada %s deu %d", stop, rec.Code)
		}
	}
	screen := f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()

	withBox := classesThatReceiveBox(t)
	if len(withBox) < 3 {
		t.Fatalf("a folha só dá caixa a %d classes: o canal não está aberto e o silêncio abaixo não é evidência", len(withBox))
	}

	positioned := 0
	z := html.NewTokenizer(strings.NewReader(screen))
	for {
		kind := z.Next()
		if kind == html.ErrorToken {
			break
		}
		if kind != html.StartTagToken && kind != html.SelfClosingTagToken {
			continue
		}
		attrs := map[string]string{}
		for {
			key, value, more := z.TagAttr()
			attrs[string(key)] = string(value)
			if !more {
				break
			}
		}
		if !strings.Contains(attrs["style"], "--col:") {
			continue
		}
		positioned++
		if !temAlgumaClasse(attrs["class"], withBox) {
			t.Errorf("o elemento de classe %q é posicionado por --col e NENHUMA classe dele recebe caixa na folha: ele sai 0x0 e a tinta não desenha", attrs["class"])
		}
	}
	// O CONTROLE do outro lado: sem ele, "todos os posicionados têm caixa" não se
	// distingue de "não há posicionado nenhum" — e a cena montada acima tem peça,
	// fantasma, terreno, trilha, paradas e alcance.
	if positioned < 10 {
		t.Fatalf("a cena só tem %d elementos posicionados por --col: a montagem não produziu o que este guarda vem medir", positioned)
	}
}

// O guarda da FAMÍLIA, e ele não é sobre o movimento.
//
// O templ NÃO aceita `else if` numa lista de atributos: ele fecha o primeiro
// `if`, escreve ` else` como TEXTO entre os atributos e abre um `if`
// INDEPENDENTE. Os dois ramos saem juntos — a peça do tabuleiro já serviu
// `data-on:pointerdown` DUAS VEZES por meses —, e o navegador guarda o primeiro
// e descarta o resto sem uma linha no console.
//
// Ele varre o HTML SERVIDO e não o código, que é a única forma de alcançar quem
// escrever `else if` num atributo amanhã sem ler nada disto.
func TestNoElementRepeatsAnAttribute(t *testing.T) {
	f := newSceneFixture(t)
	tokenID := f.onBoardAt(t, 4, 2)
	if rec := f.pede(t, f.gm, http.MethodPost,
		f.tableUrl()+"/tabuleiro/"+tokenID+"/parada", `{"from":{"X":7,"Y":3}}`); rec.Code != http.StatusOK {
		t.Fatalf("propor a parada deu %d", rec.Code)
	}
	screen := f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()

	z := html.NewTokenizer(strings.NewReader(screen))
	elements := 0
	for {
		kind := z.Next()
		if kind == html.ErrorToken {
			break
		}
		if kind != html.StartTagToken && kind != html.SelfClosingTagToken {
			continue
		}
		name, hasAttribute := z.TagName()
		if !hasAttribute {
			continue
		}
		elements++
		seen := map[string]bool{}
		for {
			key, _, more := z.TagAttr()
			if seen[string(key)] {
				t.Errorf("o <%s> repete o atributo %q: o navegador guarda o primeiro e descarta o resto em silêncio — procure um `else if` numa lista de atributos", name, key)
			}
			seen[string(key)] = true
			if !more {
				break
			}
		}
		// A palavra solta é a assinatura EXATA do defeito, e ela aparece como um
		// atributo sem valor. Vale afirmá-la à parte: um `else if` cujos dois
		// ramos escrevem atributos DIFERENTES não repete nada, e passaria pelo
		// laço acima deixando o ramo morto de pé.
		if seen["else"] {
			t.Errorf("o <%s> tem um atributo chamado `else`: um `else if` numa lista de atributos virou texto", name)
		}
	}
	// O CONTROLE: sem ele, "não achei atributo repetido" não se distingue de "não
	// achei elemento nenhum" — a cena tem centenas.
	if elements < 100 {
		t.Fatalf("a cena só tem %d elementos com atributo: o canal não está aberto, e o silêncio acima não é evidência", elements)
	}
}
