package db

// Regenerate the typed query layer (db/sqlcgen) after editing db/query.sql or the
// migrations. The generated code is committed, so the checked-in output is the
// source of truth; pin the sqlc version here if cross-version drift ever bites.
//
//	cd engine-go && go generate ./db
//
//go:generate go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate
