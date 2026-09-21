package table

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"t20engine/domain/board"
	"t20engine/domain/engine"
)

// A SETA do movimento proposto: a peça transparente marca o começo, a peça
// sólida o lugar onde foi solta, e a seta liga os dois pontos.
//
// Ela dobra nas PARADAS e não em cada casa do caminho, e a diferença é de
// natureza: a trilha pintada diz por quais quadrados a peça passa — que é a
// CONTA do custo, com a diagonal do livro (p238) dentro dela —, e a seta diz o
// GESTO, que é onde a pessoa clicou. Desenhar a seta casa a casa faria dela uma
// segunda trilha, mais grossa, contando a mesma coisa por cima.
//
// Ela carrega NÚMERO e COR — dourado até o deslocamento, azul enquanto gasta a
// ação padrão como segundo movimento, vermelho depois —, e as duas coisas contam
// a MESMA grandeza: o CUSTO em quadrados, convertido em metros. É isso que faz o
// número explicar onde cada cor começa.
//
// Os dois limiares saem do livro (T20 p233): uma ação de movimento, e a ação
// padrão trocada por uma segunda. Não há terceira, então o vermelho é
// literalmente "não cabe no turno" — e não "o servidor vai recusar".

// recuoDaSeta é o quanto a ponta PARA antes do centro da última casa, em
// quadrados.
//
// Meio quadrado põe a ponta na BORDA da casa de destino, e é o que faz a seta
// apontar para a peça em vez de riscá-la pelo meio: a peça pousou ali (é o item
// 4), e uma ponta no centro cairia em cima do monogram. Toda perna tem pelo
// menos um quadrado, então o recuo nunca inverte a última.
const recuoDaSeta = 0.5

// moveLeg é o RÓTULO de uma perna e o ponto onde ele pousa.
//
// O rótulo diz o que a perna CUSTA e não a distância geométrica dela (decisão do
// dono): sobre terreno difícil a régua diz 4,5m e a seta diz 9,0m para a mesma
// linha. A divergência é o preço, e o que ela compra é o metro do rótulo ser o
// MESMO metro do deslocamento — sem isso os números da seta não explicariam onde
// o vermelho começa.
type moveLeg struct {
	Label string
	// MidX e MeioY são o meio da perna em QUADRADOS, com sinal: o plano não tem
	// bordas. Quem os põe em pixel é a tela, porque o rótulo mora FORA do grupo
	// que escala.
	MidX float64
	MidY float64
}

// moveLegs escreve o rótulo de cada perna, no meio dela.
//
// O metro sai do `metersLeg`, que é o mesmo formatador dos rótulos da
// régua: as duas ferramentas põem número sobre uma linha, e dois formatadores
// para a mesma frase é como nasce "9,0m" numa e "9.0 m" na outra.
func moveLegs(folds []engine.Square, costs []int) []moveLeg {
	centers := foldsCenters(folds)
	legs := make([]moveLeg, 0, len(costs))
	for i, cost := range costs {
		middle := entreOsPontos(centers[i], centers[i+1], 0.5)
		legs = append(legs, moveLeg{
			Label: metersLeg(engine.Measurement{
				Squares: cost, Metres: float64(cost) * engine.SquareMetres,
			}),
			MidX: middle[0], MidY: middle[1],
		})
	}
	return legs
}

// legsCosts mede cada perna em quadrados, pela régua do MOVIMENTO.
//
// Sem orçamento na medição (-1): aqui se quer o custo INTEIRO da perna, e um
// teto faria o `PathCost` parar de somar no lugar errado. Quem confronta o
// orçamento é o `cutSpeed`, logo abaixo.
//
// A soma destes números é o custo total do movimento porque o
// `engine.PathThroughStops` concatena exatamente estes trechos, descartando a
// emenda — a mesma decomposição, medida pela mesma função.
func legsCosts(folds []engine.Square, terrain engine.MoveTerrain) []int {
	costs := make([]int, 0, len(folds))
	for i := 1; i < len(folds); i++ {
		excerpt := engine.PathBetween(folds[i-1], folds[i])
		costs = append(costs, engine.PathCost(excerpt, terrain, -1).Squares)
	}
	return costs
}

