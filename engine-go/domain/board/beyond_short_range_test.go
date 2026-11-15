package board

import (
	"slices"
	"testing"

	"t20engine/domain/engine"
)

// ALÉM DO ALCANCE CURTO (ALE-652): o alcance curto é 9m, seis quadrados (p224), medido de corpo a corpo pela
// régua da mesa (`ApartBetween`). O tabuleiro só diz a distância; o que ela tira do golpe é do motor.
func TestTheBoardSaysWhenTheTargetIsBeyondShortRange(t *testing.T) {
	for _, c := range []struct {
		name   string
		target BoardToken
		want   bool
	}{
		{"a seis quadrados ainda é curto", creatureAt("ogro", "npc", 6, 0, 1), false},
		{"a sete quadrados já não é", creatureAt("ogro", "npc", 7, 0, 1), true},
		{"o Grande a sete casas do canto, mas a seis pelo corpo", creatureAt("ogro", "npc", 7, -1, 2), true},
		{"na diagonal, quatro e quatro somam oito", creatureAt("ogro", "npc", 4, 4, 1), true},
	} {
		b := &BoardState{Tokens: []BoardToken{creatureAt("arqueira", "character", 0, 0, 1), c.target}}
		got := slices.Contains(SituationsBetween(b, "arqueira", AttackTargetOnTheBoard{EntryID: "ogro"}), engine.TargetBeyondShortRange)
		if got != c.want {
			t.Errorf("%s: além do alcance curto = %t, want %t", c.name, got, c.want)
		}
	}
}
