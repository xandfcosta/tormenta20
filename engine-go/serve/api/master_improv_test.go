package api

import (
	"net/http"
	"strings"
	"testing"
)

// Os guardas do IMPROVISO (ALE-261).
//
// O dado em si é do `engine` e tem teste lá. O que se prende aqui é a tradução
// da linha do livro para a tela — que é onde um campo trocado passa por dado
// plausível.

// TestRollingReturnsAPatchAndStacks.
func TestRollingReturnsAPatchAndStacks(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	rec := pedeNoMestre(t, s, eu, "POST", "/mestre/improviso/ruina",
		`{"ruina":[{"r":2,"t":"Vazia"}]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("Content-Type %q — a rolagem navegou em vez de remendar", ct)
	}
	// A anterior tem de sobreviver: o histórico é o que separa esta tabela de um
	// botão que só mostra o último.
	if !strings.Contains(rec.Body.String(), "Vazia") {
		t.Error("o remendo perdeu a rolagem anterior")
	}
}

// TestAnInventedTableIsRefused: a rota é montada a partir da própria lista,
// então nome errado só chega por URL digitada à mão — e devolver a cena intacta
// faria parecer que o botão não funciona.
func TestAnInventedTableIsRefused(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	rec := pedeNoMestre(t, s, eu, "POST", "/mestre/improviso/tarot", `{}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d, quero 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "tarot") {
		t.Error("a recusa não diz qual tabela foi recusada")
	}
}

func TestClearingResetsOnlyThatTable(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	sinais := `{"ruina":[{"r":4,"t":"Vazia"}],"perseguicao":[{"r":9,"t":"Obstáculo"}],` +
		`"recompensa":[{"r":2,"t":"Favor"}],"ideias":[{"r":7,"t":"Cripta"}]}`
	rec := pedeNoMestre(t, s, eu, "POST", "/mestre/improviso/ruina/limpar", sinais)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	corpo := rec.Body.String()
	if strings.Contains(corpo, "Vazia") {
		t.Error("a ruína não foi limpa")
	}
	for _, sobrevivente := range []string{"Obstáculo", "Favor", "Cripta"} {
		if !strings.Contains(corpo, sobrevivente) {
			t.Errorf("limpar a ruína levou junto %q — as tabelas são independentes", sobrevivente)
		}
	}
}

// TestClearingAnInventedTableIsRefused, como a rolagem.
func TestClearingAnInventedTableIsRefused(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")
	rec := pedeNoMestre(t, s, eu, "POST", "/mestre/improviso/tarot/limpar", `{}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status %d, quero 400", rec.Code)
	}
}
