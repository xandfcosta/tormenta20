package live

import (
	"strings"
	"testing"
)

// A mesa lê o que sobrou do turno antes de clicar, e o movimento extra tem de estar lá.
func TestTheTurnCounterSaysTheExtraMovement(t *testing.T) {
	scene := &Scene{Kind: SceneAction, StandardLeft: true, MovementLeft: true, ExtraMovementLeft: true}
	if got := TurnCounter(scene, 1, 0, 3); !strings.Contains(got, "padrão, movimento e movimento extra") {
		t.Errorf("o contador disse %q", got)
	}
	scene.StandardLeft, scene.MovementLeft = false, false
	if got := TurnCounter(scene, 1, 0, 3); !strings.HasSuffix(got, "· movimento extra") {
		t.Errorf("sem padrão e movimento, o contador disse %q", got)
	}
}
