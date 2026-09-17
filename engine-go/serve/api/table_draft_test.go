package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"t20engine/domain/board"
	"t20engine/domain/engine"
)

/*
O RASCUNHO DE LUGAR: a cena montada FORA da sessão.

O que estes casos prendem é o CAMINHO — a superfície do tabuleiro apontada para
o acervo — e as três coisas que ele não pode errar: quem entra, para onde os
gestos postam, e o que a tela promete sobre gravar.

Integração e não unitário porque é COMPOSIÇÃO que se está provando: a rota, a
trava, a view do tabuleiro reusada e a gravação no acervo. A regra de cada
mutação já está presa no `tabuleiro`, e reafirmá-la aqui seria a mesma
fronteira duas vezes.
*/

// draftPlace guarda um lugar no acervo e devolve o id dele.
func (f sceneFixture) draftPlace(t *testing.T, nome, chao string) int64 {
	t.Helper()
	lugar, err := f.s.tableHost().Boards().NewPlace(context.Background(), f.campaignID, nome, chao)
	if err != nil {
		t.Fatalf("criar o lugar %q: %v", nome, err)
	}
	return lugar.ID
}

func (f sceneFixture) draftUrl(placeID int64) string {
	return fmt.Sprintf("/campanhas/%d/lugares/%d", f.campaignID, placeID)
}

// A cena do rascunho desenha o TABULEIRO — o mesmo plano, o mesmo trilho — e
// diz, na tarja, que ninguém está vendo.
//
// A tarja é a razão de este caso existir: o mapa do rascunho é IGUALZINHO ao da
// mesa, e sem uma linha dizendo o contrário o mestre monta a emboscada sem
// saber de que lado do tempo ele está.
func TestTheDraftDrawsTheBoardAndSaysNobodyIsWatching(t *testing.T) {
	f := newSceneFixture(t)
	lugar := f.draftPlace(t, "Cripta de Thwor", "crypt")

	corpo := f.pede(t, f.mestre, http.MethodGet, f.draftUrl(lugar), "").Body.String()

	if !strings.Contains(corpo, "board-plane") {
		t.Error("o rascunho não desenhou o plano do tabuleiro")
	}
	if !strings.Contains(corpo, "Cripta de Thwor") {
		t.Error("o rascunho não diz que lugar está sendo montado")
	}
	if !strings.Contains(corpo, "a mesa não vê") {
		t.Error("a tarja não diz que ninguém está vendo — o mestre não tem como saber em que tempo está")
	}
	if !strings.Contains(corpo, "ground-crypt") {
		t.Error("o chão escolhido não foi desenhado")
	}
}

// OS GESTOS POSTAM NO ACERVO, e não numa sessão que não existe.
//
// É o guarda do `BoardView.Base`, medido no HTML que sai. Sem ele, uma chamada
// que continuasse escrevendo o caminho da mesa postaria em `/mesa/N/0/tabuleiro`
// — um endereço que RESPONDE, com 403 ou 404, e devolve uma tela que não mudou.
// O sintoma seria "o pincel não pinta", sem uma linha em lugar nenhum.
func TestTheDraftGesturesPostToTheArchiveAndNotToATable(t *testing.T) {
	f := newSceneFixture(t)
	lugar := f.draftPlace(t, "Cripta de Thwor", "crypt")

	corpo := f.pede(t, f.mestre, http.MethodGet, f.draftUrl(lugar), "").Body.String()

	esperado := fmt.Sprintf("/campanhas/%d/lugares/%d/tabuleiro", f.campaignID, lugar)
	if !strings.Contains(corpo, esperado) {
		t.Fatalf("nenhum gesto posta em %q", esperado)
	}
	// O CONTROLE, e ele é o que separa "não achei" de "não procurei": a mesma
	// página NÃO pode carregar o caminho da mesa. `/mesa/` sozinho apareceria
	// num link de navegação legítimo, então o que se procura é o caminho do
	// TABULEIRO de uma sessão.
	if strings.Contains(corpo, "/tabuleiro/terreno") && !strings.Contains(corpo, esperado+"/terreno") {
		t.Error("o pincel do rascunho posta num tabuleiro que não é o dele")
	}
	if strings.Contains(corpo, fmt.Sprintf("/mesa/%d/0/tabuleiro", f.campaignID)) {
		t.Error("um gesto escapou para a mesa com sessão ZERO — é o defeito que o `Base` existe para impedir")
	}
}

