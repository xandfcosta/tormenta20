package api

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"t20engine/events"
	"t20engine/tabuleiro"
	"t20engine/web/table"
	"testing"
	"time"
)

func (f pilotoFixture) seedOpenBoard(t *testing.T, terreno string) *tabuleiro.BoardState {
	t.Helper()
	b, err := f.s.tableHost().Boards().Open(context.Background(), f.sessionID, "Taverna do Javali", terreno)
	if err != nil {
		t.Fatalf("o tabuleiro não abriu: %v", err)
	}
	return b
}

// TestWithoutABoardTheSceneSaysThereIsNoMap.
//
// "Não há tabuleiro" e "há um vazio" são estados diferentes, e o primeiro é o
// normal — a maior parte de uma sessão não tem mapa. Desenhar uma grade vazia
// diria que o mestre abriu uma cena que ele não abriu.
func TestWithoutABoardTheSceneSaysThereIsNoMap(t *testing.T) {
	f := novoPiloto(t)
	corpo := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()

	if !strings.Contains(corpo, "Nenhum tabuleiro aberto") {
		t.Error("a cena não disse que não há mapa")
	}
	if strings.Contains(corpo, "tabuleiro-plano") {
		t.Error("desenhou a grade sem tabuleiro aberto")
	}
}

// TestTheHiddenTokenDoesNotReachThePlayer — o guarda que mais importa desta fatia.
//
// Esconder a peça é o gesto com que o mestre guarda a emboscada, e a trava não
// pode ser CSS: uma peça meio-apagada no HTML do jogador entrega a posição do
// ogro para quem abrir o inspetor. Quem a tira é o `BoardForRole`, o mesmo
// gargalo por papel que a fila usa — e este teste afirma que a cena passa por
// ele em vez de decidir por conta própria.
func TestTheHiddenTokenDoesNotReachThePlayer(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "cripta")
	if _, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab,
		tabuleiro.BoardToken{ID: "emboscada", Label: "Ogro", X: 4, Y: 3, Hidden: true}, true); err != nil {
		t.Fatalf("pôr a peça escondida: %v", err)
	}
	if _, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab,
		tabuleiro.BoardToken{ID: "avista", Label: "Arwen", X: 1, Y: 1}, true); err != nil {
		t.Fatalf("pôr a peça à vista: %v", err)
	}

	doMestre := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(doMestre, "Ogro em") {
		t.Error("o mestre não viu a própria peça escondida")
	}

	doJogador := f.pede(t, f.jogador, http.MethodGet, f.tableUrl(), "").Body.String()
	// O CONTROLE: o jogador está vendo o tabuleiro. Sem ele, "não achei o Ogro"
	// seria verdade também numa cena sem mapa nenhum.
	if !strings.Contains(doJogador, "Arwen em") {
		t.Fatal("o jogador não viu o tabuleiro; a ausência abaixo não provaria nada")
	}
	if strings.Contains(doJogador, "Ogro") {
		t.Error("a peça escondida chegou ao HTML do jogador")
	}
}

// TestTheTokenOnTurnLightsUpWithTheSameGoldAsTheTracker.
//
// O anel é o MESMO sinal que a linha da fila usa, e ligá-lo pelo `entryId` é o
// que garante isso: derivar "quem está na vez" no tabuleiro seria a segunda
// cópia da regra, e é assim que duas telas passam a apontar combatentes
// diferentes (ALE-122).
func TestTheTokenOnTurnLightsUpWithTheSameGoldAsTheTracker(t *testing.T) {
	f := novoPiloto(t)
	entryID := f.tracker(t)
	f.seedOpenBoard(t, "pedra")
	if _, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab,
		tabuleiro.BoardToken{ID: "p", Label: "Arcanista", X: 2, Y: 2, EntryID: &entryID}, true); err != nil {
		t.Fatalf("pôr a peça: %v", err)
	}
	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/scene/start", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar cena deu %d", rec.Code)
	}

	// FORA de combate ninguém está na vez, mesmo com a cena aberta e a fila
	// montada — é o `TurnIndex` negativo, e a peça não pode acender por estar
	// no mapa.
	antes := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()
	if strings.Contains(antes, "tabuleiro-peca-na-vez") {
		t.Error("a peça acendeu antes de o combate começar")
	}

	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/initiative/next-turn", ""); rec.Code != http.StatusOK {
		t.Fatalf("avançar deu %d", rec.Code)
	}
	depois := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(depois, "tabuleiro-peca-na-vez") {
		t.Error("chegou a vez do combatente e a peça dele não acendeu")
	}
	if !strings.Contains(depois, "— na vez") {
		t.Error("o anel não tem par em TEXTO: cor não existe para leitor de tela (ALE-212)")
	}
}

