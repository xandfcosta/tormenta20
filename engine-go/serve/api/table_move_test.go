package api

import (
	"context"
	"net/http"
	"strings"
	"t20engine/domain/board"
	"testing"
)

func (f sceneFixture) onBoard(t *testing.T) string {
	t.Helper()
	f.seedOpenBoard(t, "stone")
	entryID := f.tracker(t)
	placed, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab,
		board.BoardToken{Label: "Arcanista", X: 0, Y: 0, EntryID: &entryID, CharacterID: &f.charID})
	if err != nil {
		t.Fatalf("pôr a peça: %v", err)
	}
	return placed.Tokens[len(placed.Tokens)-1].ID
}

// Uma parada por clique, e o caminho ESTENDE. Se cada clique recomeçasse do
// lugar da peça, o contorno seria impossível de expressar.
func TestTheStopsAccumulateInsteadOfReplacingEachOther(t *testing.T) {
	f := newSceneFixture(t)
	tokenID := f.onBoard(t)
	base := f.tableUrl() + "/tabuleiro/" + tokenID

	// O mestre move sem orçamento, então ele serve para medir o acúmulo sem a
	// regra da vez entrar no meio.
	if rec := f.pede(t, f.gm, "POST", base+"/parada", `{"from":{"X":2,"Y":0}}`); rec.Code != http.StatusOK {
		t.Fatalf("primeira parada deu %d", rec.Code)
	}
	first := f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab).Pending
	if first == nil || len(first.Path) != 3 {
		t.Fatalf("o primeiro caminho ficou %+v", first)
	}

	if rec := f.pede(t, f.gm, "POST", base+"/parada", `{"from":{"X":2,"Y":2}}`); rec.Code != http.StatusOK {
		t.Fatalf("segunda parada deu %d", rec.Code)
	}
	after := f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab).Pending
	if after == nil {
		t.Fatal("o movimento sumiu na segunda parada")
	}
	// (0,0)→(2,0) são 3 quadrados; mais (2,1),(2,2) dão 5. Substituir daria 3.
	if len(after.Path) != 5 {
		t.Errorf("o caminho ficou com %d quadrados: %+v — a segunda parada substituiu em vez de estender", len(after.Path), after.Path)
	}
	if end := after.Path[len(after.Path)-1]; end.X != 2 || end.Y != 2 {
		t.Errorf("o caminho termina em %+v", end)
	}
}

// A peça não anda enquanto o movimento é proposta: é o que deixa a pessoa
// contornar em vários cliques sem a mesa ver a peça pulando de casa em casa.
func TestTheMoveOnlyLandsOnConfirm(t *testing.T) {
	f := newSceneFixture(t)
	tokenID := f.onBoard(t)
	base := f.tableUrl() + "/tabuleiro/" + tokenID
	where := func() (int, int) {
		p := board.FindToken(f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab), tokenID)
		return p.X, p.Y
	}

	if rec := f.pede(t, f.gm, "POST", base+"/parada", `{"from":{"X":3,"Y":1}}`); rec.Code != http.StatusOK {
		t.Fatalf("propor deu %d", rec.Code)
	}
	if x, y := where(); x != 0 || y != 0 {
		t.Errorf("a peça andou na PROPOSTA, para %d,%d", x, y)
	}

	if rec := f.pede(t, f.gm, "POST", base+"/confirmar", ""); rec.Code != http.StatusOK {
		t.Fatalf("confirmar deu %d", rec.Code)
	}
	if x, y := where(); x != 3 || y != 1 {
		t.Errorf("depois de confirmar a peça está em %d,%d", x, y)
	}
	if f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab).Pending != nil {
		t.Error("o movimento continuou pendente depois de confirmado")
	}
}

