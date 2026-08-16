package engine

import (
	"path/filepath"
	"reflect"
	"testing"
)

// TestSheetV2Parity proves the ported breakdown layer (ComputeSheetV2) on real
// data: for each seed character it primes the catalogs, computes the full sheet
// and asserts every breakdown matches the TS oracle (`sheetV2`, dumped by the
// gerado por `go run ./cmd/genoracle`) semantically. This is slice 3 / task #5's target
// — the collection (slice 2) + resolution (slice 1) tests cover the upstream
// halves.
//
// Each character is checked TWICE: with no opt-in toggled, and with every
// conditional ON. The second pass is the only golden coverage of
// `ApplyActiveConditionals` — the fold that re-runs `resolveStack` per target —
// which ran on synthetic data only until ALE-106. It bites immediately on
// `bardo-versatil-nv7`, whose two Inspiração opt-ins hit the same target with
// the same bonusType: +1 and +2 must resolve to +2 across all 29 perícias, not
// +3.
//
// Regenerate the oracle when the TS rules change:
//
//	cd engine-go && go run ./cmd/genoracle
func TestSheetV2Parity(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "parity"))
	catalogs := primeFromDump(t, dir)

	for _, slug := range parityOracleSlugs(t, dir) {
		slug := slug
		t.Run(slug, func(t *testing.T) {
			var oracle struct {
				Char               Character `json:"char"`
				SheetV2            any       `json:"sheetV2"`
				ActiveConditionals []string  `json:"activeConditionals"`
				WithConditionals   any       `json:"sheetV2WithConditionals"`
			}
			readJSON(t, filepath.Join(dir, slug), &oracle)

			got := roundTrip(t, catalogs.ComputeSheetV2(oracle.Char, map[string]bool{}))
			if !reflect.DeepEqual(got, oracle.SheetV2) {
				diffReport(t, "sheetV2", got, oracle.SheetV2)
			}

			on := toSet(oracle.ActiveConditionals)
			gotOn := roundTrip(t, catalogs.ComputeSheetV2(oracle.Char, on))
			if !reflect.DeepEqual(gotOn, oracle.WithConditionals) {
				diffReport(t, "sheetV2WithConditionals", gotOn, oracle.WithConditionals)
			}
		})
	}
}
