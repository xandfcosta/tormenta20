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
func (f sceneFixture) draftPlace(t *testing.T, name, chao string) int64 {
	t.Helper()
	place, err := f.s.tableHost().Boards().NewPlace(context.Background(), f.campaignID, name, chao)
	if err != nil {
		t.Fatalf("criar o lugar %q: %v", name, err)
	}
	return place.ID
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
	place := f.draftPlace(t, "Cripta de Thwor", "crypt")

	body := f.pede(t, f.gm, http.MethodGet, f.draftUrl(place), "").Body.String()

	if !strings.Contains(body, "board-plane") {
		t.Error("o rascunho não desenhou o plano do tabuleiro")
	}
	if !strings.Contains(body, "Cripta de Thwor") {
		t.Error("o rascunho não diz que lugar está sendo montado")
	}
	if !strings.Contains(body, "a mesa não vê") {
		t.Error("a tarja não diz que ninguém está vendo — o mestre não tem como saber em que tempo está")
	}
	if !strings.Contains(body, "ground-crypt") {
		t.Error("o chão escolhido não foi desenhado")
	}
}

// OS GESTOS POSTAM NO ACERVO, e não numa sessão que não existe.
//
// É o guarda do `BoardView.Base`, medido no HTML que sai. Sem ele, uma chamada
// que continuasse escrevendo o caminho da sessão postaria em
// `/campanhas/N/sessoes/0/tabuleiro`
// — um endereço que RESPONDE, com 403 ou 404, e devolve uma tela que não mudou.
// O sintoma seria "o pincel não pinta", sem uma linha em lugar nenhum.
func TestTheDraftGesturesPostToTheArchiveAndNotToATable(t *testing.T) {
	f := newSceneFixture(t)
	place := f.draftPlace(t, "Cripta de Thwor", "crypt")

	body := f.pede(t, f.gm, http.MethodGet, f.draftUrl(place), "").Body.String()

	want := fmt.Sprintf("/campanhas/%d/lugares/%d/tabuleiro", f.campaignID, place)
	if !strings.Contains(body, want) {
		t.Fatalf("nenhum gesto posta em %q", want)
	}
	// O CONTROLE, e ele é o que separa "não achei" de "não procurei": a mesma
	// página NÃO pode carregar o caminho de uma sessão. `/campanhas/` sozinho
	// aparece em todo link de navegação legítimo, então o que se procura é o
	// caminho do TABULEIRO de uma sessão — com a sessão ZERO, que é o valor que
	// um `Base` esquecido produz.
	if strings.Contains(body, "/tabuleiro/terreno") && !strings.Contains(body, want+"/terreno") {
		t.Error("o pincel do rascunho posta num tabuleiro que não é o dele")
	}
	if strings.Contains(body, fmt.Sprintf("/campanhas/%d/sessoes/0/tabuleiro", f.campaignID)) {
		t.Error("um gesto escapou para uma sessão ZERO — é o defeito que o `Base` existe para impedir")
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
	place := f.draftPlace(t, "Cripta de Thwor", "crypt")

	f.posta(t, f.gm, f.draftUrl(place)+"/tabuleiro/pecas/nova", `{"from":{"X":4,"Y":3},"new_token_name":"Porta da cripta","new_token_size":1,"new_token_look":"object"}`)

	scene, err := f.s.tableHost().Boards().PlaceScene(context.Background(), f.campaignID, place)
	if err != nil {
		t.Fatalf("reabrir o rascunho: %v", err)
	}
	if len(scene.Tokens) != 1 {
		t.Fatalf("o gesto não chegou ao acervo: %+v", scene.Tokens)
	}
	if scene.Tokens[0].Label != "Porta da cripta" || scene.Tokens[0].X != 4 || scene.Tokens[0].Y != 3 {
		t.Errorf("a peça não nasceu onde o clique disse: %+v", scene.Tokens[0])
	}
	// A peça nasce com id do SERVIDOR: sem ele nada consegue selecioná-la para
	// mover, editar ou remover no gesto seguinte.
	if scene.Tokens[0].ID == "" {
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
	place := f.draftPlace(t, "Cripta de Thwor", "crypt")

	page := f.pede(t, f.player, http.MethodGet, f.draftUrl(place), "")
	if page.Code != http.StatusForbidden {
		t.Errorf("o jogador abriu o rascunho com %d", page.Code)
	}
	if strings.Contains(page.Body.String(), "Cripta de Thwor") {
		t.Error("a recusa vazou o nome do lugar que ela existe para esconder")
	}

	// E POSTANDO NA MÃO, que é o caso que o botão escondido não cobre.
	gesture := f.posta(t, f.player, f.draftUrl(place)+"/tabuleiro/pecas/nova", `{"from":{"X":4,"Y":3},"new_token_name":"Intruso","new_token_size":1,"new_token_look":"object"}`)
	if strings.Contains(gesture, "datastar") {
		t.Errorf("o gesto do jogador foi atendido: %q", gesture)
	}
	scene, _ := f.s.tableHost().Boards().PlaceScene(context.Background(), f.campaignID, place)
	if len(scene.Tokens) != 0 {
		t.Errorf("o jogador escreveu no acervo: %+v", scene.Tokens)
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
	place := f.draftPlace(t, "Cripta de Thwor", "crypt")

	body := f.pede(t, f.gm, http.MethodGet, f.draftUrl(place), "").Body.String()

	for _, verb := range []string{
		"Encerrar o tabuleiro",
		"Abrir outro tabuleiro",
		"Lugares da campanha",
		"/tabuleiro/cortina/",
		"/tabuleiro/lente",
	} {
		if strings.Contains(body, verb) {
			t.Errorf("o rascunho oferece %q, que precisa de uma mesa do outro lado", verb)
		}
	}
	// O CONTROLE: o que o rascunho TEM continua lá. Sem ele, uma cena que
	// falhasse em desenhar o tabuleiro inteiro passaria neste caso — ausência de
	// botão e ausência de tela se parecem no `strings.Contains`.
	if !strings.Contains(body, "Ferramentas do mapa") {
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
		boardRead(f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab))); err != nil {
		t.Fatalf("guardar a taverna: %v", err)
	}
	place := placeNamed(t, f.s.tableHost().Boards().Places(context.Background(), f.campaignID),
		"Taverna do Javali")

	response := f.posta(t, f.gm, f.draftUrl(place.ID)+"/tabuleiro/pecas/nova", `{"from":{"X":4,"Y":3},"new_token_name":"Fantasma","new_token_size":1,"new_token_look":"object"}`)

	if !strings.Contains(response, "está aberto numa mesa agora") {
		t.Errorf("a recusa não chegou à tela: %q", response)
	}
	scene, _ := f.s.tableHost().Boards().PlaceScene(context.Background(), f.campaignID, place.ID)
	for _, token := range scene.Tokens {
		if token.Label == "Fantasma" {
			t.Error("o gesto passou por cima da trava e escreveu no acervo")
		}
	}
	// CONTROLE: o mesmo gesto num lugar que NÃO está na mesa passa. Sem ele,
	// uma recusa por qualquer outro motivo — id errado, rota que não existe —
	// seria lida como "a trava funcionou".
	other := f.draftPlace(t, "Cripta de Thwor", "crypt")
	f.posta(t, f.gm, f.draftUrl(other)+"/tabuleiro/pecas/nova", `{"from":{"X":4,"Y":3},"new_token_name":"Porta","new_token_size":1,"new_token_look":"object"}`)
	if livre, _ := f.s.tableHost().Boards().PlaceScene(context.Background(), f.campaignID, other); len(livre.Tokens) != 1 {
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
	place := f.draftPlace(t, "Cripta de Thwor", "crypt")
	// O ID vem do SERVIDOR e não do teste: o `AddToken` cunha um sempre, e
	// escolher um aqui seria arranjar um dado que a produção nunca produz.
	seeded, err := f.s.tableHost().Boards().EditPlace(context.Background(), f.campaignID, place,
		func(b *board.BoardState) error {
			return board.AddToken(b, board.BoardToken{
				Label: "Porta", X: 1, Y: 1, Footprint: 1,
			}, f.s.tableHost().Boards().NewID)
		})
	if err != nil {
		t.Fatalf("semear a peça: %v", err)
	}
	id := seeded.Tokens[0].ID

	f.posta(t, f.gm, f.draftUrl(place)+"/tabuleiro/pecas/"+id+"/mover", `{"from":{"X":6,"Y":2}}`)

	scene, _ := f.s.tableHost().Boards().PlaceScene(context.Background(), f.campaignID, place)
	token := board.FindToken(scene, id)
	if token == nil {
		t.Fatal("a peça sumiu do rascunho")
	}
	if token.X != 6 || token.Y != 2 {
		t.Errorf("a peça não foi para a casa clicada: está em (%d,%d)", token.X, token.Y)
	}
	if scene.Pending != nil {
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
	place := f.draftPlace(t, "Cripta de Thwor", "crypt")

	response := f.posta(t, f.gm, f.draftUrl(place)+"/tabuleiro/regua",
		`{"ruler_points":[[0,0],[3,0]],"ruler_phase":2}`)

	if !strings.Contains(response, "ruler_text") {
		t.Fatalf("a régua não devolveu leitura: %s", response)
	}
	// TRÊS quadrados de 1,5m são 4,5m (T20 p238). O número é escrito na mão e
	// nunca derivado do `engine.Measure`: um esperado calculado afirmaria o
	// defeito junto com a regra.
	if !strings.Contains(response, "4,5m") {
		t.Errorf("a leitura não diz 4,5m: %s", response)
	}
	// Ela NÃO remenda a cena: uma medição que devolvesse o mapa trocaria a peça
	// debaixo do dedo de quem está arrastando a régua.
	if strings.Contains(response, "draft-board") {
		t.Errorf("a régua redesenhou o mapa: %s", response)
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
	place := f.draftPlace(t, "Cripta de Thwor", "crypt")
	if _, err := f.s.tableHost().Boards().EditPlace(context.Background(), f.campaignID, place,
		func(b *board.BoardState) error {
			return board.AddToken(b, board.BoardToken{
				Label: "Assassino emboscado", X: 4, Y: 4, Footprint: 1, Hidden: true,
			}, f.s.tableHost().Boards().NewID)
		}); err != nil {
		t.Fatalf("semear a emboscada: %v", err)
	}

	// Um quadrado de lado 1 exatamente em cima dela.
	response := f.posta(t, f.gm, f.draftUrl(place)+"/tabuleiro/gabarito", templateBody("quadrado", "1", 4, 4, 4, 4))

	if !strings.Contains(response, "Assassino emboscado") {
		t.Errorf("o mestre não viu a própria peça escondida no rascunho: %s", response)
	}
	if !strings.Contains(response, "template_path") {
		t.Errorf("o gabarito não devolveu o desenho: %s", response)
	}
	// O ACERVO não muda: medir não é comandar, e um `EditPlace` aqui gravaria a
	// cada movimento do dedo.
	scene, _ := f.s.tableHost().Boards().PlaceScene(context.Background(), f.campaignID, place)
	if len(scene.Tokens) != 1 || scene.Tokens[0].X != 4 {
		t.Errorf("medir mexeu no acervo: %+v", scene.Tokens)
	}
}

// O cone SEM MIRA pede a mira em vez de apontar para um lado inventado.
//
// É a mesma decisão da Mesa, e ela precisa de caso próprio porque é o ramo que
// sai ANTES de o gabarito ser calculado — um gêmeo que esquecesse essa saída
// desenharia um cone apontando para onde o servidor achou melhor.
func TestTheDraftConeWithoutAimAsksForIt(t *testing.T) {
	f := newSceneFixture(t)
	place := f.draftPlace(t, "Cripta de Thwor", "crypt")

	response := f.posta(t, f.gm, f.draftUrl(place)+"/tabuleiro/gabarito", templateBody("cone", "6", 0, 0, 0, 0))

	if !strings.Contains(response, "Clique de novo para apontar") {
		t.Errorf("o cone sem mira não pediu a mira: %s", response)
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
	place := f.draftPlace(t, "Cripta de Thwor", "crypt")
	if _, err := f.s.tableHost().Boards().EditPlace(context.Background(), f.campaignID, place,
		func(b *board.BoardState) error {
			return board.AddToken(b, board.BoardToken{
				Label: "Assassino emboscado", X: 4, Y: 4, Footprint: 1, Hidden: true,
			}, f.s.tableHost().Boards().NewID)
		}); err != nil {
		t.Fatalf("semear a emboscada: %v", err)
	}

	for _, path := range []string{
		f.draftUrl(place) + "/tabuleiro/regua",
		f.draftUrl(place) + "/tabuleiro/gabarito",
	} {
		response := f.posta(t, f.player, path, `{"ruler_points":[[0,0],[3,0]],"ruler_phase":2}`)
		if strings.Contains(response, "Assassino emboscado") {
			t.Errorf("%s entregou a emboscada ao jogador: %s", path, response)
		}
		if strings.Contains(response, "datastar") {
			t.Errorf("%s foi atendido para o jogador: %s", path, response)
		}
	}
	// CONTROLE: o MESTRE mede as duas. Sem ele, uma rota que respondesse 404
	// para todo mundo passaria por "a trava funcionou".
	if r := f.posta(t, f.gm, f.draftUrl(place)+"/tabuleiro/gabarito", templateBody("quadrado", "1", 4, 4, 4, 4)); !strings.Contains(r, "Assassino emboscado") {
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
	scene, err := f.s.tableHost().Boards().PlaceScene(context.Background(), f.campaignID, placeID)
	if err != nil {
		t.Fatalf("ler a cena do lugar %d: %v", placeID, err)
	}
	return scene
}

// A coordenada é o assunto: uma tag `json:"from"` quebrada faz o corpo
// decodificar para (0,0) em SILÊNCIO, e o traço inteiro pousa na quina. Por isso
// nenhuma ponta deste caso é a origem.
func TestTheDraftBrushPaintsWhereTheBodySays(t *testing.T) {
	f := newSceneFixture(t)
	place := f.draftPlace(t, "Cripta de Thwor", "crypt")

	if rec := f.pede(t, f.gm, http.MethodPost,
		f.draftUrl(place)+"/tabuleiro/terreno", stroke("dificil", 3, 4, 6, 4)); rec.Code != http.StatusOK {
		t.Fatalf("pintar no rascunho deu %d", rec.Code)
	}

	squares := board.SquaresOf(f.draftScene(t, place), "dificil")
	// De (3,4) a (6,4) é uma linha reta de QUATRO casas, escritas à mão.
	if len(squares) != 4 {
		t.Errorf("o traço (3,4)→(6,4) pintou %d casas, e a linha tem 4: %v", len(squares), squares)
	}
	for _, tip := range []engine.Square{{X: 3, Y: 4}, {X: 6, Y: 4}} {
		if !contem(squares, tip) {
			t.Errorf("a casa %v não foi pintada — o traço não chegou onde o corpo mandou", tip)
		}
	}
	// E NADA NA ORIGEM. É o controle contra o `from` que parou de ser lido: (0,0)
	// é o valor-zero do struct, então um corpo ignorado pinta exatamente ali.
	if contem(squares, engine.Square{X: 0, Y: 0}) {
		t.Errorf("o traço pintou (0,0), que é o valor-zero do corpo: a coordenada não foi lida")
	}
}

// A borracha do rascunho, com a testemunha de fora do traço.
func TestTheDraftEraserClearsOnlyWhatItCrosses(t *testing.T) {
	f := newSceneFixture(t)
	place := f.draftPlace(t, "Cripta de Thwor", "crypt")

	for _, trait := range []string{stroke("dificil", 3, 4, 6, 4), stroke("cobertura", 1, 9, 1, 9)} {
		if rec := f.pede(t, f.gm, http.MethodPost,
			f.draftUrl(place)+"/tabuleiro/terreno", trait); rec.Code != http.StatusOK {
			t.Fatalf("pintar no rascunho deu %d", rec.Code)
		}
	}
	if rec := f.pede(t, f.gm, http.MethodPost,
		f.draftUrl(place)+"/tabuleiro/terreno/limpar", stroke("", 3, 4, 6, 4)); rec.Code != http.StatusOK {
		t.Fatalf("apagar no rascunho deu %d", rec.Code)
	}

	scene := f.draftScene(t, place)
	if left := board.SquaresOf(scene, "dificil"); len(left) != 0 {
		t.Errorf("a borracha do rascunho deixou %v pelo caminho", left)
	}
	// A TESTEMUNHA: "sobrou zero" não diz nada sobre o que foi apagado A MAIS.
	if witness := board.SquaresOf(scene, "cobertura"); len(witness) != 1 {
		t.Errorf("a casa (1,9), fora do traço, virou %v — a borracha apagou além do pedido", witness)
	}
}

// O retângulo é o gesto que o Shift liga, e os dois cantos vêm no corpo. Um
// canto perdido não estoura: ele vira (0,0), e a caixa cresce até a quina
// levando junto tudo que estiver no caminho.
func TestTheDraftRectangleFillsTheBoxAndTheEraserEmptiesIt(t *testing.T) {
	f := newSceneFixture(t)
	place := f.draftPlace(t, "Cripta de Thwor", "crypt")

	if rec := f.pede(t, f.gm, http.MethodPost,
		f.draftUrl(place)+"/tabuleiro/terreno/retangulo", stroke("dificil", 2, 3, 4, 5)); rec.Code != http.StatusOK {
		t.Fatalf("encher o retângulo deu %d", rec.Code)
	}
	squares := board.SquaresOf(f.draftScene(t, place), "dificil")
	// De (2,3) a (4,5) são 3×3 = NOVE casas, escritas à mão.
	if len(squares) != 9 {
		t.Errorf("o retângulo (2,3)→(4,5) encheu %d casas, e a caixa tem 9: %v", len(squares), squares)
	}
	for _, corner := range []engine.Square{{X: 2, Y: 3}, {X: 4, Y: 3}, {X: 2, Y: 5}, {X: 4, Y: 5}} {
		if !contem(squares, corner) {
			t.Errorf("a quina %v ficou de fora da caixa", corner)
		}
	}
	if contem(squares, engine.Square{X: 0, Y: 0}) {
		t.Errorf("a caixa alcançou (0,0), que é o valor-zero do corpo: um canto não foi lido")
	}

	if rec := f.pede(t, f.gm, http.MethodPost,
		f.draftUrl(place)+"/tabuleiro/terreno/limpar/retangulo", stroke("", 2, 3, 4, 5)); rec.Code != http.StatusOK {
		t.Fatalf("limpar o retângulo deu %d", rec.Code)
	}
	if left := board.SquaresOf(f.draftScene(t, place), "dificil"); len(left) != 0 {
		t.Errorf("limpar o retângulo deixou %v", left)
	}
}

// Duas afirmações, e a segunda é a razão de o marcador existir: ele nasce
// ESCONDIDO, porque marcar a armadilha na frente da mesa entrega a armadilha.
func TestTheDraftMarkerLandsWhereTheBodySaysAndIsBornHidden(t *testing.T) {
	f := newSceneFixture(t)
	place := f.draftPlace(t, "Cripta de Thwor", "crypt")

	if rec := f.pede(t, f.gm, http.MethodPost, f.draftUrl(place)+"/tabuleiro/marcadores/novo",
		`{"from":{"X":7,"Y":2}}`); rec.Code != http.StatusOK {
		t.Fatalf("pôr o marcador deu %d", rec.Code)
	}

	markers := f.draftScene(t, place).Markers
	if len(markers) != 1 {
		t.Fatalf("o rascunho ficou com %d marcadores, esperado 1", len(markers))
	}
	if m := markers[0]; m.X != 7 || m.Y != 2 {
		t.Errorf("o marcador pousou em (%d,%d) e o corpo mandou (7,2)", m.X, m.Y)
	}
	if !markers[0].Hidden {
		t.Error("o marcador do rascunho nasceu VISÍVEL — marcar a armadilha na frente da mesa a entrega")
	}
}
