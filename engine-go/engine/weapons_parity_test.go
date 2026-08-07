package engine

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// TestWeaponCardsParity proves ComputeWeaponCards (WeaponFormulaCards) on real
// data: for each seed character it primes the catalogs, assembles the wielded-
// weapon cards (no active conditionals), and asserts they match the TS oracle
// (`weaponCards`, dumped by the frontend GEN_ORACLE harness).
//
// Regenerate the oracle when the TS rules change:
//
//	GEN_ORACLE=1 pnpm --filter frontend test parity-oracle
func TestWeaponCardsParity(t *testing.T) {
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

	for _, slug := range slugs {
		slug := slug
		t.Run(slug, func(t *testing.T) {
			var oracle struct {
				Char        Character `json:"char"`
				WeaponCards any       `json:"weaponCards"`
			}
			readJSON(t, filepath.Join(dir, slug), &oracle)

			got := roundTrip(t, catalogs.ComputeWeaponCards(oracle.Char, noConditionals))
			if !reflect.DeepEqual(got, oracle.WeaponCards) {
				diffReport(t, "weaponCards", got, oracle.WeaponCards)
			}
		})
	}
}
