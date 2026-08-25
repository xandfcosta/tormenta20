package api

import (
	"context"
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
