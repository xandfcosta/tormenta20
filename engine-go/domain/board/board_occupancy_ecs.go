package board

import (
	"t20engine/domain/ecs"
	"t20engine/domain/engine"
	"t20engine/domain/live"
)

// A OCUPAÇÃO DO PLANO EM SISTEMAS (ALE-413).
//
// A entidade continua sendo a PEÇA — o mesmo `worldOfTheBoard` da redação —, e
// o que muda é a pergunta: "que quadrado está tomado, e quem já tem peça na
// mesa?". Ela se responde com dois componentes que a redação não precisa.
//
// # O CORPO vira componente, e ele estava escrito duas vezes
//
// "A peça ocupa `footprint × footprint` quadrados a partir da âncora" é a mesma
// conta no `occupied` e no `TokensInRectangle`, e nos dois com o mesmo
// `if footprint <= 0 { footprint = 1 }` na frente. Com ela num sistema, quem
// pergunta quadrado lê um retângulo pronto e a regra mora num lugar.
//
// # Montar UMA vez por gesto é o ponto, e não um detalhe
//
// O `freeSpotNear` varre anéis até o limite do plano chamando `occupied` a cada
// sonda, e cada chamada refazia a conta de corpo de TODAS as peças. Aqui o
// mundo nasce uma vez no começo do gesto e as sondas leem o que já está escrito
// — e a peça que o `PopulateBoard` acabou de pôr entra no mundo junto, senão a
// próxima nasceria em cima dela.

// body é o retângulo que a peça ocupa, em quadrados, já resolvido.
//
// Fechado nas duas pontas: `X1` e `Y1` são o último quadrado DENTRO do corpo, e
// não o primeiro fora. A forma vem do livro, que descreve tamanho em quadrados
// ocupados — uma Colossal ocupa 6×6 (p107) —, e a outra convenção obrigaria um
// `-1` em toda comparação.
type body struct{ X0, Y0, X1, Y1 int }

// boundTo é a linha da fila que a peça representa. Quem não tem o componente é
// peça AVULSA — porta, baú, barril —, e a distinção é a mesma que o `*string`
// com `omitempty` carrega no fio.
type boundTo struct{ EntryID string }

func (r body) covers(x, y int) bool {
	return x >= r.X0 && x <= r.X1 && y >= r.Y0 && y <= r.Y1
}

func (r body) touches(other body) bool {
	return r.X0 <= other.X1 && r.X1 >= other.X0 && r.Y0 <= other.Y1 && r.Y1 >= other.Y0
}

// giveEveryTokenABody escreve o corpo de cada peça.
//
// O footprint zerado vira 1 aqui e em nenhum outro lugar: peça antiga gravada
// sem o campo existe no banco, e deixar o zero passar faria o corpo dela ser
// vazio — ela deixaria de ocupar quadrado nenhum e duas nasceriam em cima.
func giveEveryTokenABody(b *BoardState) ecs.System {
	return func(w *ecs.World) {
		ecs.Each(w, func(e ecs.Entity, at tokenAt) {
			t := b.Tokens[at.Index]
			side := t.Footprint
			if side <= 0 {
				side = 1
			}
			ecs.Set(w, e, body{X0: t.X, Y0: t.Y, X1: t.X + side - 1, Y1: t.Y + side - 1})
		})
	}
}

// bindEachTokenToItsQueueRow escreve o vínculo de quem tem um.
func bindEachTokenToItsQueueRow(b *BoardState) ecs.System {
	return func(w *ecs.World) {
		ecs.Each(w, func(e ecs.Entity, at tokenAt) {
			if entryID := b.Tokens[at.Index].EntryID; entryID != nil {
				ecs.Set(w, e, boundTo{EntryID: *entryID})
			}
		})
	}
}

// occupancyWorldOf é o mundo pronto para as perguntas de quadrado e de fila.
func occupancyWorldOf(b *BoardState) *ecs.World {
	w := worldOfTheBoard(b)
	ecs.Run(w, giveEveryTokenABody(b), bindEachTokenToItsQueueRow(b))
	return w
}

// squareIsTaken diz se alguma peça cobre o quadrado.
func squareIsTaken(w *ecs.World, x, y int) bool {
	taken := false
	ecs.Each(w, func(_ ecs.Entity, r body) {
		if r.covers(x, y) {
			taken = true
		}
	})
	return taken
}

// freeSpotNear devolve o quadrado livre mais próximo, em anéis.
//
// AO LADO do original, e não na fileira de entrada: quem duplica o zumbi que
// está no canto do mapa espera o irmão dele ali do lado, não a dez quadrados de
// distância no lugar combinado onde as peças avulsas nascem.
func freeSpotNear(w *ecs.World, from boardSpot) boardSpot {
	for ring := 1; ring <= boardCoordLimit; ring++ {
		for dy := -ring; dy <= ring; dy++ {
			for dx := -ring; dx <= ring; dx++ {
				if abs(dx) != ring && abs(dy) != ring {
					continue // o miolo já foi visto nos anéis de dentro
				}
				spot := boardSpot{x: from.x + dx, y: from.y + dy}
				if !squareIsTaken(w, spot.x, spot.y) {
					return spot
				}
			}
		}
	}
	return from
}

