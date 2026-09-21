package api

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"t20engine/domain/board"
	"t20engine/infra/events"
	"t20engine/serve/web/table"
	"testing"
	"time"
)

func (f sceneFixture) seedOpenBoard(t *testing.T, terrain string) *board.BoardState {
	t.Helper()
	b, err := f.s.tableHost().Boards().Open(context.Background(), f.sessionID, "Taverna do Javali", terrain)
	if err != nil {
		t.Fatalf("o tabuleiro não abriu: %v", err)
	}
	return b
}

// "Não há tabuleiro" e "há um vazio" são estados diferentes, e o primeiro é o
// normal — a maior parte de uma sessão não tem mapa. Desenhar uma grade vazia
// diria que o mestre abriu uma cena que ele não abriu.
func TestWithoutABoardTheSceneSaysThereIsNoMap(t *testing.T) {
	f := newSceneFixture(t)
	body := f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()

	if !strings.Contains(body, "Nenhum tabuleiro aberto") {
		t.Error("a cena não disse que não há mapa")
	}
	if strings.Contains(body, "board-plane") {
		t.Error("desenhou a grade sem tabuleiro aberto")
	}
}

// A trava da peça escondida não pode ser CSS: uma peça meio-apagada no HTML do
// jogador entrega a posição do ogro para quem abrir o inspetor. Quem a tira é o
// `BoardForRole`, o mesmo gargalo por papel que a fila usa — e este teste afirma
// que a cena passa por ele em vez de decidir por conta própria.
func TestTheHiddenTokenDoesNotReachThePlayer(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "crypt")
	if _, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab,
		board.BoardToken{ID: "emboscada", Label: "Ogro", X: 4, Y: 3, Hidden: true}); err != nil {
		t.Fatalf("pôr a peça escondida: %v", err)
	}
	if _, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab,
		board.BoardToken{ID: "avista", Label: "Arwen", X: 1, Y: 1}); err != nil {
		t.Fatalf("pôr a peça à vista: %v", err)
	}

	forGM := f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(forGM, "Ogro em") {
		t.Error("o mestre não viu a própria peça escondida")
	}

	forPlayer := f.pede(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()
	// O CONTROLE: o jogador está vendo o tabuleiro. Sem ele, "não achei o Ogro"
	// seria verdade também numa cena sem mapa nenhum.
	if !strings.Contains(forPlayer, "Arwen em") {
		t.Fatal("o jogador não viu o tabuleiro; a ausência abaixo não provaria nada")
	}
	if strings.Contains(forPlayer, "Ogro") {
		t.Error("a peça escondida chegou ao HTML do jogador")
	}
}

// O anel é o MESMO sinal que a linha da fila usa, e ligá-lo pelo `entryId` é o
// que garante isso: derivar "quem está na vez" no tabuleiro seria a segunda
// cópia da regra, e é assim que duas telas passam a apontar combatentes
// diferentes.
func TestTheTokenOnTurnLightsUpWithTheSameGoldAsTheTracker(t *testing.T) {
	f := newSceneFixture(t)
	entryID := f.tracker(t)
	f.seedOpenBoard(t, "stone")
	if _, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab,
		board.BoardToken{ID: "p", Label: "Arcanista", X: 2, Y: 2, EntryID: &entryID}); err != nil {
		t.Fatalf("pôr a peça: %v", err)
	}
	if rec := f.pede(t, f.gm, "POST", f.tableUrl()+"/cena/iniciar/acao", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar cena deu %d", rec.Code)
	}

	// FORA de combate ninguém está na vez, mesmo com a cena aberta e a fila
	// montada — é o `TurnIndex` negativo, e a peça não pode acender por estar
	// no mapa.
	before := f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if strings.Contains(before, "board-token-on-turn") {
		t.Error("a peça acendeu antes de o combate começar")
	}

	if rec := f.pede(t, f.gm, "POST", f.tableUrl()+"/iniciativa/proxima-vez", ""); rec.Code != http.StatusOK {
		t.Fatalf("avançar deu %d", rec.Code)
	}
	after := f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(after, "board-token-on-turn") {
		t.Error("chegou a vez do combatente e a peça dele não acendeu")
	}
	if !strings.Contains(after, "— na vez") {
		t.Error("o anel não tem par em TEXTO: cor não existe para leitor de tela (ALE-212)")
	}
}

