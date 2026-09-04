package table

import (
	"testing"

	"t20engine/engine"
)

// A ESFERA NASCE NA INTERSEÇÃO, e as outras três formas não (p225).
//
// "Escolha uma interseção entre quadrados. A esfera se espalha a partir dela" —
// e o cone, a linha e o quadrado nascem no quadrado. Não é escolha de tela: é o
// livro, e o `engine.sphereSquares` já desenhava a partir do CANTO. Quem errava
// era a tela, que mandava o quadrado do `floor` do clique — até meio quadrado
// entre onde o dedo estava e onde a bola caía, sem nada dizendo por quê.
//
// A outra metade deste guarda ficou no hospedeiro (`table_polyline_test.go`):
// ela precisa de um servidor montado para provar que a página arredonda o
// clique e pergunta ao servidor onde a forma nasce. Separadas as duas já
// divergiram uma vez, e é por isso que as duas existem.
func TestOnlyTheSphereStartsAtTheIntersection(t *testing.T) {
	if !shapeStartsAtIntersection(engine.AreaSphere) {
		t.Error("a esfera deixou de nascer na interseção (p225)")
	}
	for _, outra := range []engine.AreaKind{engine.AreaSquare, engine.AreaCone, engine.AreaLine} {
		if shapeStartsAtIntersection(outra) {
			t.Errorf("%q passou a nascer na interseção, e o livro só diz isso da esfera (p225)", outra)
		}
	}
}
