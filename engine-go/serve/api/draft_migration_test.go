package api

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
)

// A migração põe os dados antigos na regra: quem está em campanha vira ficha que joga, e o
// rascunho perde o que tinha de jogo. Sem isso o herói ferido continuaria ferido na lista.
func TestTheMigrationCleansDraftsAndKeepsWhatPlays(t *testing.T) {
	f, draftID := draftSheet(t)
	ctx := context.Background()
	memberID := f.charID
	seat := func(id int64) {
		t.Helper()
		for _, statement := range []string{
			`INSERT OR REPLACE INTO character_damage (characterId, hpDamage, mpSpent) VALUES (?, 3, 2)`,
			`INSERT INTO active_effects (characterId, source, catalogId, scope, modifiers, createdAt) VALUES (?, 'spell', 'armadura-arcana', 'scene', '[]', '2026-11-01')`,
			`UPDATE characters SET activeConditions = '["caido"]' WHERE id = ?`,
			`INSERT INTO character_conditionals (characterId, conditionalId) VALUES (?, 'k')`,
			`INSERT INTO character_power_uses (characterId, powerId, scope, used) VALUES (?, 'p', 'scene', 1)`,
			`INSERT INTO character_stances (characterId, flag) VALUES (?, 'f')`,
			`INSERT INTO character_nonlethal_damage (characterId, amount) VALUES (?, 1)`,
		} {
			if _, err := f.s.db.ExecContext(ctx, statement, id); err != nil {
				t.Fatalf("%s: %v", statement, err)
			}
		}
	}
	seat(draftID)
	seat(memberID)
	// A ficha antiga: em campanha, e sem campaignId.
	if _, err := f.s.db.ExecContext(ctx, `UPDATE characters SET campaignId = NULL WHERE id = ?`, memberID); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile("../../infra/db/migrations/00023_draft_without_play_state.sql")
	if err != nil {
		t.Fatal(err)
	}
	up, _, _ := strings.Cut(string(raw), "-- +goose Down")
	if _, err := f.s.db.ExecContext(ctx, up); err != nil {
		t.Fatalf("rodar a migração: %v", err)
	}

	count := func(table string, id int64) int {
		t.Helper()
		var n int
		if err := f.s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table+` WHERE characterId = ?`, id).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	for _, table := range []string{"character_damage", "active_effects", "character_conditionals", "character_power_uses", "character_stances", "character_nonlethal_damage"} {
		if got := count(table, draftID); got != 0 {
			t.Errorf("o rascunho ficou com %d linhas em %s", got, table)
		}
		if got := count(table, memberID); got == 0 {
			t.Errorf("a ficha de campanha perdeu %s", table)
		}
	}
	var conditions string
	var campaignID sql.NullInt64
	if err := f.s.db.QueryRowContext(ctx, `SELECT activeConditions FROM characters WHERE id = ?`, draftID).Scan(&conditions); err != nil {
		t.Fatal(err)
	}
	if conditions != "[]" {
		t.Errorf("o rascunho ficou com as condições %s", conditions)
	}
	if err := f.s.db.QueryRowContext(ctx, `SELECT campaignId FROM characters WHERE id = ?`, memberID).Scan(&campaignID); err != nil {
		t.Fatal(err)
	}
	if !campaignID.Valid || campaignID.Int64 != f.campaignID {
		t.Errorf("a ficha em campanha devia apontar a campanha %d, e ficou %v", f.campaignID, campaignID)
	}
}
