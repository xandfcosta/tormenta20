package boards

import (
	"context"

	"t20engine/domain/board"
	"t20engine/domain/engine"
	"t20engine/domain/live"
)

// As portas dos gestos que DECIDEM sobre o tabuleiro: cada uma lê, decide e grava dentro da mesma
// mutação, sob a trava. A decisão em si é do `domain/board` (`decided_gestures.go` de lá), e o porquê
// mora lá também.

// AddNextMarker põe um marcador novo na casa, com a letra decidida sobre o tabuleiro que a mutação leu.
func (bs *Store) AddNextMarker(ctx context.Context, sessionID int64, boardID string, square engine.Square) (*board.BoardState, error) {
	return bs.apply(ctx, sessionID, boardID, func(b *board.BoardState) error {
		return board.AddNextMarker(b, square, bs.newID)
	})
}

// ToggleMarkerHidden mostra ou esconde o marcador, conforme o que a mutação leu.
func (bs *Store) ToggleMarkerHidden(ctx context.Context, sessionID int64, boardID, markerID string) (*board.BoardState, error) {
	return bs.apply(ctx, sessionID, boardID, func(b *board.BoardState) error {
		return board.ToggleMarkerHidden(b, markerID)
	})
}

// EraseMarker tira o marcador, e recusa o que não existe.
func (bs *Store) EraseMarker(ctx context.Context, sessionID int64, boardID, markerID string) (*board.BoardState, error) {
	return bs.apply(ctx, sessionID, boardID, func(b *board.BoardState) error {
		return board.EraseMarker(b, markerID)
	})
}

// ToggleTokenHidden mostra ou esconde a peça, conforme o que a mutação leu.
func (bs *Store) ToggleTokenHidden(ctx context.Context, sessionID int64, boardID, tokenID string) (*board.BoardState, error) {
	return bs.apply(ctx, sessionID, boardID, func(b *board.BoardState) error {
		return board.ToggleTokenHidden(b, tokenID)
	})
}

// ToggleTokenMoving liga ou desliga o "em movimento" do objeto, conforme o que a mutação leu.
func (bs *Store) ToggleTokenMoving(ctx context.Context, sessionID int64, boardID, tokenID string) (*board.BoardState, error) {
	return bs.apply(ctx, sessionID, boardID, func(b *board.BoardState) error {
		return board.ToggleTokenMoving(b, tokenID)
	})
}

// TakeTokenOff tira a peça do tabuleiro, e recusa a que não existe.
func (bs *Store) TakeTokenOff(ctx context.Context, sessionID int64, boardID, tokenID string) (*board.BoardState, error) {
	return bs.apply(ctx, sessionID, boardID, func(b *board.BoardState) error {
		return board.TakeTokenOff(b, tokenID)
	})
}

// UndoLastStop tira a última perna do movimento que esta pessoa monta, sobre as paradas que a mutação
// leu.
func (bs *Store) UndoLastStop(ctx context.Context, sessionID int64, boardID string, st *live.SessionRuntimeState, tokenID string, by board.Mover) (*board.BoardState, error) {
	return bs.apply(ctx, sessionID, boardID, func(b *board.BoardState) error {
		return board.UndoLastStop(b, st, tokenID, by)
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
