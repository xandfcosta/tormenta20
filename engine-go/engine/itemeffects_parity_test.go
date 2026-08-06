package engine

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// TestItemEffectsParity feeds each seed character's real `activeItems` (collected
// by the TS collection layer, dumped by the frontend parity harness) into the
// ported ComputeItemEffects and asserts the resolved ItemEffects match the TS
// oracle semantically. This proves the resolution core (slice 1) on real data,
// not just the inline unit cases. See PORT-PLAN.md §3.
//
// Regenerate the oracle when the TS rules change:
//
//	GEN_ORACLE=1 pnpm --filter frontend test parity-oracle
func TestItemEffectsParity(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "parity"))
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read parity dir %s: %v (run the frontend GEN_ORACLE harness)", dir, err)
	}

	var slugs []string
	for _, e := range entries {
		// `_`-prefixed files are shared inputs (e.g. _catalogs.json), not oracles.
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
				ActiveItems []ActiveItem `json:"activeItems"`
				ItemEffects any          `json:"itemEffects"`
			}
			readJSON(t, filepath.Join(dir, slug), &oracle)

			got := roundTrip(t, ComputeItemEffects(oracle.ActiveItems))
			if !reflect.DeepEqual(got, oracle.ItemEffects) {
				diffReport(t, "itemEffects", got, oracle.ItemEffects)
			}
		})
	}
}

func mustWd(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return wd
}
