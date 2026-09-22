package api

import (
	"fmt"

	"t20engine/domain/board"
)

// AS LEITURAS DO TABULEIRO DEVOLVEM ERRO desde a ALE-375, e estes dois ajudantes
// existem para o caso FALHAR ALTO em vez de seguir com o valor-zero.
//
// Sem eles, um `_` no erro faria o `nil` do tabuleiro querer dizer duas coisas —
// "esta sessão não tem tabuleiro" e "o banco não respondeu" —, que é exatamente
// a confusão que a fatia veio desfazer. Um caso que confundisse as duas mediria
// a ausência do tabuleiro quando o que houve foi a ausência do banco.
//
// # Por que `panic` e não `t.Fatalf`
//
// Porque eles são escritos EMBRULHANDO a chamada — `boardRead(s.boards.Get(…))`
// —, e o Go só aceita repassar uma chamada de vários retornos quando ela
// preenche TODOS os parâmetros de quem a recebe. Um `*testing.T` na frente
// proibiria a forma, e as noventa e duas chamadas teriam de virar duas linhas
// cada.
//
// O `panic` não custa diagnóstico aqui: ele reprova o caso com a frase e com a
// pilha apontando a linha da leitura, que é mais do que um `Fatalf` dá. Quem
// for "consertar" isto para `t.Fatalf` vai ter de reescrever todos os
// chamadores — a forma é a razão.

// boardRead lê UM tabuleiro e falha alto.
func boardRead(b *board.BoardState, err error) *board.BoardState {
	if err != nil {
		panic(fmt.Sprintf("ler o tabuleiro: %v", err))
	}
	return b
}

// boardsRead lê os tabuleiros ABERTOS de uma sessão e falha alto.
func boardsRead(open []*board.BoardState, err error) []*board.BoardState {
	if err != nil {
		panic(fmt.Sprintf("ler os tabuleiros abertos: %v", err))
	}
	return open
}
