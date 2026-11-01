package convention

import "strings"

func contem(tokens []string, target string) bool {
	for _, t := range tokens {
		if t == target {
			return true
		}
	}
	return false
}

func comPrefixo(tokens []string, prefix string) bool {
	for _, t := range tokens {
		if strings.HasPrefix(t, prefix) {
			return true
		}
	}
	return false
}

// semComentario corta o que vier depois de `//` em cada linha. Grosseiro de
// propósito: `//` dentro de string literal viraria corte indevido, mas em
// template isso é raro e o custo do erro é um falso NEGATIVO — o guarda deixa
// passar —, não um falso positivo que faz alguém desligá-lo.
func semComentario(rows []string) []string {
	outside := make([]string, len(rows))
	for i, l := range rows {
		if j := strings.Index(l, "//"); j >= 0 {
			l = l[:j]
		}
		outside[i] = l
	}
	return outside
}
