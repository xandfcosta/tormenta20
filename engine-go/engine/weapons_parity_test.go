package engine

import (
	"path/filepath"
	"reflect"
	"testing"
)

// TestWeaponCardsParity proves ComputeWeaponCards (WeaponFormulaCards) on real
// data: for each seed character it primes the catalogs, assembles the wielded-
// weapon cards and asserts they match the TS oracle (`weaponCards`, dumped by
// gerado por `go run ./cmd/genoracle`).
//
// Also with every conditional ON (ALE-106) — a card is exactly where an opt-in
// like a Fúria that grants +2 em ataque and dano lands, and the base pass could
// never see it.
//
// Regenerate the oracle when the TS rules change:
//
//	cd engine-go && go run ./cmd/genoracle
func TestWeaponCardsParity(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "parity"))
	catalogs := primeFromDump(t, dir)

	for _, slug := range parityOracleSlugs(t, dir) {
		slug := slug
		t.Run(slug, func(t *testing.T) {
			var oracle struct {
				Char               Character `json:"char"`
				WeaponCards        any       `json:"weaponCards"`
				ActiveConditionals []string  `json:"activeConditionals"`
				WithConditionals   any       `json:"weaponCardsWithConditionals"`
			}
			readJSON(t, filepath.Join(dir, slug), &oracle)

			got := roundTrip(t, catalogs.ComputeWeaponCards(oracle.Char, map[string]bool{}))
			if !reflect.DeepEqual(got, oracle.WeaponCards) {
				diffReport(t, "weaponCards", got, oracle.WeaponCards)
			}

			on := toSet(oracle.ActiveConditionals)
			gotOn := roundTrip(t, catalogs.ComputeWeaponCards(oracle.Char, on))
			if !reflect.DeepEqual(gotOn, oracle.WithConditionals) {
				diffReport(t, "weaponCardsWithConditionals", gotOn, oracle.WithConditionals)
			}
		})
	}
}
