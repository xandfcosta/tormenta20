package engine

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// TestSheetV2Parity proves the ported breakdown layer (ComputeSheetV2) on real
// data: for each seed character it primes the catalogs, computes the full sheet
// (no active conditionals), and asserts every breakdown matches the TS oracle
// (`sheetV2`, dumped by the frontend GEN_ORACLE harness) semantically. This is
// slice 3 / task #5's target — the collection (slice 2) + resolution (slice 1)
// tests cover the upstream halves.
//
// Regenerate the oracle when the TS rules change:
//
//	GEN_ORACLE=1 pnpm --filter frontend test parity-oracle
func TestSheetV2Parity(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "parity"))
	catalogs := primeFromDump(t, dir)
	noConditionals := map[string]bool{}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read parity dir %s: %v (run the frontend GEN_ORACLE harness)", dir, err)
	}
	var slugs []string
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".json" && e.Name()[0] != '_' {
			slugs = append(slugs, e.Name())
		}
	}
	sort.Strings(slugs)
	if len(slugs) != 16 {
		t.Fatalf("expected 16 oracle files, found %d", len(slugs))
	}

	for _, slug := range slugs {
		slug := slug
		t.Run(slug, func(t *testing.T) {
			var oracle struct {
				Char    Character `json:"char"`
				SheetV2 any       `json:"sheetV2"`
			}
			readJSON(t, filepath.Join(dir, slug), &oracle)

			got := roundTrip(t, catalogs.ComputeSheetV2(oracle.Char, noConditionals))
			if !reflect.DeepEqual(got, oracle.SheetV2) {
				diffReport(t, "sheetV2", got, oracle.SheetV2)
			}
		})
	}
}