// O terreno vem do BANCO, então é dado do cliente. Uma classe `chao-<qualquer>`
// não existiria na folha e o chão sairia transparente — o que se parece com
// defeito de CSS e manda procurar no lugar errado.
func TestAnInventedTerrainFallsBackToTheDefaultGround(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "vulcão-de-neon")

	body := f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(body, "ground-stone") {
		t.Error("o terreno inventado não caiu no chão padrão")
	}
	if strings.Contains(body, "ground-vulcão") {
		t.Error("o terreno inventado virou classe solta")
	}
}

// ── o aviso do tabuleiro ─────────────────────────────────────────────────────

// Os TRÊS pontos de escrita são medidos, e o `Open`/`Close` estão aqui porque
// eles não passam pelo `apply`: abrir e fechar são as mudanças mais VISÍVEIS do
// tabuleiro — a grade aparecendo e sumindo —, e um aviso que cobrisse só o que
// se move perderia o que nasce.
//
// O caso afirma QUAL evento chegou, e não só que algo chegou: com um sino sem
// carga, trocar abrir por fechar no código passa verde aqui.
func TestTheBoardTellsItsListenersOnEveryChange(t *testing.T) {
	f := newSceneFixture(t)
	bs := f.s.tableHost().Boards()
	ctx := context.Background()
	const session = int64(1)

	sub, stop := f.s.tableHost().Bus().Subscribe(events.OfSession(session))
	defer stop()
	drain := func() {
		for len(sub.C) > 0 {
			<-sub.C
		}
	}
	warned := func(what string, want events.Event) {
		t.Helper()
		select {
		case ev := <-sub.C:
			if fmt.Sprintf("%T", ev) != fmt.Sprintf("%T", want) {
				t.Errorf("%s publicou %T, esperado %T", what, ev, want)
			}
		default:
			t.Errorf("%s não avisou quem escuta", what)
		}
	}

	drain()
	if _, err := bs.Open(ctx, session, "Taverna", "tavern"); err != nil {
		t.Fatalf("abrir: %v", err)
	}
	warned("abrir o tabuleiro", events.BoardOpened{})

	drain()
	if _, err := bs.AddToken(ctx, session, defaultTab, board.BoardToken{ID: "p", Label: "Ogro", X: 1, Y: 1}); err != nil {
		t.Fatalf("pôr a peça: %v", err)
	}
	warned("pôr uma peça (pelo apply)", events.BoardChanged{})

	drain()
	bs.Close(ctx, session, defaultTab)
	warned("fechar o tabuleiro", events.BoardClosed{})
}

// E uma mutação RECUSADA não avisa: "mudou" tem de significar mudou, senão o
// stream relê e o hash o faz calar — trabalho para nada a cada erro de quem
// clica.
func TestARefusedMutationTellsNobody(t *testing.T) {
	f := newSceneFixture(t)
	ctx := context.Background()
	const session = int64(2)

	sub, stop := f.s.tableHost().Bus().Subscribe(events.OfSession(session))
	defer stop()
	// SEM tabuleiro aberto: o `apply` recusa antes de mexer em nada.
	if _, err := f.s.tableHost().Boards().AddToken(ctx, session, defaultTab, board.BoardToken{ID: "p", Label: "Ogro"}); err == nil {
		t.Fatal("pôr peça sem tabuleiro devia recusar; sem a recusa este teste não mede nada")
	}
	select {
	case ev := <-sub.C:
		t.Errorf("a mutação recusada publicou %T", ev)
	default:
	}
}

// Não há caso aqui para a BAIXA do ouvinte de propósito: o registro é do
// barramento, não do tabuleiro, e a baixa é medida onde ela mora, em
// `events.TestUnsubscribeRemovesTheListener`. Uma regra, uma camada.

