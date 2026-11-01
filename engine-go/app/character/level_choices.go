package character

import (
	"context"
	"database/sql"
	"fmt"

	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// syncLevelChoices acerta as linhas com o `sheet.PlanLevelRows`. Quem chama é todo gesto que
// muda classe, nível ou escolha, na mesma transação, e o boot, para as fichas que vieram da
// migração.
func syncLevelChoices(
	ctx context.Context, q *sqlcgen.Queries, cat *engine.Catalogs, row sqlcgen.Character,
) error {
	dto, err := sheet.Load(ctx, q, cat, row)
	if err != nil {
		return err
	}
	current := map[int64]sheet.LevelChoiceDTO{}
	for _, choice := range dto.LevelChoices {
		current[choice.ID] = choice
	}
	for _, wanted := range sheet.PlanLevelRows(dto) {
		if wanted.ID == 0 {
			if err := createLevelChoice(ctx, q, row.ID, wanted); err != nil {
				return err
			}
			continue
		}
		if current[wanted.ID].Level != wanted.Level {
			if err := q.SetLevelChoiceLevel(ctx, sqlcgen.SetLevelChoiceLevelParams{
				Level: sql.NullInt64{Int64: wanted.Level, Valid: wanted.Level > 0}, ID: wanted.ID,
			}); err != nil {
				return fmt.Errorf("situar a escolha %d da ficha %d: %w", wanted.ID, row.ID, err)
			}
		}
		delete(current, wanted.ID)
	}
	for id := range current {
		if err := q.DeleteLevelChoice(ctx, id); err != nil {
			return fmt.Errorf("apagar a escolha %d da ficha %d: %w", id, row.ID, err)
		}
	}
	return nil
}

func createLevelChoice(
	ctx context.Context, q *sqlcgen.Queries, characterID int64, choice sheet.LevelChoiceDTO,
) error {
	estimated := int64(0)
	if choice.Estimated {
		estimated = 1
	}
	if err := q.CreateLevelChoice(ctx, sqlcgen.CreateLevelChoiceParams{
		Characterid: characterID,
		Level:       sql.NullInt64{Int64: choice.Level, Valid: choice.Level > 0},
		Kind:        choice.Kind, Ref: choice.Ref, Pick: choice.Pick, Estimated: estimated,
	}); err != nil {
		return fmt.Errorf("gravar a escolha %s %q da ficha %d: %w", choice.Kind, choice.Ref, characterID, err)
	}
	return nil
}

// saveChoiceBlobs leva os textos do gesto para as linhas.
func saveChoiceBlobs(
	ctx context.Context, q *sqlcgen.Queries, cat *engine.Catalogs,
	row sqlcgen.Character, classPowers, powerChoices *string,
) error {
	rows, err := q.ListLevelChoicesByCharacter(ctx, row.ID)
	if err != nil {
		return fmt.Errorf("ler as escolhas da ficha %d: %w", row.ID, err)
	}
	diff := sheet.DiffChoiceBlobs(sheet.LevelChoicesFrom(rows), classPowers, powerChoices)
	for _, id := range diff.RemoveIDs {
		if err := q.DeleteLevelChoice(ctx, id); err != nil {
			return fmt.Errorf("apagar a escolha %d da ficha %d: %w", id, row.ID, err)
		}
	}
	for _, choice := range diff.Add {
		if err := createLevelChoice(ctx, q, row.ID, choice); err != nil {
			return err
		}
	}
	return syncLevelChoices(ctx, q, cat, row)
}

// SyncAllLevelChoices situa as escolhas de toda ficha. Roda no boot, porque a migração não
// tem como fazer isto: a conta precisa do catálogo, que não mora em SQL. Refazer é seguro, e
// é o que leva uma correção do catálogo às automáticas guardadas.
func (p Plays) SyncAllLevelChoices(ctx context.Context) error {
	ids, err := p.queries.ListCharacterIDs(ctx)
	if err != nil {
		return fmt.Errorf("listar as fichas: %w", err)
	}
	for _, id := range ids {
		row, err := p.queries.GetCharacter(ctx, id)
		if err != nil {
			return fmt.Errorf("ler a ficha %d: %w", id, err)
		}
		if err := p.inTx(ctx, "situar as escolhas", func(q *sqlcgen.Queries) error {
			return syncLevelChoices(ctx, q, p.catalogs, row)
		}); err != nil {
			return err
		}
	}
	return nil
}
