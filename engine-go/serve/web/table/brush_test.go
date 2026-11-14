package table

import (
	"testing"

	"t20engine/domain/board"
)

// O guarda que paga o preço de o desenho morar fora do domínio: `drawing`
// entra em pânico numa espécie sem entrada, e este caso faz o pânico acontecer
// na suíte em vez de na mesa. Sem ele, a quinta espécie nasceria com uma casa
// que não se distingue de nenhuma outra — indistinguível de "o pincel não
// funcionou", que é a família de defeito que esta issue inteira persegue.
//
// E ele afirma também que os CANTOS são distintos: duas espécies no mesmo canto
// desenham uma por cima da outra, e a casa com folhagens (difícil E camuflagem,
// p267) mostraria uma só.
func TestEveryKindHasADrawing(t *testing.T) {
	corners := map[string]string{}
	for _, brush := range board.TerrainKinds {
		d := drawing(brush.ID)
		if d.Icon == "" || d.Canto == "" {
			t.Errorf("a espécie %q tem desenho incompleto: %+v", brush.ID, d)
		}
		if owner, found := corners[d.Canto]; found {
			t.Errorf("o canto %q é de %q e de %q — uma desenha por cima da outra",
				d.Canto, owner, brush.ID)
		}
		corners[d.Canto] = string(brush.ID)
	}
}
