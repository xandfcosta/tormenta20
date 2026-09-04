package api

import (
	"context"
	"net/http"
	"strings"
	"t20engine/tabuleiro"
	"testing"
)

func (f pilotoFixture) onBoard(t *testing.T) string {
	t.Helper()
	f.seedOpenBoard(t, "pedra")
	entryID := f.tracker(t)
	posto, err := f.s.Boards().AddToken(context.Background(), f.sessionID, defaultTab,
		tabuleiro.BoardToken{Label: "Arcanista", X: 0, Y: 0, EntryID: &entryID, CharacterID: &f.charID}, true)
	if err != nil {
		t.Fatalf("pôr a peça: %v", err)
	}
	return posto.Tokens[len(posto.Tokens)-1].ID
}

// TestTheStopsAccumulateInsteadOfReplacingEachOther — o coração desta fatia.
//
// Uma parada por clique, e o caminho ESTENDE. Se cada clique recomeçasse do
// lugar da peça, o contorno seria impossível de expressar — que é exatamente o
// defeito da SPA que a ALE-266 abriu.
func TestTheStopsAccumulateInsteadOfReplacingEachOther(t *testing.T) {
	f := novoPiloto(t)
	tokenID := f.onBoard(t)
	base := f.tableUrl() + "/tabuleiro/" + tokenID

	// O mestre move sem orçamento, então ele serve para medir o acúmulo sem a
	// regra da vez entrar no meio.
	if rec := f.pede(t, f.mestre, "POST", base+"/parada/2/0", ""); rec.Code != http.StatusOK {
		t.Fatalf("primeira parada deu %d", rec.Code)
	}
	primeiro := f.s.Boards().Get(context.Background(), f.sessionID, defaultTab).Pending
	if primeiro == nil || len(primeiro.Path) != 3 {
		t.Fatalf("o primeiro caminho ficou %+v", primeiro)
	}

	if rec := f.pede(t, f.mestre, "POST", base+"/parada/2/2", ""); rec.Code != http.StatusOK {
		t.Fatalf("segunda parada deu %d", rec.Code)
	}
	depois := f.s.Boards().Get(context.Background(), f.sessionID, defaultTab).Pending
	if depois == nil {
		t.Fatal("o movimento sumiu na segunda parada")
	}
	// (0,0)→(2,0) são 3 quadrados; mais (2,1),(2,2) dão 5. Substituir daria 3.
	if len(depois.Path) != 5 {
		t.Errorf("o caminho ficou com %d quadrados: %+v — a segunda parada substituiu em vez de estender", len(depois.Path), depois.Path)
	}
	if fim := depois.Path[len(depois.Path)-1]; fim.X != 2 || fim.Y != 2 {
		t.Errorf("o caminho termina em %+v", fim)
	}
}

// TestTheMoveOnlyLandsOnConfirm: a peça não anda enquanto o movimento é
// proposta. É o que deixa a pessoa contornar em vários cliques sem a mesa ver a
// peça pulando de casa em casa.
func TestTheMoveOnlyLandsOnConfirm(t *testing.T) {
	f := novoPiloto(t)
	tokenID := f.onBoard(t)
	base := f.tableUrl() + "/tabuleiro/" + tokenID
	onde := func() (int, int) {
		p := tabuleiro.FindToken(f.s.Boards().Get(context.Background(), f.sessionID, defaultTab), tokenID)
		return p.X, p.Y
	}

	if rec := f.pede(t, f.mestre, "POST", base+"/parada/3/1", ""); rec.Code != http.StatusOK {
		t.Fatalf("propor deu %d", rec.Code)
	}
	if x, y := onde(); x != 0 || y != 0 {
		t.Errorf("a peça andou na PROPOSTA, para %d,%d", x, y)
	}

	if rec := f.pede(t, f.mestre, "POST", base+"/confirmar", ""); rec.Code != http.StatusOK {
		t.Fatalf("confirmar deu %d", rec.Code)
	}
	if x, y := onde(); x != 3 || y != 1 {
		t.Errorf("depois de confirmar a peça está em %d,%d", x, y)
	}
	if f.s.Boards().Get(context.Background(), f.sessionID, defaultTab).Pending != nil {
		t.Error("o movimento continuou pendente depois de confirmado")
	}
}