// O gesto atravessa o caminho inteiro e o acervo muda.
//
// Pela porta de VERDADE (`posta`), num servidor HTTP real: o SDK do Datastar
// fecha o corpo do pedido ao criar o gerador SSE, e o par
// `httptest.NewRequest` + recorder não reproduz esse ciclo de vida — a ordem
// trocada passa verde na suíte e quebra toda escrita no servidor.
func TestADraftGestureChangesTheArchivedScene(t *testing.T) {
	f := newSceneFixture(t)
	lugar := f.draftPlace(t, "Cripta de Thwor", "crypt")

	f.posta(t, f.mestre, f.draftUrl(lugar)+"/tabuleiro/pecas/nova", `{"from":{"X":4,"Y":3},"new_token_name":"Porta da cripta","new_token_size":1,"new_token_look":"object"}`)

	cena, err := f.s.tableHost().Boards().PlaceScene(context.Background(), f.campaignID, lugar)
	if err != nil {
		t.Fatalf("reabrir o rascunho: %v", err)
	}
	if len(cena.Tokens) != 1 {
		t.Fatalf("o gesto não chegou ao acervo: %+v", cena.Tokens)
	}
	if cena.Tokens[0].Label != "Porta da cripta" || cena.Tokens[0].X != 4 || cena.Tokens[0].Y != 3 {
		t.Errorf("a peça não nasceu onde o clique disse: %+v", cena.Tokens[0])
	}
	// A peça nasce com id do SERVIDOR: sem ele nada consegue selecioná-la para
	// mover, editar ou remover no gesto seguinte.
	if cena.Tokens[0].ID == "" {
		t.Error("a peça do rascunho nasceu sem id")
	}
}

// O ACERVO É DO MESTRE. Um jogador da campanha não monta o rascunho — e a trava
// é do SERVIDOR, não do botão escondido.
//
// O que ele veria não é um detalhe: a cripta de sábado, com a emboscada
// posicionada e os marcadores que ainda não foram revelados.
func TestAStrangerDoesNotReachThePlaceDraft(t *testing.T) {
	f := newSceneFixture(t)
	lugar := f.draftPlace(t, "Cripta de Thwor", "crypt")

	pagina := f.pede(t, f.jogador, http.MethodGet, f.draftUrl(lugar), "")
	if pagina.Code != http.StatusForbidden {
		t.Errorf("o jogador abriu o rascunho com %d", pagina.Code)
	}
	if strings.Contains(pagina.Body.String(), "Cripta de Thwor") {
		t.Error("a recusa vazou o nome do lugar que ela existe para esconder")
	}

	// E POSTANDO NA MÃO, que é o caso que o botão escondido não cobre.
	gesto := f.posta(t, f.jogador, f.draftUrl(lugar)+"/tabuleiro/pecas/nova", `{"from":{"X":4,"Y":3},"new_token_name":"Intruso","new_token_size":1,"new_token_look":"object"}`)
	if strings.Contains(gesto, "datastar") {
		t.Errorf("o gesto do jogador foi atendido: %q", gesto)
	}
	cena, _ := f.s.tableHost().Boards().PlaceScene(context.Background(), f.campaignID, lugar)
	if len(cena.Tokens) != 0 {
		t.Errorf("o jogador escreveu no acervo: %+v", cena.Tokens)
	}
}

