package sheetui

import (
	"testing"
)

// Os guardas do painel de PERÍCIAS (ALE-272, fatia 4).
//
// O que eles prendem é a REGRA e a DECISÃO: a ordem que a mesa lê, o filtro que
// ignora acento, os quatro gestos de escrita e a fronteira de cada um. Os
// NÚMEROS são do motor e já têm o oráculo de paridade.

// TestTheHeaderSaysTheTrainingForTheLevel é uma ficha com perícias de verdade e um ofício inventado.
// O TREINO POR NÍVEL tem três degraus, e o cabeçalho os diz.
func TestTheHeaderSaysTheTrainingForTheLevel(t *testing.T) {
	casos := []struct {
		nivel int64
		quer  int
	}{{1, 2}, {6, 2}, {7, 4}, {14, 4}, {15, 6}, {20, 6}}
	for _, caso := range casos {
		if got := trainingBonusFor(caso.nivel); got != caso.quer {
			t.Errorf("no nível %d o treino é %d, quer %d", caso.nivel, got, caso.quer)
		}
	}
}
