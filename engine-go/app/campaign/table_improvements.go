package campaign

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"time"

	"t20engine/app"
	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// AS MELHORIAS DA MESA: só o mestre escreve. Quem as coloca no item é o jogador, pela ficha.

// StoredTableImprovement é uma melhoria como a aba a desenha.
type StoredTableImprovement struct {
	ID          string
	Name        string
	CharacterID *int64
	Family      string
	Effect      engine.RuleEffect
}

// TableImprovementDraft é o que o editor manda gravar. ID vazio é melhoria nova.
type TableImprovementDraft struct {
	ID          string
	Name        string
	CharacterID *int64
	Family      string
	Effect      engine.RuleEffect
}

// TableImprovementGone é a frase da melhoria apagada em outra aba.
const TableImprovementGone = "Esta melhoria não existe mais: ela foi apagada depois que esta tela abriu."

// TableImprovements são as melhorias da campanha, na ordem em que nasceram.
func (l Lifecycle) TableImprovements(ctx context.Context, campaignID int64) ([]StoredTableImprovement, error) {
	rows, err := l.queries.ListCampaignImprovements(ctx, campaignID)
	if err != nil {
		return nil, fmt.Errorf("ler as melhorias da campanha %d: %w", campaignID, err)
	}
	out := make([]StoredTableImprovement, 0, len(rows))
	for _, row := range rows {
		effect, err := engine.ParseRuleEffect(row.Effect)
		if err != nil {
			return nil, fmt.Errorf("a melhoria %q da campanha %d: %w", row.Name, campaignID, err)
		}
		improvement := StoredTableImprovement{ID: row.ID, Name: row.Name, Family: row.Family, Effect: effect}
		if row.Characterid.Valid {
			improvement.CharacterID = &row.Characterid.Int64
		}
		out = append(out, improvement)
	}
	return out, nil
}

// TableImprovementUses diz em quantos itens da mesa cada melhoria está.
func (l Lifecycle) TableImprovementUses(ctx context.Context, campaignID int64) (map[string]int, error) {
	rows, err := l.queries.ListCampaignItemsWithImprovements(ctx, campaignID)
	if err != nil {
		return nil, fmt.Errorf("ler os itens da campanha %d: %w", campaignID, err)
	}
	uses := map[string]int{}
	for _, row := range rows {
		for _, id := range improvementIDsOf(row.ID, row.Improvements) {
			uses[id]++
		}
	}
	return uses, nil
}

// SaveTableImprovement grava a melhoria nova, ou reescreve a que existe.
func (l Lifecycle) SaveTableImprovement(ctx context.Context, who app.Caller, campaignID int64, draft TableImprovementDraft) error {
	if _, err := l.access.OwnedCampaign(ctx, who, campaignID); err != nil {
		return err
	}
	if err := book.TableImprovementProblem(draft.Name, draft.Family, draft.Effect); err != nil {
		return TableRuleRefusal{Reason: err.Error()}
	}
	character := sql.NullInt64{}
	if draft.CharacterID != nil {
		if err := l.characterSitsAt(ctx, campaignID, *draft.CharacterID); err != nil {
			return err
		}
		character = sql.NullInt64{Int64: *draft.CharacterID, Valid: true}
	}
	effect, err := json.Marshal(draft.Effect)
	if err != nil {
		return fmt.Errorf("escrever o efeito da melhoria %q: %w", draft.Name, err)
	}
	if draft.ID == "" {
		if err := l.queries.InsertCampaignImprovement(ctx, sqlcgen.InsertCampaignImprovementParams{
			ID: engine.TableImprovementPrefix + strconv.FormatInt(time.Now().UnixNano(), 36), Campaignid: campaignID,
			Characterid: character, Name: draft.Name, Family: draft.Family, Effect: string(effect), Updatedat: dbvalue.NowISO(),
		}); err != nil {
			return fmt.Errorf("gravar a melhoria %q: %w", draft.Name, err)
		}
		return nil
	}
	changed, err := l.queries.RewriteCampaignImprovement(ctx, sqlcgen.RewriteCampaignImprovementParams{
		Characterid: character, Name: draft.Name, Family: draft.Family, Effect: string(effect),
		Updatedat: dbvalue.NowISO(), ID: draft.ID, Campaignid: campaignID,
	})
	return oneImprovementChanged(changed, err, draft.ID)
}

// RemoveTableImprovement tira a melhoria de todo item da mesa e a apaga, numa transação: um "Aplicar"
// do jogador no meio regravaria a lista velha do item.
func (l Lifecycle) RemoveTableImprovement(ctx context.Context, who app.Caller, campaignID int64, improvementID string) error {
	if _, err := l.access.OwnedCampaign(ctx, who, campaignID); err != nil {
		return err
	}
	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("abrir a transação: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := l.queries.WithTx(tx)
	rows, err := q.ListCampaignItemsWithImprovements(ctx, campaignID)
	if err != nil {
		return fmt.Errorf("ler os itens da campanha %d: %w", campaignID, err)
	}
	for _, row := range rows {
		ids := improvementIDsOf(row.ID, row.Improvements)
		if !slices.Contains(ids, improvementID) {
			continue
		}
		kept := slices.DeleteFunc(ids, func(id string) bool { return id == improvementID })
		if err := q.SetItemOverlays(ctx, sqlcgen.SetItemOverlaysParams{
			Improvements: sheet.MarshalStrings(&kept), Material: row.Material, ID: row.ID,
		}); err != nil {
			return fmt.Errorf("tirar a melhoria %q do item %d: %w", improvementID, row.ID, err)
		}
	}
	changed, err := q.DropCampaignImprovement(ctx, sqlcgen.DropCampaignImprovementParams{ID: improvementID, Campaignid: campaignID})
	if err := oneImprovementChanged(changed, err, improvementID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("gravar a exclusão da melhoria %q: %w", improvementID, err)
	}
	return nil
}

// improvementIDsOf lê a lista do item. A ilegível conta como vazia, pela razão do `bookOverlays`: uma
// linha torta não pode derrubar a aba inteira nem travar a exclusão. O log diz qual.
func improvementIDsOf(itemID int64, stored string) []string {
	ids := []string{}
	if err := json.Unmarshal([]byte(stored), &ids); err != nil {
		slog.Warn("as melhorias de um item estão ilegíveis", "item", itemID, "error", err)
		return []string{}
	}
	return ids
}

func oneImprovementChanged(changed int64, err error, improvementID string) error {
	if err != nil {
		return fmt.Errorf("gravar a melhoria %q: %w", improvementID, err)
	}
	if changed == 0 {
		return TableRuleRefusal{Reason: TableImprovementGone}
	}
	return nil
}