// "O store avisa" e "a tela recebe" são perguntas diferentes, e só a segunda é a
// que o mestre sente — os casos acima passavam verdes com a peça levando o
// batimento inteiro para andar um quadrado.
//
// Este abre um stream de VERDADE por HTTP, move uma peça, e exige o quadro em bem
// menos que o batimento. O limite é 400ms: folgado para um round-trip local, e
// menos da metade do batimento, então um verde aqui não pode ser o relógio.
func TestMovingATokenReachesTheStreamWithoutWaitingForTheHeartbeat(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	// O id vem do SERVIDOR (`bs.newID`) e não do que se passa: dois clientes
	// criando ao mesmo tempo não podem inventar o mesmo. Por isso ele é lido do
	// estado devolvido em vez de assumido.
	placed, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab,
		board.BoardToken{Label: "Ogro", X: 2, Y: 2})
	if err != nil {
		t.Fatalf("pôr a peça: %v", err)
	}
	tokenID := placed.Tokens[len(placed.Tokens)-1].ID

	srv := httptest.NewServer(f.s.WebRouter())
	defer srv.Close()
	req, requestErr := http.NewRequest(http.MethodGet, srv.URL+f.tableUrl()+"/fluxo", nil)
	if requestErr != nil {
		t.Fatalf("montar pedido: %v", requestErr)
	}
	req.Header.Set("Authorization", "Bearer "+f.token(t, f.gm))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	resp, streamErr := http.DefaultClient.Do(req.WithContext(ctx))
	if streamErr != nil {
		t.Fatalf("abrir stream: %v", streamErr)
	}
	defer func() { _ = resp.Body.Close() }()

	frames := make(chan string, 8)
	go func() {
		reader := bufio.NewScanner(resp.Body)
		reader.Buffer(make([]byte, 0, 64*1024), 1<<20)
		var current strings.Builder
		for reader.Scan() {
			if row := reader.Text(); row != "" {
				current.WriteString(row)
				continue
			}
			select {
			case frames <- current.String():
			default:
			}
			current.Reset()
		}
	}()

	// A carga fria é o CONTROLE: sem ela, um stream que nunca abriu daria o mesmo
	// silêncio que um aviso que não chega.
	//
	// PROCURA entre os quadros porque a carga manda um por REGIÃO: esperar a peça
	// no primeiro afirmaria a ordem do render, que não é promessa.
	waitsForToken := func(where, what string) {
		t.Helper()
		cap := time.After(3 * time.Second)
		for {
			select {
			case q := <-frames:
				if strings.Contains(q, where) {
					return
				}
			case <-cap:
				t.Fatalf("%s", what)
			}
		}
	}
	waitsForToken("Ogro em 2, 2", "a carga fria não trouxe a peça onde ela está")

	start := time.Now()
	if _, err := f.s.tableHost().Boards().UpdateToken(context.Background(), f.sessionID, defaultTab, tokenID,
		board.ParseTokenPatch(map[string]any{"x": 7})); err != nil {
		t.Fatalf("mover a peça: %v", err)
	}

	limit := time.After(400 * time.Millisecond)
	for {
		select {
		case q := <-frames:
			if strings.Contains(q, "Ogro em 7, 2") {
				t.Logf("a peça chegou à tela em %v", time.Since(start))
				return
			}
		case <-limit:
			t.Fatal("a peça não chegou em 400ms — a tela está esperando o batimento do stream")
		}
	}
}

