package api

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"t20engine/tabuleiro"
)

// Os guardas do TABULEIRO na Mesa (ALE-263).
//
// O extenso e a aparência têm guarda de REGRA no `tabuleiro`, contra as bordas.
// O que se prende aqui é a COMPOSIÇÃO — que a cena pergunta a coisa certa a cada
// regra, e sobretudo que a REDAÇÃO POR PAPEL alcança o mapa.

// abreTabuleiro põe um tabuleiro na mesa com uma peça, e devolve o id dela.
func (f pilotoFixture) abreTabuleiro(t *testing.T, terreno string) *tabuleiro.BoardState {
	t.Helper()
	b := f.s.boards.Open(context.Background(), f.sessionID, "Taverna do Javali", terreno)
	if b == nil {
		t.Fatal("o tabuleiro não abriu")
	}
	return b
}

// TestSemTabuleiroACenaDizQueNaoHaMapaENaoDesenhaGrade.
//
// "Não há tabuleiro" e "há um vazio" são estados diferentes, e o primeiro é o
// normal — a maior parte de uma sessão não tem mapa. Desenhar uma grade vazia
// diria que o mestre abriu uma cena que ele não abriu.
func TestSemTabuleiroACenaDizQueNaoHaMapa(t *testing.T) {
	f := novoPiloto(t)
	corpo := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()

	if !strings.Contains(corpo, "Nenhum tabuleiro aberto") {
		t.Error("a cena não disse que não há mapa")
	}
	if strings.Contains(corpo, "tabuleiro-plano") {
		t.Error("desenhou a grade sem tabuleiro aberto")
	}
}

// TestAPecaEscondidaNaoCHEGAaoJogador — o guarda que mais importa desta fatia.
//
// Esconder a peça é o gesto com que o mestre guarda a emboscada, e a trava não
// pode ser CSS: uma peça meio-apagada no HTML do jogador entrega a posição do
// ogro para quem abrir o inspetor. Quem a tira é o `BoardForRole`, o mesmo
// gargalo por papel que a fila usa — e este teste afirma que a cena passa por
// ele em vez de decidir por conta própria.
func TestAPecaEscondidaNaoChegaAoJogador(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "cripta")
	if _, err := f.s.boards.AddToken(context.Background(), f.sessionID,
		tabuleiro.BoardToken{ID: "emboscada", Label: "Ogro", X: 4, Y: 3, Hidden: true}, true); err != nil {
		t.Fatalf("pôr a peça escondida: %v", err)
	}
	if _, err := f.s.boards.AddToken(context.Background(), f.sessionID,
		tabuleiro.BoardToken{ID: "avista", Label: "Arwen", X: 1, Y: 1}, true); err != nil {
		t.Fatalf("pôr a peça à vista: %v", err)
	}

	doMestre := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()
	if !strings.Contains(doMestre, "Ogro em") {
		t.Error("o mestre não viu a própria peça escondida")
	}

	doJogador := f.pede(t, f.jogador, http.MethodGet, f.urlDaMesa(), "").Body.String()
	// O CONTROLE: o jogador está vendo o tabuleiro. Sem ele, "não achei o Ogro"
	// seria verdade também numa cena sem mapa nenhum.
	if !strings.Contains(doJogador, "Arwen em") {
		t.Fatal("o jogador não viu o tabuleiro; a ausência abaixo não provaria nada")
	}
	if strings.Contains(doJogador, "Ogro") {
		t.Error("a peça escondida chegou ao HTML do jogador")
	}
}

// TestAPecaDaVezAcendeComOMesmoDouradoDaFila.
//
// O anel é o MESMO sinal que a linha da fila usa, e ligá-lo pelo `entryId` é o
// que garante isso: derivar "quem está na vez" no tabuleiro seria a segunda
// cópia da regra, e é assim que duas telas passam a apontar combatentes
// diferentes (ALE-122).
func TestAPecaDaVezAcendeComOMesmoDouradoDaFila(t *testing.T) {
	f := novoPiloto(t)
	entryID := f.naFila(t)
	f.abreTabuleiro(t, "pedra")
	if _, err := f.s.boards.AddToken(context.Background(), f.sessionID,
		tabuleiro.BoardToken{ID: "p", Label: "Arcanista", X: 2, Y: 2, EntryID: &entryID}, true); err != nil {
		t.Fatalf("pôr a peça: %v", err)
	}
	if rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/scene/start", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar cena deu %d", rec.Code)
	}

	// FORA de combate ninguém está na vez, mesmo com a cena aberta e a fila
	// montada — é o `TurnIndex` negativo, e a peça não pode acender por estar
	// no mapa.
	antes := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()
	if strings.Contains(antes, "tabuleiro-peca-na-vez") {
		t.Error("a peça acendeu antes de o combate começar")
	}

	if rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/initiative/next-turn", ""); rec.Code != http.StatusOK {
		t.Fatalf("avançar deu %d", rec.Code)
	}
	depois := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()
	if !strings.Contains(depois, "tabuleiro-peca-na-vez") {
		t.Error("chegou a vez do combatente e a peça dele não acendeu")
	}
	if !strings.Contains(depois, "— na vez") {
		t.Error("o anel não tem par em TEXTO: cor não existe para leitor de tela (ALE-212)")
	}
}

// TestUmTerrenoInventadoNaoVIRAclasseSolta.
//
// O terreno vem do BANCO, então é dado do cliente. Uma classe `chao-<qualquer>`
// não existiria na folha e o chão sairia transparente — o que se parece com
// defeito de CSS e manda procurar no lugar errado.
func TestUmTerrenoInventadoCaiNoChaoPadrao(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "vulcão-de-neon")

	corpo := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()
	if !strings.Contains(corpo, "chao-pedra") {
		t.Error("o terreno inventado não caiu no chão padrão")
	}
	if strings.Contains(corpo, "chao-vulcão") {
		t.Error("o terreno inventado virou classe solta")
	}
}
