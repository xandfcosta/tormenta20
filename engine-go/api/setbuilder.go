package api

import (
	"context"
	"database/sql"
	"strings"
)

// setBuilder accumulates a partial UPDATE: the columns a PATCH body actually
// carried, plus their values, in order.
//
// Four handlers (campaigns, sessions, character items, character abilities) each
// grew their own `sets []string` / `args []any` pair — and copied the
// `//nolint:gosec` alongside, which is the part that matters: the suppression
// says "the SET clause is a fixed allowlist, not input", and that promise was
// being re-asserted in four places where a fifth could quietly break it.
// Concentrating it here means the claim is made once, next to the only code
// that builds the clause.
//
// Column names come from literals at the call sites, never from the request —
// only the values are bound.
//
// @example
//
//	var set setBuilder
//	set.add("name = ?", name)
//	if err := set.exec(ctx, s.db, "UPDATE campaigns", id); err != nil { … }
type setBuilder struct {
	columns []string
	args    []any
}

// add records one column assignment. `clause` must be a literal like "name = ?".
func (b *setBuilder) add(clause string, value any) {
	b.columns = append(b.columns, clause)
	b.args = append(b.args, value)
}

// empty reports whether the PATCH carried no updatable field — the caller answers
// 400 rather than running an UPDATE that changes nothing but `updatedAt`.
func (b *setBuilder) empty() bool { return len(b.columns) == 0 }

// exec runs `<prefix> SET <clauses> WHERE id = ?` as recorded, touching no
// timestamp — `character_items` has no `updatedAt` column at all.
func (b *setBuilder) exec(ctx context.Context, db *sql.DB, prefix string, id int64) error {
	//nolint:gosec // The SET clause is built from column literals at the call
	// sites — a fixed allowlist. Only values are bound.
	_, err := db.ExecContext(ctx,
		prefix+" SET "+strings.Join(b.columns, ", ")+" WHERE id = ?",
		append(append([]any{}, b.args...), id)...)
	return err
}

// execTouched is exec plus an `updatedAt` stamp — the shape for every table that
// HAS the column. Kept as a separate call so the difference is visible where it
// matters instead of hidden behind a bool.
func (b *setBuilder) execTouched(ctx context.Context, db *sql.DB, prefix string, id int64) error {
	stamped := setBuilder{
		columns: append(append([]string{}, b.columns...), "updatedAt = ?"),
		args:    append(append([]any{}, b.args...), nowISO()),
	}
	return stamped.exec(ctx, db, prefix, id)
}