// spawnTokenBody põe no mundo o corpo de uma peça recém-posta, para a próxima
// não nascer em cima dela.
func spawnTokenBody(w *ecs.World, x, y, side int) {
	if side <= 0 {
		side = 1
	}
	ecs.Set(w, w.Spawn(), body{X0: x, Y0: y, X1: x + side - 1, Y1: y + side - 1})
}

// TokensInRectangle são os ids das peças cujo CORPO toca o retângulo.
//
// O corpo e não a âncora: uma Colossal ocupa 6×6 (p107), e marcá-la só quando o
// laço pega a quina dela faria o mestre desenhar em volta do dragão e não pegar
// o dragão.
//
// Exemplo:
//
//	TokensInRectangle(b, engine.Square{}, engine.Square{X: 5, Y: 5})
//	// → os ids das peças que aparecem no quadrado de (0,0) a (5,5)
func TokensInRectangle(b *BoardState, de, ate engine.Square) []string {
	if b == nil {
		return nil
	}
	laco := body{
		X0: min(de.X, ate.X), Y0: min(de.Y, ate.Y),
		X1: max(de.X, ate.X), Y1: max(de.Y, ate.Y),
	}
	var ids []string
	w := occupancyWorldOf(b)
	ecs.Each2(w, func(_ ecs.Entity, at tokenAt, r body) {
		if r.touches(laco) {
			ids = append(ids, b.Tokens[at.Index].ID)
		}
	})
	return ids
}

// UnbindOrphanTokens desamarra as peças cuja linha da fila não existe mais, e
// devolve quantas foram. A peça FICA no mapa — só o vínculo sai.
//
// # Por que desamarrar, e não remover a peça
//
// Porque a peça sobreviver é o desenho, e não um efeito colateral. "Reiniciar o
// combate" promete na tela que *"a partida CONTINUA no ar"*: o mestre reiniciou
// o COMBATE, não a CENA, e o mapa que ele montou é trabalho dele (decisão do
// dono, ALE-377).
//
// O que não pode sobreviver é o PONTEIRO. Uma peça apontando para uma linha
// morta mente de um jeito silencioso: ela continua se anunciando como
// combatente, com o botão "Atacar" no menu, e o gesto responde 200 sem fazer
// nada nem recusar.
//
// # Ela é a reconciliação das TRÊS fontes
//
// Tirar um combatente da fila, reiniciar o combate e reabrir um lugar do acervo
// noutra sessão deixavam o mesmo estado por três caminhos. Passar o estado da
// fila que VALE AGORA responde aos três — inclusive ao terceiro, com `st` nulo:
// no acervo da campanha não existe fila nenhuma, e nenhum `EntryID` de sessão
// tem sentido lá.
func UnbindOrphanTokens(b *BoardState, st *live.SessionRuntimeState) int {
	if b == nil {
		return 0
	}
	w := occupancyWorldOf(b)
	ecs.Run(w, forgetTheRowsTheQueueDropped(st))

	unbound := 0
	ecs.Each(w, func(e ecs.Entity, at tokenAt) {
		if b.Tokens[at.Index].EntryID == nil {
			return
		}
		if _, ainda := ecs.Get[boundTo](w, e); ainda {
			return
		}
		// O `CharacterID` vai junto: ele é a outra metade do mesmo vínculo, e
		// uma peça que diz ter ficha sem ter linha desenha barra de PV de um
		// combatente que não está na mesa.
		b.Tokens[at.Index].EntryID, b.Tokens[at.Index].CharacterID = nil, nil
		unbound++
	})
	return unbound
}

// forgetTheRowsTheQueueDropped tira o vínculo de quem aponta para linha que não
// está mais na fila.
//
// `Remove` e não `Despawn`: a peça CONTINUA no mundo e no mapa — é só o
// ponteiro que sai. O núcleo tem as duas operações justamente porque as duas
// existem, e escolher a errada aqui apagaria o trabalho do mestre.
func forgetTheRowsTheQueueDropped(st *live.SessionRuntimeState) ecs.System {
	return func(w *ecs.World) {
		naFila := map[string]bool{}
		if st != nil {
			for i := range st.Initiative {
				naFila[st.Initiative[i].ID] = true
			}
		}
		ecs.Each(w, func(e ecs.Entity, bound boundTo) {
			if !naFila[bound.EntryID] {
				ecs.Remove[boundTo](w, e)
			}
		})
	}
}

// hasTokenForEntry diz se a linha da fila já tem peça no mapa.
func hasTokenForEntry(w *ecs.World, entryID string) bool {
	achou := false
	ecs.Each(w, func(_ ecs.Entity, bound boundTo) {
		if bound.EntryID == entryID {
			achou = true
		}
	})
	return achou
}