// E CANCELAR não mexe na peça: ela volta a poder ser movida de onde estava.
func TestCancelDoesNotTouchTheToken(t *testing.T) {
	f := novoPiloto(t)
	tokenID := f.onBoard(t)
	base := f.tableUrl() + "/tabuleiro/" + tokenID

	if rec := f.pede(t, f.mestre, "POST", base+"/parada/4/4", ""); rec.Code != http.StatusOK {
		t.Fatalf("propor deu %d", rec.Code)
	}
	if rec := f.pede(t, f.mestre, "POST", base+"/cancelar", ""); rec.Code != http.StatusOK {
		t.Fatalf("cancelar deu %d", rec.Code)
	}

	b := f.s.Boards().Get(context.Background(), f.sessionID, defaultTab)
	if b.Pending != nil {
		t.Error("o cancelamento não limpou a proposta")
	}
	if p := tabuleiro.FindToken(b, tokenID); p.X != 0 || p.Y != 0 {
		t.Errorf("a peça ficou em %d,%d depois do cancelamento", p.X, p.Y)
	}
}

// TestThePlayerDoesNotMoveSomeoneElsesToken — a autorização é do `tabuleiro`, e a recusa
// vem com a FRASE que a regra escreve.
//
// Não é 403: quem chega aqui é da mesa e podia estar movendo a própria peça. A
// diferença importa porque a frase é o que a pessoa lê — "a peça não é sua" diz
// o que fazer, e "proibido" não.
func TestThePlayerDoesNotMoveSomeoneElsesToken(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")
	posto, err := f.s.Boards().AddToken(context.Background(), f.sessionID, defaultTab,
		tabuleiro.BoardToken{Label: "Ogro", X: 5, Y: 5}, true)
	if err != nil {
		t.Fatalf("pôr o Ogro: %v", err)
	}
	ogro := posto.Tokens[len(posto.Tokens)-1].ID

	corpo := f.pede(t, f.jogador, "POST",
		f.tableUrl()+"/tabuleiro/"+ogro+"/parada/6/5", "").Body.String()
	if !strings.Contains(corpo, "não é sua") {
		t.Errorf("a recusa não explica de quem é a peça; sinais = %s", trechoDeSinais(corpo))
	}
	if f.s.Boards().Get(context.Background(), f.sessionID, defaultTab).Pending != nil {
		t.Error("o movimento recusado virou proposta mesmo assim")
	}
}

