package boards

import (
	"context"

	"t20engine/domain/board"
	"t20engine/domain/engine"
	"t20engine/domain/live"
)

// ChangeMoveMode troca o modo do movimento proposto (a pé, escalando, nadando, voando) e o mede de
// novo, sob a trava do tabuleiro. A regra é do `board.ChangeMoveMode`.
func (bs *Store) ChangeMoveMode(ctx context.Context, sessionID int64, boardID string, st *live.SessionRuntimeState, tokenID string, mode board.MoveModeKind, by board.Mover) (*board.BoardState, error) {
	return bs.apply(ctx, sessionID, boardID, func(b *board.BoardState) error {
		return board.ChangeMoveMode(b, st, tokenID, mode, by)
	})
}

// ProposeNextStop acrescenta uma parada ao movimento que esta pessoa monta com a peça, numa leitura
// só: as paradas que já existem saem do tabuleiro que a mutação acabou de ler, e não de uma leitura
// antes dela.
func (bs *Store) ProposeNextStop(ctx context.Context, sessionID int64, boardID string, st *live.SessionRuntimeState, tokenID string, destination engine.Square, by board.Mover) (*board.BoardState, error) {
	return bs.apply(ctx, sessionID, boardID, func(b *board.BoardState) error {
		stops, err := board.StopsSoFar(b, tokenID, by.UserID)
		if err != nil {
			return err
		}
		return board.ProposeMoveWithStops(b, st, tokenID, append(stops, destination), by)
	})
}
