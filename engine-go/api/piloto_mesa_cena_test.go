package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// Os guardas de ABRIR e ENCERRAR a cena (ALE-264, item 3).
//
// O que se prende aqui é o CICLO, que é a composição que nenhum teste de regra
// alcança: o mestre abre pelo diálogo, o lugar e o chão que ele escolheu chegam
// ao tabuleiro, e encerrar tira a cena da mesa depois de arquivá-la.

// TestOMestreABREaCenaPeloDialogo: o lugar e o chão vêm dos SINAIS.
func TestOMestreAbreACenaPeloDialogo(t *testing.T) {
	f := novoPiloto(t)

	// O CONTROLE: não há tabuleiro antes. Sem ele, "o lugar é a Taverna" seria
	// verdade também sobre uma cena que já estava aberta desde a fixture.
	if f.s.boards.Get(context.Background(), f.sessionID) != nil {
		t.Fatal("a sessão já nasceu com tabuleiro — o guarda mediria a cena errada")
	}

	rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/tabuleiro/abrir",
		`{"novolugar":"Taverna do Javali","novochao":"taverna"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("abrir deu %d", rec.Code)
	}
	b := f.s.boards.Get(context.Background(), f.sessionID)
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

// TestLugarEmBrancoVIRAcena, e chão desconhecido cai no padrão.
//
// Os dois defaults são a mesma decisão: o mestre que só quer a grade não deve
// ser barrado por um campo, e um chão que a tela não oferece só chega por posse
// do fio — a resposta a isso é desenhar pedra, não discutir.
func TestLugarEmBrancoViraCenaEChaoDesconhecidoCaiNoPadrao(t *testing.T) {
	f := novoPiloto(t)
	if rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/tabuleiro/abrir",
		`{"novolugar":"   ","novochao":"lava"}`); rec.Code != http.StatusOK {
		t.Fatalf("abrir deu %d", rec.Code)
	}
	b := f.s.boards.Get(context.Background(), f.sessionID)
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

// TestSOoMESTREmontaEdesmontaAcena: a trava é do servidor.
//
// O botão escondido é cortesia para quem não pode; quem postar na mão leva 403.
// Este guarda é de HANDLER e não uma asserção de que o botão sumiu, porque a
// fronteira de segurança é o servidor e a tela é UX.
func TestSoOMestreMontaEDesmontaACena(t *testing.T) {
	f := novoPiloto(t)
	if rec := f.pede(t, f.jogador, "POST", f.urlDaMesa()+"/tabuleiro/abrir",
		`{"novolugar":"Cripta","novochao":"cripta"}`); rec.Code != http.StatusForbidden {
		t.Errorf("o jogador abriu a cena: %d", rec.Code)
	}
	if f.s.boards.Get(context.Background(), f.sessionID) != nil {
		t.Error("a cena do jogador abriu mesmo assim")
	}

	f.abreTabuleiro(t, "pedra")
	if rec := f.pede(t, f.jogador, "POST", f.urlDaMesa()+"/tabuleiro/encerrar", ""); rec.Code != http.StatusForbidden {
		t.Errorf("o jogador encerrou a cena: %d", rec.Code)
	}
	if f.s.boards.Get(context.Background(), f.sessionID) == nil {
		t.Error("a cena sumiu quando o jogador mandou encerrar")
	}
}

// TestEncerrarTIRAaCenaDaMesaEaGUARDAnoAcervo.
//
// As duas metades juntas porque uma sem a outra não é o gesto: encerrar sem
// arquivar perde a noite de trabalho, e arquivar sem tirar deixa a mesa presa
// numa cena que já acabou.
func TestEncerrarTiraACenaDaMesaEAGuardaNoAcervo(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "taverna")

	if rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/tabuleiro/encerrar", ""); rec.Code != http.StatusOK {
		t.Fatalf("encerrar deu %d", rec.Code)
	}
	if f.s.boards.Get(context.Background(), f.sessionID) != nil {
		t.Error("a cena continuou na mesa depois de encerrada")
	}

	lugares := f.s.boards.Places(context.Background(), f.campaignID)
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

// TestACenaVAZIAdizCoisasDIFERENTESaosDois.
//
// Não é a mesma frase com um botão a mais: o jogador não tem o que fazer além de
// esperar, e o mestre tem. "O mestre abre quando a cena tiver lugar" DITO AO
// PRÓPRIO MESTRE é a tela mandando ele fazer o que ela não deixa — que foi
// exatamente o texto que o piloto carregou até esta fatia.
func TestACenaVaziaDizCoisasDiferentesAosDois(t *testing.T) {
	f := novoPiloto(t)

	// O CONTROLE: os dois chegam na cena vazia. Sem ele, "o jogador não vê
	// 'Abrir tabuleiro'" seria verdade também sobre uma página que não carregou.
	doMestre := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()
	doJogador := f.pede(t, f.jogador, http.MethodGet, f.urlDaMesa(), "").Body.String()
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

// TestOAcervoLISTAoQueFoiEncerrado, com a contagem de peças.
//
// A contagem é o que separa a cena montada da cena aberta e abandonada, e é por
// ela que o mestre decide o que reabrir e o que apagar.
func TestOAcervoListaOQueFoiEncerrado(t *testing.T) {
	f := novoPiloto(t)
	f.noTabuleiro(t) // abre a Taverna e põe UMA peça
	if rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/tabuleiro/encerrar", ""); rec.Code != http.StatusOK {
		t.Fatalf("encerrar deu %d", rec.Code)
	}

	tela := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()
	if !strings.Contains(tela, "Lugares da campanha · 1") {
		t.Errorf("o acervo não apareceu com a cena encerrada")
	}
	// LITERAL e nunca `pecasEmPortugues(1)`: o esperado derivado da produção
	// afirmaria o defeito junto com a regra, e foi assim que "1 peças" chegou à
	// tela na primeira medição.
	if !strings.Contains(tela, "1 peça") {
		t.Errorf("o acervo não diz quantas peças a cena guardada tem")
	}
	if strings.Contains(tela, "1 peças") {
		t.Errorf("a concordância quebrou: a linha diz \"1 peças\"")
	}

	// O acervo é do MESTRE: a mesa não escolhe onde joga.
	doJogador := f.pede(t, f.jogador, http.MethodGet, f.urlDaMesa(), "").Body.String()
	if strings.Contains(doJogador, "Lugares da campanha") {
		t.Error("o jogador recebeu o acervo da campanha")
	}
}

// TestSemLugarGUARDADOnaoHAbotaoDeAcervo.
//
// Um botão que abre uma lista vazia ensina que o acervo não serve para nada. É a
// metade que faz o guarda acima significar alguma coisa — sem ela, "o acervo
// apareceu" seria verdade sobre um botão que aparece sempre.
func TestSemLugarGuardadoNaoHaBotaoDeAcervo(t *testing.T) {
	f := novoPiloto(t)
	tela := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()
	if !strings.Contains(tela, "Abrir tabuleiro") {
		t.Fatal("a cena vazia do mestre não desenhou — o guarda mediria a tela errada")
	}
	if strings.Contains(tela, "Lugares da campanha") {
		t.Error("o acervo vazio ofereceu um menu que não tem o que mostrar")
	}
}

// TestReabrirTROCAacenaEGUARDAaqueEstavaNaMesa.
//
// As duas metades juntas porque é isso que deixa o mestre pular da taverna para
// a cripta com a mesa jogando: reabrir sem guardar a atual perderia a cena em
// que eles estavam.
func TestReabrirTrocaACenaEGuardaAQueEstavaNaMesa(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "taverna") // "Taverna do Javali"
	if rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/tabuleiro/encerrar", ""); rec.Code != http.StatusOK {
		t.Fatalf("encerrar a taverna deu %d", rec.Code)
	}
	if rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/tabuleiro/abrir",
		`{"novolugar":"Cripta","novochao":"cripta"}`); rec.Code != http.StatusOK {
		t.Fatalf("abrir a cripta deu %d", rec.Code)
	}

	taverna := int64(0)
	for _, l := range f.s.boards.Places(context.Background(), f.campaignID) {
		if l.Name == "Taverna do Javali" {
			taverna = l.ID
		}
	}
	if taverna == 0 {
		t.Fatal("a taverna não está no acervo — o guarda mediria a troca errada")
	}

	if rec := f.pede(t, f.mestre, "POST",
		fmt.Sprintf("%s/tabuleiro/lugares/%d/reabrir", f.urlDaMesa(), taverna), ""); rec.Code != http.StatusOK {
		t.Fatalf("reabrir deu %d", rec.Code)
	}
	b := f.s.boards.Get(context.Background(), f.sessionID)
	if b == nil || b.Place != "Taverna do Javali" {
		t.Fatalf("a mesa não voltou para a taverna: %+v", b)
	}
	// E a CRIPTA foi guardada na troca, em vez de perdida.
	achouCripta := false
	for _, l := range f.s.boards.Places(context.Background(), f.campaignID) {
		if l.Name == "Cripta" {
			achouCripta = true
		}
	}
	if !achouCripta {
		t.Error("a cena que estava na mesa se perdeu na troca")
	}
}

// TestApagarUmLugarNAOderrubaAcenaDaMesa.
//
// O `removeOLugar` devolve o tabuleiro ATUAL e não nil, e é por isso: nil faria
// o caminho publicar "não há tabuleiro" para a mesa inteira — o mestre limparia
// o acervo e a mesa perderia a cena em que estava jogando.
func TestApagarUmLugarNaoDerrubaACenaDaMesa(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "taverna")
	if rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/tabuleiro/encerrar", ""); rec.Code != http.StatusOK {
		t.Fatalf("encerrar deu %d", rec.Code)
	}
	if rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/tabuleiro/abrir",
		`{"novolugar":"Cripta","novochao":"cripta"}`); rec.Code != http.StatusOK {
		t.Fatalf("abrir a cripta deu %d", rec.Code)
	}
	guardados := f.s.boards.Places(context.Background(), f.campaignID)
	if len(guardados) == 0 {
		t.Fatal("o acervo está vazio — o guarda não teria o que apagar")
	}

	if rec := f.pede(t, f.mestre, "POST",
		fmt.Sprintf("%s/tabuleiro/lugares/%d/remover", f.urlDaMesa(), guardados[0].ID), ""); rec.Code != http.StatusOK {
		t.Fatalf("remover deu %d", rec.Code)
	}
	if b := f.s.boards.Get(context.Background(), f.sessionID); b == nil {
		t.Error("apagar um lugar do acervo derrubou a cena que estava na mesa")
	} else if b.Place != "Cripta" {
		t.Errorf("a cena da mesa virou %q", b.Place)
	}
}

// TestSOoMESTREmexeNoAcervo: a trava é do servidor.
func TestSoOMestreMexeNoAcervo(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "taverna")
	if rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/tabuleiro/encerrar", ""); rec.Code != http.StatusOK {
		t.Fatalf("encerrar deu %d", rec.Code)
	}
	guardados := f.s.boards.Places(context.Background(), f.campaignID)
	if len(guardados) == 0 {
		t.Fatal("o acervo está vazio — o guarda mediria uma rota sem alvo")
	}
	id := guardados[0].ID

	for _, acao := range []string{"reabrir", "remover"} {
		rec := f.pede(t, f.jogador, "POST",
			fmt.Sprintf("%s/tabuleiro/lugares/%d/%s", f.urlDaMesa(), id, acao), "")
		if rec.Code != http.StatusForbidden {
			t.Errorf("o jogador conseguiu %s um lugar: %d", acao, rec.Code)
		}
	}
	if len(f.s.boards.Places(context.Background(), f.campaignID)) != len(guardados) {
		t.Error("o acervo mudou de tamanho com o jogador mexendo nele")
	}
}

// TestACenaVAZIAnoAcervoSeANUNCIAcomoTal.
//
// "0 peças" descreve mal o que a linha é. Cena aberta e abandonada é justamente
// o que o mestre procura quando abre o acervo para limpar, e a linha tem de
// dizer isso em vez de fazer ele contar zeros.
func TestACenaVaziaNoAcervoSeAnunciaComoTal(t *testing.T) {
	f := novoPiloto(t)
	if rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/tabuleiro/abrir",
		`{"novolugar":"Sala esquecida","novochao":"pedra"}`); rec.Code != http.StatusOK {
		t.Fatalf("abrir deu %d", rec.Code)
	}
	if rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/tabuleiro/encerrar", ""); rec.Code != http.StatusOK {
		t.Fatalf("encerrar deu %d", rec.Code)
	}

	tela := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()
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