// E CANCELAR não mexe na peça: ela volta a poder ser movida de onde estava.
func TestCancelDoesNotTouchTheToken(t *testing.T) {
	f := newSceneFixture(t)
	tokenID := f.onBoard(t)
	base := f.tableUrl() + "/tabuleiro/" + tokenID

	if rec := f.pede(t, f.gm, "POST", base+"/parada", `{"from":{"X":4,"Y":4}}`); rec.Code != http.StatusOK {
		t.Fatalf("propor deu %d", rec.Code)
	}
	if rec := f.pede(t, f.gm, "POST", base+"/cancelar", ""); rec.Code != http.StatusOK {
		t.Fatalf("cancelar deu %d", rec.Code)
	}

	b := f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)
	if b.Pending != nil {
		t.Error("o cancelamento não limpou a proposta")
	}
	if p := board.FindToken(b, tokenID); p.X != 0 || p.Y != 0 {
		t.Errorf("a peça ficou em %d,%d depois do cancelamento", p.X, p.Y)
	}
}

// A autorização é do `tabuleiro`, e a recusa vem com a FRASE que a regra
// escreve.
//
// Não é 403: quem chega aqui é da mesa e podia estar movendo a própria peça. A
// diferença importa porque a frase é o que a pessoa lê — "a peça não é sua" diz
// o que fazer, e "proibido" não.
func TestThePlayerDoesNotMoveSomeoneElsesToken(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	placed, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab,
		board.BoardToken{Label: "Ogro", X: 5, Y: 5})
	if err != nil {
		t.Fatalf("pôr o Ogro: %v", err)
	}
	ogre := placed.Tokens[len(placed.Tokens)-1].ID

	body := f.pede(t, f.player, "POST",
		f.tableUrl()+"/tabuleiro/"+ogre+"/parada", `{"from":{"X":6,"Y":5}}`).Body.String()
	if !strings.Contains(body, "não é sua") {
		t.Errorf("a recusa não explica de quem é a peça; sinais = %s", trechoDeSinais(body))
	}
	if f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab).Pending != nil {
		t.Error("o movimento recusado virou proposta mesmo assim")
	}
}

