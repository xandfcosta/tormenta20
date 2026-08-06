package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// benchDir locates bench/ relative to the repo root (two levels up from
// engine-go/engine).
func benchDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", "..", "bench"))
}

// TestParity runs ComputeCharacterSheet on every bench payload and compares
// the result to the pre-computed TS oracle SEMANTICALLY (values, not bytes).
func TestParity(t *testing.T) {
	bench := benchDir(t)
	payloadDir := filepath.Join(bench, "payloads")
	expectedDir := filepath.Join(bench, "expected")

	entries, err := os.ReadDir(payloadDir)
	if err != nil {
		t.Fatalf("read payloads dir %s: %v", payloadDir, err)
	}

	var slugs []string
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".json" {
			slugs = append(slugs, e.Name())
		}
	}
	sort.Strings(slugs)
	if len(slugs) != 16 {
		t.Fatalf("expected 16 payloads, found %d", len(slugs))
	}

	for _, slug := range slugs {
		slug := slug
		t.Run(slug, func(t *testing.T) {
			var input CharacterInput
			readJSON(t, filepath.Join(payloadDir, slug), &input)

			got := ComputeCharacterSheet(&input)
			gotAny := roundTrip(t, got)

			var wantAny any
			readJSON(t, filepath.Join(expectedDir, slug), &wantAny)

			if !reflect.DeepEqual(gotAny, wantAny) {
				diffReport(t, "", gotAny, wantAny)
			}
		})
	}
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
			diffReport(t, path+"["+itoa(i)+"]", g[i], w[i])
		}
	default:
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got %v want %v", path, got, want)
		}
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
