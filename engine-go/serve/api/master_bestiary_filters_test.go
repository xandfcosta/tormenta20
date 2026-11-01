package api

import (
	"strings"
	"testing"
)

// O Bestiário oferece os filtros num diálogo, como os catálogos, e o campo de ND do
// diálogo não pode repetir o id do campo da gaveta: dois `id` iguais fazem o rótulo
// de um focar o outro.
func TestTheBestiaryFiltersOpenInADialog(t *testing.T) {
	s := newTestServer(t)
	gm := seedUser(t, s, "mestre@t20.local")

	body := pedeNoMestre(t, s, gm, "GET", "/mestre/bestiario", "").Body.String()
	// O id é montado do prefixo de quem chama, e a varredura de ids só casa o literal.
	filters := "bestiario" + "-filters"
	if !strings.Contains(body, `commandfor="`+filters+`"`) || !strings.Contains(body, `id="`+filters+`"`) {
		t.Fatal("o Bestiário não oferece os filtros num diálogo")
	}
	if n := strings.Count(body, `id="bestiario-cr-min"`); n != 1 {
		t.Errorf("o campo de ND mínimo saiu com o mesmo id %d vezes", n)
	}
	if !strings.Contains(body, `id="bestiario-sheet-cr-min"`) {
		t.Error("o diálogo não tem o próprio campo de ND mínimo")
	}
}
