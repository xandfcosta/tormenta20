package api

import (
	"context"
	"net/http"
	"strings"
	"t20engine/tabuleiro"
	"testing"

	"golang.org/x/net/html"
)

func (f pilotoFixture) onBoardAt(t *testing.T, x, y int) string {
	t.Helper()
	f.seedOpenBoard(t, "pedra")
	entryID := f.tracker(t)
	posto, err := f.s.Boards().AddToken(context.Background(), f.sessionID, defaultTab,
		tabuleiro.BoardToken{Label: "Arcanista", X: x, Y: y, EntryID: &entryID, CharacterID: &f.charID}, true)
	if err != nil {
		t.Fatalf("pôr a peça em %d,%d: %v", x, y, err)
	}
	return posto.Tokens[len(posto.Tokens)-1].ID
}

// TestTheTokenIsDrawnWhereItWasDropped — o coração do item 4.
//
// O dono descreveu o defeito como "ela volta para o início do movimento": o
// gesto acabava se desfazendo aos olhos de quem arrastou, e o que marcava o
// destino era um losango que não parecia a peça.
//
// O CONTROLE vem antes: sem ele, "achei a peça em 3,1" não distingue "a peça
// andou" de "eu procurei a coisa errada e casei com outro nó".
func TestTheTokenIsDrawnWhereItWasDropped(t *testing.T) {
	f := novoPiloto(t)
	tokenID := f.onBoardAt(t, 4, 2)
	f.turnPlayer(t)

	antes := f.pede(t, f.jogador, http.MethodGet, f.tableUrl(), "").Body.String()
	if peca := element(t, antes, "aria-label", "Arcanista em 4, 2"); peca == nil {
		t.Fatal("a peça não está em 4,2 ANTES da proposta: o canal não está aberto e o que vem abaixo não é evidência")
	} else if !strings.Contains(peca["style"], "--col:4; --lin:2;") {
		t.Fatalf("a peça parada está desenhada em %q", peca["style"])
	}

	// O JOGADOR desenha, e é pelos olhos DELE que se lê: para quem pede, o
	// destino é o fato — a peça sólida vai para lá. Para o mestre é o contrário,
	// e o guarda disso é o `TestForTheGmTheTokenStaysAndTheGhostGoes`.
	if rec := f.pede(t, f.jogador, http.MethodPost,
		f.tableUrl()+"/tabuleiro/"+tokenID+"/parada/7/3", ""); rec.Code != http.StatusOK {
		t.Fatalf("propor a parada deu %d", rec.Code)
	}
	depois := f.pede(t, f.jogador, http.MethodGet, f.tableUrl(), "").Body.String()

	peca := element(t, depois, "aria-label", "Arcanista em 7, 3")
	if peca == nil {
		t.Fatal("a peça não é desenhada onde foi solta: ela voltou para a origem, que é o defeito do dono")
	}
	if !strings.Contains(peca["style"], "--col:7; --lin:3;") {
		t.Errorf("a peça solta em 7,3 está desenhada em %q", peca["style"])
	}
	// O ARRASTO conta do lugar DESENHADO, senão a próxima parada cai longe do
	// dedo — é a regra do `nextStepOrigin`, que antes morava no losango.
	if !strings.Contains(peca["data-on:pointerup__window"], "(7 + dx)") {
		t.Errorf("o arrasto da peça proposta conta da origem: %q", peca["data-on:pointerup__window"])
	}

	// E a METADE QUE NÃO PODE TER MUDADO: a peça continua GRAVADA em 4,2. Sem
	// esta asserção o guarda acima passaria verde sobre uma peça que ANDOU sem
	// confirmação, que é pior que o defeito que ele conserta.
	gravada := tabuleiro.FindToken(f.s.Boards().Get(context.Background(), f.sessionID, defaultTab), tokenID)
	if gravada.X != 4 || gravada.Y != 2 {
		t.Errorf("a peça ANDOU na proposta, para %d,%d — o desenho virou gravação", gravada.X, gravada.Y)
	}
}