// O guarda que as REGIÕES existem para dar.
//
// Remendar o `<main id="table">` inteiro a cada mudança de qualquer um não é só
// desperdício: com o arrasto, um jogador registrando iniciativa substituiria o
// elemento debaixo do dedo do mestre e cancelaria o gesto. O que se mede é a
// separação — mexer na FILA manda o quadro da fila e NÃO manda o do mapa.
func TestATrackerChangeDoesNotPatchTheMap(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	if _, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab,
		board.BoardToken{Label: "Ogro", X: 2, Y: 2}); err != nil {
		t.Fatalf("pôr a peça: %v", err)
	}

	srv := httptest.NewServer(f.s.WebRouter())
	defer srv.Close()
	req, requestErr := http.NewRequest(http.MethodGet, srv.URL+f.tableUrl()+"/fluxo", nil)
	if requestErr != nil {
		t.Fatalf("montar pedido: %v", requestErr)
	}
	req.Header.Set("Authorization", "Bearer "+f.token(t, f.gm))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	resp, streamErr := http.DefaultClient.Do(req.WithContext(ctx))
	if streamErr != nil {
		t.Fatalf("abrir stream: %v", streamErr)
	}
	defer func() { _ = resp.Body.Close() }()

	frames := make(chan string, 32)
	go func() {
		reader := bufio.NewScanner(resp.Body)
		reader.Buffer(make([]byte, 0, 64*1024), 1<<20)
		var current strings.Builder
		for reader.Scan() {
			if row := reader.Text(); row != "" {
				current.WriteString(row)
				continue
			}
			select {
			case frames <- current.String():
			default:
			}
			current.Reset()
		}
	}()

	// Deixa a carga fria passar drenando até o SILÊNCIO, e não contando quadros:
	// contar afirmaria quantos eventos o SDK emite por região, que não é promessa
	// nenhuma, e faria a própria carga passar por mudança.
	seenOnLoad := 0
	for stopped := false; !stopped; {
		select {
		case <-frames:
			seenOnLoad++
		case <-time.After(300 * time.Millisecond):
			stopped = true
		}
	}
	if seenOnLoad == 0 {
		t.Fatal("a carga fria não mandou nada — sem ela o silêncio abaixo não prova nada")
	}

	// Agora UMA mudança na fila, e mais nada.
	if _, err := f.s.tableHost().Sessions().AddInitiativeEntry(f.sessionID,
		sheetCombatant("Arwen", 17, f.charID)); err != nil {
		t.Fatalf("pôr na fila: %v", err)
	}

	var sawQueue, sawBoard bool
	limit := time.After(700 * time.Millisecond)
	for !sawQueue {
		select {
		case q := <-frames:
			if strings.Contains(q, `id="table-tracker"`) {
				sawQueue = true
			}
			if strings.Contains(q, `id="table-board"`) {
				sawBoard = true
			}
		case <-limit:
			t.Fatal("a mudança da fila não chegou — o teste não alcançou o que queria medir")
		}
	}
	// O CONTROLE é o `sawQueue` acima: sem ele, "não vi o mapa" seria verdade
	// também num stream que parou de mandar qualquer coisa.
	if sawBoard {
		t.Error("mexer na FILA remendou o MAPA — a peça debaixo do dedo do mestre seria trocada no meio do arrasto")
	}
}

// A cortina não pode parecer um tabuleiro vazio: os dois estados têm de se
// parecer o menos possível, porque o jogador resolve um esperando e o outro
// cutucando o mestre.
//
// O nome do lugar não pode aparecer — "Covil do Dragão" já conta a cena que a
// cortina existe para esconder. Quem o apaga é o `BoardForRole`; o que se prende
// aqui é que a cena não o reintroduz.
func TestTheCurtainHidesTheSceneAndDoesNotLookLikeAnEmptyBoard(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "crypt")
	if _, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab,
		board.BoardToken{Label: "Dragão", X: 3, Y: 3}); err != nil {
		t.Fatalf("pôr a peça: %v", err)
	}
	if _, _, err := f.s.tableHost().Boards().SetCurtain(context.Background(), f.sessionID, defaultTab, true); err != nil {
		t.Fatalf("fechar a cortina: %v", err)
	}

	forPlayer := f.pede(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(forPlayer, "O mestre está montando a cena") {
		t.Error("a mesa não viu a cortina")
	}
	// As TRÊS ausências, e cada uma é um vazamento diferente: a grade contaria
	// que há cena montada, a peça contaria o que há nela, e o nome contaria qual
	// é ela.
	for _, leak := range []string{"board-plane", "Dragão", "Taverna do Javali"} {
		if strings.Contains(forPlayer, leak) {
			t.Errorf("a cortina deixou passar %q", leak)
		}
	}

	// O CONTROLE: o mestre continua vendo a cena inteira, senão "a mesa não viu"
	// seria verdade também num tabuleiro que ninguém abriu.
	forGM := f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(forGM, "Dragão") || !strings.Contains(forGM, "board-plane") {
		t.Error("o mestre perdeu a própria cena com a cortina fechada")
	}
	// E ele é AVISADO. Sem a tira, o mapa dele fica igualzinho com a cortina
	// aberta ou fechada, e ele narra a taverna para uma mesa que vê um aviso.
	if !strings.Contains(forGM, "a mesa não vê esta cena") {
		t.Error("o mestre não foi avisado de que a cortina está fechada")
	}
}

