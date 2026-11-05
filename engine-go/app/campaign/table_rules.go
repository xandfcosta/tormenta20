package campaign

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"t20engine/app"
	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// AS REGRAS DA MESA: o mestre escreve e liga, e todo gesto passa pelo `OwnedCampaign`.

// StoredTableRule é uma regra como a aba a desenha.
type StoredTableRule struct {
	ID          string
	Name        string
	CharacterID *int64
	Enabled     bool
	When        []engine.RuleCondition
	Effect      engine.RuleEffect
}

// TableRuleDraft é o que o editor manda gravar. ID vazio é regra nova.
type TableRuleDraft struct {
	ID          string
	Name        string
	CharacterID *int64
	When        []engine.RuleCondition
	Effect      engine.RuleEffect
}

// TableRuleGone é a frase da regra apagada em outra aba.
const TableRuleGone = "Esta regra não existe mais: ela foi apagada depois que esta tela abriu."

// TableRuleRefusal é a recusa que a tela mostra dentro do editor, com o que se esperava.
type TableRuleRefusal struct{ Reason string }

func (r TableRuleRefusal) Error() string { return r.Reason }

// TableRules são as regras da campanha, na ordem em que nasceram.
func (l Lifecycle) TableRules(ctx context.Context, campaignID int64) ([]StoredTableRule, error) {
	rows, err := l.queries.ListCampaignRules(ctx, campaignID)
	if err != nil {
		return nil, fmt.Errorf("ler as regras da campanha %d: %w", campaignID, err)
	}
	out := make([]StoredTableRule, 0, len(rows))
	for _, row := range rows {
		when, then, err := engine.ParseTableRule(row.Conditions, row.Effect)
		if err != nil {
			return nil, fmt.Errorf("a regra %q da campanha %d: %w", row.Name, campaignID, err)
		}
		rule := StoredTableRule{ID: row.ID, Name: row.Name, Enabled: row.Enabled == 1, When: when, Effect: then}
		if row.Characterid.Valid {
			rule.CharacterID = &row.Characterid.Int64
		}
		out = append(out, rule)
	}
	return out, nil
}

// SaveTableRule grava a regra nova, que nasce desligada, ou reescreve a que existe.
func (l Lifecycle) SaveTableRule(ctx context.Context, who app.Caller, campaignID int64, draft TableRuleDraft) error {
	if _, err := l.access.OwnedCampaign(ctx, who, campaignID); err != nil {
		return err
	}
	if err := book.TableRuleProblem(draft.Name, draft.When, draft.Effect); err != nil {
		return TableRuleRefusal{Reason: err.Error()}
	}
	character := sql.NullInt64{}
	if draft.CharacterID != nil {
		if err := l.characterSitsAt(ctx, campaignID, *draft.CharacterID); err != nil {
			return err
		}
		character = sql.NullInt64{Int64: *draft.CharacterID, Valid: true}
	}
	conditions, err := json.Marshal(draft.When)
	if err != nil {
		return fmt.Errorf("escrever as condições da regra: %w", err)
	}
	effect, err := json.Marshal(draft.Effect)
	if err != nil {
		return fmt.Errorf("escrever o efeito da regra: %w", err)
	}
	if draft.ID == "" {
		return l.queries.InsertCampaignRule(ctx, sqlcgen.InsertCampaignRuleParams{
			ID: newTableRuleID(), Campaignid: campaignID, Characterid: character, Name: draft.Name,
			Conditions: string(conditions), Effect: string(effect), Updatedat: dbvalue.NowISO(),
		})
	}
	changed, err := l.queries.RewriteCampaignRule(ctx, sqlcgen.RewriteCampaignRuleParams{
		Characterid: character, Name: draft.Name, Conditions: string(conditions), Effect: string(effect),
		Updatedat: dbvalue.NowISO(), ID: draft.ID, Campaignid: campaignID,
	})
	return oneRuleChanged(changed, err, draft.ID)
}

// FlipTableRule liga a regra desligada, e desliga a ligada.
func (l Lifecycle) FlipTableRule(ctx context.Context, who app.Caller, campaignID int64, ruleID string) error {
	if _, err := l.access.OwnedCampaign(ctx, who, campaignID); err != nil {
		return err
	}
	changed, err := l.queries.FlipCampaignRule(ctx, sqlcgen.FlipCampaignRuleParams{
		Updatedat: dbvalue.NowISO(), ID: ruleID, Campaignid: campaignID,
	})
	return oneRuleChanged(changed, err, ruleID)
}

// RemoveTableRule apaga a regra.
func (l Lifecycle) RemoveTableRule(ctx context.Context, who app.Caller, campaignID int64, ruleID string) error {
	if _, err := l.access.OwnedCampaign(ctx, who, campaignID); err != nil {
		return err
	}
	changed, err := l.queries.DropCampaignRule(ctx, sqlcgen.DropCampaignRuleParams{ID: ruleID, Campaignid: campaignID})
	return oneRuleChanged(changed, err, ruleID)
}

func oneRuleChanged(changed int64, err error, ruleID string) error {
	if err != nil {
		return fmt.Errorf("gravar a regra %q: %w", ruleID, err)
	}
	if changed == 0 {
		return TableRuleRefusal{Reason: TableRuleGone}
	}
	return nil
}

func (l Lifecycle) characterSitsAt(ctx context.Context, campaignID, characterID int64) error {
	if characterID == 0 {
		return TableRuleRefusal{Reason: "Escolha o personagem a quem a regra vale."}
	}
	members, err := l.queries.ListMembers(ctx, campaignID)
	if err != nil {
		return fmt.Errorf("ler a mesa %d: %w", campaignID, err)
	}
	for _, member := range members {
		if member.Characterid == characterID {
			return nil
		}
	}
	return TableRuleRefusal{Reason: fmt.Sprintf("o personagem %d não está nesta mesa", characterID)}
}

// newTableRuleID nasce do relógio, porque a lista se lê na ordem do id: a mais antiga primeiro.
func newTableRuleID() string {
	return "regra-" + strconv.FormatInt(time.Now().UnixNano(), 36)
}