// TestAnInventedTerrainFallsBackToTheDefaultGround.
//
// O terreno vem do BANCO, então é dado do cliente. Uma classe `chao-<qualquer>`
// não existiria na folha e o chão sairia transparente — o que se parece com
// defeito de CSS e manda procurar no lugar errado.
func TestAnInventedTerrainFallsBackToTheDefaultGround(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "vulcão-de-neon")

	corpo := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(corpo, "chao-pedra") {
		t.Error("o terreno inventado não caiu no chão padrão")
	}
	if strings.Contains(corpo, "chao-vulcão") {
		t.Error("o terreno inventado virou classe solta")
	}
}

// ── o aviso do tabuleiro (ALE-264) ───────────────────────────────────────────

// TestTheBoardTellsItsListenersOnEveryChange.
//
// O stream da Mesa assinava só o store da SESSÃO, e o tabuleiro é outro: mover
// uma peça não acordava ninguém, e a mudança só chegava no batimento de reserva
// — 1310ms cronometrados no navegador.
//
// Os TRÊS pontos de escrita são medidos, e o `Open`/`Close` estão aqui porque
// eles não passam pelo `apply`: abrir e fechar são as mudanças mais VISÍVEIS do
// tabuleiro — a grade aparecendo e sumindo —, e um aviso que cobrisse só o que
// se move perderia o que nasce.
//
// Desde a ALE-279 o caso afirma QUAL evento chegou, e não só que algo chegou. É
// a diferença que o barramento comprou: com `chan struct{}` abrir e fechar eram
// o mesmo sino, então trocar um pelo outro no código passava verde aqui.
func TestTheBoardTellsItsListenersOnEveryChange(t *testing.T) {
	f := novoPiloto(t)
	bs := f.s.tableHost().Boards()
	ctx := context.Background()
	const sessao = int64(1)

	sub, parar := f.s.tableHost().Bus().Subscribe(events.OfSession(sessao))
	defer parar()
	drenar := func() {
		for len(sub.C) > 0 {
			<-sub.C
		}
	}
	avisou := func(oque string, esperado events.Event) {
		t.Helper()
		select {
		case ev := <-sub.C:
			if fmt.Sprintf("%T", ev) != fmt.Sprintf("%T", esperado) {
				t.Errorf("%s publicou %T, esperado %T", oque, ev, esperado)
			}
		default:
			t.Errorf("%s não avisou quem escuta", oque)
		}
	}

	drenar()
	if _, err := bs.Open(ctx, sessao, "Taverna", "taverna"); err != nil {
		t.Fatalf("abrir: %v", err)
	}
	avisou("abrir o tabuleiro", events.BoardOpened{})

	drenar()
	if _, err := bs.AddToken(ctx, sessao, defaultTab, tabuleiro.BoardToken{ID: "p", Label: "Ogro", X: 1, Y: 1}, true); err != nil {
		t.Fatalf("pôr a peça: %v", err)
	}
	avisou("pôr uma peça (pelo apply)", events.BoardChanged{})

	drenar()
	bs.Close(ctx, sessao, defaultTab)
	avisou("fechar o tabuleiro", events.BoardClosed{})
}

// E uma mutação RECUSADA não avisa: "mudou" tem de significar mudou, senão o
// stream relê e o hash o faz calar — trabalho para nada a cada erro de quem
// clica.
func TestARefusedMutationTellsNobody(t *testing.T) {
	f := novoPiloto(t)
	ctx := context.Background()
	const sessao = int64(2)

	sub, parar := f.s.tableHost().Bus().Subscribe(events.OfSession(sessao))
	defer parar()
	// SEM tabuleiro aberto: o `apply` recusa antes de mexer em nada.
	if _, err := f.s.tableHost().Boards().AddToken(ctx, sessao, defaultTab, tabuleiro.BoardToken{ID: "p", Label: "Ogro"}, true); err == nil {
		t.Fatal("pôr peça sem tabuleiro devia recusar; sem a recusa este teste não mede nada")
	}
	select {
	case ev := <-sub.C:
		t.Errorf("a mutação recusada publicou %T", ev)
	default:
	}
}

