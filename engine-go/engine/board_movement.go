package engine

import (
	"fmt"
	"sort"
)

// Movimento no mapa de batalha (Tormenta 20, p236-238).
//
// A conta é feita em QUADRADOS inteiros, nunca em metros. O livro autoriza
// ("para simplificar, você pode se referir a distâncias em 'quadrados' (de
// 1,5m)", p236) e é o que mantém a aritmética exata: diagonal e terreno difícil
// são multiplicadores inteiros sobre um passo, enquanto em metros a mesma conta
// vira 1,5 / 3,0 / 4,5 em ponto flutuante, com comparação de orçamento sujeita a
// epsilon. Metro é coisa de tela.

// SquareMetres é quanto um quadrado vale no mundo (T20 p236). Existe como
// constante para a conversão da tela sair de um lugar só.
const SquareMetres = 1.5

// Square é um quadrado do mapa, em coordenadas inteiras.
type Square struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// MoveTerrain descreve o chão para efeito de custo. Esparso de propósito: um
// mapa 40×30 tem 1200 células e quase todas são chão limpo.
type MoveTerrain struct {
	Difficult map[Square]bool
}

// MoveCost é o veredito sobre UM caminho.
type MoveCost struct {
	// Squares é o custo total em quadrados (não o número de passos: uma
	// diagonal é um passo que custa dois).
	Squares int  `json:"squares"`
	Budget  int  `json:"budget"`
	Legal   bool `json:"legal"`
	// StoppedAt é o índice do passo que estourou o orçamento, ou -1 se coube.
	// A tela precisa dele para pintar o trecho recusado em vez de recusar o
	// caminho inteiro: quem estourou quer ver ONDE estourou.
	StoppedAt int    `json:"stoppedAt"`
	Reason    string `json:"reason,omitempty"`
}

// stepCost devolve o custo de UM passo entre quadrados adjacentes, em quadrados.
//
// Ortogonal custa 1 (1,5m). Diagonal custa 2 — "mover-se na diagonal custa o
// dobro… andar 1,5m (1 quadrado) na diagonal conta como 3m (2 quadrados)"
// (p238). NÃO existe a alternância 1-2-1 de outros d20: o livro diz "o dobro",
// sem exceção declarada.
//
// Entrar em terreno difícil DOBRA o custo do passo — "gasta 3m de deslocamento
// por quadrado, em vez de 1,5m" (p238). O livro enuncia as duas dobras em
// frases separadas e NUNCA as compõe; a leitura desta casa é multiplicativa, ou
// seja, uma diagonal em terreno difícil custa 4 quadrados (6m). É decisão de
// mesa registrada, não texto do livro.
func stepCost(from, to Square, terrain MoveTerrain) (int, error) {
	dx, dy := abs(to.X-from.X), abs(to.Y-from.Y)
	if dx > 1 || dy > 1 || (dx == 0 && dy == 0) {
		return 0, fmt.Errorf("passo inválido de (%d,%d) para (%d,%d): o caminho tem de ser quadrado a quadrado", from.X, from.Y, to.X, to.Y)
	}
	cost := 1
	if dx == 1 && dy == 1 {
		cost = 2
	}
	if terrain.Difficult[to] {
		cost *= 2
	}
	return cost, nil
}

// PathCost soma o custo de um caminho, passo a passo, em quadrados (T20 p238).
//
// `budgetSquares` negativo significa SEM orçamento — a cena fora de combate não
// tem vez nem deslocamento de turno, e ali a régua mede sem recusar.
//
//	PathCost([]Square{{0, 0}, {1, 1}}, MoveTerrain{}, 6)
//	  → MoveCost{Squares: 2, Budget: 6, Legal: true, StoppedAt: -1}
func PathCost(path []Square, terrain MoveTerrain, budgetSquares int) MoveCost {
	out := MoveCost{Budget: budgetSquares, Legal: true, StoppedAt: -1}
	for i := 1; i < len(path); i++ {
		cost, err := stepCost(path[i-1], path[i], terrain)
		if err != nil {
			return MoveCost{Budget: budgetSquares, Legal: false, StoppedAt: i, Reason: err.Error()}
		}
		out.Squares += cost
		if budgetSquares >= 0 && out.Squares > budgetSquares && out.Legal {
			out.Legal = false
			out.StoppedAt = i
			out.Reason = fmt.Sprintf("o caminho custa %d quadrados e o deslocamento alcança %d", out.Squares, budgetSquares)
		}
	}
	return out
}