// TestTheReachOnlyShowsWhenThereIsABudget.
//
// Quem tem teto é o jogador NA VEZ dele. O mestre move sem orçamento (-1), e
// desenhar alcance para ele seria inventar um limite que a regra não põe — foi
// isto que fez a casa alcançável deixar de ser o alvo do clique e virar pintura.
func TestTheReachOnlyShowsWhenThereIsABudget(t *testing.T) {
	f := novoPiloto(t)
	f.onBoard(t)
	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/scene/start", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar cena deu %d", rec.Code)
	}
	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/initiative/next-turn", ""); rec.Code != http.StatusOK {
		t.Fatalf("avançar deu %d", rec.Code)
	}

	// O CONTROLE: o jogador vê o tabuleiro. Sem isto, "não achei alcance" seria
	// verdade também numa cena sem mapa.
	doJogador := f.pede(t, f.jogador, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(doJogador, "tabuleiro-plano") {
		t.Fatal("o jogador não viu o tabuleiro")
	}
	if !strings.Contains(doJogador, "tabuleiro-alcance") {
		t.Error("é a vez do jogador e ele não viu até onde pode andar")
	}
	// AS DUAS FAIXAS (T20 p233): ouro é o que a ação de movimento alcança, azul é
	// o que só se alcança gastando a ação padrão junto. É a resposta de relance à
	// pergunta do dono — "se ele precisa gastar a ação de movimento e a ação
	// principal" — sem desenhar caminho nenhum.
	if !strings.Contains(doJogador, "tabuleiro-alcance-segundo") {
		t.Error("o jogador não viu até onde chega gastando a ação principal também")
	}

	// O MESTRE VÊ O MESMO SOMBREADO (decisão do dono: "o mestre não tem limite,
	// mas a parte visual serve para todos"). Ele não é barrado por ele — a trava
	// saiu do servidor —, e esconder as faixas dele tiraria da pessoa que decide
	// exatamente o que a mesa está lendo.
	doMestre := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(doMestre, "tabuleiro-alcance-segundo") {
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
	f := novoPiloto(t)
	f.onBoard(t)

	for quem, quemChama := range map[string]int64{"jogador": f.jogador, "mestre": f.mestre} {
		tela := f.pede(t, quemChama, http.MethodGet, f.tableUrl(), "").Body.String()
		if !strings.Contains(tela, "tabuleiro-plano") {
			t.Fatalf("o %s não viu o tabuleiro: a ausência abaixo não é evidência", quem)
		}
		if strings.Contains(tela, "tabuleiro-alcance") {
			t.Errorf("fora de combate o %s viu um teto desenhado", quem)
		}
	}
}

// TestARefusedStopSpeaksOnTheBoard.
//
// O arrasto (ALE-264) quebrou a invariante em que este arquivo se apoiava: com
// CLIQUE só se acerta casa oferecida, mas soltar acontece onde o dedo estiver,
// inclusive fora do alcance. A recusa passou a ser alcançável de verdade — e
// ela saía em `erroDoComando`, que é o sinal do RODAPÉ DO MESTRE. O jogador não
// renderiza rodapé nenhum: a frase existia no fio e não tinha onde pousar, e a
// parada era engolida em silêncio.
//
// A ALE-203 mudou QUAL comando recusa — a parada cara passou a ser aceita e
// desenhada, e quem barra é o confirmar —, e não mudou nada do que este guarda
// prende: a frase continua tendo de sair no sinal do MOVIMENTO e a região do
// jogador continua tendo de ter onde acendê-la. Trocar o gatilho e manter as
// duas asserções é o que separa "o guarda ainda mede" de "o guarda ficou verde".
//
// Prende as DUAS metades, porque uma sem a outra não é o conserto: que a frase
// sai no sinal certo, e que a região do tabuleiro tem onde acendê-la.
func TestARefusedStopSpeaksOnTheBoard(t *testing.T) {
	f := novoPiloto(t)
	tokenID := f.onBoard(t)
	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/scene/start", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar cena deu %d", rec.Code)
	}
	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/initiative/next-turn", ""); rec.Code != http.StatusOK {
		t.Fatalf("avançar deu %d", rec.Code)
	}
	base := f.tableUrl() + "/tabuleiro/" + tokenID

	// O CANAL: a região do tabuleiro tem o elemento ligado ao sinal. Sem esta
	// asserção, "a frase saiu" seria verdade sobre uma tela que não a mostra —
	// que é exatamente o defeito que este guarda existe para pegar.
	doJogador := f.pede(t, f.jogador, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(doJogador, "$erroDoMovimento") {
		t.Fatal("o tabuleiro do jogador não tem onde acender a recusa de uma parada")
	}

	// O deslocamento padrão são 6 quadrados (T20 p106); 9 não cabem — e desde a
	// ALE-203 a PARADA os aceita, porque é o desenho que conta à pessoa onde ela
	// estourou. Quem recusa é o CONFIRMAR, e é a recusa dele que precisa pousar
	// aqui.
	proposta := f.pede(t, f.jogador, "POST", base+"/parada/9/0", "")
	if proposta.Code != http.StatusOK {
		t.Fatalf("a parada cara deu %d: sem provisório não há trecho vermelho para desenhar", proposta.Code)
	}
	if sinais := trechoDeSinais(proposta.Body.String()); !strings.Contains(sinais, `"erroDoMovimento":""`) {
		t.Errorf("a parada cara acendeu uma recusa que já não é dela; sinais = %s", sinais)
	}

	recusado := f.pede(t, f.jogador, "POST", base+"/confirmar", "").Body.String()
	sinais := trechoDeSinais(recusado)
	if !strings.Contains(sinais, "erroDoMovimento") {
		t.Errorf("a recusa não saiu no sinal do movimento; sinais = %s", sinais)
	}
	if strings.Contains(sinais, `"erroDoMovimento":""`) {
		t.Errorf("a recusa saiu VAZIA — o confirmar foi engolido em silêncio; sinais = %s", sinais)
	}

	// E APAGA no acerto: um sinal que só se escreve quando dá errado deixa a
	// recusa de duas paradas atrás acesa sobre uma que funcionou.
	aceito := f.pede(t, f.jogador, "POST", base+"/parada/2/0", "").Body.String()
	if !strings.Contains(trechoDeSinais(aceito), `"erroDoMovimento":""`) {
		t.Errorf("a parada válida não apagou a recusa anterior; sinais = %s", trechoDeSinais(aceito))
	}
}

