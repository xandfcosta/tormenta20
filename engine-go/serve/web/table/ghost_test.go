package table

import (
	"testing"

	"t20engine/domain/board"
	"t20engine/domain/engine"
)

// `Stops` NULO é valor legítimo: o `ProposeMove` deixa o caminho pronto sem
// passar por paradas. Deduzir as dobras do `Path` não é possível — um trecho
// legítimo já dobra sozinho, porque a diagonal vem primeiro —, então a seta vira
// a reta entre o começo e o fim.
func TestTheArrowWithoutStopsJoinsBothEndsOfThePath(t *testing.T) {
	noStops := &board.PendingMove{
		Path: []engine.Square{{}, {X: 1}, {X: 2}, {X: 3}},
	}
	folds := moveFolds(noStops)
	if len(folds) != 2 || folds[0] != (engine.Square{}) || folds[1] != (engine.Square{X: 3}) {
		t.Fatalf("as dobras de um caminho sem paradas saíram %+v", folds)
	}
	// A perna anda 3 e a ponta recua meio quadrado: de 0,5 até 3,0. Sem orçamento
	// (-1) ela sai inteira de dourado — o vermelho tem guarda próprio em
	// `move_drawing_test.go`.
	if wire, _, _ := moveWires(folds, []int{3}, -1); wire != "M 0.5 0.5 L 3 0.5" {
		t.Errorf("a seta reta saiu %q", wire)
	}
	// E com uma dobra só não há o que ligar: `d` vazio é o jeito de o `<path>`
	// não desenhar sem um `data-show` a mais, que é a combinação que congela a aba.
	if wire, blue, beyond := moveWires([]engine.Square{{}}, nil, -1); wire != "" || blue != "" || beyond != "" {
		t.Errorf("uma dobra só desenhou %q, %q e %q", wire, blue, beyond)
	}
}
