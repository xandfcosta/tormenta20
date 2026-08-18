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

	s.boards.open(ctx, sid, "Taverna do Javali", "taverna")
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
	if voltou.Place != "Taverna do Javali" || voltou.Terrain != "taverna" {
		t.Errorf("o lugar ou o cenário se perderam: %+v", voltou)
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
	s.boards.open(ctx, sid, "Cripta", "pedra")
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
	primeiro := s.boards.open(ctx, sid, "Taverna", "taverna")
	if _, err := s.boards.addToken(ctx, sid, BoardToken{Label: "Bandido"}); err != nil {
		t.Fatalf("adicionar: %v", err)
	}
	antes := s.boards.get(ctx, sid).Version

	segundo := s.boards.open(ctx, sid, "Masmorra", "pedra")

	if segundo.Version <= antes {
		t.Errorf("a versão voltou no tempo: %d depois de %d (primeiro abriu em %d)",
			segundo.Version, antes, primeiro.Version)
	}
	if len(segundo.Tokens) != 0 {
		t.Errorf("a masmorra nasceu com as peças da taverna: %+v", segundo.Tokens)
	}
}

// Gravação que falha PARA DE SER SILENCIOSA (ALE-124).
//
// Este teste existe por um defeito de verdade: a `session_boards` sumiu do
// banco de desenvolvimento — a migração constava aplicada, a tabela não existia
// — e o tabuleiro passou um dia inteiro vivendo só em memória. A tela estava
// impecável, e cada gravação falhava numa linha de log que ninguém lê. O que
// faltava não era a gravação: era a mesa SABER que ela parou.
//
// A transição é o que importa: avisa quando começa a falhar e avisa quando
// volta, e não a cada mensagem — um aviso por tique de peça viraria ruído e
// ninguém leria esse também.
func TestBoardPersistFailureIsReported(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	sid := seedSession(t, s, seedCampaign(t, s, seedUser(t, s, "gm@t.com")))
	s.boards.open(ctx, sid, "Cripta", "pedra")

	if dirty, changed := s.boards.persist(ctx, sid); dirty || changed {
		t.Fatalf("gravação saudável já saiu como falha: dirty=%v changed=%v", dirty, changed)
	}

	if _, err := s.db.Exec("DROP TABLE session_boards"); err != nil {
		t.Fatalf("derrubar a tabela: %v", err)
	}
	dirty, changed := s.boards.persist(ctx, sid)
	if !dirty || !changed {
		t.Fatalf("a tabela sumiu e ninguém avisou: dirty=%v changed=%v", dirty, changed)
	}
	// Segunda falha seguida: continua falhando, mas NÃO é notícia nova.
	if _, changed := s.boards.persist(ctx, sid); changed {
		t.Error("a mesa levou um aviso a cada gravação, e não só na transição")
	}

	if _, err := s.db.Exec(`CREATE TABLE session_boards (
		sessionId INTEGER PRIMARY KEY, state TEXT NOT NULL, updatedAt TEXT NOT NULL)`); err != nil {
		t.Fatalf("recriar a tabela: %v", err)
	}
	if dirty, changed := s.boards.persist(ctx, sid); dirty || !changed {
		t.Errorf("a recuperação não foi anunciada: dirty=%v changed=%v", dirty, changed)
	}
}
