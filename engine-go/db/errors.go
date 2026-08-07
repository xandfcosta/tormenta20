package db

import (
	"errors"

	sqlite "modernc.org/sqlite"
	sqlitelib "modernc.org/sqlite/lib"
)

// IsUniqueViolation reports whether err is a SQLite UNIQUE constraint failure —
// the backstop for racing inserts that pass an app-level pre-check (mirrors the
// Nest backend's isPrismaUniqueViolation → P2002).
func IsUniqueViolation(err error) bool {
	var e *sqlite.Error
	if errors.As(err, &e) {
		return e.Code() == sqlitelib.SQLITE_CONSTRAINT_UNIQUE
	}
	return false
}