// A cena do rascunho NÃO oferece os verbos da sessão.
//
// Não é cosmético: "Encerrar o tabuleiro" aqui arquivaria uma cena que já é
// acervo, e a cortina prometeria esconder de uma mesa que não existe. Cada um
// deles precisa de uma sessão do outro lado, e um botão que não pode funcionar é
// pior que a ausência dele — ele ensina um gesto errado.
func TestTheDraftDoesNotOfferTheSessionVerbs(t *testing.T) {
	f := newSceneFixture(t)
	lugar := f.draftPlace(t, "Cripta de Thwor", "crypt")

	corpo := f.pede(t, f.mestre, http.MethodGet, f.draftUrl(lugar), "").Body.String()

	for _, verbo := range []string{
		"Encerrar o tabuleiro",
		"Abrir outro tabuleiro",
		"Lugares da campanha",
		"/tabuleiro/cortina/",
		"/tabuleiro/lente",
	} {
		if strings.Contains(corpo, verbo) {
			t.Errorf("o rascunho oferece %q, que precisa de uma mesa do outro lado", verbo)
		}
	}
	// O CONTROLE: o que o rascunho TEM continua lá. Sem ele, uma cena que
	// falhasse em desenhar o tabuleiro inteiro passaria neste caso — ausência de
	// botão e ausência de tela se parecem no `strings.Contains`.
	if !strings.Contains(corpo, "Ferramentas do mapa") {
		t.Fatal("o trilho de ferramentas sumiu junto: o guarda mediu uma tela vazia")
	}
}

// O lugar que está numa MESA AO VIVO não se monta — e a recusa chega à tela.
//
// A trava mora no `EditPlace` e está presa lá; o que este caso prende é o
// CAMINHO até ela: a sessão viva da campanha é resolvida pelo gateway e chega
// à regra. Sem isso a regra existiria e nunca seria consultada.
func TestTheDraftOfAPlaceOnALiveTableIsRefused(t *testing.T) {
	f := newSceneFixture(t)
	// A taverna é aberta na sessão da fixture — o estado normal de uma partida
	// em andamento. O status da sessão NÃO importa para a trava, e é de
	// propósito: uma sessão encerrada guarda os tabuleiros dela e reabre com
	// eles, então "está na mesa" é sobre o tabuleiro, não sobre a partida.
	f.seedOpenBoard(t, "tavern")
	if err := f.s.tableHost().Boards().Archive(context.Background(), f.campaignID,
		f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)); err != nil {
		t.Fatalf("guardar a taverna: %v", err)
	}
	lugar := placeNamed(t, f.s.tableHost().Boards().Places(context.Background(), f.campaignID),
		"Taverna do Javali")

	resposta := f.posta(t, f.mestre, f.draftUrl(lugar.ID)+"/tabuleiro/pecas/nova", `{"from":{"X":4,"Y":3},"new_token_name":"Fantasma","new_token_size":1,"new_token_look":"object"}`)

	if !strings.Contains(resposta, "está aberto numa mesa agora") {
		t.Errorf("a recusa não chegou à tela: %q", resposta)
	}
	cena, _ := f.s.tableHost().Boards().PlaceScene(context.Background(), f.campaignID, lugar.ID)
	for _, peca := range cena.Tokens {
		if peca.Label == "Fantasma" {
			t.Error("o gesto passou por cima da trava e escreveu no acervo")
		}
	}
	// CONTROLE: o mesmo gesto num lugar que NÃO está na mesa passa. Sem ele,
	// uma recusa por qualquer outro motivo — id errado, rota que não existe —
	// seria lida como "a trava funcionou".
	outro := f.draftPlace(t, "Cripta de Thwor", "crypt")
	f.posta(t, f.mestre, f.draftUrl(outro)+"/tabuleiro/pecas/nova", `{"from":{"X":4,"Y":3},"new_token_name":"Porta","new_token_size":1,"new_token_look":"object"}`)
	if livre, _ := f.s.tableHost().Boards().PlaceScene(context.Background(), f.campaignID, outro); len(livre.Tokens) != 1 {
		t.Fatalf("o gesto foi recusado no lugar que NÃO está na mesa: %+v", livre.Tokens)
	}
}