// TestTheGhostMarksTheOriginWithTheTokenMonogram.
//
// Ele é a peça e não um disco genérico: com três zumbis em campo, uma sombra
// anônima na casa não responde qual deles está a caminho.
func TestTheGhostMarksTheOriginWithTheTokenMonogram(t *testing.T) {
	f := novoPiloto(t)
	tokenID := f.onBoardAt(t, 4, 2)

	semMovimento := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()
	if strings.Contains(semMovimento, "tabuleiro-peca-fantasma") {
		t.Fatal("há fantasma SEM movimento proposto: ele estaria marcando um começo que não existe")
	}

	if rec := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/"+tokenID+"/parada/7/3", ""); rec.Code != http.StatusOK {
		t.Fatalf("propor a parada deu %d", rec.Code)
	}
	tela := f.pede(t, f.jogador, http.MethodGet, f.tableUrl(), "").Body.String()

	fantasma := element(t, tela, "class", "tabuleiro-peca-fantasma")
	if fantasma == nil {
		t.Fatal("a origem do movimento não tem fantasma: o começo do caminho não está marcado em lugar nenhum")
	}
	if !strings.Contains(fantasma["style"], "--col:4; --lin:2;") {
		t.Errorf("o fantasma está em %q, e a peça saiu de 4,2", fantasma["style"])
	}
	// Ele veste a PEÇA, e é o `--matiz` que prova: sem ele o disco sairia cinza,
	// e a cor da espécie é metade de quem ele diz que é (ALE-179).
	if !strings.Contains(fantasma["style"], "--matiz:") {
		t.Errorf("o fantasma saiu sem a cor da espécie: %q", fantasma["style"])
	}
	// E o leitor de tela não perde de onde ela saiu: quem conta é o nome da PEÇA,
	// porque o fantasma é `aria-hidden` para não anunciar o mesmo combatente duas
	// vezes na mesma cena.
	peca := element(t, tela, "aria-label", "Arcanista em 7, 3")
	if peca == nil || !strings.Contains(peca["aria-label"], "saiu de 4, 2") {
		t.Fatalf("o nome da peça não diz de onde ela saiu: %+v", peca)
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
	f := novoPiloto(t)
	tokenID := f.onBoardAt(t, 4, 2)

	if rec := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/"+tokenID+"/parada/7/3", ""); rec.Code != http.StatusOK {
		t.Fatalf("propor a parada deu %d", rec.Code)
	}
	tela := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()

	if peca := element(t, tela, "aria-label", "Arcanista em 4, 2"); peca == nil {
		t.Error("a peça do mestre saiu da casa dela numa proposta que ele ainda não confirmou")
	}
	fantasma := element(t, tela, "class", "tabuleiro-peca-fantasma")
	if fantasma == nil {
		t.Fatal("o mestre não vê fantasma nenhum: o destino proposto não está marcado")
	}
	if !strings.Contains(fantasma["style"], "--col:7; --lin:3;") {
		t.Errorf("o fantasma do mestre está em %q, e o destino proposto é 7,3", fantasma["style"])
	}
}

