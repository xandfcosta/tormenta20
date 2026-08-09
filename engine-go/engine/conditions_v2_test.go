package engine

import (
	"path/filepath"
	"testing"
)

// TestConditionEffects checks that p394 status conditions move the v2 sheet
// numbers (ALE-28). Cross-language parity is proven separately by the sheetV2
// oracle (recruta-nv1-simples carries conditions); this asserts the actual
// deltas + the "aplique o mais severo" non-stacking rule Go-side.
func TestConditionEffects(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "parity"))
	catalogs := primeFromDump(t, dir)
	none := map[string]bool{}

	var oracle struct {
		Char Character `json:"char"`
	}
	// aprendiz-nv1-simples ships with no conditions — a clean baseline.
	readJSON(t, filepath.Join(dir, "aprendiz-nv1-simples.json"), &oracle)
	clean := oracle.Char

	with := func(ids string) ComputedSheetV2 {
		c := clean
		c.ActiveConditions = ids
		return catalogs.ComputeSheetV2(c, none)
	}
	expertise := func(s ComputedSheetV2, name string) int {
		for _, e := range s.Expertises {
			if e.Name == name {
				return e.Total
			}
		}
		t.Fatalf("expertise %s not found", name)
		return 0
	}

	base := catalogs.ComputeSheetV2(clean, none)

	if got := with(`["vulneravel"]`).Defense.Total; got != base.Defense.Total-2 {
		t.Errorf("Vulnerável: Defesa = %d, quer %d", got, base.Defense.Total-2)
	}
	if got := expertise(with(`["abalado"]`), "Fortitude"); got != expertise(base, "Fortitude")-2 {
		t.Errorf("Abalado: Fortitude = %d, quer %d", got, expertise(base, "Fortitude")-2)
	}
	// mais severo (p394): −2 + −5 na Defesa = −5, não −7.
	if got := with(`["vulneravel","desprevenido"]`).Defense.Total; got != base.Defense.Total-5 {
		t.Errorf("Vulnerável+Desprevenido: Defesa = %d, quer %d", got, base.Defense.Total-5)
	}
	// Caído penaliza Luta (corpo-a-corpo), não Pontaria.
	caido := with(`["caido"]`)
	if got := expertise(caido, "Luta"); got != expertise(base, "Luta")-5 {
		t.Errorf("Caído: Luta = %d, quer %d", got, expertise(base, "Luta")-5)
	}
	if got := expertise(caido, "Pontaria"); got != expertise(base, "Pontaria") {
		t.Errorf("Caído não deveria afetar Pontaria: %d != %d", got, expertise(base, "Pontaria"))
	}
	// Condição só-lembrete não altera nada.
	if with(`["lento"]`).Defense.Total != base.Defense.Total {
		t.Errorf("Lento não deveria mexer na Defesa")
	}
}
