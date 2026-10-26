package engine

import "testing"

// O TREINO POR NÍVEL (p114): +2, +4 no 7º e +6 no 15º. Um caso por fronteira —
// o que quebra numa tabela de degraus é o limite.
func TestTheTrainingBonusClimbsAtTheSeventhAndFifteenthLevels(t *testing.T) {
	for _, tc := range []struct{ level, want int }{{1, 2}, {6, 2}, {7, 4}, {14, 4}, {15, 6}, {20, 6}} {
		if got := TrainingBonusForLevel(tc.level); got != tc.want {
			t.Errorf("no nível %d o treino é %d, quer %d", tc.level, got, tc.want)
		}
	}
}
