package api

import (
	"fmt"
	"testing"
)

// O tabuleiro tático da sessão (ALE-124). O que se prova aqui é o que alguém na
// mesa notaria quebrar: peça que sai da grade, peça escondida que vaza para o
// jogador, e "adicionar grupo" duplicando quem já está no tabuleiro.

func boardCounter() func() string {
	n := 0
	return func() string { n++; return fmt.Sprintf("t%d", n) }
}

func openBoard(t *testing.T, cols, rows int) *BoardState {
	t.Helper()
	b, err := newBoard("Taverna do Javali", cols, rows, "pedra")
	if err != nil {
		t.Fatalf("abrir tabuleiro %dx%d: %v", cols, rows, err)
	}
	return b
}

func TestNewBoardRejectsUnplayableGrid(t *testing.T) {
	if _, err := newBoard("Vazio", 0, 10, "pedra"); err == nil {
		t.Error("grade de largura zero foi aceita")
	}
	// 60 quadrados são 90m, o alcance longo do livro (p224) — além disso não
	// cabe em magia nenhuma nem em vista nenhuma.
	if _, err := newBoard("Enorme", boardMaxSide+1, 10, "pedra"); err == nil {
		t.Error("grade acima do teto foi aceita")
	}
}

// Uma peça Grande ocupa 2×2 (T20 p107, Tab. 1-21): ancorada na última coluna,
// metade dela ficaria fora da grade. A validação cobra o CORPO, não o canto.
func TestTokenMustFitInsideTheGrid(t *testing.T) {
	b := openBoard(t, 10, 10)
	id := boardCounter()

	if err := addToken(b, BoardToken{Label: "Ogro", Footprint: 2, X: 9, Y: 4}, id); err == nil {
		t.Error("peça 2x2 na última coluna foi aceita e metade dela está fora")
	}
	if err := addToken(b, BoardToken{Label: "Ogro", Footprint: 2, X: 8, Y: 4}, id); err != nil {
		t.Errorf("peça 2x2 que CABE foi recusada: %v", err)
	}
	if err := addToken(b, BoardToken{Label: "Rato", X: -1, Y: 0}, id); err == nil {
		t.Error("posição negativa foi aceita")
	}
}

// Crescer a peça é mover a borda dela sem mexer em X/Y: um Médio no canto que
// vira Colossal (6×6, p107) sai da grade sem que ninguém tenha arrastado nada.
func TestGrowingATokenIsAlsoBoundsChecked(t *testing.T) {
	b := openBoard(t, 8, 8)
	id := boardCounter()
	if err := addToken(b, BoardToken{Label: "Dragão", X: 5, Y: 5}, id); err != nil {
		t.Fatalf("adicionar: %v", err)
	}
	seis := 6

	if err := updateToken(b, "t1", tokenPatch{Footprint: &seis}); err == nil {
		t.Error("peça 6x6 ancorada em (5,5) numa grade 8x8 foi aceita")
	}
	if b.Tokens[0].Footprint != 1 {
		t.Errorf("a peça recusada mudou assim mesmo: footprint %d", b.Tokens[0].Footprint)
	}
}

func TestBoardVersionRisesOnEveryAcceptedChange(t *testing.T) {
	b := openBoard(t, 10, 10)
	id := boardCounter()
	inicio := b.Version

	_ = addToken(b, BoardToken{Label: "Goblin"}, id)
	depoisDeAdicionar := b.Version
	if depoisDeAdicionar <= inicio {
		t.Error("adicionar peça não moveu a versão")
	}
	// Recusa NÃO conta: uma versão que sobe sem o estado mudar faria o cliente
	// descartar broadcast bom.
	_ = addToken(b, BoardToken{Label: "Fora", X: 99}, id)
	if b.Version != depoisDeAdicionar {
		t.Error("uma mutação RECUSADA mexeu na versão")
	}
	removeToken(b, "t1")
	if b.Version <= depoisDeAdicionar {
		t.Error("remover peça não moveu a versão")
	}
}

// A peça escondida some INTEIRA da cópia do jogador. É a assimetria deliberada
// em relação ao `hpHidden` da iniciativa, onde a linha sobrevive sem os números:
// aqui a existência da peça é a emboscada (ALE-124).
func TestHiddenTokenVanishesForPlayers(t *testing.T) {
	b := openBoard(t, 10, 10)
	id := boardCounter()
	_ = addToken(b, BoardToken{Label: "Bandido", X: 1, Y: 1}, id)
	_ = addToken(b, BoardToken{Label: "Assassino na viga", X: 2, Y: 2, Hidden: true}, id)

	doJogador := boardForRole("player", b)

	if len(doJogador.Tokens) != 1 || doJogador.Tokens[0].Label != "Bandido" {
		t.Errorf("o jogador recebeu %d peças: %+v", len(doJogador.Tokens), doJogador.Tokens)
	}
	if doMestre := boardForRole("gm", b); len(doMestre.Tokens) != 2 {
		t.Errorf("o mestre perdeu a própria emboscada: %d peças", len(doMestre.Tokens))
	}
	// Papel desconhecido cai em jogador: errar para o lado que MOSTRA seria
	// vazar por omissão.
	if len(boardForRole("", b).Tokens) != 1 {
		t.Error("papel vazio recebeu o tabuleiro inteiro")
	}
	if len(b.Tokens) != 2 {
		t.Error("a redação mexeu no tabuleiro original — o mestre perderia a peça")
	}
}

func TestPopulateBoardIsIdempotent(t *testing.T) {
	b := openBoard(t, 10, 10)
	id := boardCounter()
	st := emptyRuntimeState()
	entryID := counter()
	_ = addEntry(st, npc("Ogro", 12), entryID)
	_ = addEntry(st, npc("Bandido", 8), entryID)

	if placed := populateBoard(b, st, id); placed != 2 {
		t.Errorf("primeira chamada colocou %d peças, esperado 2", placed)
	}
	if placed := populateBoard(b, st, id); placed != 0 {
		t.Errorf("segunda chamada colocou %d peças — quem já está no tabuleiro duplicou", placed)
	}
	if len(b.Tokens) != 2 {
		t.Errorf("o tabuleiro ficou com %d peças", len(b.Tokens))
	}
	// Ninguém empilhado: duas peças no mesmo quadrado seriam uma peça invisível.
	if b.Tokens[0].X == b.Tokens[1].X && b.Tokens[0].Y == b.Tokens[1].Y {
		t.Errorf("as duas peças nasceram no mesmo quadrado: %+v", b.Tokens)
	}
}