// A PEÇA AVULSA NASCE ONDE O MESTRE CLICOU.
//
// O GLOSSARY promete que "uma peça pode existir sem linha na fila (a porta, o
// baú)"; o `poeNoMapa` itera a INICIATIVA e só faz nascer peça de combatente.
//
// A asserção é sobre a CASA e não só sobre a existência: uma peça que nasce no
// lugar errado é pior que nenhuma, porque o mestre põe a porta e ela aparece do
// outro lado da cripta.
func TestALoosePieceIsBornOnTheSquareTheGmClicked(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "crypt")

	body := f.posta(t, f.gm, f.tableUrl()+"/tabuleiro/pecas/nova", `{"from":{"X":-3,"Y":7},"new_token_name":"  Porta da cripta  ","new_token_size":1,"new_token_look":"object"}`)

	board := f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)
	if len(board.Tokens) != 1 {
		t.Fatalf("o mapa ficou com %d peças; a resposta foi:\n%s", len(board.Tokens), firstRows(body, 6))
	}
	token := board.Tokens[0]
	// COORDENADA NEGATIVA é lugar legítimo — o plano não tem bordas —, e é por
	// isso que ela viaja no caminho. O -3 está aqui de propósito.
	if token.X != -3 || token.Y != 7 {
		t.Errorf("a peça nasceu em (%d,%d) e o clique foi em (-3,7)", token.X, token.Y)
	}
	// O nome vai APARADO, como o do combatente: espaço sobrando não deve
	// produzir uma peça que ordena e se lê diferente do que o mestre digitou.
	if token.Label != "Porta da cripta" {
		t.Errorf("o rótulo ficou %q", token.Label)
	}
	if token.Kind != "object" {
		t.Errorf("a aparência ficou %q", token.Kind)
	}
	// E ela NÃO tem linha na fila: é isso que a distingue do `poeNoMapa`, e é a
	// promessa do glossário.
	if token.EntryID != nil {
		t.Errorf("a peça avulsa nasceu ligada à fila: %v", *token.EntryID)
	}
}

// A TIRA RECUSA o que não desenha peça nenhuma, e a recusa nomeia o valor.
//
// Os três casos são de naturezas diferentes de propósito: o nome vazio é o que
// deixaria a peça muda no mapa e no leitor de tela; o tamanho fora da Tabela
// 1-21 (p107) desenharia uma criatura que o livro não tem; e a aparência
// `character` é a que criaria uma peça que PARECE de jogador sem ninguém atrás.
func TestTheLoosePieceRefusesWhatDrawsNoPiece(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "crypt")
	cases := []struct{ name, signals, waits string }{
		{"sem nome", `{"from":{"X":1,"Y":1},"new_token_name":"   ","new_token_size":1,"new_token_look":"object"}`, "dê um nome"},
		{"tamanho de nada", `{"from":{"X":1,"Y":1},"new_token_name":"Carroça","new_token_size":4,"new_token_look":"object"}`, "p107"},
		{"ficha solta", `{"from":{"X":1,"Y":1},"new_token_name":"Falso herói","new_token_size":1,"new_token_look":"character"}`, "aparência"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := f.posta(t, f.gm, f.tableUrl()+"/tabuleiro/pecas/nova", tc.signals)
			if !strings.Contains(body, tc.waits) {
				t.Errorf("a recusa não citou %q; a resposta foi:\n%s", tc.waits, firstRows(body, 6))
			}
		})
	}
	// O CONTROLE: nenhuma das três recusas pode ter escrito. Sem ele, uma recusa
	// que já tivesse criado a peça passaria pelas asserções acima.
	if board := f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab); len(board.Tokens) != 0 {
		t.Errorf("as recusas deixaram %d peças no mapa", len(board.Tokens))
	}
}

// PÔR PEÇA É DO MESTRE, e a trava é o servidor.
//
// Esconder o botão do jogador é UX; quem impede é o handler. O caso posta na
// mão, como quem abre o console: botão ausente nunca foi prova de trava.
func TestOnlyTheGmPutsALoosePieceOnTheMap(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "crypt")

	rec := f.pede(t, f.player, "POST", f.tableUrl()+"/tabuleiro/pecas/nova", `{"from":{"X":1,"Y":1},"new_token_name":"Porta","new_token_size":1,"new_token_look":"object"}`)

	if rec.Code != http.StatusForbidden {
		t.Errorf("o jogador pôs peça e levou %d, queria 403", rec.Code)
	}
	if board := f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab); len(board.Tokens) != 0 {
		t.Errorf("a recusa deixou %d peças no mapa", len(board.Tokens))
	}
}