// moveWires parte a seta em DOURADO — o que o deslocamento paga — e VERMELHO, o
// que passa dele.
//
// Devolve "" nos dois com menos de duas dobras: uma seta de um ponto só não liga
// nada, e um `d` vazio é o jeito de o `<path>` não desenhar sem um `data-show` a
// mais.
//
// SEM ORÇAMENTO (mestre, ou cena fora de combate: `orcamento < 0`) sai tudo
// dourado. Não há teto, e desenhar um seria INVENTÁ-LO — é o mesmo argumento que
// já governa o desenho do `Alcance`, que também não aparece ali.
//
// @example moveWires([]engine.Square{{}, {X: 3}}, []int{3}, -1) // ouro "M 0.5 0.5 L 3 0.5", resto vazio
func moveWires(folds []engine.Square, costs []int, budget int) (fits, segundo, beyond string) {
	centers := foldsCenters(folds)
	if len(centers) < 2 {
		return "", "", ""
	}
	points := endIndent(centers)
	if budget < 0 {
		return pointsWire(points), "", ""
	}
	// AS DUAS TESOURAS são medidas contra o caminho INTEIRO, e não uma sobre o
	// resto da outra: cortar em cadeia obrigaria a traduzir o índice da segunda
	// para dentro do trecho que a primeira devolveu, e essa aritmética erra em
	// silêncio — a linha continua saindo, só com a cor virando no lugar errado.
	// Aqui os dois índices falam da mesma lista, e `pontos[i1:i2]` é o miolo azul.
	i1, advance1, passesFirst := cutSpeed(costs, budget)
	if !passesFirst {
		return pointsWire(points), "", ""
	}
	cut1 := entreOsPontos(centers[i1-1], centers[i1], advance1)
	i2, advance2, passesBoth := cutSpeed(costs, 2*budget)
	if !passesBoth {
		return pointsWire(pointUntil(points[:i1], cut1)),
			pointsWire(point(cut1, points[i1:])), ""
	}
	cut2 := entreOsPontos(centers[i2-1], centers[i2], advance2)
	return pointsWire(pointUntil(points[:i1], cut1)),
		pointsWire(pointUntil(point(cut1, points[i1:i2]), cut2)),
		pointsWire(point(cut2, points[i2:]))
}

// pointUntil fecha uma polilinha num ponto solto; point a abre num.
//
// Os dois COPIAM em vez de fatiar por cima: as três faixas saem da mesma lista
// de pontos, e um `append` sobre o array de trás compartilhado faria a faixa
// seguinte sobrescrever a anterior — o clássico do slice em Go, e aqui ele
// apareceria como uma cor comendo a outra.
func pointUntil(start [][2]float64, end [2]float64) [][2]float64 {
	return append(append([][2]float64(nil), start...), end)
}

func point(start [2]float64, rest [][2]float64) [][2]float64 {
	return append([][2]float64{start}, rest...)
}

