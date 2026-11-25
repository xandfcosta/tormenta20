package table

import (
	"testing"

	"t20engine/domain/board"
	"t20engine/domain/live"
)

// O ENJOADO tem UMA ação (p394): o caminho que pede duas ações de movimento não
// cabe no turno dele, e a tela tem de dizer isso antes do mestre confirmar —
// a mesma pergunta que a cobrança faz depois.
func TestTheSickenedTurnCannotPayAPathOfTwoMovements(t *testing.T) {
	entry := "goblin"
	b := &board.BoardState{
		Tokens:  []board.BoardToken{{ID: "peca", EntryID: &entry}},
		Pending: &board.PendingMove{TokenID: "peca"},
	}
	st := live.EmptyRuntimeState()
	st.Initiative = []live.InitiativeEntry{{ID: entry, Label: "Goblin"}}
	st.TurnIndex = 0
	st.Scene = &live.Scene{Kind: live.SceneAction, StandardLeft: true, MovementLeft: true}
	if turnCannotPay(b, st, 12, 9) {
		t.Fatal("o controle falhou: o turno cheio paga duas pernas de movimento (a padrão trocada, p233)")
	}
	st.Scene.OneActionOnly = true
	if !turnCannotPay(b, st, 12, 9) {
		t.Error("enjoado, o turno pagou duas ações de movimento (p394)")
	}
	if turnCannotPay(b, st, 6, 9) {
		t.Error("enjoado, uma ação de movimento ainda cabe")
	}
}