// Aqui morava `TestABaixaLimpaOOuvinte`, que afirmava que o `desassinar` do
// tabuleiro limpava o registro. O registro deixou de ser do tabuleiro: ele é do
// barramento, e a baixa é medida onde ela mora, em `events.TestUnsubscribeRemovesTheListener`.
// Uma regra, uma camada.

// TestMovingATokenReachesTheStreamWithoutWaitingForTheHeartbeat.
//
// Este é o teste que a MEDIÇÃO pediu, e ele nasceu porque o guarda de unidade
// não bastou: os testes acima provam que o store AVISA, e mesmo assim o
// cronômetro no navegador deu 1000ms redondos — o batimento — para uma peça
// andar um quadrado. "O store avisa" e "a tela recebe" são perguntas diferentes,
// e só esta segunda é a que o mestre sente.
//
// Ele abre um stream de VERDADE por HTTP, move uma peça, e exige o quadro em bem
// menos que o batimento. O limite é 400ms: folgado para um round-trip local, e
// menos da metade do batimento, então um verde aqui não pode ser o relógio.
func TestMovingATokenReachesTheStreamWithoutWaitingForTheHeartbeat(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")
	// O id vem do SERVIDOR (`bs.newID`), não do que eu passo: dois clientes
	// criando ao mesmo tempo não podem inventar o mesmo. Por isso ele é lido do
	// estado devolvido em vez de assumido — a primeira versão deste teste
	// assumiu "p" e morreu em `peça "p" não está no tabuleiro`.
	posto, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab,
		tabuleiro.BoardToken{Label: "Ogro", X: 2, Y: 2}, true)
	if err != nil {
		t.Fatalf("pôr a peça: %v", err)
	}
	pecaID := posto.Tokens[len(posto.Tokens)-1].ID

	srv := httptest.NewServer(f.s.WebRouter())
	defer srv.Close()
	req, erroDoPedido := http.NewRequest(http.MethodGet, srv.URL+f.tableUrl()+"/stream", nil)
	if erroDoPedido != nil {
		t.Fatalf("montar pedido: %v", erroDoPedido)
	}
	req.Header.Set("Authorization", "Bearer "+f.token(t, f.mestre))
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	resp, erroDoStream := http.DefaultClient.Do(req.WithContext(ctx))
	if erroDoStream != nil {
		t.Fatalf("abrir stream: %v", erroDoStream)
	}
	defer func() { _ = resp.Body.Close() }()

	quadros := make(chan string, 8)
	go func() {
		leitor := bufio.NewScanner(resp.Body)
		leitor.Buffer(make([]byte, 0, 64*1024), 1<<20)
		var atual strings.Builder
		for leitor.Scan() {
			if linha := leitor.Text(); linha != "" {
				atual.WriteString(linha)
				continue
			}
			select {
			case quadros <- atual.String():
			default:
			}
			atual.Reset()
		}
	}()

	// A carga fria é o CONTROLE: sem ela, um stream que nunca abriu daria o mesmo
	// silêncio que um aviso que não chega.
	//
	// PROCURA entre os quadros porque desde as regiões (ALE-264) a carga manda um
	// por região, e o tabuleiro é o quarto. Esperar a peça no PRIMEIRO afirmaria
	// a ordem do render, que não é promessa.
	esperaAPeca := func(onde, oque string) {
		t.Helper()
		limite := time.After(3 * time.Second)
		for {
			select {
			case q := <-quadros:
				if strings.Contains(q, onde) {
					return
				}
			case <-limite:
				t.Fatalf("%s", oque)
			}
		}
	}
	esperaAPeca("Ogro em 2, 2", "a carga fria não trouxe a peça onde ela está")

	inicio := time.Now()
	if _, err := f.s.tableHost().Boards().UpdateToken(context.Background(), f.sessionID, defaultTab, pecaID,
		tabuleiro.ParseTokenPatch(map[string]any{"x": 7})); err != nil {
		t.Fatalf("mover a peça: %v", err)
	}

	limite := time.After(400 * time.Millisecond)
	for {
		select {
		case q := <-quadros:
			if strings.Contains(q, "Ogro em 7, 2") {
				t.Logf("a peça chegou à tela em %v", time.Since(inicio))
				return
			}
		case <-limite:
			t.Fatal("a peça não chegou em 400ms — a tela está esperando o batimento do stream")
		}
	}
}

