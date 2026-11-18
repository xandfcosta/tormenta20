package boards

import (
	"context"

	"t20engine/domain/board"
	"t20engine/domain/live"
)

// ChangeMoveMode troca o modo do movimento proposto (a pé, escalando, nadando, voando) e o mede de
// novo, sob a trava do tabuleiro. A regra é do `board.ChangeMoveMode`.
func (bs *Store) ChangeMoveMode(ctx context.Context, sessionID int64, boardID string, st *live.SessionRuntimeState, tokenID string, mode board.MoveModeKind, by board.Mover) (*board.BoardState, error) {
	return bs.apply(ctx, sessionID, boardID, func(b *board.BoardState) error {
		return board.ChangeMoveMode(b, st, tokenID, mode, by)
	})
}
