package table

import (
	"testing"

	"t20engine/domain/board"
	"t20engine/domain/live"
)

// Com a iniciativa rodando, a peça que o mestre move com custo é a DA VEZ. Pegar a
// primeira que ele pode mover prendia o gesto numa peça só: passar a vez trocava o nome
// na fila, e o arrasto continuava levando a mesma.
func TestTheGMTargetFollowsTheTurn(t *testing.T) {
	first, second := "entrada-1", "entrada-2"
	tokens := &board.BoardState{Tokens: []board.BoardToken{
		{ID: "alfa", Label: "Alfa", EntryID: &first},
		{ID: "beta", Label: "Beta", EntryID: &second},
		{ID: "gama", Label: "Gama"},
	}}
	combat := func(turn int) *live.SessionRuntimeState {
		return &live.SessionRuntimeState{
			TurnIndex:  turn,
			Initiative: []live.InitiativeEntry{{ID: first}, {ID: second}, {ID: "entrada-sem-peca"}},
		}
	}
	gm := board.Mover{Role: "gm"}

	cases := []struct {
		name string
		turn int
		want string
	}{
		{"a vez é da primeira", 0, "alfa"},
		{"a vez passou para a segunda", 1, "beta"},
		{"a vez é de quem não tem peça: nenhuma paga custo", 2, ""},
	}
	for _, c := range cases {
		if got := reachAndTarget(tokens, combat(c.turn), gm, nil).Target; got != c.want {
			t.Errorf("%s: o alvo do mestre é %q, esperava %q", c.name, got, c.want)
		}
	}

	// Fora da iniciativa não há vez para seguir: nenhuma peça propõe, e todas pousam direto.
	outside := &live.SessionRuntimeState{TurnIndex: -1}
	if got := reachAndTarget(tokens, outside, gm, nil).Target; got != "" {
		t.Errorf("fora da iniciativa o alvo é %q, esperava nenhum", got)
	}
}
