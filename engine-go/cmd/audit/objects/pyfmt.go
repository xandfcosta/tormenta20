package main

import (
	"strings"
	"t20engine/cmd/internal/pyfmt"
)

// pyDict é o `str()` de um dict de str do Python, na ordem de inserção dada.
func pyDict(m map[string]string, order []string) string {
	parts := make([]string, len(order))
	for i, k := range order {
		parts[i] = pyfmt.Repr(k) + ": " + pyfmt.Repr(m[k])
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