// Quem tem teto é o jogador NA VEZ dele. O mestre move sem orçamento (-1), e
// desenhar alcance para ele seria inventar um limite que a regra não põe.
func TestTheReachOnlyShowsWhenThereIsABudget(t *testing.T) {
	f := newSceneFixture(t)
	f.onBoard(t)
	if rec := f.pede(t, f.gm, "POST", f.tableUrl()+"/cena/iniciar/acao", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar cena deu %d", rec.Code)
	}
	if rec := f.pede(t, f.gm, "POST", f.tableUrl()+"/iniciativa/proxima-vez", ""); rec.Code != http.StatusOK {
		t.Fatalf("avançar deu %d", rec.Code)
	}

	// O CONTROLE: o jogador vê o tabuleiro. Sem isto, "não achei alcance" seria
	// verdade também numa cena sem mapa.
	forPlayer := f.pede(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(forPlayer, "board-plane") {
		t.Fatal("o jogador não viu o tabuleiro")
	}
	if !strings.Contains(forPlayer, "board-range") {
		t.Error("é a vez do jogador e ele não viu até onde pode andar")
	}
	// AS DUAS FAIXAS (T20 p233): ouro é o que a ação de movimento alcança, azul é
	// o que só se alcança gastando a ação padrão junto.
	if !strings.Contains(forPlayer, "board-range-second") {
		t.Error("o jogador não viu até onde chega gastando a ação principal também")
	}

	// O MESTRE VÊ O MESMO SOMBREADO sem ser barrado por ele (decisão do dono):
	// esconder as faixas dele tiraria da pessoa que decide exatamente o que a
	// mesa está lendo.
	forGM := f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(forGM, "board-range-second") {
		t.Error("o mestre não viu as faixas de alcance da peça que ele move")
	}
}

// FORA DE COMBATE não há faixa nenhuma, para ninguém.
//
// O CONTROLE do guarda acima: sem ele, "o mestre viu o alcance" não se distingue
// de "o alcance é desenhado sempre", e a regra que o dono escolheu — sem vez não
// há ação padrão para trocar, então não há teto a desenhar — não estaria sendo
// medida por ninguém.
func TestOutOfCombatNobodySeesReach(t *testing.T) {
	f := newSceneFixture(t)
	f.onBoard(t)

	for who, caller := range map[string]int64{"jogador": f.player, "mestre": f.gm} {
		screen := f.pede(t, caller, http.MethodGet, f.tableUrl(), "").Body.String()
		if !strings.Contains(screen, "board-plane") {
			t.Fatalf("o %s não viu o tabuleiro: a ausência abaixo não é evidência", who)
		}
		if strings.Contains(screen, "board-range") {
			t.Errorf("fora de combate o %s viu um teto desenhado", who)
		}
	}
}

// A recusa é alcançável de verdade porque soltar acontece onde o dedo estiver,
// inclusive fora do alcance. Se ela sair em `command_error` — o sinal do RODAPÉ
// DO MESTRE —, a frase existe no fio e não tem onde pousar, porque o jogador não
// renderiza rodapé nenhum: a recusa é engolida em silêncio.
//
// Prende as DUAS metades, porque uma sem a outra não é o conserto: que a frase
// sai no sinal do MOVIMENTO, e que a região do tabuleiro tem onde acendê-la.
func TestARefusedStopSpeaksOnTheBoard(t *testing.T) {
	f := newSceneFixture(t)
	tokenID := f.onBoard(t)
	if rec := f.pede(t, f.gm, "POST", f.tableUrl()+"/cena/iniciar/acao", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar cena deu %d", rec.Code)
	}
	if rec := f.pede(t, f.gm, "POST", f.tableUrl()+"/iniciativa/proxima-vez", ""); rec.Code != http.StatusOK {
		t.Fatalf("avançar deu %d", rec.Code)
	}
	base := f.tableUrl() + "/tabuleiro/" + tokenID

	// O CANAL: a região do tabuleiro tem o elemento ligado ao sinal. Sem esta
	// asserção, "a frase saiu" seria verdade sobre uma tela que não a mostra —
	// que é exatamente o defeito que este guarda existe para pegar.
	forPlayer := f.pede(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(forPlayer, "$move_error") {
		t.Fatal("o tabuleiro do jogador não tem onde acender a recusa de uma parada")
	}

	// O deslocamento padrão são 6 quadrados (T20 p106); 9 não cabem — e a PARADA
	// os aceita de propósito, porque é o desenho que conta à pessoa onde ela
	// estourou. Quem recusa é o CONFIRMAR, e é a recusa dele que precisa pousar
	// aqui.
	proposal := f.pede(t, f.player, "POST", base+"/parada", `{"from":{"X":9,"Y":0}}`)
	if proposal.Code != http.StatusOK {
		t.Fatalf("a parada cara deu %d: sem provisório não há trecho vermelho para desenhar", proposal.Code)
	}
	if signals := trechoDeSinais(proposal.Body.String()); !strings.Contains(signals, `"move_error":""`) {
		t.Errorf("a parada cara acendeu uma recusa que já não é dela; sinais = %s", signals)
	}

	refused := f.pede(t, f.player, "POST", base+"/confirmar", "").Body.String()
	sig := trechoDeSinais(refused)
	if !strings.Contains(sig, "move_error") {
		t.Errorf("a recusa não saiu no sinal do movimento; sinais = %s", sig)
	}
	if strings.Contains(sig, `"move_error":""`) {
		t.Errorf("a recusa saiu VAZIA — o confirmar foi engolido em silêncio; sinais = %s", sig)
	}

	// E APAGA no acerto: um sinal que só se escreve quando dá errado deixa a
	// recusa de duas paradas atrás acesa sobre uma que funcionou.
	accepted := f.pede(t, f.player, "POST", base+"/parada", `{"from":{"X":2,"Y":0}}`).Body.String()
	if !strings.Contains(trechoDeSinais(accepted), `"move_error":""`) {
		t.Errorf("a parada válida não apagou a recusa anterior; sinais = %s", trechoDeSinais(accepted))
	}
}

// Sem o número escrito a pessoa empilha paradas que no fim somam mais do que ela
// anda, e descobre no bloqueio sem saber o que desfazer. O alcance desenhado é o
// aviso mudo; este número é o falado.
//
// Guarda também a CONTA: `Alcance` e `Restante` são os dois valores de UMA
// chamada de `reachAndTarget`, e sem ninguém afirmando o segundo dá para
// movê-lo de lugar sem nenhum teste piscar.
func TestWhatIsLeftOfTheDisplacementAppearsInWriting(t *testing.T) {
	f := newSceneFixture(t)
	tokenID := f.onBoard(t)
	if rec := f.pede(t, f.gm, "POST", f.tableUrl()+"/cena/iniciar/acao", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar cena deu %d", rec.Code)
	}
	if rec := f.pede(t, f.gm, "POST", f.tableUrl()+"/iniciativa/proxima-vez", ""); rec.Code != http.StatusOK {
		t.Fatalf("avançar deu %d", rec.Code)
	}

	// Duas casas em linha reta custam 2 do deslocamento padrão de 6 (T20 p106).
	if rec := f.pede(t, f.player, "POST", f.tableUrl()+"/tabuleiro/"+tokenID+"/parada", `{"from":{"X":2,"Y":0}}`); rec.Code != http.StatusOK {
		t.Fatalf("a parada deu %d", rec.Code)
	}
	screen := f.pede(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()

	// O CONTROLE: a frase do movimento está na tela. Sem ele, não achar "sobram"
	// seria verdade também numa tela sem movimento proposto nenhum.
	if !strings.Contains(screen, "quadrados") {
		t.Fatal("o jogador não viu o movimento proposto")
	}
	if !strings.Contains(screen, "2 de 6") {
		t.Errorf("a tela não diz o gasto contra o teto")
	}
	if !strings.Contains(screen, "sobram 4") {
		t.Errorf("a tela não diz quanto ainda dá para andar")
	}
}

// turnPlayer põe a cena em combate e passa a vez para o jogador, que é a
// única condição em que existe DESLOCAMENTO para estourar: o mestre tem
// orçamento -1 e nunca vê vermelho.
func (f sceneFixture) turnPlayer(t *testing.T) {
	t.Helper()
	if rec := f.pede(t, f.gm, "POST", f.tableUrl()+"/cena/iniciar/acao", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar cena deu %d", rec.Code)
	}
	if rec := f.pede(t, f.gm, "POST", f.tableUrl()+"/iniciativa/proxima-vez", ""); rec.Code != http.StatusOK {
		t.Fatalf("avançar deu %d", rec.Code)
	}
}

// A COMPOSIÇÃO, que é o que nenhum dos guardas de unidade alcança: que o caminho
// caro chega até o HTML do JOGADOR com o trecho vermelho desenhado, com a ponta
// da cor dele, e com os metros de cada perna escritos por cima.
//
// Vale pela porta de verdade — a parada, o mesmo POST que o dedo faz — porque o
// desenho só existe se o `ProposeMove` tiver ACEITADO o caminho caro. Chamar o
// `moveWires` direto provaria a aritmética sobre uma proposta que a cena
// talvez recusasse.
func TestTheArrowComesOutInTwoColorsWhenThePathOverruns(t *testing.T) {
	f := newSceneFixture(t)
	tokenID := f.onBoard(t)
	f.turnPlayer(t)

	// O deslocamento padrão são 6 quadrados (T20 p106); nove para o leste custam 9.
	if rec := f.pede(t, f.player, "POST", f.tableUrl()+"/tabuleiro/"+tokenID+"/parada", `{"from":{"X":9,"Y":0}}`); rec.Code != http.StatusOK {
		t.Fatalf("a parada cara deu %d: sem provisório não há seta para pintar", rec.Code)
	}
	screen := f.pede(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()

	if !strings.Contains(screen, "board-move-second") {
		t.Error("o caminho passou da ação de movimento e a seta saiu inteira dourada")
	}
	if !strings.Contains(screen, "url(#board-tip-second)") {
		t.Error("o trecho azul saiu sem ponta, ou com a ponta dourada")
	}
	// NOVE cabe em DOZE: passa da ação de movimento e ainda cabe na ação padrão
	// trocada por ela (p233). Nada de vermelho — o vermelho é só o que não cabe
	// no turno, e confundir os dois apagaria a distinção que o dono pediu.
	if strings.Contains(screen, "board-move-beyond") {
		t.Error("nove quadrados sobre um deslocamento de seis pintaram vermelho: eles cabem em duas ações")
	}
	if !strings.Contains(screen, "ação de movimento + ação principal") {
		t.Error("o rodapé não nomeia as ações que o caminho gasta")
	}
	// A seta tem UMA ponta: com trecho vermelho o dourado termina no MEIO do
	// plano, e uma ponta ali pareceria um segundo destino.
	if !strings.Contains(screen, `marker-end="none"`) {
		t.Error("o fio dourado ficou com ponta no meio do caminho")
	}
	// O corte no centro de (6,0): o deslocamento paga seis casas para o leste.
	if !strings.Contains(screen, `d="M 0.5 0.5 L 6.5 0.5"`) {
		t.Error("o dourado não parou onde o deslocamento acaba")
	}
	// Os METROS da perna, e o rodapé contando a mesma história: nove quadrados
	// são 13,5m, e passam três (4,5m) do deslocamento.
	if !strings.Contains(screen, ">13,5m<") {
		t.Error("a seta não diz a distância da perna em metros")
	}
	// E o rodapé NÃO repete a conta em metros: com dois limiares, "4,5m além do
	// deslocamento" é ambíguo — além de qual dos dois? — e diz a mesma coisa que a
	// frase das ações ao lado. O metro por perna fica sobre a seta, que é onde ele
	// explica a cor.
	if strings.Contains(screen, "além do deslocamento") {
		t.Error("o rodapé voltou a medir o excesso em metros, ao lado da frase que já o nomeia")
	}
}

// O CONTROLE da seta de duas cores: o caminho que CABE sai inteiro dourado.
//
// Sem ele, "a tela tem `board-move-beyond`" não se distingue de "a tela
// tem sempre", e o vermelho poderia aparecer em todo movimento sem nenhum guarda
// reclamar. O mesmo jogador, na mesma vez, com um caminho que o deslocamento
// paga.
func TestTheControlForTheTwoColorArrow(t *testing.T) {
	f := newSceneFixture(t)
	tokenID := f.onBoard(t)
	f.turnPlayer(t)

	if rec := f.pede(t, f.player, "POST", f.tableUrl()+"/tabuleiro/"+tokenID+"/parada", `{"from":{"X":4,"Y":0}}`); rec.Code != http.StatusOK {
		t.Fatalf("a parada que cabe deu %d", rec.Code)
	}
	screen := f.pede(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()

	if strings.Contains(screen, "board-move-beyond") {
		t.Error("quatro quadrados sobre um deslocamento de seis pintaram vermelho")
	}
	if strings.Contains(screen, "além do deslocamento") {
		t.Error("o rodapé falou de excesso num caminho que cabe")
	}
	// O CANAL continua aberto: a seta e o rótulo em metros saem do mesmo jeito,
	// senão "não achei vermelho" seria verdade sobre uma cena sem seta nenhuma.
	if !strings.Contains(screen, "board-move-arrow") {
		t.Fatal("não há seta na tela: a ausência de vermelho não prova nada")
	}
	if !strings.Contains(screen, ">6,0m<") {
		t.Error("a seta que cabe não diz a distância da perna em metros")
	}
}
