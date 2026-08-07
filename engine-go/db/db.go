// Package db owns the API server's SQLite data layer: schema migrations (goose)
// and the sqlc-generated typed queries. Kept OUT of the engine package so the
// WASM build (engine/ + cmd/wasm) stays dependency-free.
package db

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite" // pure-Go SQLite driver (no cgo) — registers "sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Open opens the SQLite database at path with foreign keys enforced + WAL, then
// applies all pending goose migrations. Uses the pure-Go modernc driver so the
// server cross-compiles cleanly like the rest of engine-go.
func Open(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)",
		path,
	)
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", path, err)
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite %q: %w", path, err)
	}
	if err := migrate(sqlDB); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return sqlDB, nil
}

func migrate(sqlDB *sql.DB) error {
	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("goose dialect: %w", err)
	}
	if err := goose.Up(sqlDB, "migrations"); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}