// A peça se MOVE no rascunho, direto, sem proposta.
//
// A diferença é o que a issue nomeia: na mesa o arrasto propõe um movimento com
// custo e vez, e alguém confirma; aqui não há vez para gastar. Uma proposta
// pendurada num rascunho seria um movimento que ninguém pode confirmar — e o
// `PlaceScene` a descarta na leitura seguinte, então ela sumiria em silêncio.
func TestTheDraftMovesThePieceWithoutAProposal(t *testing.T) {
	f := newSceneFixture(t)
	lugar := f.draftPlace(t, "Cripta de Thwor", "crypt")
	// O ID vem do SERVIDOR e não do teste: o `AddToken` cunha um sempre, e
	// escolher um aqui seria arranjar um dado que a produção nunca produz.
	semeada, err := f.s.tableHost().Boards().EditPlace(context.Background(), f.campaignID, lugar,
		func(b *board.BoardState) error {
			return board.AddToken(b, board.BoardToken{
				Label: "Porta", X: 1, Y: 1, Footprint: 1,
			}, f.s.tableHost().Boards().NewID)
		})
	if err != nil {
		t.Fatalf("semear a peça: %v", err)
	}
	id := semeada.Tokens[0].ID

	f.posta(t, f.mestre, f.draftUrl(lugar)+"/tabuleiro/pecas/"+id+"/mover", `{"from":{"X":6,"Y":2}}`)

	cena, _ := f.s.tableHost().Boards().PlaceScene(context.Background(), f.campaignID, lugar)
	peca := board.FindToken(cena, id)
	if peca == nil {
		t.Fatal("a peça sumiu do rascunho")
	}
	if peca.X != 6 || peca.Y != 2 {
		t.Errorf("a peça não foi para a casa clicada: está em (%d,%d)", peca.X, peca.Y)
	}
	if cena.Pending != nil {
		t.Error("o rascunho ficou com um movimento PROPOSTO, que ninguém pode confirmar")
	}
}

/*
MEDIR o rascunho.

A régua e o gabarito são desenhadas no rascunho junto com o trilho do
tabuleiro, e um gesto oferecido que o servidor não atende é o pior defeito
desta casa: 404, tela que não muda, e nada explicando por quê.

As duas são LEITURA. O que se prende aqui é que elas medem a cena GUARDADA, que
a resposta não mexe no acervo, e que um estranho não as alcança.
*/

// A régua mede no rascunho, e a resposta é só SINAL.
func TestTheRulerMeasuresInsideTheDraft(t *testing.T) {
	f := newSceneFixture(t)
	lugar := f.draftPlace(t, "Cripta de Thwor", "crypt")

	resposta := f.posta(t, f.mestre, f.draftUrl(lugar)+"/tabuleiro/regua",
		`{"ruler_points":[[0,0],[3,0]],"ruler_phase":2}`)

	if !strings.Contains(resposta, "ruler_text") {
		t.Fatalf("a régua não devolveu leitura: %s", resposta)
	}
	// TRÊS quadrados de 1,5m são 4,5m (T20 p238). O número é escrito na mão e
	// nunca derivado do `engine.Measure`: um esperado calculado afirmaria o
	// defeito junto com a regra.
	if !strings.Contains(resposta, "4,5m") {
		t.Errorf("a leitura não diz 4,5m: %s", resposta)
	}
	// Ela NÃO remenda a cena: uma medição que devolvesse o mapa trocaria a peça
	// debaixo do dedo de quem está arrastando a régua.
	if strings.Contains(resposta, "draft-board") {
		t.Errorf("a régua redesenhou o mapa: %s", resposta)
	}
}

