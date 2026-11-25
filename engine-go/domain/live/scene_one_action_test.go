package live

import (
	"strings"
	"testing"
)

// O enjoado tem UMA ação, padrão ou de movimento (p394), e o contador tem de
// dizer a escolha, e não "padrão e movimento".
func TestTheTurnCounterSaysTheSickenedHasOneAction(t *testing.T) {
	scene := &Scene{Kind: SceneAction, StandardLeft: true, MovementLeft: true, OneActionOnly: true}
	if got := TurnCounter(scene, 1, 0, 3); !strings.HasSuffix(got, "· padrão ou movimento") {
		t.Errorf("enjoado, o contador disse %q", got)
	}
}

// A vez seguinte começa cheia: o "uma ação só" é de quem estava enjoado.
func TestRefreshingTheTurnDropsTheOneActionLimit(t *testing.T) {
	st := EmptyRuntimeState()
	st.Scene = &Scene{Kind: SceneAction, OneActionOnly: true}
	RefreshTurn(st)
	if st.Scene.OneActionOnly {
		t.Error("a vez nova herdou o limite do enjoado da anterior")
	}
}
