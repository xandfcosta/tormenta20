package api

import "strings"

// AcceptsEncoding responde se o cabeçalho `Accept-Encoding` lista a codificação.
//
// Comparação por TOKEN e não por `strings.Contains`: "gzip;q=0" é uma RECUSA
// explícita, e um `Contains` a leria como aceitação, mandando conteúdo
// comprimido para quem disse que não quer (ALE-153).
//
// Mora aqui, e não em `cmd/api`, porque quem serve a SPA e quem serve o
// catálogo precisam da MESMA leitura — duas cópias divergiriam justamente no
// caso do `q=0`, que é o que ninguém lembra de tratar (ALE-159).
func AcceptsEncoding(header, encoding string) bool {
	for _, part := range strings.Split(header, ",") {
		fields := strings.Split(strings.TrimSpace(part), ";")
		if !strings.EqualFold(strings.TrimSpace(fields[0]), encoding) {
			continue
		}
		for _, param := range fields[1:] {
			if strings.EqualFold(strings.TrimSpace(param), "q=0") {
				return false
			}
		}
		return true
	}
	return false
}
