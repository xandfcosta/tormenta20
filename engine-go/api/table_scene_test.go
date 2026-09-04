package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestTheGmOpensTheSceneThroughTheDialog(t *testing.T) {
	f := novoPiloto(t)

	// O CONTROLE: não há tabuleiro antes. Sem ele, "o lugar é a Taverna" seria
	// verdade também sobre uma cena que já estava aberta desde a fixture.
	if f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab) != nil {
		t.Fatal("a sessão já nasceu com tabuleiro — o guarda mediria a cena errada")
	}

	rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/tabuleiro/abrir",
		`{"novolugar":"Taverna do Javali","novochao":"taverna"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("abrir deu %d", rec.Code)
	}
	b := f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)
	if b == nil {
		t.Fatal("o tabuleiro não abriu")
	}
	if b.Place != "Taverna do Javali" {
		t.Errorf("o lugar ficou %q — o sinal do diálogo não chegou", b.Place)
	}
	if b.Terrain != "taverna" {
		t.Errorf("o chão ficou %q — o mestre escolheu taverna", b.Terrain)
	}

	// O formulário volta ao zero: sem isto a cena seguinte nasce com o nome da
	// anterior, e no fim da noite ninguém confere o campo antes de clicar.
	if !strings.Contains(rec.Body.String(), `"novolugar":""`) {
		t.Errorf("o campo do lugar não foi limpo; sinais = %s", trechoDeSinais(rec.Body.String()))
	}
}

// TestABlankPlaceBecomesASceneAndAnUnknownGroundFallsBackToTheDefault:
// lugar em branco VIRA cena, e chão desconhecido cai no padrão.
//
// Os dois defaults são a mesma decisão: o mestre que só quer a grade não deve
// ser barrado por um campo, e um chão que a tela não oferece só chega por posse
// do fio — a resposta a isso é desenhar pedra, não discutir.
func TestABlankPlaceBecomesASceneAndAnUnknownGroundFallsBackToTheDefault(t *testing.T) {
	f := novoPiloto(t)
	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/tabuleiro/abrir",
		`{"novolugar":"   ","novochao":"lava"}`); rec.Code != http.StatusOK {
		t.Fatalf("abrir deu %d", rec.Code)
	}
	b := f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)
	if b == nil {
		t.Fatal("o tabuleiro não abriu")
	}
	if b.Place != "Cena" {
		t.Errorf("o lugar em branco virou %q", b.Place)
	}
	if b.Terrain != "pedra" {
		t.Errorf("o chão inventado virou %q — devia cair no padrão", b.Terrain)
	}
}

// TestOnlyTheGmBuildsAndTearsDownTheScene: a trava é do servidor.
//
// O botão escondido é cortesia para quem não pode; quem postar na mão leva 403.
// Este guarda é de HANDLER e não uma asserção de que o botão sumiu, porque a
// fronteira de segurança é o servidor e a tela é UX.
func TestOnlyTheGmBuildsAndTearsDownTheScene(t *testing.T) {
	f := novoPiloto(t)
	if rec := f.pede(t, f.jogador, "POST", f.tableUrl()+"/tabuleiro/abrir",
		`{"novolugar":"Cripta","novochao":"cripta"}`); rec.Code != http.StatusForbidden {
		t.Errorf("o jogador abriu a cena: %d", rec.Code)
	}
	if f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab) != nil {
		t.Error("a cena do jogador abriu mesmo assim")
	}

	f.seedOpenBoard(t, "pedra")
	if rec := f.pede(t, f.jogador, "POST", f.tableUrl()+"/tabuleiro/encerrar", ""); rec.Code != http.StatusForbidden {
		t.Errorf("o jogador encerrou a cena: %d", rec.Code)
	}
	if f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab) == nil {
		t.Error("a cena sumiu quando o jogador mandou encerrar")
	}
}

// TestEndingTakesTheSceneOffTheTableAndStoresItInTheArchive.
//
// As duas metades juntas porque uma sem a outra não é o gesto: encerrar sem
// arquivar perde a noite de trabalho, e arquivar sem tirar deixa a mesa presa
// numa cena que já acabou.
func TestEndingTakesTheSceneOffTheTableAndStoresItInTheArchive(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "taverna")

	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/tabuleiro/encerrar", ""); rec.Code != http.StatusOK {
		t.Fatalf("encerrar deu %d", rec.Code)
	}
	if f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab) != nil {
		t.Error("a cena continuou na mesa depois de encerrada")
	}

	lugares := f.s.tableHost().Boards().Places(context.Background(), f.campaignID)
	achou := false
	for _, l := range lugares {
		if l.Name == "Taverna do Javali" {
			achou = true
		}
	}
	if !achou {
		t.Errorf("a cena encerrada não foi para o acervo; lá tem %d lugares", len(lugares))
	}
}

// TestAnEmptySceneSaysDifferentThingsToEachOfThem.
//
// Não é a mesma frase com um botão a mais: o jogador não tem o que fazer além de
// esperar, e o mestre tem. "O mestre abre quando a cena tiver lugar" DITO AO
// PRÓPRIO MESTRE é a tela mandando ele fazer o que ela não deixa — que foi
// exatamente o texto que o piloto carregou até esta fatia.
func TestAnEmptySceneSaysDifferentThingsToEachOfThem(t *testing.T) {
	f := novoPiloto(t)

	// O CONTROLE: os dois chegam na cena vazia. Sem ele, "o jogador não vê
	// 'Abrir tabuleiro'" seria verdade também sobre uma página que não carregou.
	doMestre := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()
	doJogador := f.pede(t, f.jogador, http.MethodGet, f.tableUrl(), "").Body.String()
	for quem, corpo := range map[string]string{"mestre": doMestre, "jogador": doJogador} {
		if !strings.Contains(corpo, "Nenhum tabuleiro aberto") && !strings.Contains(corpo, "ainda não abriu") {
			t.Fatalf("a cena vazia do %s não desenhou nada reconhecível", quem)
		}
	}

	if !strings.Contains(doMestre, "Abrir tabuleiro") {
		t.Error("o mestre não tem como abrir a cena")
	}
	if !strings.Contains(doMestre, "cena de interpretação") {
		t.Error("a frase do mestre não diz que o tabuleiro serve fora de combate")
	}
	if strings.Contains(doJogador, "Abrir tabuleiro") {
		t.Error("o jogador recebeu o gesto de abrir a cena")
	}
	if !strings.Contains(doJogador, "O mestre ainda não abriu um tabuleiro") {
		t.Error("a frase do jogador não diz de quem ele está esperando")
	}
}

// Os guardas do ACERVO de lugares (ALE-264, item 4).

// TestTheArchiveListsWhatWasEnded, com a contagem de peças.
//
// A contagem é o que separa a cena montada da cena aberta e abandonada, e é por
// ela que o mestre decide o que reabrir e o que apagar.
func TestTheArchiveListsWhatWasEnded(t *testing.T) {
	f := novoPiloto(t)
	f.onBoard(t) // abre a Taverna e põe UMA peça
	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/tabuleiro/encerrar", ""); rec.Code != http.StatusOK {
		t.Fatalf("encerrar deu %d", rec.Code)
	}

	tela := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(tela, "Lugares da campanha · 1") {
		t.Errorf("o acervo não apareceu com a cena encerrada")
	}
	// LITERAL e nunca `ui.TokenCount(1)`: o esperado derivado da produção
	// afirmaria o defeito junto com a regra, e foi assim que "1 peças" chegou à
	// tela na primeira medição.
	if !strings.Contains(tela, "1 peça") {
		t.Errorf("o acervo não diz quantas peças a cena guardada tem")
	}
	if strings.Contains(tela, "1 peças") {
		t.Errorf("a concordância quebrou: a linha diz \"1 peças\"")
	}

	// O acervo é do MESTRE: a mesa não escolhe onde joga.
	doJogador := f.pede(t, f.jogador, http.MethodGet, f.tableUrl(), "").Body.String()
	if strings.Contains(doJogador, "Lugares da campanha") {
		t.Error("o jogador recebeu o acervo da campanha")
	}
}

// TestWithoutAStoredPlaceThereIsNoArchiveButton.
//
// Um botão que abre uma lista vazia ensina que o acervo não serve para nada. É a
// metade que faz o guarda acima significar alguma coisa — sem ela, "o acervo
// apareceu" seria verdade sobre um botão que aparece sempre.
func TestWithoutAStoredPlaceThereIsNoArchiveButton(t *testing.T) {
	f := novoPiloto(t)
	tela := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(tela, "Abrir tabuleiro") {
		t.Fatal("a cena vazia do mestre não desenhou — o guarda mediria a tela errada")
	}
	if strings.Contains(tela, "Lugares da campanha") {
		t.Error("o acervo vazio ofereceu um menu que não tem o que mostrar")
	}
}

// TestReopeningAddsATabAndSwapsNothing (ALE-205, fatia 3).
//
// Aqui morava `TestReabrirTrocaACenaEGuardaAQueEstavaNaMesa`, que prendia a
// regra da ALE-191: reabrir ARQUIVAVA a cena que estava na mesa e entrava no
// lugar dela. Aquilo existia porque a sessão tinha UM tabuleiro — o
// arquivamento preventivo era o que impedia a taverna de se perder quando a
// cripta entrava.
//
// Com abas, o problema que ele resolvia deixou de existir: nada é substituído,
// então não há o que guardar antes. Um teste sobre a regra antiga ficaria verde
// afirmando um mundo que não é este.
func TestReopeningAddsATabAndSwapsNothing(t *testing.T) {
	f := novoPiloto(t)
	ctx := context.Background()
	f.seedOpenBoard(t, "taverna") // "Taverna do Javali"
	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/tabuleiro/encerrar", ""); rec.Code != http.StatusOK {
		t.Fatalf("encerrar a taverna deu %d", rec.Code)
	}
	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/tabuleiro/abrir",
		`{"novolugar":"Cripta","novochao":"cripta"}`); rec.Code != http.StatusOK {
		t.Fatalf("abrir a cripta deu %d", rec.Code)
	}

	taverna := int64(0)
	for _, l := range f.s.tableHost().Boards().Places(ctx, f.campaignID) {
		if l.Name == "Taverna do Javali" {
			taverna = l.ID
		}
	}
	if taverna == 0 {
		t.Fatal("a taverna não está no acervo — o guarda mediria a troca errada")
	}

	if rec := f.pede(t, f.mestre, "POST",
		fmt.Sprintf("%s/tabuleiro/lugares/%d/reabrir", f.tableUrl(), taverna), ""); rec.Code != http.StatusOK {
		t.Fatalf("reabrir deu %d", rec.Code)
	}

	abertos := f.s.tableHost().Boards().OpenBoards(ctx, f.sessionID)
	if len(abertos) != 2 {
		t.Fatalf("a sessão ficou com %d cenas abertas, esperado 2 (a cripta e a taverna)", len(abertos))
	}
	if abertos[0].Place != "Cripta" {
		t.Errorf("a cripta saiu da mesa quando a taverna entrou: %q", abertos[0].Place)
	}
	if abertos[1].Place != "Taverna do Javali" {
		t.Errorf("a taverna não entrou como aba nova: %q", abertos[1].Place)
	}
	// E ela NÃO foi arquivada, porque não foi tirada de lugar nenhum: a cripta
	// no acervo com a cripta na mesa seriam duas verdades sobre a mesma cena.
	for _, l := range f.s.tableHost().Boards().Places(ctx, f.campaignID) {
		if l.Name == "Cripta" {
			t.Error("a cena que continua aberta foi arquivada: o acervo passou a ter uma cópia dela")
		}
	}
	// Quem reabriu VAI para a aba nova — ele acabou de escolher aquele lugar
	// numa lista, e ficar na cena anterior faria o gesto parecer que não pegou.
	tela := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(tela, "Taverna do Javali</h2>") {
		t.Error("o mestre reabriu a taverna e continuou olhando a cripta")
	}
}

// TestDeletingAPlaceDoesNotTakeTheSceneOffTheTable.
//
// O `removeOLugar` devolve o tabuleiro ATUAL e não nil, e é por isso: nil faria
// o caminho publicar "não há tabuleiro" para a mesa inteira — o mestre limparia
// o acervo e a mesa perderia a cena em que estava jogando.
func TestDeletingAPlaceDoesNotTakeTheSceneOffTheTable(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "taverna")
	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/tabuleiro/encerrar", ""); rec.Code != http.StatusOK {
		t.Fatalf("encerrar deu %d", rec.Code)
	}
	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/tabuleiro/abrir",
		`{"novolugar":"Cripta","novochao":"cripta"}`); rec.Code != http.StatusOK {
		t.Fatalf("abrir a cripta deu %d", rec.Code)
	}
	guardados := f.s.tableHost().Boards().Places(context.Background(), f.campaignID)
	if len(guardados) == 0 {
		t.Fatal("o acervo está vazio — o guarda não teria o que apagar")
	}

	if rec := f.pede(t, f.mestre, "POST",
		fmt.Sprintf("%s/tabuleiro/lugares/%d/remover", f.tableUrl(), guardados[0].ID), ""); rec.Code != http.StatusOK {
		t.Fatalf("remover deu %d", rec.Code)
	}
	if b := f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab); b == nil {
		t.Error("apagar um lugar do acervo derrubou a cena que estava na mesa")
	} else if b.Place != "Cripta" {
		t.Errorf("a cena da mesa virou %q", b.Place)
	}
}

// TestOnlyTheGmTouchesTheArchive: a trava é do servidor.
func TestOnlyTheGmTouchesTheArchive(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "taverna")
	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/tabuleiro/encerrar", ""); rec.Code != http.StatusOK {
		t.Fatalf("encerrar deu %d", rec.Code)
	}
	guardados := f.s.tableHost().Boards().Places(context.Background(), f.campaignID)
	if len(guardados) == 0 {
		t.Fatal("o acervo está vazio — o guarda mediria uma rota sem alvo")
	}
	id := guardados[0].ID

	for _, acao := range []string{"reabrir", "remover"} {
		rec := f.pede(t, f.jogador, "POST",
			fmt.Sprintf("%s/tabuleiro/lugares/%d/%s", f.tableUrl(), id, acao), "")
		if rec.Code != http.StatusForbidden {
			t.Errorf("o jogador conseguiu %s um lugar: %d", acao, rec.Code)
		}
	}
	if len(f.s.tableHost().Boards().Places(context.Background(), f.campaignID)) != len(guardados) {
		t.Error("o acervo mudou de tamanho com o jogador mexendo nele")
	}
}

// TestAnEmptySceneInTheArchiveAnnouncesItselfAsSuch.
//
// "0 peças" descreve mal o que a linha é. Cena aberta e abandonada é justamente
// o que o mestre procura quando abre o acervo para limpar, e a linha tem de
// dizer isso em vez de fazer ele contar zeros.
func TestAnEmptySceneInTheArchiveAnnouncesItselfAsSuch(t *testing.T) {
	f := novoPiloto(t)
	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/tabuleiro/abrir",
		`{"novolugar":"Sala esquecida","novochao":"pedra"}`); rec.Code != http.StatusOK {
		t.Fatalf("abrir deu %d", rec.Code)
	}
	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/tabuleiro/encerrar", ""); rec.Code != http.StatusOK {
		t.Fatalf("encerrar deu %d", rec.Code)
	}

	tela := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()
	// O CONTROLE: a cena guardada está listada. Sem ele, não achar "0 peças"
	// seria verdade também sobre um acervo que não desenhou nada.
	if !strings.Contains(tela, "Sala esquecida") {
		t.Fatal("a cena encerrada não apareceu no acervo")
	}
	if strings.Contains(tela, "0 peças") {
		t.Error(`a linha diz "0 peças" em vez de dizer que a cena está vazia`)
	}
	if !strings.Contains(tela, "cena vazia") {
		t.Error("a cena sem peça nenhuma não se anuncia como vazia")
	}
}

/*
A FAIXA DE QUEM VEM DEPOIS na tela (ALE-290).

O `aovivo.UpcomingTurns` estava no ar desde a ALE-179 com cinco guardas e
nenhuma tela. Os casos de tradução moram no `web/table`; o que se prende aqui é
que ela CHEGA ao HTML dos dois papéis, e que a do jogador obedece à mesma
redação que o resto da Mesa.
*/

// A faixa desenha os três, e o jogador se lê como "você".
func TestTheTurnStripReachesBothScreens(t *testing.T) {
	f := novoPiloto(t)
	f.scene(t)
	// A CENA COMEÇADA não basta: o `StartScene` deixa o turno em -1, que é "em
	// cena, ninguém na vez ainda". A faixa só existe DENTRO do combate, e é o
	// avanço que o inicia — pela porta de verdade, que é o botão mais clicado da
	// sessão.
	//
	// DOIS avanços, e o segundo é o caso que a ALE-179 nomeia: a fila tem dois,
	// então o segundo turno é o ÚLTIMO da rodada, e "quem vem depois" está no
	// TOPO da lista. É o turno em que ler de cima para baixo não acha ninguém.
	f.posta(t, f.mestre, f.tableUrl()+"/initiative/next-turn", "")
	f.posta(t, f.mestre, f.tableUrl()+"/initiative/next-turn", "")

	doMestre := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()
	// ANCORADO NA VIRADA DA RODADA, que é texto que SÓ a faixa escreve: o nome
	// do Ogro aparece na lista de iniciativa de qualquer jeito, e procurá-lo
	// mediria a lista em vez da faixa.
	if !strings.Contains(doMestre, "começa a rodada 2") {
		t.Errorf("a faixa do mestre não marca onde a rodada vira")
	}
	// E ela DÁ A VOLTA: no último da rodada, o próximo é o primeiro da fila.
	// Sem esta linha o caso passaria com uma faixa que só desenha a vez atual.
	if !strings.Contains(doMestre, "Ogro cansado") {
		t.Errorf("a faixa não deu a volta: o primeiro da fila não aparece depois do último")
	}
	// O MESTRE não é personagem nenhum da fila, então nenhuma linha é dele.
	if strings.Contains(doMestre, ">você<") {
		t.Errorf("a faixa do mestre marcou uma linha como dele")
	}

	doJogador := f.pede(t, f.jogador, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(doJogador, ">você<") {
		t.Errorf("a faixa do jogador não marca o personagem dele como \"você\"")
	}
	// E ele continua vendo quem está na vez — a faixa é sobre ORDEM, e esconder
	// os outros a deixaria sem sentido.
	if !strings.Contains(doJogador, "Ogro cansado") {
		t.Errorf("a faixa do jogador não diz quem está na vez")
	}
}

// A faixa do jogador passa pela MESMA redação do resto da Mesa.
//
// Ela lê a fila, e a fila é o que o `StateForRole` redige. Um segundo caminho
// até os nomes seria um segundo lugar por onde vazar o que a mesa não deve ver —
// e este caso prende que ela usa o primeiro, e não um seu.
//
// O PV OCULTO é a prova concreta: o Ogro da cena tem PV escondidos, e eles não
// podem aparecer por causa da faixa nova.
func TestThePlayerTurnStripObeysTheSameRedaction(t *testing.T) {
	f := novoPiloto(t)
	f.scene(t)
	f.posta(t, f.mestre, f.tableUrl()+"/initiative/next-turn", "")
	f.posta(t, f.mestre, f.tableUrl()+"/initiative/next-turn", "")

	doJogador := f.pede(t, f.jogador, http.MethodGet, f.tableUrl(), "").Body.String()

	if strings.Contains(doJogador, "12/130") {
		t.Errorf("os PV que o mestre escondeu chegaram à tela do jogador")
	}
	// CONTROLE: o MESTRE vê os PV. Sem ele, um HTML que não trouxesse a fila
	// nenhuma passaria por "a redação funcionou".
	if doMestre := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String(); !strings.Contains(doMestre, "12") {
		t.Fatal("o mestre também não viu os PV — o guarda mediu uma tela vazia")
	}
}

// FORA DE COMBATE não há faixa, e a frase de espera continua.
//
// É o estado em que a maior parte de uma sessão vive, e uma faixa vazia com as
// setas soltas seria enfeite dizendo nada.
func TestOutOfCombatTheHeaderKeepsWaiting(t *testing.T) {
	f := novoPiloto(t)

	corpo := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()

	if !strings.Contains(corpo, "Aguardando iniciativa") {
		t.Error("fora de combate o cabeçalho não diz que espera a iniciativa")
	}
	if strings.Contains(corpo, "começa a rodada") {
		t.Error("desenhou a virada de rodada fora de combate")
	}
}
