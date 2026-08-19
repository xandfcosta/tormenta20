package engine

import "testing"

/*
Os gabaritos de área (Tormenta 20, p225).

A figura da p225 É a regra — o livro diz, sobre o cone, "conforme os modelos da
ilustração" —, então estes testes guardam a TRANSCRIÇÃO dela: a contagem de
casas de cada modelo desenhado e, nos casos que carregam a forma, o desenho
inteiro.

A transcrição foi feita medindo a figura do PDF quadrado a quadrado, e o achado
que ela trouxe está registrado no `board_area.go`: as formas do livro são
LOSANGOS, não círculos — a mesma régua do movimento (p238) e do alcance (p224).
*/

// desenho devolve o gabarito como linhas de texto, para o teste comparar FORMA e
// não uma lista de pares que ninguém consegue ler.
func desenho(casas []Square) []string {
	if len(casas) == 0 {
		return nil
	}
	minX, maxX, minY, maxY := casas[0].X, casas[0].X, casas[0].Y, casas[0].Y
	for _, c := range casas {
		if c.X < minX {
			minX = c.X
		}
		if c.X > maxX {
			maxX = c.X
		}
		if c.Y < minY {
			minY = c.Y
		}
		if c.Y > maxY {
			maxY = c.Y
		}
	}
	dentro := map[Square]bool{}
	for _, c := range casas {
		dentro[c] = true
	}
	var linhas []string
	for y := minY; y <= maxY; y++ {
		linha := ""
		for x := minX; x <= maxX; x++ {
			if dentro[Square{X: x, Y: y}] {
				linha += "#"
				continue
			}
			linha += "."
		}
		linhas = append(linhas, linha)
	}
	return linhas
}

func exigeDesenho(t *testing.T, casas []Square, quer []string) {
	t.Helper()
	got := desenho(casas)
	if len(got) != len(quer) {
		t.Fatalf("o gabarito tem %d linhas, esperado %d:\n%v", len(got), len(quer), got)
	}
	for i := range quer {
		if got[i] != quer[i] {
			t.Errorf("linha %d:\n  veio  %q\n  quer  %q\n(inteiro: %v)", i, got[i], quer[i], got)
		}
	}
}

// As três esferas DESENHADAS na figura: raio de 1,5m (1 quadrado), 3m (2) e 6m
// (4). A esfera nasce na interseção de quatro quadrados, então não há casa
// central — há quatro.
func TestAsEsferasDaIlustracao(t *testing.T) {
	quatro := AreaSquares(Square{X: 0, Y: 0}, Area{Kind: AreaSphere, Size: 1})
	if len(quatro) != 4 {
		t.Errorf("o raio de 1,5m cobriu %d casas, a figura mostra 4", len(quatro))
	}
	exigeDesenho(t, quatro, []string{
		"##",
		"##",
	})

	doze := AreaSquares(Square{X: 0, Y: 0}, Area{Kind: AreaSphere, Size: 2})
	if len(doze) != 12 {
		t.Errorf("o raio de 3m cobriu %d casas, a figura mostra 12", len(doze))
	}
	exigeDesenho(t, doze, []string{
		".##.",
		"####",
		"####",
		".##.",
	})

	quarenta := AreaSquares(Square{X: 0, Y: 0}, Area{Kind: AreaSphere, Size: 4})
	if len(quarenta) != 40 {
		t.Errorf("o raio de 6m cobriu %d casas, a figura mostra 40", len(quarenta))
	}
	exigeDesenho(t, quarenta, []string{
		"...##...",
		"..####..",
		".######.",
		"########",
		"########",
		".######.",
		"..####..",
		"...##...",
	})
}

