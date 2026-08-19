package api

import (
	"context"
	"testing"
)

/*
Lugares da crônica (ALE-124, fatia 5).

Até esta fatia, encerrar o tabuleiro DESTRUÍA a cena: o `close` apagava a linha
e a taverna que o mestre montou peça por peça morria junto. A épica prometia o
contrário — "encerrar ARQUIVA, e devolve o tabuleiro à lista de Lugares da
crônica" —, e era a única promessa que o código contradizia.

O que se prova aqui é o ciclo que a mesa vive: montar, encerrar, voltar semana
que vem e achar tudo onde estava.
*/

func mesaComTaverna(t *testing.T) (*Server, int64, int64) {
	t.Helper()
	s := newTestServer(t)
	campanha := seedCampaign(t, s, seedUser(t, s, "gm@t.com"))
	sessao := seedSession(t, s, campanha)
	ctx := context.Background()

	s.boards.open(ctx, sessao, "Taverna do Javali", "taverna")
	if _, err := s.boards.addToken(ctx, sessao, BoardToken{Label: "Ogro", X: 3, Y: 4, Footprint: 2}, true); err != nil {
		t.Fatalf("adicionar peça: %v", err)
	}
	return s, campanha, sessao
}

func TestEncerrarArquivaACenaEmVezDeDestruir(t *testing.T) {
	s, campanha, sessao := mesaComTaverna(t)
	ctx := context.Background()

	if err := s.boards.archive(ctx, campanha, s.boards.get(ctx, sessao)); err != nil {
		t.Fatalf("arquivar: %v", err)
	}
	s.boards.close(ctx, sessao)

	lugares := s.boards.places(ctx, campanha)
	if len(lugares) != 1 {
		t.Fatalf("depois de encerrar, a crônica tem %d lugares: %+v", len(lugares), lugares)
	}
	if lugares[0].Name != "Taverna do Javali" {
		t.Errorf("o lugar guardado se chama %q", lugares[0].Name)
	}
	// A contagem existe para o mestre escolher onde jogar sem baixar o acervo.
	if lugares[0].Tokens != 1 {
		t.Errorf("a taverna guardada tem %d peças, esperado 1", lugares[0].Tokens)
	}
	// E a mesa fica MESMO sem tabuleiro: arquivar não é deixar a cena aberta.
	if b := s.boards.get(ctx, sessao); b != nil {
		t.Errorf("a sessão continuou com tabuleiro depois de encerrar: %+v", b)
	}
}

func TestReabrirTrazAsPecasOndeEstavam(t *testing.T) {
	s, campanha, sessao := mesaComTaverna(t)
	ctx := context.Background()
	if err := s.boards.archive(ctx, campanha, s.boards.get(ctx, sessao)); err != nil {
		t.Fatalf("arquivar: %v", err)
	}
	s.boards.close(ctx, sessao)
	guardado := s.boards.places(ctx, campanha)[0]

	volta, err := s.boards.reopen(ctx, sessao, guardado.ID)
	if err != nil {
		t.Fatalf("reabrir: %v", err)
	}

	if volta.Place != "Taverna do Javali" {
		t.Errorf("reabriu como %q", volta.Place)
	}
	if len(volta.Tokens) != 1 || volta.Tokens[0].X != 3 || volta.Tokens[0].Y != 4 {
		t.Fatalf("as peças não voltaram onde estavam: %+v", volta.Tokens)
	}
	if volta.Tokens[0].Footprint != 2 {
		t.Errorf("o tamanho da peça se perdeu: %d", volta.Tokens[0].Footprint)
	}
}

// Encerrar a MESMA taverna de novo sobrescreve: quem reabre, move duas peças e
// encerra espera UMA taverna, não uma pilha de tavernas quase iguais.
func TestArquivarDuasVezesNaoEmpilhaOMesmoLugar(t *testing.T) {
	s, campanha, sessao := mesaComTaverna(t)
	ctx := context.Background()

	if err := s.boards.archive(ctx, campanha, s.boards.get(ctx, sessao)); err != nil {
		t.Fatalf("arquivar: %v", err)
	}
	if _, err := s.boards.addToken(ctx, sessao, BoardToken{Label: "Bandido", X: 9, Y: 9}, true); err != nil {
		t.Fatalf("segunda peça: %v", err)
	}
	if err := s.boards.archive(ctx, campanha, s.boards.get(ctx, sessao)); err != nil {
		t.Fatalf("arquivar de novo: %v", err)
	}

	lugares := s.boards.places(ctx, campanha)
	if len(lugares) != 1 {
		t.Fatalf("a crônica ficou com %d tavernas: %+v", len(lugares), lugares)
	}
	if lugares[0].Tokens != 2 {
		t.Errorf("o lugar guardou %d peças, esperado 2 (a versão mais recente)", lugares[0].Tokens)
	}
}

// O provisório é de uma cena que já acabou: a mesa que reabre a taverna não
// deve nada a um movimento proposto na semana passada.
func TestOProvisorioNaoVoltaComOLugar(t *testing.T) {
	s, campanha, sessao := mesaComTaverna(t)
	ctx := context.Background()
	board := s.boards.get(ctx, sessao)
	board.Pending = &PendingMove{TokenID: "t1", Cost: 3, Budget: 6}

	if err := s.boards.archive(ctx, campanha, board); err != nil {
		t.Fatalf("arquivar: %v", err)
	}
	s.boards.close(ctx, sessao)
	volta, err := s.boards.reopen(ctx, sessao, s.boards.places(ctx, campanha)[0].ID)
	if err != nil {
		t.Fatalf("reabrir: %v", err)
	}

	if volta.Pending != nil {
		t.Errorf("o provisório voltou junto com o lugar: %+v", volta.Pending)
	}
}

// O id do lugar vem do cliente: sem conferir a crônica, um mestre apagaria a
// cena de OUTRA mesa mandando um id que não é dele.
func TestNaoSeApagaLugarDeOutraCronica(t *testing.T) {
	s, campanha, sessao := mesaComTaverna(t)
	ctx := context.Background()
	if err := s.boards.archive(ctx, campanha, s.boards.get(ctx, sessao)); err != nil {
		t.Fatalf("arquivar: %v", err)
	}
	guardado := s.boards.places(ctx, campanha)[0]
	outra := seedCampaign(t, s, seedUser(t, s, "outro@t.com"))

	if err := s.boards.removePlace(ctx, outra, guardado.ID); err == nil {
		t.Fatal("apagou o lugar de outra crônica")
	}
	if len(s.boards.places(ctx, campanha)) != 1 {
		t.Error("o lugar sumiu mesmo com a recusa")
	}
}
