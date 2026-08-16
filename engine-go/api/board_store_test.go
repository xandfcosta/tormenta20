package api

import (
	"context"
	"testing"
)

// O tabuleiro sobrevive ao reinício do servidor — a memória é a verdade da
// sessão, mas o servidor cai, e uma mesa que perde as posições no meio da noite
// perde a cena inteira (ALE-124).

func TestBoardPersistsAndComesBack(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	sid := seedSession(t, s, seedCampaign(t, s, seedUser(t, s, "gm@t.com")))

	if _, err := s.boards.open(ctx, sid, "Taverna do Javali", 20, 15, "taverna"); err != nil {
		t.Fatalf("abrir: %v", err)
	}
	if _, err := s.boards.addToken(ctx, sid, BoardToken{Label: "Ogro", X: 3, Y: 4, Footprint: 2}); err != nil {
		t.Fatalf("adicionar peça: %v", err)
	}
	s.boards.persist(ctx, sid)

	// Um servidor novo sobre o MESMO banco: é o reinício, sem fingir.
	frio := newBoardStore(s.queries, newUUID)
	voltou := frio.get(ctx, sid)

	if voltou == nil {
		t.Fatal("o tabuleiro não voltou do banco")
	}
	if voltou.Place != "Taverna do Javali" || voltou.Cols != 20 || voltou.Rows != 15 {
		t.Errorf("o lugar ou a grade se perderam: %+v", voltou)
	}
	if len(voltou.Tokens) != 1 || voltou.Tokens[0].X != 3 || voltou.Tokens[0].Y != 4 {
		t.Errorf("a peça voltou fora do lugar: %+v", voltou.Tokens)
	}
	// Quadrado é a unidade guardada: um footprint que volta 0 desenharia uma
	// peça sem corpo e o teto de alcance sairia errado.
	if voltou.Tokens[0].Footprint != 2 {
		t.Errorf("o tamanho da peça se perdeu: %d", voltou.Tokens[0].Footprint)
	}
}

// "Sem tabuleiro" tem de voltar como AUSÊNCIA. Um `BoardState{}` de cortesia
// desenharia uma grade de 0×0 e o mestre acharia que abriu alguma coisa.
func TestSessionWithoutBoardStaysWithout(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	sid := seedSession(t, s, seedCampaign(t, s, seedUser(t, s, "gm@t.com")))

	if b := s.boards.get(ctx, sid); b != nil {
		t.Errorf("sessão nova já veio com tabuleiro: %+v", b)
	}
	if _, err := s.boards.addToken(ctx, sid, BoardToken{Label: "Ninguém"}); err == nil {
		t.Error("pôs peça num tabuleiro que não existe")
	}
}

func TestClosingBoardErasesItFromDiskToo(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	sid := seedSession(t, s, seedCampaign(t, s, seedUser(t, s, "gm@t.com")))
	if _, err := s.boards.open(ctx, sid, "Cripta", 10, 10, "pedra"); err != nil {
		t.Fatalf("abrir: %v", err)
	}
	s.boards.persist(ctx, sid)

	s.boards.close(ctx, sid)

	if b := s.boards.get(ctx, sid); b != nil {
		t.Error("o tabuleiro encerrado continua na memória")
	}
	if b := newBoardStore(s.queries, newUUID).get(ctx, sid); b != nil {
		t.Error("o tabuleiro encerrado voltou do banco no próximo reinício")
	}
}

// Reabrir não pode fazer a versão VOLTAR: o cliente descarta o que tem número
// menor que o dele, e um tabuleiro novo com versão velha seria ignorado pela
// tela de quem estava na sessão desde antes.
func TestReopeningKeepsVersionMovingForward(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	sid := seedSession(t, s, seedCampaign(t, s, seedUser(t, s, "gm@t.com")))
	primeiro, err := s.boards.open(ctx, sid, "Taverna", 10, 10, "taverna")
	if err != nil {
		t.Fatalf("abrir: %v", err)
	}
	if _, err := s.boards.addToken(ctx, sid, BoardToken{Label: "Bandido"}); err != nil {
		t.Fatalf("adicionar: %v", err)
	}
	antes := s.boards.get(ctx, sid).Version

	segundo, err := s.boards.open(ctx, sid, "Masmorra", 12, 12, "pedra")
	if err != nil {
		t.Fatalf("reabrir: %v", err)
	}

	if segundo.Version <= antes {
		t.Errorf("a versão voltou no tempo: %d depois de %d (primeiro abriu em %d)",
			segundo.Version, antes, primeiro.Version)
	}
	if len(segundo.Tokens) != 0 {
		t.Errorf("a masmorra nasceu com as peças da taverna: %+v", segundo.Tokens)
	}
}
