package engine

// Alcance no mapa de batalha (Tormenta 20, p224).
//
// O livro dá as faixas em metros e a conversão para quadrados já feita: curto
// 9m (6 quadrados), médio 30m (20 quadrados), longo 90m (60 quadrados). Pessoal,
// Toque e Ilimitado não são distâncias de mapa — o primeiro não sai do
// personagem, o segundo depende do alcance natural da criatura (que varia com o
// tamanho) e o terceiro não tem régua.
const (
	ShortRangeSquares  = 6
	MediumRangeSquares = 20
	LongRangeSquares   = 60
)

// RangeBand é a faixa do livro em que uma distância cai.
type RangeBand string

const (
	RangeShort  RangeBand = "curto"
	RangeMedium RangeBand = "médio"
	RangeLong   RangeBand = "longo"
	// RangeBeyond é "além do longo": não existe faixa entre o longo e o
	// ilimitado, e o ilimitado não se mede na mesa.
	RangeBeyond RangeBand = "além"
)

// Measurement é a leitura da régua entre dois quadrados.
type Measurement struct {
	Squares int `json:"squares"`
	// Metres existe para a tela não ter uma segunda conversão: quadrado é a
	// unidade da regra (p236) e metro é a unidade da conversa na mesa.
	Metres float64   `json:"metres"`
	Band   RangeBand `json:"band"`
}

// RangeSquares mede a distância entre dois quadrados, em quadrados.
//
// Usa a MESMA régua do movimento — a diagonal custa o dobro (p238) —, e isso é
// DECISÃO DE MESA, não texto do livro: a p224 dá o alcance em metros e a p236
// dá a conversão, mas nenhuma das duas diz como se conta uma diagonal para
// efeito de alcance. Duas réguas diferentes no mesmo mapa seriam duas verdades
// sobre a mesma distância, e a tela já desenha o losango do movimento com esta
// aqui.
//
// Com a diagonal valendo dois, o caminho mais barato entre dois quadrados custa
// exatamente Δx + Δy: usa-se a diagonal enquanto os dois eixos precisam andar
// (dois pelo preço de um passo, custando dois) e a ortogonal no resto.
//
// Terreno difícil NÃO entra: ele encarece o ANDAR (p238), e uma flecha não
// atravessa o brejo mais devagar.
func RangeSquares(from, to Square) int {
	return abs(to.X-from.X) + abs(to.Y-from.Y)
}

// BandFor diz em que faixa do livro a distância cai (p224).
func BandFor(squares int) RangeBand {
	switch {
	case squares <= ShortRangeSquares:
		return RangeShort
	case squares <= MediumRangeSquares:
		return RangeMedium
	case squares <= LongRangeSquares:
		return RangeLong
	default:
		return RangeBeyond
	}
}

// Measure é a régua sob demanda da mesa: "dá para acertar daqui?".
//
// @example Measure(Square{0, 0}, Square{3, 0}) // 3 quadrados, 4,5m, curto
func Measure(from, to Square) Measurement {
	squares := RangeSquares(from, to)
	return Measurement{
		Squares: squares,
		Metres:  float64(squares) * SquareMetres,
		Band:    BandFor(squares),
	}
}