// TestATrackerChangeDoesNotPatchTheMap — o guarda que as REGIÕES existem para dar
// (ALE-264).
//
// A cena era um fragmento só, e o stream remendava o `<main id="mesa">` inteiro
// a cada mudança de qualquer um: 39.742 bytes medidos para mover uma peça. O
// desperdício é o menor dos problemas — o problema é de COMPORTAMENTO, e o dono
// o nomeou: com o arrasto, um jogador registrando iniciativa substituiria o
// elemento debaixo do dedo do mestre e cancelaria o gesto.
//
// Este teste mede a separação onde ela importa: mexer na FILA manda o quadro da
// fila e NÃO manda o do mapa.
func TestATrackerChangeDoesNotPatchTheMap(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")
	if _, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab,
		tabuleiro.BoardToken{Label: "Ogro", X: 2, Y: 2}, true); err != nil {
		t.Fatalf("pôr a peça: %v", err)
	}

	srv := httptest.NewServer(f.s.WebRouter())
	defer srv.Close()
	req, erroDoPedido := http.NewRequest(http.MethodGet, srv.URL+f.tableUrl()+"/stream", nil)
	if erroDoPedido != nil {
		t.Fatalf("montar pedido: %v", erroDoPedido)
	}
	req.Header.Set("Authorization", "Bearer "+f.token(t, f.mestre))
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	resp, erroDoStream := http.DefaultClient.Do(req.WithContext(ctx))
	if erroDoStream != nil {
		t.Fatalf("abrir stream: %v", erroDoStream)
	}
	defer func() { _ = resp.Body.Close() }()

	quadros := make(chan string, 32)
	go func() {
		leitor := bufio.NewScanner(resp.Body)
		leitor.Buffer(make([]byte, 0, 64*1024), 1<<20)
		var atual strings.Builder
		for leitor.Scan() {
			if linha := leitor.Text(); linha != "" {
				atual.WriteString(linha)
				continue
			}
			select {
			case quadros <- atual.String():
			default:
			}
			atual.Reset()
		}
	}()

	// Deixa a carga fria passar, drenando até o SILÊNCIO em vez de contar quadros.
	// Contar afirmaria quantos eventos o SDK emite por região, que não é promessa
	// nenhuma — e a primeira versão deste teste contou 6 e mediu a própria
	// carga como se fosse a mudança.
	vistosNaCarga := 0
	for parou := false; !parou; {
		select {
		case <-quadros:
			vistosNaCarga++
		case <-time.After(300 * time.Millisecond):
			parou = true
		}
	}
	if vistosNaCarga == 0 {
		t.Fatal("a carga fria não mandou nada — sem ela o silêncio abaixo não prova nada")
	}

	// Agora UMA mudança na fila, e mais nada.
	if _, err := f.s.tableHost().Sessions().AddInitiativeEntry(f.sessionID,
		sheetCombatant("Arwen", 17, f.charID)); err != nil {
		t.Fatalf("pôr na fila: %v", err)
	}

	var viuAFila, viuOMapa bool
	limite := time.After(700 * time.Millisecond)
	for !viuAFila {
		select {
		case q := <-quadros:
			if strings.Contains(q, `id="mesa-fila"`) {
				viuAFila = true
			}
			if strings.Contains(q, `id="mesa-tabuleiro"`) {
				viuOMapa = true
			}
		case <-limite:
			t.Fatal("a mudança da fila não chegou — o teste não alcançou o que queria medir")
		}
	}
	// O CONTROLE é o `viuAFila` acima: sem ele, "não vi o mapa" seria verdade
	// também num stream que parou de mandar qualquer coisa.
	if viuOMapa {
		t.Error("mexer na FILA remendou o MAPA — a peça debaixo do dedo do mestre seria trocada no meio do arrasto")
	}
}

