package api

import (
	"net/http"
	"strings"
	"testing"
)

// Os guardas do CONSTRUTOR DE ENCONTROS (ALE-259).
//
// A conta em si é do `engine` e tem os testes dela lá, contra o livro. O que se
// prende aqui é a ÁLGEBRA DO RASCUNHO e a tradução do gesto — que é onde um
// erro apaga o encontro do mestre sem avisar.

// TestTheAddGestureReturnsAPatch, e não uma página: recarregar no meio
// da montagem perderia o rascunho, que só vive nos sinais.
func TestTheAddGestureReturnsAPatch(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	rec := pedeNoMestre(t, s, eu, "POST", "/mestre/encontros/adicionar/ogro", `{"encontro":[]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("Content-Type %q — o gesto navegou em vez de remendar", ct)
	}
	if !strings.Contains(rec.Body.String(), "Ogro") {
		t.Error("o remendo não trouxe a criatura acrescentada")
	}
}
