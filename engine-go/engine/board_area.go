package engine

import "sort"

// Áreas de efeito no mapa de batalha (Tormenta 20, p225).
//
// O livro descreve as áreas em uma frase cada e manda ver a ILUSTRAÇÃO da
// p225 ("conforme os modelos da ilustração") — ou seja, a figura É a regra, e é
// contra ela que este arquivo foi escrito. Os seis modelos desenhados (cones de
// 4,5m, 6m e 9m, cada um em duas orientações), as três esferas (raios de 1,5m,
// 3m e 6m), a linha de 15m e os dois cubos foram TRANSCRITOS quadrado a
// quadrado da figura, e o teste de regra guarda a contagem de cada um.
//
// O achado da transcrição: as formas da figura são LOSANGOS, e não círculos.
// Isso não é detalhe estético — significa que a área usa a mesma régua que o
// movimento (a diagonal custa o dobro, p238) e que o alcance (p224). Uma só
// régua no mapa inteiro.

// AreaKind é a categoria de área do livro (p225).
type AreaKind string

const (
	// AreaSphere é a esfera/raio: "surge na interseção de quatro quadrados,
	// estendendo-se em todas as direções até o limite de seu raio".
	AreaSphere AreaKind = "esfera"
	// AreaCone: "surge adjacente a você e se afasta na direção escolhida,
	// ficando mais largo com a distância".
	AreaCone AreaKind = "cone"
	// AreaLine: "surge adjacente a você e se afasta reta até o fim do alcance";
	// 1,5m de largura salvo indicação em contrário — um quadrado.
	AreaLine AreaKind = "linha"
	// AreaSquare é o quadrado/cubo: "surge no quadrado ou quadrados escolhidos".
	AreaSquare AreaKind = "quadrado"
)

// Area descreve o gabarito a desenhar.
//
// Size é sempre em QUADRADOS: raio da esfera, comprimento do cone e da linha,
// lado do quadrado. Direction é o passo unitário (dx,dy ∈ {-1,0,1}) para cone e
// linha, e é ignorada pelos outros — a esfera vai para todos os lados e o
// quadrado não aponta para lugar nenhum.
type Area struct {
	Kind      AreaKind `json:"kind"`
	Size      int      `json:"size"`
	Direction Square   `json:"direction"`
}

// AreaSquares devolve as casas cobertas pelo gabarito, ordenadas para o
// resultado ser estável entre chamadas (a tela desenha e o teste compara).
//
// `origin` é o quadrado clicado para o cone, a linha e o quadrado. Para a
// ESFERA ele é a INTERSEÇÃO, dada pelo canto superior-esquerdo do quadrado
// clicado: o livro põe a esfera no encontro de quatro quadrados, então ela não
// tem casa central — tem quatro.
//
// @example AreaSquares(Square{0,0}, Area{Kind: AreaSphere, Size: 1}) // as 4 casas em volta
func AreaSquares(origin Square, area Area) []Square {
	var casas []Square
	switch area.Kind {
	case AreaSphere:
		casas = sphereSquares(origin, area.Size)
	case AreaCone:
		casas = coneSquares(origin, area.Size, area.Direction)
	case AreaLine:
		casas = lineSquares(origin, area.Size, area.Direction)
	case AreaSquare:
		casas = squareSquares(origin, area.Size)
	}
	sort.Slice(casas, func(a, b int) bool {
		if casas[a].Y != casas[b].Y {
			return casas[a].Y < casas[b].Y
		}
		return casas[a].X < casas[b].X
	})
	return casas
}

