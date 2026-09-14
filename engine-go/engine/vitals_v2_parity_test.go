package engine

import (
	"path/filepath"
	"reflect"
	"testing"
)

// TestVitalsParity proves the catalog-driven vitals (ComputeVitals) on real data:
// for each seed it builds the VitalContext (attrTotals from engine effects, no
// conditionals) and asserts {pvMax,pmMax} match the oracle's `vitals`. The lone
// god-power case is arcanista-erudito (Bênção do Mana). Regenerate:
//
//	cd engine-go && go run ./cmd/genoracle
func TestVitalsParity(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "parity"))
	catalogs := primeFromDump(t, dir)

	slugs := parityOracleSlugs(t, dir)

	for _, slug := range slugs {
		slug := slug
		t.Run(slug, func(t *testing.T) {
			var oracle struct {
				Char   Character `json:"char"`
				Vitals any       `json:"vitals"`
			}
			readJSON(t, filepath.Join(dir, slug), &oracle)

			ctx := catalogs.VitalContextFor(oracle.Char)
			got := roundTrip(t, catalogs.ComputeVitals(ctx))
			if !reflect.DeepEqual(got, oracle.Vitals) {
				diffReport(t, "vitals", got, oracle.Vitals)
			}
		})
	}
}
