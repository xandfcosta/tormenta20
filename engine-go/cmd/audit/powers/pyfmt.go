package main

import (
	"sort"
	"strconv"
	"strings"
	"t20engine/cmd/internal/pyfmt"
)

// A saída deste auditor é a do script Python que ele substituiu, palavra por
// palavra — inclusive as listas e os dicts que o f-string imprimia com `repr`.

// stringSet é o `set[str]` do Python; `nil` faz o papel do `None`.
type stringSet map[string]bool

// sorted é o `sorted(set)`: a ordem por ponto de código, que é a ordem dos bytes
// em UTF-8.
func (s stringSet) sorted() []string {
	out := make([]string, 0, len(s))
	for x := range s {
		out = append(out, x)
	}
	sort.Strings(out)
	return out
}

func (s stringSet) minus(o stringSet) stringSet {
	out := stringSet{}
	for x := range s {
		if !o[x] {
			out[x] = true
		}
	}
	return out
}

func (s stringSet) equal(o stringSet) bool {
	if len(s) != len(o) {
		return false
	}
	for x := range s {
		if !o[x] {
			return false
		}
	}
	return true
}

// countEntry é um par de um dict `{str: int}` em ordem de inserção.
type countEntry struct {
	key string
	n   int
}

// pyCountDict é o `str()` de um dict `{str: int}`.
func pyCountDict(entries []countEntry) string {
	parts := make([]string, len(entries))
	for i, e := range entries {
		parts[i] = pyfmt.Repr(e.key) + ": " + strconv.Itoa(e.n)
	}
	return "{" + strings.Join(parts, ", ") + "}"
}
