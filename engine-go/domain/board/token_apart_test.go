package board

import "testing"

// A DISTÂNCIA ENTRE DUAS PEÇAS (ALE-660) é de CORPO a corpo: do quadrado de uma ao quadrado da outra que estão
// mais perto, pela régua da mesa (a diagonal vale dois, `engine.RangeSquares`). Uma criatura Grande ocupa 2×2
// (Tabela 1-21, p107), e medir do canto dela diria que o ogro está longe do anão que ele está pisando. "Alcança"
// é o alcance natural de quem age: adjacente, diagonal incluída (p233), e mais longe para os corpos maiores.
func TestApartIsMeasuredFromBodyToBody(t *testing.T) {
	at := func(x, y, side int) BoardToken { return BoardToken{X: x, Y: y, Footprint: side} }
	for _, tt := range []struct {
		name     string
		from, to BoardToken
		squares  int
		metres   float64
		reaches  bool
	}{
		{"ao lado", at(0, 0, 1), at(1, 0, 1), 1, 1.5, true},
		{"na diagonal", at(0, 0, 1), at(1, 1, 1), 2, 3, true},
		{"a três quadrados", at(0, 0, 1), at(3, 0, 1), 3, 4.5, false},
		{"o Grande ao lado, pelo corpo e não pelo canto", at(0, 0, 2), at(2, 1, 1), 1, 1.5, true},
		{"o Grande alcança dois quadrados", at(0, 0, 2), at(3, 0, 1), 2, 3, true},
		{"o Médio não alcança o Grande a dois quadrados", at(3, 0, 1), at(0, 0, 2), 2, 3, false},
	} {
		got := ApartBetween(tt.from, tt.to)
		if got.Squares != tt.squares || got.Metres != tt.metres || got.Reaches != tt.reaches {
			t.Errorf("%s: %d quadrados (%.1fm), alcança %t; want %d (%.1fm), %t",
				tt.name, got.Squares, got.Metres, got.Reaches, tt.squares, tt.metres, tt.reaches)
		}
	}
}
