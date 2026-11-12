package api

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// O e2e do tabuleiro reprovou duas vezes com "criar a campanha descartável:
// 500", em 5,0s, e o servidor não deixou linha nenhuma: o handler descartava o
// erro e respondia uma frase genérica. Este caso derruba a tabela para a criação
// falhar de verdade, pelo roteador inteiro, e exige que o log diga QUAL
// requisição falhou e POR QUÊ.
func TestACampaignThatFailsToOpenLogsTheCause(t *testing.T) {
	s := newTestServer(t)
	var logged bytes.Buffer
	s.log = slog.New(slog.NewJSONHandler(&logged, nil))
	owner := seedUser(t, s, "mestre-do-log@t20.local")
	if _, err := s.db.Exec("DROP TABLE campaigns"); err != nil {
		t.Fatalf("derrubar a tabela das campanhas: %v", err)
	}

	user, err := s.queries.GetUserByID(t.Context(), owner)
	if err != nil {
		t.Fatalf("ler o usuário %d: %v", owner, err)
	}
	token, err := s.accountGate().SignSession(user)
	if err != nil {
		t.Fatalf("assinar a sessão: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/campanhas", strings.NewReader(`{"name":"Mesa que não abre"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	s.WebRouter().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("a criação respondeu %d sem a tabela; o caso precisa de um 500 para medir o log", rec.Code)
	}
	out := logged.String()
	if !strings.Contains(out, `"path":"/api/campanhas"`) {
		t.Errorf("o log não diz qual requisição respondeu 500:\n%s", out)
	}
	if !strings.Contains(out, "no such table: campaigns") {
		t.Errorf("o log não traz a causa do 500 (no such table: campaigns):\n%s", out)
	}
}