// TestTheArrowBendsAtTheStopsAndEndsAtTheDestinationEdge.
//
// Duas afirmações num caso só porque elas são a MESMA decisão vista de dois
// lados: a seta é o GESTO (dobra onde a pessoa clicou) e não a trilha (que dobra
// em cada casa), e ela para antes do centro para apontar a peça em vez de riscá-la.
func TestTheArrowBendsAtTheStopsAndEndsAtTheDestinationEdge(t *testing.T) {
	f := novoPiloto(t)
	tokenID := f.onBoardAt(t, 0, 0)
	base := f.tableUrl() + "/tabuleiro/" + tokenID

	for _, parada := range []string{"/parada/3/0", "/parada/3/4"} {
		if rec := f.pede(t, f.mestre, http.MethodPost, base+parada, ""); rec.Code != http.StatusOK {
			t.Fatalf("a parada %s deu %d", parada, rec.Code)
		}
	}
	tela := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()

	fio := element(t, tela, "class", "tabuleiro-movimento-fio")
	if fio == nil {
		t.Fatal("o movimento não tem seta")
	}
	// Três paradas, três pontos: (0,0) → (3,0) → (3,4). A ÚLTIMA perna desce 4
	// quadrados e a ponta recua meio, então ela termina em y=4, não em 4,5.
	//
	// A trilha desta proposta tem NOVE casas; uma seta que dobrasse nelas teria
	// nove pontos, e é essa a confusão que o caso separa.
	if fio["d"] != "M 0.5 0.5 L 3.5 0.5 L 3.5 4" {
		t.Errorf("a seta saiu %q, esperado \"M 0.5 0.5 L 3.5 0.5 L 3.5 4\"", fio["d"])
	}
	if !strings.Contains(fio["marker-end"], "tabuleiro-ponta-do-movimento") {
		t.Errorf("a seta não tem ponta: %q — sem ela o desenho é uma régua, que não tem sentido", fio["marker-end"])
	}
}

// TestEveryClassPositionedByColAndRowHasABox — o guarda da FAMÍLIA de um defeito
// que o CSS não denuncia.
//
// `posicaoNoPlano` escreve `--col`/`--lin`/`--pegada` no `style`, e quem os
// transforma em pixels é UMA regra da folha, com a lista das classes que a
// recebem. Quem escreve a tinta de uma classe nova não volta a essa lista — e o
// resultado é um `<div>` estático de 0×0 com o `background` certo: `position` é
// `static`, a caixa tem área zero, e uma tinta de área zero simplesmente não
// desenha. Nada no DOM diz "isto está sem caixa".
//
// Medido no navegador (ALE-203): a TRILHA do movimento e o ALCANCE estavam assim
// desde que a moldura saiu — o caminho proposto e as casas alcançáveis não
// apareciam para ninguém. A `.tabuleiro-parada` escapou por ter copiado a
// geometria para dentro de si, que é o remendo que fecha um buraco e deixa a
// família aberta.
//
// Ele cruza as duas pontas: o HTML SERVIDO diz quem é posicionado por `--col`, e
// a FOLHA COMPILADA diz quem recebe caixa. É a mesma forma do
// `TestNoLayerReadsThePointWithoutAddingTheViewport`, e é por AMOSTRAGEM: quem escrever
// a classe nova amanhã cai aqui sem acrescentar uma linha, porque a pergunta é
// sobre o `--col` e não sobre um nome.
func TestEveryClassPositionedByColAndRowHasABox(t *testing.T) {
	f := novoPiloto(t)
	tokenID := f.onBoardAt(t, 4, 2)
	// A cena precisa ter as três famílias no ar, senão o guarda mede o que
	// sobrou: terreno pintado, movimento proposto (trilha e paradas) e alcance.
	if rec := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/terreno/dificil/5/2/ate/5/2", ""); rec.Code != http.StatusOK {
		t.Fatalf("pintar terreno deu %d", rec.Code)
	}
	for _, parada := range []string{"/parada/7/3", "/parada/7/6"} {
		if rec := f.pede(t, f.mestre, http.MethodPost,
			f.tableUrl()+"/tabuleiro/"+tokenID+parada, ""); rec.Code != http.StatusOK {
			t.Fatalf("a parada %s deu %d", parada, rec.Code)
		}
	}
	tela := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()

	comCaixa := classesThatReceiveBox(t)
	if len(comCaixa) < 3 {
		t.Fatalf("a folha só dá caixa a %d classes: o canal não está aberto e o silêncio abaixo não é evidência", len(comCaixa))
	}

	posicionados := 0
	z := html.NewTokenizer(strings.NewReader(tela))
	for {
		tipo := z.Next()
		if tipo == html.ErrorToken {
			break
		}
		if tipo != html.StartTagToken && tipo != html.SelfClosingTagToken {
			continue
		}
		attrs := map[string]string{}
		for {
			chave, valor, mais := z.TagAttr()
			attrs[string(chave)] = string(valor)
			if !mais {
				break
			}
		}
		if !strings.Contains(attrs["style"], "--col:") {
			continue
		}
		posicionados++
		if !temAlgumaClasse(attrs["class"], comCaixa) {
			t.Errorf("o elemento de classe %q é posicionado por --col e NENHUMA classe dele recebe caixa na folha: ele sai 0x0 e a tinta não desenha", attrs["class"])
		}
	}
	// O CONTROLE do outro lado: sem ele, "todos os posicionados têm caixa" não se
	// distingue de "não há posicionado nenhum" — e a cena montada acima tem peça,
	// fantasma, terreno, trilha, paradas e alcance.
	if posicionados < 10 {
		t.Fatalf("a cena só tem %d elementos posicionados por --col: a montagem não produziu o que este guarda vem medir", posicionados)
	}
}