// TestWhatIsLeftOfTheDisplacementAppearsInWriting.
//
// A realimentação que o dono pediu por nome: sem ela a pessoa empilha paradas
// que no fim somam mais do que ela anda, e descobre no bloqueio sem saber o que
// desfazer. O alcance desenhado é o aviso mudo; este número é o falado.
//
// Guarda também a CONTA, que estava sem dono: `Alcance` e `Restante` são os dois
// valores de UMA chamada de `reachAndTarget`, e enquanto ninguém
// afirmava o segundo dava para movê-lo de lugar sem nenhum teste piscar.
func TestWhatIsLeftOfTheDisplacementAppearsInWriting(t *testing.T) {
	f := novoPiloto(t)
	tokenID := f.onBoard(t)
	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/scene/start", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar cena deu %d", rec.Code)
	}
	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/initiative/next-turn", ""); rec.Code != http.StatusOK {
		t.Fatalf("avançar deu %d", rec.Code)
	}

	// Duas casas em linha reta custam 2 do deslocamento padrão de 6 (T20 p106).
	if rec := f.pede(t, f.jogador, "POST", f.tableUrl()+"/tabuleiro/"+tokenID+"/parada/2/0", ""); rec.Code != http.StatusOK {
		t.Fatalf("a parada deu %d", rec.Code)
	}
	tela := f.pede(t, f.jogador, http.MethodGet, f.tableUrl(), "").Body.String()

	// O CONTROLE: a frase do movimento está na tela. Sem ele, não achar "sobram"
	// seria verdade também numa tela sem movimento proposto nenhum.
	if !strings.Contains(tela, "quadrados") {
		t.Fatal("o jogador não viu o movimento proposto")
	}
	if !strings.Contains(tela, "2 de 6") {
		t.Errorf("a tela não diz o gasto contra o teto")
	}
	if !strings.Contains(tela, "sobram 4") {
		t.Errorf("a tela não diz quanto ainda dá para andar")
	}
}

// turnPlayer põe a cena em combate e passa a vez para o jogador, que é a
// única condição em que existe DESLOCAMENTO para estourar: o mestre tem
// orçamento -1 e nunca vê vermelho.
func (f pilotoFixture) turnPlayer(t *testing.T) {
	t.Helper()
	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/scene/start", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar cena deu %d", rec.Code)
	}
	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/initiative/next-turn", ""); rec.Code != http.StatusOK {
		t.Fatalf("avançar deu %d", rec.Code)
	}
}

