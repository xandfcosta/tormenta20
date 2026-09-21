package table

import (
	"testing"

	"t20engine/domain/engine"
)

// Os guardas do DESFAZER UMA PARADA na Mesa (ALE-269, item 10).
//
// A régua e a autorização continuam com dono: o custo é do `engine` e o
// `assertMovable` é do `tabuleiro`. O que se prende aqui é o que só existe desde
// esta fatia — que as PARADAS são lembradas, que desfazer tira exatamente a
// última e recalcula o custo, e que a última parada de todas vira cancelar.

// boardStops lê as paradas guardadas no provisório.
// TestThePathDoesNotLetTheStopsBeUncovered — a razão de o campo existir.
//
// Este é o teste que justifica ter acrescentado estado ao provisório em vez de
// deduzir a última perna do caminho, e ele mede a AMBIGUIDADE diretamente: um
// trecho reto de (0,0) a (2,2) e um trecho de (0,0) a (1,1) emendado com outro
// até (2,2) produzem o MESMO caminho, quadrado a quadrado.
//
// Enquanto isso for verdade, "cortar o fim do caminho" é um palpite sobre o
// movimento que a mesa está vendo — e a lista de paradas é a única resposta.
func TestThePathDoesNotLetTheStopsBeUncovered(t *testing.T) {
	direct := engine.PathThroughStops([]engine.Square{{X: 0, Y: 0}, {X: 2, Y: 2}})
	withStop := engine.PathThroughStops([]engine.Square{{X: 0, Y: 0}, {X: 1, Y: 1}, {X: 2, Y: 2}})
	if len(direct) != len(withStop) {
		t.Fatalf("os dois caminhos têm tamanhos diferentes (%d e %d) — a premissa mudou", len(direct), len(withStop))
	}
	for i := range direct {
		if direct[i] != withStop[i] {
			t.Fatalf("os caminhos divergem no quadrado %d — a premissa mudou", i)
		}
	}
}
