package boards

import (
	"context"

	"t20engine/domain/board"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// MovementOf lê da ficha computada o que o tabuleiro pede de uma peça (mora aqui, e não na ficha, porque
// a ficha não conhece o tabuleiro): o deslocamento em QUADRADOS
// (o motor já conta assim), se o terreno difícil cobra dela (p129) e os outros deslocamentos —
// escalada, natação, voo —, que a ficha guarda em metros e a peça conta em quadrados. Juntos porque
// saem da MESMA ficha, e computá-la uma vez por campo seria pagar o caro por nada.
//
// Uso: `boards.MovementOf(ctx, q, cat, 7)` devolve o movimento da peça do personagem 7.
func MovementOf(ctx context.Context, q *sqlcgen.Queries, cat *engine.Catalogs, characterID int64) (board.TokenMovement, error) {
	row, err := q.GetCharacter(ctx, characterID)
	if err != nil {
		return board.TokenMovement{}, err
	}
	character, err := sheet.LoadAndCompute(ctx, q, cat, row)
	if err != nil {
		return board.TokenMovement{}, err
	}
	land := character.Displacement.Total
	climb := character.ClimbSpeed * 2 / 3
	if character.ClimbsAsItWalks {
		climb = max(climb, land)
	}
	movement := board.TokenMovement{Squares: land, IgnoresDifficult: character.IgnoresDifficultTerrain}
	for _, other := range []board.MoveMode{
		{Kind: board.MoveModeClimb, Squares: climb},
		{Kind: board.MoveModeSwim, Squares: character.SwimSpeed * 2 / 3},
		{Kind: board.MoveModeFly, Squares: character.FlySpeed * 2 / 3},
	} {
		if other.Squares > 0 {
			movement.Modes = append(movement.Modes, other)
		}
	}
	return movement, nil
}