// SquaresForDisplacement converte deslocamento em metros para orçamento em
// quadrados (T20 p106: "seu deslocamento é 9 metros (6 quadrados no mapa)").
//
// TRUNCA: 10m não compra um sétimo quadrado, porque o sétimo custa 1,5m
// inteiros. Deslocamento negativo (sobrecarga acima do que a criatura tem) vira
// zero — quem não anda, não anda; não anda para trás.
//
//	SquaresForDisplacement(9) // → 6
func SquaresForDisplacement(metres float64) int {
	if metres <= 0 {
		return 0
	}
	return int(metres / SquareMetres)
}

// FootprintForSize devolve o LADO do token em quadrados (T20 p107, Tab. 1-21):
// Minúsculo, Pequeno e Médio ocupam 1; Grande 2; Enorme 3; Colossal 6. Não
// existe 4 nem 5 na tabela, e é por isso que isto é uma tabela e não uma conta.
//
// Aceita as duas grafias que circulam no projeto — o bestiário guarda "medio" e
// a ficha guarda "Médio" —, porque normalizar em um lugar só é mais barato que
// descobrir a divergência na tela.
//
//	FootprintForSize("Grande") // → 2
func FootprintForSize(size string) int {
	switch normalizeSizeKey(size) {
	case "grande":
		return 2
	case "enorme":
		return 3
	case "colossal":
		return 6
	default:
		return 1
	}
}

// normalizeSizeKey tira acento e caixa do nome do tamanho.
func normalizeSizeKey(size string) string {
	lowered := []rune{}
	for _, r := range size {
		switch r {
		case 'é', 'É', 'ê', 'Ê':
			r = 'e'
		case 'í', 'Í':
			r = 'i'
		case 'ú', 'Ú':
			r = 'u'
		case 'ç', 'Ç':
			r = 'c'
		}
		if r >= 'A' && r <= 'Z' {
			r += 'a' - 'A'
		}
		lowered = append(lowered, r)
	}
	return string(lowered)
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// ReachableSquares devolve todo quadrado que a criatura ALCANÇA com o orçamento
// dado, partindo de `from` (T20 p238).
//
// Existe para a tela poder ACENDER as casas alcançáveis em vez de fazer a mesa
// contar quadrado — e mora aqui, e não no cliente, porque a conta é a mesma
// regra do livro: quem escrevesse uma cópia em TS acabaria com duas verdades
// sobre a diagonal, que foi exatamente o que a ALE-104 apagou.
//
// O resultado NÃO é um quadrado de lado `budget`: como a diagonal custa o
// dobro, o alcance é um LOSANGO — com 6 quadrados de deslocamento dá para
// andar 6 em linha reta e só 3 na diagonal. É a forma que ensina a regra sem
// ninguém explicar.
//
// Dijkstra e não um losango calculado de cabeça: o terreno difícil (fatia do
// mapa) muda o custo por casa, e a busca já vai estar pronta para ele. A área
// é limitada pelo orçamento, então o custo é O(budget²).
//
//	ReachableSquares(Square{0, 0}, 2, MoveTerrain{}) // → 12 quadrados (o losango de raio 2)
func ReachableSquares(from Square, budget int, terrain MoveTerrain) []Square {
	if budget <= 0 {
		return []Square{}
	}
	cost := map[Square]int{from: 0}
	frontier := []Square{from}
	for len(frontier) > 0 {
		current := frontier[0]
		frontier = frontier[1:]
		for _, next := range neighbours(current) {
			step, err := stepCost(current, next, terrain)
			if err != nil {
				continue
			}
			total := cost[current] + step
			if total > budget {
				continue
			}
			if seen, ok := cost[next]; ok && seen <= total {
				continue
			}
			cost[next] = total
			frontier = append(frontier, next)
		}
	}
	out := make([]Square, 0, len(cost))
	for square := range cost {
		if square != from {
			out = append(out, square)
		}
	}
	sortSquares(out)
	return out
}

func neighbours(s Square) []Square {
	out := make([]Square, 0, 8)
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			out = append(out, Square{X: s.X + dx, Y: s.Y + dy})
		}
	}
	return out
}

// sortSquares dá ordem estável à saída: um mapa em Go itera fora de ordem, e um
// resultado que muda de ordem a cada chamada faria a lista do cliente
// reconciliar tudo sem nada ter mudado.
func sortSquares(list []Square) {
	sort.Slice(list, func(i, j int) bool {
		if list[i].Y != list[j].Y {
			return list[i].Y < list[j].Y
		}
		return list[i].X < list[j].X
	})
}
