package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"t20engine/tabuleiro"
)

// Os guardas do PINCEL de terreno na Mesa (ALE-264, item 5).

// TestOPincelPINTAaEspecieQUEfoiPEDIDA, e só ela.
//
// Amostragem sobre `EspeciesDeTerreno`: a quinta espécie já nasce medida.
func TestOPincelPintaAEspecieQuePediu(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "pedra")

	for i, pincel := range tabuleiro.EspeciesDeTerreno {
		casa := fmt.Sprintf("/%d/0", i)
		rec := f.pede(t, f.mestre, "POST",
			f.urlDaMesa()+"/tabuleiro/terreno/"+string(pincel.ID)+casa, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("pintar %s deu %d", pincel.ID, rec.Code)
		}
	}

	b := f.s.boards.Get(context.Background(), f.sessionID)
	for i, pincel := range tabuleiro.EspeciesDeTerreno {
		casas := tabuleiro.QuadradosDe(b, pincel.ID)
		if len(casas) != 1 || casas[0].X != i {
			t.Errorf("%s ficou com %v, esperado só a casa %d", pincel.ID, casas, i)
		}
	}
}

// TestABORRACHAapagaSOaEspecieEscolhida.
//
// É por isso que ela é um MODO e não uma espécie: numa casa com duas, uma
// "espécie borracha" teria de decidir qual apagar, e a resposta certa — a que
// está selecionada — já é o que o modo faz. Folhagens são difícil E camuflagem
// (p267), então a casa com duas não é hipótese.
func TestABorrachaApagaSoAEspecieEscolhida(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "pedra")
	base := f.urlDaMesa() + "/tabuleiro/terreno"

	for _, especie := range []string{"dificil", "camuflagem"} {
		if rec := f.pede(t, f.mestre, "POST", base+"/"+especie+"/3/3", ""); rec.Code != http.StatusOK {
			t.Fatalf("pintar %s deu %d", especie, rec.Code)
		}
	}
	if rec := f.pede(t, f.mestre, "POST", base+"/camuflagem/3/3?apagar=1", ""); rec.Code != http.StatusOK {
		t.Fatalf("apagar deu %d", rec.Code)
	}

	b := f.s.boards.Get(context.Background(), f.sessionID)
	if n := len(tabuleiro.QuadradosDe(b, tabuleiro.TerrenoCamuflagem)); n != 0 {
		t.Errorf("a camuflagem não foi apagada (%d casas)", n)
	}
	if n := len(tabuleiro.QuadradosDe(b, tabuleiro.TerrenoDificil)); n != 1 {
		t.Errorf("a borracha levou o difícil junto (%d casas) — a casa tinha as duas", n)
	}
}

// TestAsQUATROespeciesSAOdesenhadasDISTINTAS.
//
// O guarda de leiaute que esta casa cobra: um traço pintado que não vira classe
// própria some no desenho das outras, e o mestre lê a cena errada sem nada
// estourar. Amostragem sobre a lista.
func TestAsQuatroEspeciesSaoDesenhadasDistintas(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "pedra")
	for i, pincel := range tabuleiro.EspeciesDeTerreno {
		if rec := f.pede(t, f.mestre, "POST",
			fmt.Sprintf("%s/tabuleiro/terreno/%s/%d/0", f.urlDaMesa(), pincel.ID, i), ""); rec.Code != http.StatusOK {
			t.Fatalf("pintar %s deu %d", pincel.ID, rec.Code)
		}
	}

	tela := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()
	// O CONTROLE: o tabuleiro desenhou. Sem ele, não achar as classes seria
	// verdade também sobre uma cena que não abriu.
	if !strings.Contains(tela, "tabuleiro-plano") {
		t.Fatal("o tabuleiro não desenhou — o guarda mediria a tela errada")
	}
	for _, pincel := range tabuleiro.EspeciesDeTerreno {
		if !strings.Contains(tela, "tabuleiro-"+string(pincel.ID)) {
			t.Errorf("a espécie %s foi pintada e não tem desenho próprio na cena", pincel.ID)
		}
	}
}

// TestOTRILHOdizOEFEITOdeCadaEspecie.
//
// "Cobertura" sozinho não lembra ninguém de que são +5 na Defesa, e o mestre que
// precisa da regra sai da mesa para procurá-la no livro. É a mesma razão de o
// diálogo de abrir dizer que um quadrado são 1,5m.
func TestOTrilhoDizOEfeitoDeCadaEspecie(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "pedra")
	tela := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()

	if !strings.Contains(tela, "Pincel de terreno") {
		t.Fatal("o mestre não tem trilho de pincel na cena aberta")
	}
	// O nome acessível da camada NÃO cita a espécie: ela sairia do sinal, que
	// guarda o ID, e o leitor de tela anunciaria "Pintar dificil" sem acento.
	// Quem diz qual é a espécie é o botão `aria-pressed` do trilho.
	if strings.Contains(tela, "'Pintar ' + $pincel") {
		t.Error("o nome acessível da camada monta o rótulo com o id do pincel")
	}
	for _, pincel := range tabuleiro.EspeciesDeTerreno {
		if !strings.Contains(tela, pincel.Efeito) {
			t.Errorf("o trilho não diz o que %s faz (%q)", pincel.ID, pincel.Efeito)
		}
	}
	// A PÁGINA, e não a regra: a citação vai junto para conferir sem reabrir o
	// livro. O número é da Tabela 5-3.
	if !strings.Contains(tela, "p238") {
		t.Error("o trilho não cita a página da regra")
	}

	// E o pincel é do MESTRE: o jogador não pinta chão.
	doJogador := f.pede(t, f.jogador, http.MethodGet, f.urlDaMesa(), "").Body.String()
	if strings.Contains(doJogador, "Pincel de terreno") {
		t.Error("o jogador recebeu o pincel de terreno")
	}
}

// TestSOoMESTREpinta: a trava é do servidor, e não o botão escondido.
func TestSoOMestrePinta(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "pedra")

	rec := f.pede(t, f.jogador, "POST", f.urlDaMesa()+"/tabuleiro/terreno/dificil/1/1", "")
	if rec.Code != http.StatusForbidden {
		t.Errorf("o jogador pintou o chão: %d", rec.Code)
	}
	b := f.s.boards.Get(context.Background(), f.sessionID)
	if n := len(tabuleiro.QuadradosDe(b, tabuleiro.TerrenoDificil)); n != 0 {
		t.Errorf("a pintura do jogador entrou mesmo assim (%d casas)", n)
	}
}

// TestPintarSEMtabuleiroRECUSAcomFRASE.
//
// Não é 500 nem silêncio: pintar chão de uma cena que não está na mesa não tem
// onde acontecer, e a recusa fala no `erroDoComando` do rodapé do mestre.
func TestPintarSemTabuleiroRecusaComFrase(t *testing.T) {
	f := novoPiloto(t)
	corpo := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/tabuleiro/terreno/dificil/1/1", "").Body.String()
	if !strings.Contains(corpo, "não há tabuleiro aberto") {
		t.Errorf("a recusa não explica o que faltou; sinais = %s", trechoDeSinais(corpo))
	}
}
