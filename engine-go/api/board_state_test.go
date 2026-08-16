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

func openBoard(t *testing.T) *BoardState {
	t.Helper()
	return newBoard("Taverna do Javali", "pedra")
}

// O plano NÃO tem bordas: quadrado negativo é lugar legítimo, e é para lá que a
// cena cresce quando o mestre empurra a briga para a esquerda (ALE-124).
func TestBoardHasNoEdges(t *testing.T) {
	b := openBoard(t)
	id := boardCounter()

	if err := addToken(b, BoardToken{Label: "Batedor", X: -40, Y: -12}, id); err != nil {
		t.Errorf("coordenada negativa recusada num plano infinito: %v", err)
	}
	if err := addToken(b, BoardToken{Label: "Ogro", Footprint: 2, X: 999, Y: 4}, id); err != nil {
		t.Errorf("peça longe da origem recusada: %v", err)
	}
}

// O limite de sanidade não é borda do mapa: é o guarda contra o cliente que
// manda lixo, porque um número absurdo estoura a serialização e a tela de todo
// mundo na mesa.
func TestAbsurdCoordinatesAreRefused(t *testing.T) {
	b := openBoard(t)
	id := boardCounter()

	if err := addToken(b, BoardToken{Label: "Lixo", X: boardCoordLimit + 1}, id); err == nil {
		t.Error("coordenada absurda foi aceita")
	}
	if err := addToken(b, BoardToken{Label: "Lixo", Y: -(boardCoordLimit + 1)}, id); err == nil {
		t.Error("coordenada absurda negativa foi aceita")
	}
	if len(b.Tokens) != 0 {
		t.Errorf("a peça recusada entrou assim mesmo: %+v", b.Tokens)
	}
}

func TestBoardVersionRisesOnEveryAcceptedChange(t *testing.T) {
	b := openBoard(t)
	id := boardCounter()
	inicio := b.Version

	_ = addToken(b, BoardToken{Label: "Goblin"}, id)
	depoisDeAdicionar := b.Version
	if depoisDeAdicionar <= inicio {
		t.Error("adicionar peça não moveu a versão")
	}
	// Recusa NÃO conta: uma versão que sobe sem o estado mudar faria o cliente
	// descartar broadcast bom.
	_ = addToken(b, BoardToken{Label: "Lixo", X: boardCoordLimit + 1}, id)
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
	b := openBoard(t)
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
	b := openBoard(t)
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