// TestTheCurtainHidesTheSceneAndDoesNotLookLikeAnEmptyBoard: a cortina esconde a cena e não parece um tabuleiro vazio (ALE-202).
//
// A cortina chegou pela `main` e a Mesa em Datastar desenharia uma GRADE VAZIA
// no lugar dela — o que é justamente a tela do "o mestre ainda não abriu um
// tabuleiro". Os dois estados têm de se parecer o menos possível: o jogador
// resolve um esperando e o outro cutucando o mestre.
//
// O nome do lugar não pode aparecer, e a razão está no glossário: "Covil do
// Dragão" já conta a cena que a cortina existe para esconder. Quem o apaga é o
// `BoardForRole`; o que se prende aqui é que a cena não o reintroduz.
func TestTheCurtainHidesTheSceneAndDoesNotLookLikeAnEmptyBoard(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "cripta")
	if _, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab,
		tabuleiro.BoardToken{Label: "Dragão", X: 3, Y: 3}, true); err != nil {
		t.Fatalf("pôr a peça: %v", err)
	}
	if _, _, err := f.s.tableHost().Boards().SetCurtain(context.Background(), f.sessionID, defaultTab, true); err != nil {
		t.Fatalf("fechar a cortina: %v", err)
	}

	doJogador := f.pede(t, f.jogador, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(doJogador, "O mestre está montando a cena") {
		t.Error("a mesa não viu a cortina")
	}
	// As TRÊS ausências, e cada uma é um vazamento diferente: a grade contaria
	// que há cena montada, a peça contaria o que há nela, e o nome contaria qual
	// é ela.
	for _, vazamento := range []string{"tabuleiro-plano", "Dragão", "Taverna do Javali"} {
		if strings.Contains(doJogador, vazamento) {
			t.Errorf("a cortina deixou passar %q", vazamento)
		}
	}

	// O CONTROLE: o mestre continua vendo a cena inteira, senão "a mesa não viu"
	// seria verdade também num tabuleiro que ninguém abriu.
	doMestre := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(doMestre, "Dragão") || !strings.Contains(doMestre, "tabuleiro-plano") {
		t.Error("o mestre perdeu a própria cena com a cortina fechada")
	}
	// E ele é AVISADO. Sem a tira, o mapa dele fica igualzinho com a cortina
	// aberta ou fechada, e ele narra a taverna para uma mesa que vê um aviso.
	if !strings.Contains(doMestre, "a mesa não vê esta cena") {
		t.Error("o mestre não foi avisado de que a cortina está fechada")
	}
}

// A PEÇA AVULSA NASCE ONDE O MESTRE CLICOU (ALE-291).
//
// O GLOSSARIO promete que "uma peça pode existir sem linha na fila (a porta, o
// baú)", e até aqui não havia caminho: a única rota que criava peça era o
// `poeNoMapa`, cujo `populateBoard` itera a INICIATIVA — só nascia peça para
// quem já era combatente.
//
// A asserção é sobre a CASA e não só sobre a existência: uma peça que nasce no
// lugar errado é pior que nenhuma, porque o mestre põe a porta e ela aparece do
// outro lado da cripta.
func TestALoosePieceIsBornOnTheSquareTheGmClicked(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "cripta")

	corpo := f.posta(t, f.mestre, f.tableUrl()+"/tabuleiro/pecas/nova/-3/7",
		`{"novapecanome":"  Porta da cripta  ","novapecatamanho":1,"novapecaaparencia":"object"}`)

	mapa := f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)
	if len(mapa.Tokens) != 1 {
		t.Fatalf("o mapa ficou com %d peças; a resposta foi:\n%s", len(mapa.Tokens), firstRows(corpo, 6))
	}
	peca := mapa.Tokens[0]
	// COORDENADA NEGATIVA é lugar legítimo — o plano não tem bordas —, e é por
	// isso que ela viaja no caminho. O -3 está aqui de propósito.
	if peca.X != -3 || peca.Y != 7 {
		t.Errorf("a peça nasceu em (%d,%d) e o clique foi em (-3,7)", peca.X, peca.Y)
	}
	// O nome vai APARADO, como o do combatente: espaço sobrando não deve
	// produzir uma peça que ordena e se lê diferente do que o mestre digitou.
	if peca.Label != "Porta da cripta" {
		t.Errorf("o rótulo ficou %q", peca.Label)
	}
	if peca.Kind != "object" {
		t.Errorf("a aparência ficou %q", peca.Kind)
	}
	// E ela NÃO tem linha na fila: é isso que a distingue do `poeNoMapa`, e é a
	// promessa do glossário.
	if peca.EntryID != nil {
		t.Errorf("a peça avulsa nasceu ligada à fila: %v", *peca.EntryID)
	}
}

