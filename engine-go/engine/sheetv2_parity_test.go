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
	comConditionais := 0

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

			// A segunda passada só EXISTE para exercitar `ApplyActiveConditionals`,
			// e 15 dos 18 personagens não têm condicional nenhum: ali ela repetia a
			// primeira e contava como cobertura. Rodar só onde há o que ligar deixa
			// claro quantos realmente exercitam a dobra.
			if len(oracle.ActiveConditionals) == 0 {
				return
			}
			comConditionais++
			on := toSet(oracle.ActiveConditionals)
			gotOn := roundTrip(t, catalogs.ComputeSheetV2(oracle.Char, on))
			if !reflect.DeepEqual(gotOn, oracle.WithConditionals) {
				diffReport(t, "sheetV2WithConditionals", gotOn, oracle.WithConditionals)
			}
		})
	}

	// A dobra dos condicionais é o que pegou a Inspiração dupla do bardo
	// (ALE-106): se um dia NENHUM oráculo tiver condicional, a cobertura dela
	// vira zero em silêncio — e é isso que esta linha impede.
	if comConditionais == 0 {
		t.Error("nenhum oráculo exercitou ApplyActiveConditionals — a dobra ficou sem prova")
	}
}
