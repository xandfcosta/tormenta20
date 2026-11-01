package character

import (
	"context"
	"fmt"

	"t20engine/domain/sheet"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// ApplyLevelSwap grava a troca planejada pelo `sheet.SwapLevelChoice`.
func (p Plays) ApplyLevelSwap(ctx context.Context, row sqlcgen.Character, swap sheet.LevelSwap) error {
	return p.inTx(ctx, "trocar a escolha", func(q *sqlcgen.Queries) error {
		for _, id := range swap.RemoveIDs {
			if err := q.DeleteLevelChoice(ctx, id); err != nil {
				return fmt.Errorf("apagar a escolha %d: %w", id, err)
			}
		}
		for _, choice := range swap.Add {
			if err := createLevelChoice(ctx, q, row.ID, choice); err != nil {
				return err
			}
		}
		if err := q.TouchCharacter(ctx, sqlcgen.TouchCharacterParams{UpdatedAt: dbvalue.NowISO(), ID: row.ID}); err != nil {
			return fmt.Errorf("carimbar a ficha %d: %w", row.ID, err)
		}
		return syncLevelChoices(ctx, q, p.catalogs, row)
	})
}

// LowerLastLevel tira o último nível e o que foi escolhido nele.
func (p Plays) LowerLastLevel(ctx context.Context, row sqlcgen.Character, loss sheet.LastLevelLoss) error {
	return p.inTx(ctx, "descer de nível", func(q *sqlcgen.Queries) error {
		for _, id := range loss.RemoveIDs {
			if err := q.DeleteLevelChoice(ctx, id); err != nil {
				return fmt.Errorf("apagar a linha %d: %w", id, err)
			}
		}
		if err := q.SetCharacterLevel(ctx, sqlcgen.SetCharacterLevelParams{
			Level: loss.Level - 1, UpdatedAt: dbvalue.NowISO(), ID: row.ID,
		}); err != nil {
			return fmt.Errorf("gravar o nível da ficha %d: %w", row.ID, err)
		}
		return syncLevelChoices(ctx, q, p.catalogs, row)
	})
}
