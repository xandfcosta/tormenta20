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

/*
Duplicar peça (ALE-192).

"Mais um zumbi" é a operação mais repetida ao montar encontro, e até agora ela
custava abrir a forma, digitar o nome, escolher o tamanho e posicionar — para
uma criatura idêntica à que já está ali ao lado.

A tabela de exemplos aqui é a MESMA de `token-appearance.test.ts`, no front: as
duas pontas carregam a convenção "espécie + número", e se elas divergirem a
cópia nasce com um nome que o desenho colore como outra espécie.
*/

func tabuleiroCom(labels ...string) *BoardState {
	b := newBoard("Cripta", "pedra")
	for i, label := range labels {
		b.Tokens = append(b.Tokens, BoardToken{
			ID: fmt.Sprintf("t%d", i), Label: label, X: i, Y: 0, Footprint: 1, Kind: "npc",
		})
	}
	return b
}

func TestACopiaGanhaOProximoNumeroLivre(t *testing.T) {
	casos := []struct {
		nome    string
		cena    []string
		duplica string
		quer    string
	}{
		{"a fila continua", []string{"Zumbi 1", "Zumbi 2"}, "Zumbi 1", "Zumbi 3"},
		// Menor livre e não maior+1: tirado o Zumbi 2, a próxima cópia volta a
		// ser o Zumbi 2 e a numeração continua colada.
		{"o buraco é preenchido", []string{"Zumbi 1", "Zumbi 3"}, "Zumbi 3", "Zumbi 2"},
		// A peça SEM número ocupa o 1: um "Ogro 1" ninguém distingue do "Ogro".
		{"o sem número vira o 1", []string{"Ogro"}, "Ogro", "Ogro 2"},
		// A ARMADILHA: número no meio do nome não é instância.
		{"número no meio não conta", []string{"Recruta Nv1 Simples"}, "Recruta Nv1 Simples", "Recruta Nv1 Simples 2"},
		// Espécies diferentes não disputam número.
		{"outra espécie não atrapalha", []string{"Zumbi 1", "Goblin 2"}, "Zumbi 1", "Zumbi 2"},
	}
	for _, caso := range casos {
		b := tabuleiroCom(caso.cena...)
		alvo := ""
		for _, token := range b.Tokens {
			if token.Label == caso.duplica {
				alvo = token.ID
			}
		}
		if err := duplicateToken(b, alvo, novoIDFixo()); err != nil {
			t.Fatalf("%s: duplicar: %v", caso.nome, err)
		}
		copia := b.Tokens[len(b.Tokens)-1]
		if copia.Label != caso.quer {
			t.Errorf("%s: a cópia se chama %q, esperado %q", caso.nome, copia.Label, caso.quer)
		}
	}
}

// A cópia é uma PEÇA NOVA: ela leva o corpo e deixa o vínculo para trás.
func TestACopiaLevaOCorpoENaoOVinculo(t *testing.T) {
	b := tabuleiroCom("Zumbi 1")
	entrada := "e7"
	var personagem int64 = 42
	b.Tokens[0].EntryID = &entrada
	b.Tokens[0].CharacterID = &personagem
	b.Tokens[0].Footprint = 2
	b.Tokens[0].Hidden = true

	if err := duplicateToken(b, "t0", novoIDFixo()); err != nil {
		t.Fatalf("duplicar: %v", err)
	}

	copia := b.Tokens[len(b.Tokens)-1]
	if copia.EntryID != nil || copia.CharacterID != nil {
		t.Errorf("a cópia levou o vínculo junto: entryId=%v characterId=%v", copia.EntryID, copia.CharacterID)
	}
	if copia.Footprint != 2 {
		t.Errorf("a cópia nasceu com tamanho %d, esperado 2", copia.Footprint)
	}
	// O segundo zumbi da emboscada também está escondido.
	if !copia.Hidden {
		t.Error("a cópia de uma peça escondida nasceu visível")
	}
}

// Ao LADO, e não em cima nem na fileira de entrada: quem duplica o zumbi do
// canto do mapa espera o irmão dele ali do lado.
func TestACopiaNasceAoLadoENaoEmCima(t *testing.T) {
	b := tabuleiroCom("Zumbi 1")
	b.Tokens[0].X, b.Tokens[0].Y = 30, 12

	if err := duplicateToken(b, "t0", novoIDFixo()); err != nil {
		t.Fatalf("duplicar: %v", err)
	}

	copia := b.Tokens[len(b.Tokens)-1]
	if copia.X == 30 && copia.Y == 12 {
		t.Error("a cópia nasceu em cima do original")
	}
	if abs(copia.X-30) > 1 || abs(copia.Y-12) > 1 {
		t.Errorf("a cópia nasceu em (%d,%d), longe do original (30,12)", copia.X, copia.Y)
	}
}

func novoIDFixo() func() string {
	n := 0
	return func() string {
		n++
		return fmt.Sprintf("copia%d", n)
	}
}
