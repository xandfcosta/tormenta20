package boards

import (
	"context"
	"log/slog"

	"t20engine/domain/board"
	"t20engine/infra/events"
)

// Close encerra UM tabuleiro: a linha sai do banco. As posições não são guardadas — quem guarda a cena
// para outro dia é o `Archive`, e o encerrar da mesa é o `EndScene`.
//
// O DELETE QUE FALHA RECUSA o gesto (ALE-375), e a aba continua onde estava: uma aba que sumisse da
// tela com a linha viva no banco voltaria no próximo boot como um tabuleiro FANTASMA, com as peças de
// uma cena que a mesa já encerrou.
func (bs *Store) Close(ctx context.Context, sessionID int64, boardID string) error {
	_, closed, err := bs.closeLocked(ctx, sessionID, boardID, nil)
	if err != nil || !closed {
		return err
	}
	bs.bus.Publish(events.BoardClosed{SessionID: sessionID})
	return nil
}

// EndScene ARQUIVA o tabuleiro no acervo da campanha e o tira da mesa, numa leitura só, e diz quantas
// abas sobraram. Eram três leituras: a do arquivar, a do fechar e a de contar o que sobrou.
//
// A falha ao ARQUIVAR não impede o encerrar: o mestre mandou tirar a cena da mesa, e recusar isso
// porque o acervo falhou deixaria a mesa presa numa cena que já acabou. Ela fica no log.
func (bs *Store) EndScene(ctx context.Context, campaignID, sessionID int64, boardID string) (left int, err error) {
	archive := func(b *board.BoardState) {
		if err := bs.Archive(ctx, campaignID, b); err != nil {
			slog.Warn("o lugar não foi arquivado ao encerrar a cena",
				"campaign_id", campaignID, "session_id", sessionID, "board_id", b.ID, "err", err)
		}
	}
	left, closed, err := bs.closeLocked(ctx, sessionID, boardID, archive)
	if err != nil || !closed {
		return left, err
	}
	bs.bus.Publish(events.BoardClosed{SessionID: sessionID})
	return left, nil
}

// closeLocked apaga a linha do tabuleiro sob a trava, e diz quantas abas sobraram e se havia o que
// fechar. A função de antes do DELETE roda sobre o tabuleiro lido (o arquivar do `EndScene`). O
// aviso sai DEPOIS, fora da trava, como no `apply`.
func (bs *Store) closeLocked(ctx context.Context, sessionID int64, boardID string, beforeDelete func(*board.BoardState)) (left int, closed bool, err error) {
	bs.Mu.Lock()
	defer bs.Mu.Unlock()
	open, err := bs.snapshotsFor(ctx).Read(ctx, sessionID)
	if err != nil {
		return 0, false, err
	}
	target := FindBoard(open, boardID)
	if target == nil {
		return len(open), false, nil
	}
	if beforeDelete != nil {
		beforeDelete(target)
	}
	if err := bs.snapshotsFor(ctx).Delete(ctx, sessionID, target.ID); err != nil {
		return len(open), false, err
	}
	return len(open) - 1, true, nil
}