// O gabarito conta a peça ESCONDIDA no rascunho, e isso é o desenho e não um
// vazamento.
//
// Na Mesa a redação por papel existe porque quem pergunta "quem o cone pega?"
// não pode descobrir por aí a peça que a cortina esconde DELE. Aqui não há outro
// papel: é o mestre montando a emboscada, e a pergunta que ele veio fazer é se a
// bola de fogo pega o assassino que a mesa ainda não vê. Redigir aqui esconderia
// dele a própria resposta.
func TestTheDraftTemplateCountsTheHiddenTokenBecauseItIsTheMastersOwn(t *testing.T) {
	f := newSceneFixture(t)
	lugar := f.draftPlace(t, "Cripta de Thwor", "crypt")
	if _, err := f.s.tableHost().Boards().EditPlace(context.Background(), f.campaignID, lugar,
		func(b *board.BoardState) error {
			return board.AddToken(b, board.BoardToken{
				Label: "Assassino emboscado", X: 4, Y: 4, Footprint: 1, Hidden: true,
			}, f.s.tableHost().Boards().NewID)
		}); err != nil {
		t.Fatalf("semear a emboscada: %v", err)
	}

	// Um quadrado de lado 1 exatamente em cima dela.
	resposta := f.posta(t, f.mestre, f.draftUrl(lugar)+"/tabuleiro/gabarito", templateBody("quadrado", "1", 4, 4, 4, 4))

	if !strings.Contains(resposta, "Assassino emboscado") {
		t.Errorf("o mestre não viu a própria peça escondida no rascunho: %s", resposta)
	}
	if !strings.Contains(resposta, "template_path") {
		t.Errorf("o gabarito não devolveu o desenho: %s", resposta)
	}
	// O ACERVO não muda: medir não é comandar, e um `EditPlace` aqui gravaria a
	// cada movimento do dedo.
	cena, _ := f.s.tableHost().Boards().PlaceScene(context.Background(), f.campaignID, lugar)
	if len(cena.Tokens) != 1 || cena.Tokens[0].X != 4 {
		t.Errorf("medir mexeu no acervo: %+v", cena.Tokens)
	}
}

// O cone SEM MIRA pede a mira em vez de apontar para um lado inventado.
//
// É a mesma decisão da Mesa, e ela precisa de caso próprio porque é o ramo que
// sai ANTES de o gabarito ser calculado — um gêmeo que esquecesse essa saída
// desenharia um cone apontando para onde o servidor achou melhor.
func TestTheDraftConeWithoutAimAsksForIt(t *testing.T) {
	f := newSceneFixture(t)
	lugar := f.draftPlace(t, "Cripta de Thwor", "crypt")

	resposta := f.posta(t, f.mestre, f.draftUrl(lugar)+"/tabuleiro/gabarito", templateBody("cone", "6", 0, 0, 0, 0))

	if !strings.Contains(resposta, "Clique de novo para apontar") {
		t.Errorf("o cone sem mira não pediu a mira: %s", resposta)
	}
}

// E um estranho não MEDE o rascunho.
//
// Caso próprio, e não coberto pelo dos gestos: a régua e o gabarito não passam
// pelo `draftCommand`, então a trava delas é escrita à parte — é exatamente aí
// que ela pode ser esquecida. O gabarito devolve os NOMES das peças, então uma
// rota aberta entregaria a emboscada por sinal, que é mais fácil de ler que o
// DOM.
func TestAStrangerDoesNotMeasureThePlaceDraft(t *testing.T) {
	f := newSceneFixture(t)
	lugar := f.draftPlace(t, "Cripta de Thwor", "crypt")
	if _, err := f.s.tableHost().Boards().EditPlace(context.Background(), f.campaignID, lugar,
		func(b *board.BoardState) error {
			return board.AddToken(b, board.BoardToken{
				Label: "Assassino emboscado", X: 4, Y: 4, Footprint: 1, Hidden: true,
			}, f.s.tableHost().Boards().NewID)
		}); err != nil {
		t.Fatalf("semear a emboscada: %v", err)
	}

	for _, caminho := range []string{
		f.draftUrl(lugar) + "/tabuleiro/regua",
		f.draftUrl(lugar) + "/tabuleiro/gabarito",
	} {
		resposta := f.posta(t, f.jogador, caminho, `{"ruler_points":[[0,0],[3,0]],"ruler_phase":2}`)
		if strings.Contains(resposta, "Assassino emboscado") {
			t.Errorf("%s entregou a emboscada ao jogador: %s", caminho, resposta)
		}
		if strings.Contains(resposta, "datastar") {
			t.Errorf("%s foi atendido para o jogador: %s", caminho, resposta)
		}
	}
	// CONTROLE: o MESTRE mede as duas. Sem ele, uma rota que respondesse 404
	// para todo mundo passaria por "a trava funcionou".
	if r := f.posta(t, f.mestre, f.draftUrl(lugar)+"/tabuleiro/gabarito", templateBody("quadrado", "1", 4, 4, 4, 4)); !strings.Contains(r, "Assassino emboscado") {
		t.Fatalf("o mestre também não mediu — o guarda mediu uma rota morta: %s", r)
	}
}

