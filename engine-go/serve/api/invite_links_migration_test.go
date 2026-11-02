package api

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// O link que já foi gasto no uso único não pode acordar reutilizável: ele está
// em histórico de conversa, e foi mandado achando que valia uma vez.
func TestTheMigrationKeepsSpentLinksDeadAndOpenLinksOpen(t *testing.T) {
	ctx := context.Background()
	database, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "old.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	for _, statement := range []string{
		`CREATE TABLE users (id INTEGER PRIMARY KEY)`,
		`INSERT INTO users (id) VALUES (1), (2)`,
		`CREATE TABLE account_invites (
			id INTEGER PRIMARY KEY AUTOINCREMENT, token TEXT NOT NULL UNIQUE,
			createdBy INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			createdAt TEXT NOT NULL, expiresAt TEXT NOT NULL, usedAt TEXT,
			usedBy INTEGER REFERENCES users(id) ON DELETE SET NULL)`,
		`INSERT INTO account_invites (token, createdBy, createdAt, expiresAt, usedAt, usedBy) VALUES
			('gasto', 1, '2026-11-01', '2026-11-08', '2026-11-02', 2),
			('gasto-por-quem-saiu', 1, '2026-11-01', '2026-11-08', '2026-11-02', NULL),
			('aberto', 1, '2026-11-01', '2026-11-08', NULL, NULL)`,
	} {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			t.Fatalf("%s: %v", statement, err)
		}
	}

	raw, err := os.ReadFile("../../infra/db/migrations/00026_reusable_account_invites.sql")
	if err != nil {
		t.Fatal(err)
	}
	up, _, _ := strings.Cut(string(raw), "-- +goose Down")
	if _, err := database.ExecContext(ctx, up); err != nil {
		t.Fatalf("rodar a migração: %v", err)
	}

	for token, wantRevoked := range map[string]bool{"gasto": true, "gasto-por-quem-saiu": true, "aberto": false} {
		var revoked sql.NullString
		if err := database.QueryRowContext(ctx,
			`SELECT revokedAt FROM account_invite_links WHERE token = ?`, token).Scan(&revoked); err != nil {
			t.Fatalf("achar %q: %v", token, err)
		}
		if revoked.Valid != wantRevoked {
			t.Errorf("%q: revogado = %v, want %v", token, revoked.Valid, wantRevoked)
		}
	}
	var user int64
	if err := database.QueryRowContext(ctx, `SELECT userId FROM account_invite_uses`).Scan(&user); err != nil {
		t.Fatalf("o registro de quem entrou não veio: %v", err)
	}
	if user != 2 {
		t.Errorf("o registro diz que entrou a conta %d, want 2", user)
	}
}