// cutSpeed diz em QUE perna o deslocamento acaba e ONDE dentro dela.
//
// Devolve o índice da dobra que FECHA a perna (ela vai de `dobras[i-1]` a
// `dobras[i]`) e a fração do CUSTO dela já paga, entre 0 e 1.
//
// A cor vira no ponto EXATO em que o orçamento acaba — decisão do dono —, e não
// no começo da perna que estoura: "a distância possível" é um COMPRIMENTO, e
// arredondar por perna inteira mentiria em metros sobre quanto falta encurtar.
//
// **Reparte o CUSTO, e não os passos do caminho.** A tentação é achar o quadrado
// exato em que a peça para e cortar ali, e é o que quebra o desenho: a seta é uma
// RETA entre duas paradas e o caminho no grid não é — numa perna de (-5,0) a
// (11,6) o caminho anda seis diagonais e dez retos, e a peça para FORA da reta. O
// que a reta representa é o que o RÓTULO dela diz, o custo em metros, então quem
// a divide tem de ser o custo; senão os dois se contradizem sobre a mesma linha.
// Quem mostra as casas percorridas é a TRILHA, que é outro desenho.
func cutSpeed(costs []int, budget int) (int, float64, bool) {
	if budget < 0 {
		return 0, 0, false
	}
	spent := 0
	for i, cost := range costs {
		if spent+cost <= budget {
			spent += cost
			continue
		}
		// `cost` é maior que zero aqui: só se chega nesta linha com
		// `gasto+custo > orcamento` e `gasto <= orcamento`, que é o invariante do
		// ramo de cima.
		return i + 1, float64(budget-spent) / float64(cost), true
	}
	return 0, 0, false
}

// foldsCenters põe cada dobra no CENTRO da casa dela.
func foldsCenters(folds []engine.Square) [][2]float64 {
	centers := make([][2]float64, len(folds))
	for i, q := range folds {
		centers[i] = [2]float64{float64(q.X) + 0.5, float64(q.Y) + 0.5}
	}
	return centers
}

// endIndent encolhe a ÚLTIMA perna pelo `recuoDaSeta`, sem tocar no resto.
func endIndent(centers [][2]float64) [][2]float64 {
	points := append([][2]float64(nil), centers...)
	last := len(points) - 1
	dx := points[last][0] - points[last-1][0]
	dy := points[last][1] - points[last-1][1]
	if leg := math.Hypot(dx, dy); leg > recuoDaSeta {
		points[last][0] -= dx / leg * recuoDaSeta
		points[last][1] -= dy / leg * recuoDaSeta
	}
	return points
}

// entreOsPontos é o ponto a uma fração `t` do caminho de `a` até `b`.
func entreOsPontos(a, b [2]float64, t float64) [2]float64 {
	return [2]float64{a[0] + (b[0]-a[0])*t, a[1] + (b[1]-a[1])*t}
}

// pointsWire escreve o `d` de uma polilinha.
func pointsWire(points [][2]float64) string {
	var b strings.Builder
	for i, p := range points {
		if i == 0 {
			b.WriteString("M ")
		} else {
			b.WriteString(" L ")
		}
		fmt.Fprintf(&b, "%s %s", numeroDoFio(p[0]), numeroDoFio(p[1]))
	}
	return b.String()
}

// numeroDoFio escreve o número com três casas e sem zeros à toa.
//
// TRÊS casas porque o recuo da ponta é irracional em toda perna diagonal, e a
// precisão do `float64` inteira punha `3.025658350974743` num atributo que um
// guarda precisa comparar. Um milésimo de quadrado é um centésimo de pixel no
// zoom máximo — nada que o desenho perceba, e a diferença entre um `d` que se lê
// e um que não se lê.
func numeroDoFio(n float64) string {
	return strconv.FormatFloat(math.Round(n*1000)/1000, 'f', -1, 64)
}

// moveFolds são os pontos onde a seta DOBRA: as paradas quando se sabe
// onde a pessoa clicou, e as duas pontas do caminho quando não.
//
// O `Stops` nulo é um valor legítimo — o `ProposeMove` deixa o caminho pronto
// sem passar por paradas (ver o `PendingMove`) —, e nesse caso a seta é a reta
// entre o começo e o fim. Deduzir as dobras do `Path` não é possível: um trecho
// legítimo já dobra sozinho, porque a diagonal vem primeiro.
func moveFolds(p *board.PendingMove) []engine.Square {
	if len(p.Stops) >= 2 {
		return p.Stops
	}
	if len(p.Path) < 2 {
		return nil
	}
	return []engine.Square{p.Path[0], p.Path[len(p.Path)-1]}
}