// O MODO da peça avulsa é do MESTRE, e ele aparece FORA da fileira numerada.
//
// As duas metades se medem juntas: o botão sem a tira seria um modo que liga e
// não diz o que vai criar, e a tira sem o botão seria um formulário sem gesto.
//
// O 10 é a decisão de desenho: o `railKeys` tem dez dígitos e o `numberRail`
// estoura na décima primeira de propósito. Dar uma tecla a este modo é mudar
// este caso junto, em vez de a gramática dos dez virar dez-e-meio em silêncio.
func TestTheNewPieceModeBelongsToTheGmAndHasNoNumber(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "crypt")

	forGM := f.pede(t, f.gm, "GET", f.tableUrl(), "").Body.String()
	for _, chunk := range []string{
		"Nova peça — o clique escolhe a casa",       // o botão do modo
		"Nova peça — escolha a casa onde ela nasce", // a camada de clique
		"new_token_name", // a tira
		"Colossal",       // o tamanho do livro (p107)
	} {
		if !strings.Contains(forGM, chunk) {
			t.Errorf("o mestre não recebeu %q", chunk)
		}
	}
	// As dez ferramentas continuam sendo dez, e o modo não entrou na conta.
	if n := len(table.MapTools()); n != 10 {
		t.Errorf("o trilho numerado ficou com %d ferramentas — a gramática dos dez dígitos quebrou", n)
	}

	// O QUE NÃO PODE VAZAR É A AFORDÂNCIA, e não o sinal: a lista de sinais é da
	// PÁGINA e sai igual para os dois papéis, e um sinal sem escritor é inerte.
	// O que conta é não haver botão nem camada de clique — e a trava de verdade é
	// o 403 do handler, medido no caso vizinho.
	forPlayer := f.pede(t, f.player, "GET", f.tableUrl(), "").Body.String()
	if strings.Contains(forPlayer, "Nova peça") {
		t.Error("o gesto de criar peça vazou para o HTML do jogador")
	}
}

// A PEÇA DE CENÁRIO SE DESENHA DIFERENTE, e é aqui que o `Kind` ganha leitor.
//
// A distinção é a FORMA e não a cor: redondo é criatura em toda mesa de VTT, e
// tinta nova entraria na conta do medidor de contraste — onde uma variante que
// só aparece com cenário no mapa nasceria sem medição.
func TestTheSceneryPieceIsDrawnSquareAndTheCreatureIsNot(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "crypt")
	base := f.tableUrl() + "/tabuleiro/pecas/nova"
	f.posta(t, f.gm, base, `{"from":{"X":1,"Y":1},"new_token_name":"Porta","new_token_size":1,"new_token_look":"object"}`)
	f.posta(t, f.gm, base, `{"from":{"X":5,"Y":5},"new_token_name":"Lobo","new_token_size":1,"new_token_look":"npc"}`)

	html := f.pede(t, f.gm, "GET", f.tableUrl(), "").Body.String()

	// O CONTROLE vem primeiro: as duas peças TÊM de estar no mapa, senão o resto
	// mede a ausência das duas e passa verde dizendo nada.
	for _, name := range []string{"Porta", "Lobo"} {
		if !strings.Contains(html, name) {
			t.Fatalf("a peça %q não chegou ao mapa", name)
		}
	}
	// UMA classe de objeto e só uma: a porta a tem, o lobo não.
	if n := strings.Count(html, "board-token-object"); n != 1 {
		t.Errorf("a classe de cenário aparece %d vezes; a porta a tem e o lobo não", n)
	}
}

// ANDAR GASTA A AÇÃO DE MOVIMENTO DO TURNO (T20 p233), e a segunda sai da
// padrão — "você pode trocar sua ação padrão por uma ação de movimento".
//
// O caso é de INTEGRAÇÃO e não de unidade porque o que ele prende é a COSTURA:
// a regra mora no `engine.TurnBudget`, o que sobrou mora no `live.Scene`, e
// quem os junta é o comando da cena. Três lugares que um teste de cada um
// deixaria verde sem ninguém cobrando ninguém.
// tokenSquare devolve onde a peça está, pelo tabuleiro de verdade.
func tokenSquare(t *testing.T, f sceneFixture, tokenID string) [2]int {
	t.Helper()
	boardState := f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)
	token := board.FindToken(boardState, tokenID)
	if token == nil {
		t.Fatalf("a peça %q sumiu do tabuleiro", tokenID)
	}
	return [2]int{token.X, token.Y}
}

