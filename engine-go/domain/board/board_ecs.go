package board

import "t20engine/domain/ecs"

// A REDAÇÃO DO TABULEIRO EM SISTEMAS: aqui a ENTIDADE é a COISA DESENHADA
// (ALE-413).
//
// Peça e marcador viram a mesma espécie de entidade porque a pergunta da
// redação é a mesma para as duas — "quem olha daqui enxerga isto?" —, e a
// resposta é a mesma: o que o mestre escondeu SOME INTEIRO. Uma peça "presente
// porém anônima" entregaria a emboscada do mesmo jeito.
//
// # Por que é AQUI que o `Despawn` volta
//
// O `domain/ecs` perdeu o ciclo de vida na ALE-381 por falta de chamador, e o
// comentário do `World` deixou escrito onde ele voltaria: *"no dia em que o
// tabuleiro entrar: lá a peça sai do mapa de verdade"*. É este o dia, e a
// redação é o melhor caso possível — o modo de falha que o `Despawn` descreve
// (matar a entidade e deixar os componentes, com a consulta continuando a
// visitar) aqui não é um número errado na ficha: é a peça escondida chegando à
// tela de quem não devia vê-la.
//
// # O mundo é EFÊMERO, e o `BoardState` continua sendo a verdade
//
// Monta, roda, extrai — é a decisão da ALE-413 para todo o combate. O estado
// gravado e o que viaja no fio não mudam, e por isso os guardas de regime e a
// redação continuam valendo sem uma linha tocada.

// drawnToken e drawnMarker guardam o ÍNDICE no `BoardState`, e não uma cópia.
//
// O índice basta porque o mundo nasce e morre dentro de uma chamada, e a cópia
// duplicaria a peça inteira para devolvê-la igual. São dois componentes e não
// um com discriminador porque a leitura de volta pergunta por espécie, e é
// assim que o ECS separa: quem tem `drawnToken` é peça.
type drawnToken struct{ At int }
type drawnMarker struct{ At int }

// tokenWorldOf monta o mundo de uma redação: uma entidade por peça e uma por
// marcador, na ORDEM em que elas estão no tabuleiro.
//
// A ordem é a do fio, e o `ecs.World` varre por inserção justamente para isto —
// é a propriedade em volta da qual o núcleo foi desenhado, porque `map` em Go
// não a tem.
func tokenWorldOf(b *BoardState) *ecs.World {
	w := ecs.NewWorld()
	for at := range b.Tokens {
		ecs.Set(w, w.Spawn(), drawnToken{At: at})
	}
	for at := range b.Markers {
		ecs.Set(w, w.Spawn(), drawnMarker{At: at})
	}
	return w
}

// concealWhatTheMasterHid mata quem o mestre escondeu.
//
// MATA e não marca: uma tag deixaria a entidade no mundo, e quem lesse a lista
// sem perguntar pela tag devolveria a peça escondida. O `Despawn` apaga os
// componentes, então a consulta seguinte não a encontra de jeito nenhum — a
// diferença entre um descuido que vaza e um que não compila.
func concealWhatTheMasterHid(b *BoardState) ecs.System {
	return func(w *ecs.World) {
		ecs.Each(w, func(e ecs.Entity, drawn drawnToken) {
			if b.Tokens[drawn.At].Hidden {
				w.Despawn(e)
			}
		})
		ecs.Each(w, func(e ecs.Entity, drawn drawnMarker) {
			if b.Markers[drawn.At].Hidden {
				w.Despawn(e)
			}
		})
	}
}

// redactBoardForPlayers apaga da cópia do jogador o que o mestre escondeu.
//
// Some a coisa INTEIRA, e essa é a assimetria deliberada em relação ao
// `hpHidden` da iniciativa, onde a linha fica sem os números: aqui a EXISTÊNCIA
// é a informação.
func redactBoardForPlayers(b *BoardState) *BoardState {
	w := tokenWorldOf(b)
	ecs.Run(w, concealWhatTheMasterHid(b))

	out := *b
	out.Tokens = make([]BoardToken, 0, len(b.Tokens))
	ecs.Each(w, func(_ ecs.Entity, drawn drawnToken) {
		out.Tokens = append(out.Tokens, b.Tokens[drawn.At])
	})
	out.Markers = make([]BoardMarker, 0, len(b.Markers))
	ecs.Each(w, func(_ ecs.Entity, drawn drawnMarker) {
		out.Markers = append(out.Markers, b.Markers[drawn.At])
	})

	// O PROVISÓRIO DE QUEM NÃO SOBROU morre junto: um caminho desenhado saindo
	// do nada é a peça sem o círculo, e entregaria a emboscada por outro
	// caminho.
	//
	// A pergunta é "sobrou?" e não "estava escondida?", e as duas não são a
	// mesma: a segunda deixaria passar um provisório apontando para peça que
	// nem existe. Hoje não há como chegar nesse estado — o `RemoveToken` limpa
	// o provisório junto —, e perguntar pelo sobrevivente é a forma que
	// continua certa se um dia houver.
	if out.Pending != nil && !hasTokenWithID(out.Tokens, out.Pending.TokenID) {
		out.Pending = nil
	}
	return &out
}

func hasTokenWithID(tokens []BoardToken, tokenID string) bool {
	for i := range tokens {
		if tokens[i].ID == tokenID {
			return true
		}
	}
	return false
}
