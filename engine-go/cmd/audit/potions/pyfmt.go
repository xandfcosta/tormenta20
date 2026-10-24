package main

import (
	"strconv"
	"t20engine/cmd/internal/pyfmt"
)

// A saída é a do script Python que este comando substituiu, palavra por
// palavra; estes são os pedaços de formatação do Python que o `fmt` não tem.

// optionalRepr é o `repr()` de um `get()` de texto: no dado tipado "ausente" e
// "" são o mesmo, e o ausente é o que o Python via — `None`.
func optionalRepr(s string) string {
	if s == "" {
		return "None"
	}
	return pyfmt.Repr(s)
}

// optionalInt é o mesmo para número: zero é o campo que o JSON omite.
func optionalInt(n int) string {
	if n == 0 {
		return "None"
	}
	return strconv.Itoa(n)
}
