package engine

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// TestEquippedFlagsParity proves ComputeEquippedFlags (effect-source.ts
// equippedItemFlagEffects) on real data: for each seed character it primes the
// catalogs, resolves the equipped-item flags, and asserts they match the TS
// oracle (`equippedFlags`, dumped by the frontend GEN_ORACLE harness) — the last
// effects-cru rule to move off TS (Fase A.3.3).
//
// Regenerate the oracle when the TS rules change:
//
//	GEN_ORACLE=1 pnpm --filter frontend test parity-oracle
func TestEquippedFlagsParity(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "parity"))
	catalogs := primeFromDump(t, dir)

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

	for _, slug := range slugs {
		slug := slug
		t.Run(slug, func(t *testing.T) {
			var oracle struct {
				Char          Character `json:"char"`
				EquippedFlags any       `json:"equippedFlags"`
			}
			readJSON(t, filepath.Join(dir, slug), &oracle)

			got := roundTrip(t, catalogs.ComputeEquippedFlags(oracle.Char.Items))
			if !reflect.DeepEqual(got, oracle.EquippedFlags) {
				diffReport(t, "equippedFlags", got, oracle.EquippedFlags)
			}
		})
	}
}
