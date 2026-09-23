package boards

import (
	"context"

	"t20engine/app"
	"t20engine/app/session"
	"t20engine/domain/board"
	"t20engine/domain/live"
)

// O VÍNCULO MORTO: quando a linha da fila some, a peça FICA e o ponteiro SAI
// (ALE-377).
//
// Uma peça apontando para uma linha que não existe mais mente de um jeito
// silencioso. Medido: ela continua se anunciando como combatente, com o botão
// "Atacar Fulano" no menu, e o gesto responde 200 sem fazer nada nem recusar.
//
// Eram TRÊS caminhos deixando o mesmo estado — tirar da fila, reiniciar o
// combate, e reabrir um lugar do acervo noutra sessão. Os dois primeiros moram
// aqui porque escrevem nos DOIS donos de dado e precisam da transação; o
// terceiro é escrita só do tabuleiro e mora no `places.go`.

// RemoveFromQueue tira o combatente da fila E desamarra a peça dele.
//
// A PEÇA FICA, e isso é o desenho: o mestre tirou alguém da INICIATIVA, não do
// mapa — o comentário do `removesToken` já diz o contrário para o gesto
// contrário ("tira a peça do tabuleiro, e SÓ do tabuleiro… A linha da
// iniciativa fica"). Os dois verbos são independentes de propósito.
func (g Gestures) RemoveFromQueue(
	ctx context.Context, sessionID int64, entryID string,
) (*live.SessionRuntimeState, error) {
	var out *live.SessionRuntimeState
	err := g.units.Do(ctx, func(u session.Unit) error {
		inUnit := g.inUnit(ctx, u)
		queue, err := g.sessions.RemoveInitiativeEntry(inUnit, sessionID, entryID)
		if err != nil {
			return err
		}
		if err := g.unbindOrphans(inUnit, sessionID, queue); err != nil {
			return err
		}
		out = queue
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RestartCombat esvazia a fila e desamarra TODAS as peças do mapa.
//
// "Reiniciar o combate" promete na tela que *"a partida CONTINUA no ar"*: o
// mestre reiniciou o COMBATE e não a CENA, então o mapa que ele montou fica de
// pé — peças, terreno e posições (decisão do dono, ALE-377). O que não fica é o
// vínculo, porque a fila que ele apontava deixou de existir inteira.
func (g Gestures) RestartCombat(
	ctx context.Context, who app.Caller, campaignID, sessionID int64,
) (*live.SessionRuntimeState, error) {
	if _, err := g.access.GM(ctx, who, campaignID, sessionID); err != nil {
		return nil, err
	}
	var out *live.SessionRuntimeState
	err := g.units.Do(ctx, func(u session.Unit) error {
		inUnit := g.inUnit(ctx, u)
		queue, err := g.sessions.RestartCombat(inUnit, sessionID)
		if err != nil {
			return err
		}
		if err := g.unbindOrphans(inUnit, sessionID, queue); err != nil {
			return err
		}
		out = queue
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// unbindOrphans reconcilia TODAS as abas da sessão contra a fila que vale agora.
//
// Todas, e não a que está na tela: o mestre monta a cripta numa aba enquanto a
// mesa olha a taverna, e a linha que sumiu da fila sumiu para as oito. Varrer só
// a aba do gesto deixaria a mentira nas outras sete — e ela só apareceria quando
// alguém trocasse de aba, longe do gesto que a causou.
//
// A GRAVAÇÃO só acontece para a aba que MUDOU. O `OpenBoards` devolve CÓPIAS,
// então a primeira passada desamarra o clone e não muda nada de verdade — ela
// só responde "esta aba mudaria?". Sem ela, uma linha removida regravaria os
// oito blobs abertos, e sete seriam idênticos ao que já estava no disco.
func (g Gestures) unbindOrphans(
	ctx context.Context, sessionID int64, queue *live.SessionRuntimeState,
) error {
	open, err := g.boards.OpenBoards(ctx, sessionID)
	if err != nil {
		return err
	}
	for _, aberta := range open {
		if board.UnbindOrphanTokens(aberta, queue) == 0 {
			continue
		}
		if _, err := g.boards.UnbindTokens(ctx, sessionID, aberta.ID, queue); err != nil {
			return err
		}
	}
	return nil
}