// TestTheArrowComesOutInTwoColorsWhenThePathOverruns (ALE-203, item 13).
//
// A COMPOSIÇÃO, que é o que nenhum dos guardas de unidade alcança: que o caminho
// caro chega até o HTML do JOGADOR com o trecho vermelho desenhado, com a ponta
// da cor dele, e com os metros de cada perna escritos por cima.
//
// Vale pela porta de verdade — a parada, o mesmo POST que o dedo faz — porque o
// desenho só existe se o `ProposeMove` tiver ACEITADO o caminho caro. Chamar o
// `moveWires` direto provaria a aritmética sobre uma proposta que a cena
// talvez recusasse.
func TestTheArrowComesOutInTwoColorsWhenThePathOverruns(t *testing.T) {
	f := novoPiloto(t)
	tokenID := f.onBoard(t)
	f.turnPlayer(t)

	// O deslocamento padrão são 6 quadrados (T20 p106); nove para o leste custam 9.
	if rec := f.pede(t, f.jogador, "POST", f.tableUrl()+"/tabuleiro/"+tokenID+"/parada/9/0", ""); rec.Code != http.StatusOK {
		t.Fatalf("a parada cara deu %d: sem provisório não há seta para pintar", rec.Code)
	}
	tela := f.pede(t, f.jogador, http.MethodGet, f.tableUrl(), "").Body.String()

	if !strings.Contains(tela, "tabuleiro-movimento-segundo") {
		t.Error("o caminho passou da ação de movimento e a seta saiu inteira dourada")
	}
	if !strings.Contains(tela, "url(#tabuleiro-ponta-do-segundo)") {
		t.Error("o trecho azul saiu sem ponta, ou com a ponta dourada")
	}
	// NOVE cabe em DOZE: passa da ação de movimento e ainda cabe na ação padrão
	// trocada por ela (p233). Nada de vermelho — o vermelho é só o que não cabe
	// no turno, e confundir os dois apagaria a distinção que o dono pediu.
	if strings.Contains(tela, "tabuleiro-movimento-alem") {
		t.Error("nove quadrados sobre um deslocamento de seis pintaram vermelho: eles cabem em duas ações")
	}
	if !strings.Contains(tela, "ação de movimento + ação principal") {
		t.Error("o rodapé não nomeia as ações que o caminho gasta")
	}
	// A seta tem UMA ponta: com trecho vermelho o dourado termina no MEIO do
	// plano, e uma ponta ali pareceria um segundo destino.
	if !strings.Contains(tela, `marker-end="none"`) {
		t.Error("o fio dourado ficou com ponta no meio do caminho")
	}
	// O corte no centro de (6,0): o deslocamento paga seis casas para o leste.
	if !strings.Contains(tela, `d="M 0.5 0.5 L 6.5 0.5"`) {
		t.Error("o dourado não parou onde o deslocamento acaba")
	}
	// Os METROS da perna, e o rodapé contando a mesma história: nove quadrados
	// são 13,5m, e passam três (4,5m) do deslocamento.
	if !strings.Contains(tela, ">13,5m<") {
		t.Error("a seta não diz a distância da perna em metros")
	}
	// E o rodapé NÃO repete a conta em metros: com dois limiares, "4,5m além do
	// deslocamento" ficou ambíguo — além de qual dos dois? — e dizia a mesma coisa
	// que a frase das ações ao lado. O metro por perna continua sobre a seta, que
	// é onde ele explica a cor.
	if strings.Contains(tela, "além do deslocamento") {
		t.Error("o rodapé voltou a medir o excesso em metros, ao lado da frase que já o nomeia")
	}
}

// TestTheControlForTheTwoColorArrow: o caminho que CABE sai inteiro dourado.
//
// Sem ele, "a tela tem `tabuleiro-movimento-alem`" não se distingue de "a tela
// tem sempre", e o vermelho poderia aparecer em todo movimento sem nenhum guarda
// reclamar. O mesmo jogador, na mesma vez, com um caminho que o deslocamento
// paga.
func TestTheControlForTheTwoColorArrow(t *testing.T) {
	f := novoPiloto(t)
	tokenID := f.onBoard(t)
	f.turnPlayer(t)

	if rec := f.pede(t, f.jogador, "POST", f.tableUrl()+"/tabuleiro/"+tokenID+"/parada/4/0", ""); rec.Code != http.StatusOK {
		t.Fatalf("a parada que cabe deu %d", rec.Code)
	}
	tela := f.pede(t, f.jogador, http.MethodGet, f.tableUrl(), "").Body.String()

	if strings.Contains(tela, "tabuleiro-movimento-alem") {
		t.Error("quatro quadrados sobre um deslocamento de seis pintaram vermelho")
	}
	if strings.Contains(tela, "além do deslocamento") {
		t.Error("o rodapé falou de excesso num caminho que cabe")
	}
	// O CANAL continua aberto: a seta e o rótulo em metros saem do mesmo jeito,
	// senão "não achei vermelho" seria verdade sobre uma cena sem seta nenhuma.
	if !strings.Contains(tela, "tabuleiro-movimento-fio") {
		t.Fatal("não há seta na tela: a ausência de vermelho não prova nada")
	}
	if !strings.Contains(tela, ">6,0m<") {
		t.Error("a seta que cabe não diz a distância da perna em metros")
	}
}