// ── OS CINCO GESTOS DE TERRENO E MARCADOR DO RASCUNHO ────────────────────────
//
// `draftPaintsTerrain`, `draftClearsTerrain`, `draftFillsRect`, `draftClearsRect`
// e `draftMarksTheSpot` já estiveram sem teste nenhum: transformados em `return
// nil` puro, a suíte inteira ficava verde.
//
// Eles são as rotas irmãs das da Mesa, e o que se prende aqui é o que cada
// gêmeo tem de próprio: a coordenada chega pelo CORPO e pousa no ACERVO, não
// numa sessão. A regra de pintura em si já está presa no `board`.

// draftScene lê o que ficou gravado no acervo.
func (f sceneFixture) draftScene(t *testing.T, placeID int64) *board.BoardState {
	t.Helper()
	cena, err := f.s.tableHost().Boards().PlaceScene(context.Background(), f.campaignID, placeID)
	if err != nil {
		t.Fatalf("ler a cena do lugar %d: %v", placeID, err)
	}
	return cena
}

// A coordenada é o assunto: uma tag `json:"from"` quebrada faz o corpo
// decodificar para (0,0) em SILÊNCIO, e o traço inteiro pousa na quina. Por isso
// nenhuma ponta deste caso é a origem.
func TestTheDraftBrushPaintsWhereTheBodySays(t *testing.T) {
	f := newSceneFixture(t)
	lugar := f.draftPlace(t, "Cripta de Thwor", "crypt")

	if rec := f.pede(t, f.mestre, http.MethodPost,
		f.draftUrl(lugar)+"/tabuleiro/terreno", stroke("dificil", 3, 4, 6, 4)); rec.Code != http.StatusOK {
		t.Fatalf("pintar no rascunho deu %d", rec.Code)
	}

	casas := board.SquaresOf(f.draftScene(t, lugar), "dificil")
	// De (3,4) a (6,4) é uma linha reta de QUATRO casas, escritas à mão.
	if len(casas) != 4 {
		t.Errorf("o traço (3,4)→(6,4) pintou %d casas, e a linha tem 4: %v", len(casas), casas)
	}
	for _, ponta := range []engine.Square{{X: 3, Y: 4}, {X: 6, Y: 4}} {
		if !contem(casas, ponta) {
			t.Errorf("a casa %v não foi pintada — o traço não chegou onde o corpo mandou", ponta)
		}
	}
	// E NADA NA ORIGEM. É o controle contra o `from` que parou de ser lido: (0,0)
	// é o valor-zero do struct, então um corpo ignorado pinta exatamente ali.
	if contem(casas, engine.Square{X: 0, Y: 0}) {
		t.Errorf("o traço pintou (0,0), que é o valor-zero do corpo: a coordenada não foi lida")
	}
}

// A borracha do rascunho, com a testemunha de fora do traço.
func TestTheDraftEraserClearsOnlyWhatItCrosses(t *testing.T) {
	f := newSceneFixture(t)
	lugar := f.draftPlace(t, "Cripta de Thwor", "crypt")

	for _, traco := range []string{stroke("dificil", 3, 4, 6, 4), stroke("cobertura", 1, 9, 1, 9)} {
		if rec := f.pede(t, f.mestre, http.MethodPost,
			f.draftUrl(lugar)+"/tabuleiro/terreno", traco); rec.Code != http.StatusOK {
			t.Fatalf("pintar no rascunho deu %d", rec.Code)
		}
	}
	if rec := f.pede(t, f.mestre, http.MethodPost,
		f.draftUrl(lugar)+"/tabuleiro/terreno/limpar", stroke("", 3, 4, 6, 4)); rec.Code != http.StatusOK {
		t.Fatalf("apagar no rascunho deu %d", rec.Code)
	}

	cena := f.draftScene(t, lugar)
	if sobrou := board.SquaresOf(cena, "dificil"); len(sobrou) != 0 {
		t.Errorf("a borracha do rascunho deixou %v pelo caminho", sobrou)
	}
	// A TESTEMUNHA: "sobrou zero" não diz nada sobre o que foi apagado A MAIS.
	if testemunha := board.SquaresOf(cena, "cobertura"); len(testemunha) != 1 {
		t.Errorf("a casa (1,9), fora do traço, virou %v — a borracha apagou além do pedido", testemunha)
	}
}

