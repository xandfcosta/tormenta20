package table

import "strconv"

// myInitiativeOf é a iniciativa da MINHA linha na fila — o "Você está na fila
// com 22" do registro encolhido. Zero quando não estou na fila.
func myInitiativeOf(v View) int {
	for _, row := range v.Queue {
		if row.Mine {
			return row.Initiative
		}
	}
	return 0
}

// roundWritten é o "rodada 2" do cabeçalho da fila, vazio fora de combate.
func roundWritten(v View) string {
	if v.Turn.Kind == "idle" {
		return ""
	}
	return "rodada " + strconv.Itoa(v.Round)
}

// signed escreve o bônus com o sinal: "+7", "−1".
func signed(n int64) string {
	if n < 0 {
		return "−" + strconv.FormatInt(-n, 10)
	}
	return "+" + strconv.FormatInt(n, 10)
}
