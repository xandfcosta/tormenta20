package board

import (
	"strings"
	"testing"

	"t20engine/domain/engine"
)

// O MODO DE DESLOCAMENTO troca o orçamento do caminho proposto: a sílfide que voa mede contra o voo,
// e o voo não paga o terreno difícil, que é do chão (decisão do dono). Escalando ou nadando, a peça
// paga como a pé — "uma parede acidentada pode ser considerada terreno difícil" (p228).
func TestTheMoveModeMeasuresThePathWithItsOwnBudget(t *testing.T) {
	b, st := mesaEmCombate(t)
	b.Tokens[0].OtherModes = []MoveMode{{Kind: MoveModeFly, Squares: 8}, {Kind: MoveModeClimb, Squares: 6}}
	b.Difficult = []engine.Square{{X: 1, Y: 0}, {X: 2, Y: 0}}

	if err := ProposeMoveWithStops(b, st, "t1", caminho([2]int{0, 0}, [2]int{3, 0}), jogadorDono); err != nil {
		t.Fatalf("propor a pé: %v", err)
	}
	if b.Pending.Budget != 6 || b.Pending.Cost != 5 {
		t.Fatalf("a pé saiu %d de %d; esperava 5 de 6 (duas casas difíceis custam o dobro, p238)", b.Pending.Cost, b.Pending.Budget)
	}

	if err := ChangeMoveMode(b, st, "t1", MoveModeFly, jogadorDono); err != nil {
		t.Fatalf("trocar para o voo: %v", err)
	}
	if b.Pending.Mode != MoveModeFly || b.Pending.Budget != 8 || b.Pending.Cost != 3 {
		t.Errorf("voando saiu %d de %d no modo %q; esperava 3 de 8, o voo por cima do terreno difícil", b.Pending.Cost, b.Pending.Budget, b.Pending.Mode)
	}

	// A PARADA NOVA herda o modo: estender o caminho voando não o devolve ao chão.
	if err := ProposeMoveWithStops(b, st, "t1", caminho([2]int{0, 0}, [2]int{3, 0}, [2]int{4, 0}), jogadorDono); err != nil {
		t.Fatalf("estender voando: %v", err)
	}
	if b.Pending.Mode != MoveModeFly || b.Pending.Budget != 8 {
		t.Errorf("a parada nova voltou o modo para %q com orçamento %d", b.Pending.Mode, b.Pending.Budget)
	}

	if err := ChangeMoveMode(b, st, "t1", MoveModeClimb, jogadorDono); err != nil {
		t.Fatalf("trocar para a escalada: %v", err)
	}
	if b.Pending.Budget != 6 || b.Pending.Cost != 6 {
		t.Errorf("escalando saiu %d de %d; esperava 6 de 6, pagando o terreno difícil", b.Pending.Cost, b.Pending.Budget)
	}

	if err := ChangeMoveMode(b, st, "t1", MoveModeLand, jogadorDono); err != nil {
		t.Fatalf("voltar a pé: %v", err)
	}
	if b.Pending.Mode != MoveModeLand || b.Pending.Budget != 6 {
		t.Errorf("a pé de novo saiu no modo %q com orçamento %d", b.Pending.Mode, b.Pending.Budget)
	}
}

// O modo que a peça não tem é recusado, dizendo qual foi pedido e quais ela tem.
func TestTheMoveModeTheTokenLacksIsRefused(t *testing.T) {
	b, st := mesaEmCombate(t)
	b.Tokens[0].OtherModes = []MoveMode{{Kind: MoveModeFly, Squares: 8}}
	if err := ProposeMoveWithStops(b, st, "t1", caminho([2]int{0, 0}, [2]int{1, 0}), jogadorDono); err != nil {
		t.Fatalf("propor: %v", err)
	}
	err := ChangeMoveMode(b, st, "t1", MoveModeSwim, jogadorDono)
	if err == nil || !strings.Contains(err.Error(), "natacao") || !strings.Contains(err.Error(), "voo") {
		t.Errorf("a natação que a sílfide não tem saiu com %v; a recusa diz o pedido e o que ela tem", err)
	}
	if err := ChangeMoveMode(b, st, "t1", MoveModeFly, Mover{UserID: 99, Role: "player"}); err == nil {
		t.Error("quem não propôs trocou o modo do movimento de outra pessoa")
	}
}
