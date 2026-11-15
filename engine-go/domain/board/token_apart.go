package board

import "t20engine/domain/engine"

// Apart é a distância entre duas peças, medida de CORPO a corpo (ALE-660): do quadrado de uma ao quadrado da
// outra que estão mais perto, pela régua da mesa (`engine.Measure`, a diagonal valendo dois). `Reaches` diz que o
// alcance natural de quem age chega ao outro corpo — o mesmo teste das situações da Tabela 5-3 (`body.reaches`).
//
// Do CORPO e não do canto: uma criatura Grande ocupa 2×2 (Tabela 1-21, p107), e medir do canto diria que o ogro
// está a três metros do anão que ele está pisando.
type Apart struct {
	engine.Measurement
	Reaches bool
}

// ApartBetween mede de `from` até `to`.
//
//	ApartBetween(BoardToken{X: 0}, BoardToken{X: 1}) // 1 quadrado (1,5m), curto, alcança
func ApartBetween(from, to BoardToken) Apart {
	a, b := bodyOf(from), bodyOf(to)
	dx := max(0, b.X0-a.X1, a.X0-b.X1)
	dy := max(0, b.Y0-a.Y1, a.Y0-b.Y1)
	return Apart{Measurement: engine.Measure(engine.Square{}, engine.Square{X: dx, Y: dy}), Reaches: a.reaches(b)}
}