// A TIRA RECUSA o que não desenha peça nenhuma, e a recusa nomeia o valor.
//
// Os três casos são de naturezas diferentes de propósito: o nome vazio é o que
// deixaria a peça muda no mapa e no leitor de tela; o tamanho fora da Tabela
// 1-21 (p107) desenharia uma criatura que o livro não tem; e a aparência
// `character` é a que criaria uma peça que PARECE de jogador sem ninguém atrás.
func TestTheLoosePieceRefusesWhatDrawsNoPiece(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "cripta")
	casos := []struct{ nome, sinais, espera string }{
		{"sem nome", `{"novapecanome":"   ","novapecatamanho":1,"novapecaaparencia":"object"}`, "dê um nome"},
		{"tamanho de nada", `{"novapecanome":"Carroça","novapecatamanho":4,"novapecaaparencia":"object"}`, "p107"},
		{"ficha solta", `{"novapecanome":"Falso herói","novapecatamanho":1,"novapecaaparencia":"character"}`, "aparência"},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			corpo := f.posta(t, f.mestre, f.tableUrl()+"/tabuleiro/pecas/nova/1/1", caso.sinais)
			if !strings.Contains(corpo, caso.espera) {
				t.Errorf("a recusa não citou %q; a resposta foi:\n%s", caso.espera, firstRows(corpo, 6))
			}
		})
	}
	// O CONTROLE: nenhuma das três recusas pode ter escrito. Sem ele, uma recusa
	// que já tivesse criado a peça passaria pelas asserções acima.
	if mapa := f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab); len(mapa.Tokens) != 0 {
		t.Errorf("as recusas deixaram %d peças no mapa", len(mapa.Tokens))
	}
}

// PÔR PEÇA É DO MESTRE, e a trava é o servidor.
//
// Esconder o botão do jogador é UX; quem impede é o handler. O caso posta na
// mão, como quem abre o console — é o que a ALE-144 registrou ao tirar três
// asserções de AUSÊNCIA da suíte: botão ausente nunca foi prova de trava.
func TestOnlyTheGmPutsALoosePieceOnTheMap(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "cripta")

	rec := f.pede(t, f.jogador, "POST", f.tableUrl()+"/tabuleiro/pecas/nova/1/1",
		`{"novapecanome":"Porta","novapecatamanho":1,"novapecaaparencia":"object"}`)

	if rec.Code != http.StatusForbidden {
		t.Errorf("o jogador pôs peça e levou %d, queria 403", rec.Code)
	}
	if mapa := f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab); len(mapa.Tokens) != 0 {
		t.Errorf("a recusa deixou %d peças no mapa", len(mapa.Tokens))
	}
}

// O MODO da peça avulsa é do MESTRE, e ele aparece FORA da fileira numerada
// (ALE-291).
//
// As duas metades se medem juntas: o botão sem a tira seria um modo que liga e
// não diz o que vai criar, e a tira sem o botão seria um formulário sem gesto.
//
// E o caso afirma que ele NÃO ganhou número, que é a decisão de desenho desta
// fatia: o `railKeys` tem dez dígitos e o `numberRail` estoura na décima
// primeira de propósito. Um dia alguém vai querer dar uma tecla a este modo —
// que ele mude este caso junto, em vez de a gramática dos dez virar dez-e-meio
// em silêncio.
func TestTheNewPieceModeBelongsToTheGmAndHasNoNumber(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "cripta")

	doMestre := f.pede(t, f.mestre, "GET", f.tableUrl(), "").Body.String()
	for _, pedaco := range []string{
		"Nova peça — o clique escolhe a casa",       // o botão do modo
		"Nova peça — escolha a casa onde ela nasce", // a camada de clique
		"novapecanome", // a tira
		"Colossal",     // o tamanho do livro (p107)
	} {
		if !strings.Contains(doMestre, pedaco) {
			t.Errorf("o mestre não recebeu %q", pedaco)
		}
	}
	// As dez ferramentas continuam sendo dez, e o modo não entrou na conta.
	if n := len(table.MapTools()); n != 10 {
		t.Errorf("o trilho numerado ficou com %d ferramentas — a gramática dos dez dígitos quebrou", n)
	}

	// O QUE NÃO PODE VAZAR É A AFORDÂNCIA, e não o sinal.
	//
	// A primeira versão deste caso cobrava também a ausência de `novapecanome`, e
	// ela reprovou: a lista de sinais é da PÁGINA e sai igual para os dois
	// papéis, como já acontece com `pecaescolhida`, `rascunhode` e os outros do
	// mestre. Um sinal sem escritor é inerte; o que conta é não haver botão nem
	// camada de clique — e a trava de verdade é o 403 do handler, medido no caso
	// vizinho (ALE-144).
	doJogador := f.pede(t, f.jogador, "GET", f.tableUrl(), "").Body.String()
	if strings.Contains(doJogador, "Nova peça") {
		t.Error("o gesto de criar peça vazou para o HTML do jogador")
	}
}
