package engine

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestItemEffectsParity feeds each seed character's real `activeItems` into
// ComputeItemEffects and asserts the resolved ItemEffects match the oracle
// semantically. It proves the resolution core on real data, not just the inline
// unit cases.
//
// Regenere o oráculo quando a regra mudar:
//
//	cd engine-go && go run ./cmd/genoracle
func TestItemEffectsParity(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "parity"))
	slugs := parityOracleSlugs(t, dir)

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
