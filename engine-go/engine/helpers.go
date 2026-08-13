package engine

import (
	"fmt"
	"strings"
)

// stripAccentsLower normalizes a PT expertise name to its ASCII slug form
// (NFD-strip + lowercase), mirroring deformidade.ts expertiseNameToSkillId.
func stripAccentsLower(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		if repl, ok := accentFold[r]; ok {
			b.WriteRune(repl)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

var accentFold = map[rune]rune{
	'á': 'a', 'à': 'a', 'â': 'a', 'ã': 'a', 'ä': 'a',
	'é': 'e', 'è': 'e', 'ê': 'e', 'ë': 'e',
	'í': 'i', 'ì': 'i', 'î': 'i', 'ï': 'i',
	'ó': 'o', 'ò': 'o', 'ô': 'o', 'õ': 'o', 'ö': 'o',
	'ú': 'u', 'ù': 'u', 'û': 'u', 'ü': 'u',
	'ç': 'c', 'ñ': 'n',
}

func errf(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func hasDuplicates(s []string) bool {
	seen := map[string]bool{}
	for _, x := range s {
		if seen[x] {
			return true
		}
		seen[x] = true
	}
	return false
}

func toSet(s []string) map[string]bool {
	out := make(map[string]bool, len(s))
	for _, x := range s {
		out[x] = true
	}
	return out
}
