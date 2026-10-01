package board

import (
	"t20engine/domain/ecs"
	"t20engine/domain/engine"
)

// O TERRENO VIRA SITUAÇÃO ESPECIAL (ALE-423, p238-239).
//
// Até aqui as quatro espécies alimentavam só o OLHO. O `board_state.go` dizia
// por extenso que três delas "mudam Defesa, chance de falha e ataque, e hoje
// não são consumidos por nada" — este arquivo é o consumidor.
//
// A entidade continua sendo a PEÇA, pelo terceiro motivo diferente: a redação
// pergunta "quem olha daqui enxerga isto?", a ocupação pergunta "que quadrado
// está tomado?", e esta pergunta "que terreno esta peça está pisando?". Os três
// mundos nascem do mesmo `worldOfTheBoard` e divergem nos SISTEMAS.
//
// # O que NÃO está aqui, e é decisão de modelagem já tomada
//
// O livro dá cobertura a quem está "atrás de algo que bloqueia o ataque dos
// inimigos", e tem um método de linha-entre-cantos para decidir (p239). O
// tabuleiro modela cobertura como ESPÉCIE DE CASA, e a decisão é anterior a
// esta fatia — está escrita no `BoardState.Cover`: "+5 na Defesa de quem está
// NELA". Uma trincheira é exatamente isso; a árvore entre dois é que fica de
// fora, e ela precisaria da linha-entre-cantos.
//
// E o FLANQUEIO não está aqui porque a peça não sabe quem é ALIADA — ela tem
// `Kind` ("character" | "npc" | "object"), que é outra pergunta: um NPC aliado
// do grupo flanquearia com um personagem, e por `Kind` não flanqueia.

// Os três componentes abaixo são a espécie de terreno que o corpo da peça toca.
//
// Um componente por espécie, e não um campo com a espécie dentro: uma casa pode
// ser difícil E camuflagem ao mesmo tempo (folhagens, p267), e uma peça Grande
// cobre quatro casas que podem ter espécies diferentes.
type standsOnCover struct{}
type standsOnConcealment struct{}
type standsOnHigherGround struct{}

// readTheTerrainUnderEachToken marca cada peça com o que ela está pisando.
//
// BASTA UM QUADRADO do corpo: uma peça Grande com metade na trincheira está
// coberta. O livro não decide isso — ele não fala de peças de dois quadrados em
// terreno misto —, e a leitura generosa é a que não faz a cobertura sumir
// quando o mestre amplia a criatura.
func readTheTerrainUnderEachToken(b *BoardState) ecs.System {
	marks := []struct {
		squares []engine.Square
		mark    func(*ecs.World, ecs.Entity)
	}{
		{b.Cover, func(w *ecs.World, e ecs.Entity) { ecs.Set(w, e, standsOnCover{}) }},
		{b.Concealment, func(w *ecs.World, e ecs.Entity) { ecs.Set(w, e, standsOnConcealment{}) }},
		{b.Elevated, func(w *ecs.World, e ecs.Entity) { ecs.Set(w, e, standsOnHigherGround{}) }},
	}
	return func(w *ecs.World) {
		ecs.Each(w, func(e ecs.Entity, r body) {
			for _, kind := range marks {
				for _, square := range kind.squares {
					if r.covers(square.X, square.Y) {
						kind.mark(w, e)
						break
					}
				}
			}
		})
	}
}

// SituationsBetween são as linhas da Tabela 5-3 que o TABULEIRO produz para
// este ataque — as outras chegam da ficha, como condição.
//
// Devolve vazio quando qualquer uma das duas peças não está no mapa: uma mesa
// sem tabuleiro ataca sem situação nenhuma, e é esse o caso comum.
//
// @example board.SituationsBetween(b, "entry-do-atacante", "entry-do-alvo")
func SituationsBetween(b *BoardState, attackerEntry, targetEntry string) []engine.SpecialSituation {
	if b == nil || attackerEntry == "" || targetEntry == "" {
		return nil
	}
	w := worldOfTheBoard(b)
	ecs.Run(w, giveEveryTokenABody(b), bindEachTokenToItsQueueRow(b), readTheTerrainUnderEachToken(b))

	out := []engine.SpecialSituation{}
	ecs.Each2(w, func(e ecs.Entity, bound boundTo, _ body) {
		switch bound.EntryID {
		case attackerEntry:
			// O ELEVADO é a única espécie que beneficia quem está NELA: "+2 no
			// ataque de quem ataca DE LÁ" (p239).
			if _, alto := ecs.Get[standsOnHigherGround](w, e); alto {
				out = append(out, engine.AttackerOnHigherGround)
			}
		case targetEntry:
			if _, coberto := ecs.Get[standsOnCover](w, e); coberto {
				out = append(out, engine.TargetUnderLightCover)
			}
			if _, escondido := ecs.Get[standsOnConcealment](w, e); escondido {
				out = append(out, engine.TargetUnderLightConcealment)
			}
		}
	})
	return out
}
