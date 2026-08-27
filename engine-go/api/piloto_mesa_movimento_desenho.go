package api

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"t20engine/engine"
	"t20engine/tabuleiro"
)

// A SETA do movimento proposto (ALE-203, item 4 da lista do dono).
//
// As palavras dele: *"ao soltar a peça, ela vai ser renderizada no lugar que foi
// solta e o início mostra a peça transparente para marcar o início do movimento.
// A seta da régua conecta os dois pontos."*
//
// Ela dobra nas PARADAS e não em cada casa do caminho, e a diferença é de
// natureza: a trilha pintada diz por quais quadrados a peça passa — que é a
// CONTA do custo, com a diagonal do livro (p238) dentro dela —, e a seta diz o
// GESTO, que é onde a pessoa clicou. Desenhar a seta casa a casa faria dela uma
// segunda trilha, mais grossa, contando a mesma coisa por cima.

// recuoDaSeta é o quanto a ponta PARA antes do centro da última casa, em
// quadrados.
//
// Meio quadrado põe a ponta na BORDA da casa de destino, e é o que faz a seta
// apontar para a peça em vez de riscá-la pelo meio: a peça pousou ali (é o item
// 4), e uma ponta no centro cairia em cima do monograma. Toda perna tem pelo
// menos um quadrado, então o recuo nunca inverte a última.
const recuoDaSeta = 0.5

// oFioDoMovimento escreve o `d` da seta, do CENTRO da origem até a borda da casa
// de destino, dobrando em cada parada.
//
// Devolve "" com menos de duas paradas: uma seta de um ponto só não liga nada, e
// um `d` vazio é o jeito de o `<path>` não desenhar sem um `data-show` a mais.
//
// @example oFioDoMovimento([]engine.Square{{}, {X: 3}}) // "M 0.5 0.5 L 3 0.5"
func oFioDoMovimento(paradas []engine.Square) string {
	if len(paradas) < 2 {
		return ""
	}
	pontos := make([][2]float64, len(paradas))
	for i, q := range paradas {
		pontos[i] = [2]float64{float64(q.X) + 0.5, float64(q.Y) + 0.5}
	}
	ultimo := len(pontos) - 1
	dx := pontos[ultimo][0] - pontos[ultimo-1][0]
	dy := pontos[ultimo][1] - pontos[ultimo-1][1]
	if perna := math.Hypot(dx, dy); perna > recuoDaSeta {
		pontos[ultimo][0] -= dx / perna * recuoDaSeta
		pontos[ultimo][1] -= dy / perna * recuoDaSeta
	}
	var b strings.Builder
	for i, p := range pontos {
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

// asDobrasDoMovimento são os pontos onde a seta DOBRA: as paradas quando se sabe
// onde a pessoa clicou, e as duas pontas do caminho quando não.
//
// O `Stops` nulo é um valor legítimo — o `ProposeMove` deixa o caminho pronto
// sem passar por paradas (ver o `PendingMove`) —, e nesse caso a seta é a reta
// entre o começo e o fim. Deduzir as dobras do `Path` não é possível: um trecho
// legítimo já dobra sozinho, porque a diagonal vem primeiro.
func asDobrasDoMovimento(p *tabuleiro.PendingMove) []engine.Square {
	if len(p.Stops) >= 2 {
		return p.Stops
	}
	if len(p.Path) < 2 {
		return nil
	}
	return []engine.Square{p.Path[0], p.Path[len(p.Path)-1]}
}