func TestMovingOnYourTurnSpendsTheMovementAction(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	f.seedOpenBoard(t, "stone")

	// A VEZ é a do Ogro, que tem a iniciativa mais alta. A peça dele precisa
	// apontar para a LINHA — é o `entryId` que liga o gesto ao turno.
	state := f.s.sessions.GetState(f.sessionID)
	if _, err := f.s.sessions.NextTurn(f.sessionID); err != nil {
		t.Fatalf("começar o turno: %v", err)
	}
	state = f.s.sessions.GetState(f.sessionID)
	onTurn := state.Initiative[state.TurnIndex]
	placed, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab,
		board.BoardToken{Label: onTurn.Label, X: 2, Y: 2, EntryID: &onTurn.ID})
	if err != nil {
		t.Fatalf("pôr a peça: %v", err)
	}
	tokenID := placed.Tokens[len(placed.Tokens)-1].ID

	walk := func(destX int) *httptest.ResponseRecorder {
		base := f.tableUrl() + "/tabuleiro/" + tokenID
		if rec := f.pede(t, f.gm, "POST", base+"/parada",
			`{"from":{"X":`+strconv.Itoa(destX)+`,"Y":2}}`); rec.Code != http.StatusOK {
			t.Fatalf("propor a parada deu %d", rec.Code)
		}
		return f.pede(t, f.gm, "POST", base+"/confirmar", "")
	}

	if rec := walk(3); rec.Code != http.StatusOK {
		t.Fatalf("o primeiro movimento deu %d", rec.Code)
	}
	after := f.s.sessions.GetState(f.sessionID)
	if after.Scene.MovementLeft {
		t.Error("andar gastou a ação de MOVIMENTO, e ela continua de pé")
	}
	if !after.Scene.StandardLeft {
		t.Error("andar não toca na ação padrão")
	}

	// A SEGUNDA sai da padrão, pela troca de mão única da p233.
	if rec := walk(4); rec.Code != http.StatusOK {
		t.Fatalf("o segundo movimento, que sai da padrão, deu %d", rec.Code)
	}
	after = f.s.sessions.GetState(f.sessionID)
	if after.Scene.StandardLeft || after.Scene.MovementLeft {
		t.Errorf("o turno inteiro foi gasto em dois movimentos, e sobrou %+v", after.Scene)
	}

	// A TERCEIRA NÃO ACONTECE — e o que se prende é a PEÇA, não a frase: uma
	// recusa escrita sobre um movimento que pousou é a pior das duas saídas,
	// porque a mesa lê o erro e vê a peça no lugar novo.
	wasAt := tokenSquare(t, f, tokenID)
	rec := walk(5)
	if now := tokenSquare(t, f, tokenID); now != wasAt {
		t.Errorf("sem ação no turno a peça não anda, e ela foi de %v para %v", wasAt, now)
	}
	if !strings.Contains(rec.Body.String(), "não sobrou ação neste turno") {
		t.Errorf("a terceira andada tinha de ser recusada por falta de ação, e veio %q",
			rec.Body.String()[:min(220, len(rec.Body.String()))])
	}
	if !strings.Contains(rec.Body.String(), onTurn.Label) {
		t.Errorf("a recusa diz de quem é o turno, e veio sem %q", onTurn.Label)
	}

	// E A TELA NÃO OFERECE O QUE O SERVIDOR RECUSA: com o turno gasto, a
	// proposta continua desenhável — propor é rascunho — mas o painel dela diz
	// que não há ação e não põe um "Confirmar" na frente de quem vai ouvir não.
	if rec := f.pede(t, f.gm, "POST", f.tableUrl()+"/tabuleiro/"+tokenID+"/parada",
		`{"from":{"X":6,"Y":2}}`); rec.Code != http.StatusOK {
		t.Fatalf("propor com o turno gasto deu %d", rec.Code)
	}
	screen := f.pede(t, f.gm, "GET", f.tableUrl(), "").Body.String()
	if !strings.Contains(screen, "não sobrou ação neste turno") {
		t.Error("o painel do movimento não diz que o turno acabou")
	}
	if strings.Contains(screen, ">Confirmar</button>") {
		t.Error("o painel oferece Confirmar num movimento que o servidor vai recusar")
	}

	// PASSAR A VEZ devolve o turno inteiro a quem entra nele.
	if _, err := f.s.sessions.NextTurn(f.sessionID); err != nil {
		t.Fatalf("passar a vez: %v", err)
	}
	if fresh := f.s.sessions.GetState(f.sessionID).Scene; !fresh.StandardLeft || !fresh.MovementLeft {
		t.Errorf("quem entra no turno o recebe inteiro, e veio %+v", fresh)
	}
}