// TestNoElementRepeatsAnAttribute — o guarda da FAMÍLIA, e ele não é sobre o
// movimento.
//
// O templ NÃO aceita `else if` numa lista de atributos: ele fecha o primeiro
// `if`, escreve ` else` como TEXTO entre os atributos e abre um `if`
// INDEPENDENTE. Os dois ramos saem juntos, e a peça do tabuleiro serviu
// `data-on:pointerdown` DUAS VEZES durante toda a ALE-203 — o navegador guarda o
// primeiro e descarta o resto sem uma linha no console.
//
// Ele varre o HTML SERVIDO e não o código, que é a única forma de alcançar quem
// escrever `else if` num atributo amanhã sem ler nada disto. É o mesmo molde do
// `TestNoLayerReadsThePointWithoutAddingTheViewport`.
func TestNoElementRepeatsAnAttribute(t *testing.T) {
	f := novoPiloto(t)
	tokenID := f.onBoardAt(t, 4, 2)
	if rec := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/"+tokenID+"/parada/7/3", ""); rec.Code != http.StatusOK {
		t.Fatalf("propor a parada deu %d", rec.Code)
	}
	tela := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()

	z := html.NewTokenizer(strings.NewReader(tela))
	elementos := 0
	for {
		tipo := z.Next()
		if tipo == html.ErrorToken {
			break
		}
		if tipo != html.StartTagToken && tipo != html.SelfClosingTagToken {
			continue
		}
		nome, temAtributo := z.TagName()
		if !temAtributo {
			continue
		}
		elementos++
		vistos := map[string]bool{}
		for {
			chave, _, mais := z.TagAttr()
			if vistos[string(chave)] {
				t.Errorf("o <%s> repete o atributo %q: o navegador guarda o primeiro e descarta o resto em silêncio — procure um `else if` numa lista de atributos", nome, chave)
			}
			vistos[string(chave)] = true
			if !mais {
				break
			}
		}
		// A palavra solta é a assinatura EXATA do defeito, e ela aparece como um
		// atributo sem valor. Vale afirmá-la à parte: um `else if` cujos dois
		// ramos escrevem atributos DIFERENTES não repete nada, e passaria pelo
		// laço acima deixando o ramo morto de pé.
		if vistos["else"] {
			t.Errorf("o <%s> tem um atributo chamado `else`: um `else if` numa lista de atributos virou texto", nome)
		}
	}
	// O CONTROLE: sem ele, "não achei atributo repetido" não se distingue de "não
	// achei elemento nenhum" — a cena tem centenas.
	if elementos < 100 {
		t.Fatalf("a cena só tem %d elementos com atributo: o canal não está aberto, e o silêncio acima não é evidência", elementos)
	}
}
