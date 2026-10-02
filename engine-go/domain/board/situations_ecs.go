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

// AttackTargetOnTheBoard é quem está sendo atacado, do ponto de vista do mapa.
//
// DOIS campos e não um, porque são dois endereços diferentes para a mesma
// pergunta: criatura se endereça pela LINHA da fila, e objeto pela PEÇA — ele
// não tem turno, então não tem linha. Exatamente um vem preenchido.
//
// Uma struct e não um terceiro parâmetro `string`: três strings em sequência
// numa chamada é onde se troca a ordem de duas delas sem o compilador reclamar.
type AttackTargetOnTheBoard struct {
	EntryID string
	TokenID string
}

// SituationsBetween são as linhas da Tabela 5-3 que o TABULEIRO produz para
// este ataque — as outras chegam da ficha, como condição.
//
// Devolve vazio quando qualquer uma das duas peças não está no mapa: uma mesa
// sem tabuleiro ataca sem situação nenhuma, e é esse o caso comum.
//
// @example board.SituationsBetween(b, "entry-do-atacante", board.AttackTargetOnTheBoard{EntryID: "entry-do-alvo"})
func SituationsBetween(
	b *BoardState, attackerEntry string, target AttackTargetOnTheBoard,
) []engine.SpecialSituation {
	if b == nil || attackerEntry == "" || (target.EntryID == "" && target.TokenID == "") {
		return nil
	}
	w := worldOfTheBoard(b)
	ecs.Run(w, giveEveryTokenABody(b), bindEachTokenToItsQueueRow(b), readTheTerrainUnderEachToken(b))

	out := []engine.SpecialSituation{}
	// A VARREDURA É POR `tokenAt` e não por `boundTo`, e isso é a mudança: o
	// `boundTo` só existe para peça com linha na fila, e um OBJETO não tem — ele
	// nunca entrava neste laço, então uma porta atrás de uma trincheira era
	// medida sem a cobertura, calado.
	ecs.Each2(w, func(e ecs.Entity, at tokenAt, _ body) {
		peca := b.Tokens[at.Index]
		if bound, tem := ecs.Get[boundTo](w, e); tem && bound.EntryID == attackerEntry {
			// O ELEVADO é a única espécie que beneficia quem está NELA: "+2 no
			// ataque de quem ataca DE LÁ" (p239).
			if _, alto := ecs.Get[standsOnHigherGround](w, e); alto {
				out = append(out, engine.AttackerOnHigherGround)
			}
			return
		}
		if !isTheTarget(w, e, peca, target) {
			return
		}
		if _, coberto := ecs.Get[standsOnCover](w, e); coberto {
			out = append(out, engine.TargetUnderLightCover)
		}
		if _, escondido := ecs.Get[standsOnConcealment](w, e); escondido {
			out = append(out, engine.TargetUnderLightConcealment)
		}
		// "Se o objeto estiver em movimento, recebe +5 na Defesa" (p239). É o
		// único estado da Tabela 5-3 que o MESTRE liga à mão: o tabuleiro não
		// consegue derivá-lo, porque uma peça arrastada está sendo POSTA noutra
		// casa, e isso não é estar em movimento durante o ataque.
		if peca.Moving && peca.HasObjectStats() {
			out = append(out, engine.TargetObjectInMotion)
		}
	})
	return out
}

// isTheTarget casa a peça com o endereço que chegou — a linha ou a peça.
func isTheTarget(w *ecs.World, e ecs.Entity, peca BoardToken, target AttackTargetOnTheBoard) bool {
	if target.TokenID != "" {
		return peca.ID == target.TokenID
	}
	bound, tem := ecs.Get[boundTo](w, e)
	return tem && bound.EntryID == target.EntryID
}
