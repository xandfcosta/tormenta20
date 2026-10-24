package main

import (
	"strconv"
)

// pyIntRepr é o `repr()` de um int que pode ser `None`.
func pyIntRepr(n *int) string {
	if n == nil {
		return "None"
	}
	return strconv.Itoa(*n)
}
