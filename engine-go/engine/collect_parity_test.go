package engine

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestActiveItemsParity proves the ported collection layer (ActiveItemsFor) on
// real data: for each seed character it primes the catalogs from _catalogs.json,
// re-collects the raw Character, and asserts the []ActiveItem match the TS oracle
// (`activeItems`, dumped by the frontend GEN_ORACLE harness) semantically. This is
// slice 2's target — the resolution test (slice 1) covers the downstream half.
//
// Regenerate the oracle + catalog dump when the TS rules change:
//
//	GEN_ORACLE=1 pnpm --filter frontend test parity-oracle
func TestActiveItemsParity(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "parity"))
	catalogs := primeFromDump(t, dir)

	slugs := parityOracleSlugs(t, dir)

	for _, slug := range slugs {
		slug := slug
		t.Run(slug, func(t *testing.T) {
			var oracle struct {
				Char        Character `json:"char"`
				ActiveItems any       `json:"activeItems"`
			}
			readJSON(t, filepath.Join(dir, slug), &oracle)

			got := roundTrip(t, catalogs.ActiveItemsFor(oracle.Char))
			if !reflect.DeepEqual(got, oracle.ActiveItems) {
				diffReport(t, "activeItems", got, oracle.ActiveItems)
			}
		})
	}
}

// primeFromDump loads engine-go/parity/_catalogs.json into an indexed Catalogs —
// the same data ensureCatalogs primes the frontend caches with.
func primeFromDump(t *testing.T, dir string) *Catalogs {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, "_catalogs.json"))
	if err != nil {
		t.Fatalf("read _catalogs.json: %v (run the frontend GEN_ORACLE harness)", err)
	}
	catalogs, err := PrimeEngineCatalogs(raw)
	if err != nil {
		t.Fatalf("prime catalogs: %v", err)
	}
	return catalogs
}