// O retângulo é o gesto que o Shift liga, e os dois cantos vêm no corpo. Um
// canto perdido não estoura: ele vira (0,0), e a caixa cresce até a quina
// levando junto tudo que estiver no caminho.
func TestTheDraftRectangleFillsTheBoxAndTheEraserEmptiesIt(t *testing.T) {
	f := newSceneFixture(t)
	lugar := f.draftPlace(t, "Cripta de Thwor", "crypt")

	if rec := f.pede(t, f.mestre, http.MethodPost,
		f.draftUrl(lugar)+"/tabuleiro/terreno/retangulo", stroke("dificil", 2, 3, 4, 5)); rec.Code != http.StatusOK {
		t.Fatalf("encher o retângulo deu %d", rec.Code)
	}
	casas := board.SquaresOf(f.draftScene(t, lugar), "dificil")
	// De (2,3) a (4,5) são 3×3 = NOVE casas, escritas à mão.
	if len(casas) != 9 {
		t.Errorf("o retângulo (2,3)→(4,5) encheu %d casas, e a caixa tem 9: %v", len(casas), casas)
	}
	for _, quina := range []engine.Square{{X: 2, Y: 3}, {X: 4, Y: 3}, {X: 2, Y: 5}, {X: 4, Y: 5}} {
		if !contem(casas, quina) {
			t.Errorf("a quina %v ficou de fora da caixa", quina)
		}
	}
	if contem(casas, engine.Square{X: 0, Y: 0}) {
		t.Errorf("a caixa alcançou (0,0), que é o valor-zero do corpo: um canto não foi lido")
	}

	if rec := f.pede(t, f.mestre, http.MethodPost,
		f.draftUrl(lugar)+"/tabuleiro/terreno/limpar/retangulo", stroke("", 2, 3, 4, 5)); rec.Code != http.StatusOK {
		t.Fatalf("limpar o retângulo deu %d", rec.Code)
	}
	if sobrou := board.SquaresOf(f.draftScene(t, lugar), "dificil"); len(sobrou) != 0 {
		t.Errorf("limpar o retângulo deixou %v", sobrou)
	}
}

// Duas afirmações, e a segunda é a razão de o marcador existir: ele nasce
// ESCONDIDO, porque marcar a armadilha na frente da mesa entrega a armadilha.
func TestTheDraftMarkerLandsWhereTheBodySaysAndIsBornHidden(t *testing.T) {
	f := newSceneFixture(t)
	lugar := f.draftPlace(t, "Cripta de Thwor", "crypt")

	if rec := f.pede(t, f.mestre, http.MethodPost, f.draftUrl(lugar)+"/tabuleiro/marcadores/novo",
		`{"from":{"X":7,"Y":2}}`); rec.Code != http.StatusOK {
		t.Fatalf("pôr o marcador deu %d", rec.Code)
	}

	marcadores := f.draftScene(t, lugar).Markers
	if len(marcadores) != 1 {
		t.Fatalf("o rascunho ficou com %d marcadores, esperado 1", len(marcadores))
	}
	if m := marcadores[0]; m.X != 7 || m.Y != 2 {
		t.Errorf("o marcador pousou em (%d,%d) e o corpo mandou (7,2)", m.X, m.Y)
	}
	if !marcadores[0].Hidden {
		t.Error("o marcador do rascunho nasceu VISÍVEL — marcar a armadilha na frente da mesa a entrega")
	}
}