// sphereSquares desenha a esfera a partir da interseção (p225).
//
// A conta é `i + j ≤ raio + 1` com i e j contados A PARTIR DE 1 (a casa colada
// na interseção é a 1), espelhada nos quatro quadrantes. É a transcrição exata
// dos três modelos da figura: raio 1 dá as 4 casas em volta do ponto, raio 2 dá
// 12 e raio 4 dá 40.
func sphereSquares(corner Square, radius int) []Square {
	if radius < 1 {
		return nil
	}
	var casas []Square
	for i := 1; i <= radius; i++ {
		for j := 1; j <= radius; j++ {
			if i+j > radius+1 {
				continue
			}
			casas = append(casas,
				Square{X: corner.X + i - 1, Y: corner.Y + j - 1},
				Square{X: corner.X - i, Y: corner.Y + j - 1},
				Square{X: corner.X + i - 1, Y: corner.Y - j},
				Square{X: corner.X - i, Y: corner.Y - j},
			)
		}
	}
	return casas
}

// coneSquares desenha o cone, e são DUAS formas porque a figura desenha duas.
//
// Na ORTOGONAL o cone abre de dois em dois: a `d` quadrados de distância ele
// tem `2·⌊d/2⌋+1` de largura — 1, 3, 3, 5, 5, 7… —, que é exatamente o que os
// modelos de 4,5m (7 casas), 6m (12) e 9m (24) mostram.
//
// Na DIAGONAL ele é o quadrante de um losango — `dx+dy ≤ n+1` —, o que dá a
// escada de 6, 10 e 21 casas dos mesmos três modelos. As duas formas têm
// contagens diferentes para o mesmo tamanho, e isso é da figura, não um
// arredondamento nosso.
func coneSquares(origin Square, length int, dir Square) []Square {
	if length < 1 || (dir.X == 0 && dir.Y == 0) {
		return nil
	}
	if dir.X != 0 && dir.Y != 0 {
		return diagonalCone(origin, length, dir)
	}
	return orthogonalCone(origin, length, dir)
}

func orthogonalCone(origin Square, length int, dir Square) []Square {
	var casas []Square
	for d := 1; d <= length; d++ {
		meia := d / 2 // 2·⌊d/2⌋+1 de largura, centrada na fileira da origem
		for k := -meia; k <= meia; k++ {
			if dir.X != 0 {
				casas = append(casas, Square{X: origin.X + dir.X*d, Y: origin.Y + k})
				continue
			}
			casas = append(casas, Square{X: origin.X + k, Y: origin.Y + dir.Y*d})
		}
	}
	return casas
}

func diagonalCone(origin Square, length int, dir Square) []Square {
	var casas []Square
	for dx := 1; dx <= length; dx++ {
		for dy := 1; dy <= length; dy++ {
			if dx+dy > length+1 {
				continue
			}
			casas = append(casas, Square{X: origin.X + dir.X*dx, Y: origin.Y + dir.Y*dy})
		}
	}
	return casas
}

// lineSquares desenha a linha: reta, de um quadrado de largura (1,5m, p225),
// saindo da casa vizinha à origem.
//
// A figura só desenha a linha ORTOGONAL. Na diagonal ela vira a escada de
// quadrados que a régua do movimento já usa — extrapolação desta casa, dita
// aqui para ninguém a confundir com texto do livro.
func lineSquares(origin Square, length int, dir Square) []Square {
	if length < 1 || (dir.X == 0 && dir.Y == 0) {
		return nil
	}
	casas := make([]Square, 0, length)
	for d := 1; d <= length; d++ {
		casas = append(casas, Square{X: origin.X + dir.X*d, Y: origin.Y + dir.Y*d})
	}
	return casas
}

// squareSquares desenha o quadrado/cubo com a casa clicada no canto
// superior-esquerdo: "surge no quadrado ou quadrados escolhidos" (p225).
func squareSquares(origin Square, side int) []Square {
	if side < 1 {
		return nil
	}
	casas := make([]Square, 0, side*side)
	for j := 0; j < side; j++ {
		for i := 0; i < side; i++ {
			casas = append(casas, Square{X: origin.X + i, Y: origin.Y + j})
		}
	}
	return casas
}
