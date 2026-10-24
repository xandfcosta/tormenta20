// Package pyfmt imprime valores como o Python os imprimia.
//
// Os auditores do catálogo (`cmd/audit/...`) e as ferramentas que gravam nele
// nasceram em Python e viraram Go na ALE-498 com a saída provada igual linha a
// linha. O texto que eles imprimem tem listas e strings no formato do `repr` —
// é o que quem lê o relatório reconhece —, e este pacote é a única cópia dessas
// regras: elas estavam repetidas em quinze pacotes, em quatro variantes.
//
//	fmt.Println(pyfmt.List([]string{"Luta", "d'água"})) // ['Luta', "d'água"]
package pyfmt

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

// Repr é o `repr()` de uma str: aspas simples salvo quando o texto tem aspa
// simples e não tem dupla, e escape do que o Python não considera imprimível —
// o livro traz hífen discricionário (U+00AD), que sai `\xad`.
func Repr(s string) string {
	quote := '\''
	if strings.ContainsRune(s, '\'') && !strings.ContainsRune(s, '"') {
		quote = '"'
	}
	var b strings.Builder
	b.WriteRune(quote)
	for _, r := range s {
		switch {
		case r == '\\' || r == quote:
			b.WriteRune('\\')
			b.WriteRune(r)
		case r == '\t':
			b.WriteString(`\t`)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case printable(r):
			b.WriteRune(r)
		case r <= 0xff:
			fmt.Fprintf(&b, `\x%02x`, r)
		case r <= 0xffff:
			fmt.Fprintf(&b, `\u%04x`, r)
		default:
			fmt.Fprintf(&b, `\U%08x`, r)
		}
	}
	b.WriteRune(quote)
	return b.String()
}

// printable é o `str.isprintable` de um caractere: tudo menos as categorias
// Outro e Separador, salvo o espaço ASCII.
func printable(r rune) bool {
	if r == ' ' {
		return true
	}
	if unicode.In(r, unicode.Cc, unicode.Cf, unicode.Cs, unicode.Co, unicode.Zl, unicode.Zp, unicode.Zs) {
		return false
	}
	return unicode.In(r, unicode.L, unicode.M, unicode.N, unicode.P, unicode.S)
}

// List é o `str()` de uma lista de str.
func List(xs []string) string {
	parts := make([]string, len(xs))
	for i, x := range xs {
		parts[i] = Repr(x)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// SortedList é o `str(sorted(xs))`: a ordem por ponto de código, que é a dos
// bytes em UTF-8.
func SortedList(xs []string) string { return List(slices.Sorted(slices.Values(xs))) }

// Sorted é o `str(sorted(conjunto))`.
func Sorted(set map[string]bool) string {
	xs := make([]string, 0, len(set))
	for x := range set {
		xs = append(xs, x)
	}
	return SortedList(xs)
}

// Percent é o `format(x, '.0%')`: o arredondamento correto do `FormatFloat` é o
// mesmo do Python.
func Percent(x float64) string { return strconv.FormatFloat(x*100, 'f', 0, 64) + "%" }

// Optional é o número de um campo que pode faltar: o zero sai `None`, porque é
// assim que o auditor via o campo ausente.
func Optional(x float64) string {
	if x == 0 {
		return "None"
	}
	return strconv.FormatFloat(x, 'f', -1, 64)
}

// OptionalRepr é o `repr` de uma str que pode faltar: a vazia sai `None`.
func OptionalRepr(s string) string {
	if s == "" {
		return "None"
	}
	return Repr(s)
}

// Bool é o `str()` de um bool.
func Bool(b bool) string {
	if b {
		return "True"
	}
	return "False"
}

// FirstRunes é o `s[:n]`, que corta por ponto de código e não por byte.
func FirstRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
