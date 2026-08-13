package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"testing"
)

// Shared helpers for the parity suites (collect, itemeffects, sheetV2, vitals,
// equipped flags, weapon cards). They used to live in `parity_test.go` — the MVP
// engine's own test — which is why deleting that engine required moving them
// out first rather than dropping the file wholesale.

// parityOracleCount é quantos personagens-oráculo existem. Vive aqui, uma vez:
// seis suítes de paridade varriam o diretório com o mesmo laço e três delas
// repetiam o número, então acrescentar uma fixture obrigava a caçar cópias.
const parityOracleCount = 18

// parityOracleSlugs lista os arquivos de oráculo em ordem estável. Os
// `_`-prefixados são entrada compartilhada (`_catalogs.json`), não oráculo.
//
// A contagem é verificada aqui porque o modo de falha silencioso é o oposto do
// esperado: uma fixture nova do lado TS sem regenerar não quebra nada, ela
// simplesmente não é testada.
func parityOracleSlugs(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ler o diretório de paridade %s: %v (rode o harness GEN_ORACLE do front)", dir, err)
	}
	var slugs []string
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".json" && e.Name()[0] != '_' {
			slugs = append(slugs, e.Name())
		}
	}
	sort.Strings(slugs)
	if len(slugs) != parityOracleCount {
		t.Fatalf("esperados %d oráculos, achados %d — regenere com `GEN_ORACLE=1 pnpm --filter frontend test parity-oracle`", parityOracleCount, len(slugs))
	}
	return slugs
}

func readJSON(t *testing.T, path string, dst any) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if err := json.Unmarshal(b, dst); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
}

// roundTrip marshals the computed sheet and re-parses it into an untyped tree,
// so the comparison is value-based and key-order independent.
func roundTrip(t *testing.T, v any) any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal computed sheet: %v", err)
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("re-unmarshal computed sheet: %v", err)
	}
	return out
}

// diffReport walks both trees and reports the first mismatching leaf paths —
// the numeric diffs that matter for an integer-stats engine.
func diffReport(t *testing.T, path string, got, want any) {
	switch w := want.(type) {
	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok {
			t.Errorf("%s: type mismatch got %T want map", path, got)
			return
		}
		keys := map[string]bool{}
		for k := range w {
			keys[k] = true
		}
		for k := range g {
			keys[k] = true
		}
		var sorted []string
		for k := range keys {
			sorted = append(sorted, k)
		}
		sort.Strings(sorted)
		for _, k := range sorted {
			diffReport(t, path+"."+k, g[k], w[k])
		}
	case []any:
		g, ok := got.([]any)
		if !ok {
			t.Errorf("%s: type mismatch got %T want slice", path, got)
			return
		}
		if len(g) != len(w) {
			t.Errorf("%s: slice length got %d want %d", path, len(g), len(w))
			return
		}
		for i := range w {
			diffReport(t, path+"["+strconv.Itoa(i)+"]", g[i], w[i])
		}
	default:
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got %v want %v", path, got, want)
		}
	}
}