// Os cones da figura, nas DUAS orientações que ela desenha. As contagens são
// diferentes entre elas para o mesmo tamanho, e isso é da figura.
func TestOsConesDaIlustracao(t *testing.T) {
	direita := Square{X: 1, Y: 0}
	casos := []struct {
		nome  string
		lado  int
		casas int
	}{
		{"cone de 4,5m (3 quadrados), ortogonal", 3, 7},
		{"cone de 6m (4 quadrados), ortogonal", 4, 12},
		{"cone de 9m (6 quadrados), ortogonal", 6, 24},
	}
	for _, caso := range casos {
		got := AreaSquares(Square{X: 0, Y: 0}, Area{Kind: AreaCone, Size: caso.lado, Direction: direita})
		if len(got) != caso.casas {
			t.Errorf("%s cobriu %d casas, a figura mostra %d", caso.nome, len(got), caso.casas)
		}
	}

	// A forma do cone de 6m ortogonal, que é a que mostra o alargamento de dois
	// em dois: 1, 3, 3, 5.
	exigeDesenho(t, AreaSquares(Square{X: 0, Y: 0}, Area{Kind: AreaCone, Size: 4, Direction: direita}), []string{
		"...#",
		".###",
		"####",
		".###",
		"...#",
	})

	diagonais := []struct {
		lado  int
		casas int
	}{{3, 6}, {4, 10}, {6, 21}}
	for _, caso := range diagonais {
		got := AreaSquares(Square{X: 0, Y: 0}, Area{Kind: AreaCone, Size: caso.lado, Direction: Square{X: 1, Y: 1}})
		if len(got) != caso.casas {
			t.Errorf("o cone diagonal de %d quadrados cobriu %d casas, a figura mostra %d", caso.lado, len(got), caso.casas)
		}
	}
	// E a forma da escada, no modelo de 4,5m.
	exigeDesenho(t, AreaSquares(Square{X: 0, Y: 0}, Area{Kind: AreaCone, Size: 3, Direction: Square{X: 1, Y: 1}}), []string{
		"###",
		"##.",
		"#..",
	})
}

// A linha de 15m da figura: dez quadrados, um de largura (p225).
func TestALinhaDaIlustracao(t *testing.T) {
	linha := AreaSquares(Square{X: 0, Y: 0}, Area{Kind: AreaLine, Size: 10, Direction: Square{X: 1, Y: 0}})

	if len(linha) != 10 {
		t.Errorf("a linha de 15m cobriu %d casas, a figura mostra 10", len(linha))
	}
	exigeDesenho(t, linha, []string{"##########"})
	// E ela começa ADJACENTE à origem, não em cima dela: "surge adjacente a você".
	if linha[0].X != 1 {
		t.Errorf("a linha começou em x=%d; ela surge adjacente à origem", linha[0].X)
	}
}

// Os dois cubos da figura: 1,5m é uma casa, 3m é 2×2.
func TestOsCubosDaIlustracao(t *testing.T) {
	um := AreaSquares(Square{X: 0, Y: 0}, Area{Kind: AreaSquare, Size: 1})
	if len(um) != 1 {
		t.Errorf("o cubo de 1,5m cobriu %d casas, a figura mostra 1", len(um))
	}
	quatro := AreaSquares(Square{X: 0, Y: 0}, Area{Kind: AreaSquare, Size: 2})
	exigeDesenho(t, quatro, []string{
		"##",
		"##",
	})
}

// Tamanho zero ou negativo não desenha nada: é o estado em que a tela está
// antes de a pessoa escolher, e devolver uma casa "de cortesia" faria o mapa
// piscar um gabarito que ninguém pediu.
func TestGabaritoSemTamanhoNaoDesenhaNada(t *testing.T) {
	for _, area := range []Area{
		{Kind: AreaSphere, Size: 0},
		{Kind: AreaCone, Size: 3},                           // sem direção
		{Kind: AreaLine, Size: -1, Direction: Square{X: 1}}, //nolint:exhaustruct
		{Kind: AreaSquare, Size: 0},
	} {
		if got := AreaSquares(Square{X: 0, Y: 0}, area); len(got) != 0 {
			t.Errorf("%+v desenhou %d casas", area, len(got))
		}
	}
}
