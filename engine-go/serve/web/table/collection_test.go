package table

import (
	"strings"
	"testing"
)

// Os guardas do ACERVO que distingue ABERTO de FECHADO (ALE-205, fatia 3).
//
// O papercut que os motivou está nas palavras do dono: com 148 lugares, o que
// está na mesa AGORA aparecia no meio dos outros sem nada que o distinguisse —
// porque o `ShowPlace` arquivava a cena atual antes de trocar, e ela voltava
// para a lista como qualquer outra.

// savePlace encerra a cena aberta, o que a manda para o acervo, e devolve o
// id dela.
// collectionRow recorta o `<li>` que contém uma marca, e FALHA quando não acha.
//
// Falhar em vez de devolver vazio é o que separa este helper de um instrumento
// mudo: uma busca que não acha nada faria toda asserção seguinte passar sobre
// uma string vazia — o `strings.Contains(vazio, x)` é falso, e "não contém" é
// exatamente o que a maioria dos guardas daqui afirma.
func collectionRow(t *testing.T, screen, mark string) string {
	t.Helper()
	pos := strings.Index(screen, mark)
	if pos < 0 {
		t.Fatalf("não achei %q na tela: a asserção seguinte mediria uma string vazia", mark)
	}
	start := strings.LastIndex(screen[:pos], "<li ")
	end := strings.Index(screen[pos:], "</li>")
	if start < 0 || end < 0 {
		t.Fatalf("a marca %q não está dentro de um <li> do acervo", mark)
	}
	return screen[start : pos+end]
}
