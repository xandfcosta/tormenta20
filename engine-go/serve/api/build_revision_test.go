package api

import (
	"bufio"
	"context"
	"encoding/json"
	"html"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// O `/health` diz qual commit está no ar: é por ele que o deploy confere que a versão nova
// subiu, e não um processo velho que continuou respondendo na porta.
func TestTheHealthSaysWhichRevisionIsRunning(t *testing.T) {
	s := newTestServer(t)
	s.revision = "abc123"

	rec := httptest.NewRecorder()
	s.HealthProbe().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	var body struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("o /health não é JSON: %v", err)
	}
	if body.Version != "abc123" {
		t.Errorf("o /health diz a versão %q, esperava abc123", body.Version)
	}
}

// A página guarda a versão com que nasceu, e o fluxo diz a versão do servidor a cada
// conexão. Depois de um deploy, o fluxo reconecta sozinho e a diferença acende o aviso.
func TestThePageKeepsItsRevisionAndTheStreamSaysTheServers(t *testing.T) {
	f := newSceneFixture(t)
	f.s.revision = "versao-da-pagina"

	page := html.UnescapeString(f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String())
	if !strings.Contains(page, "page_build: 'versao-da-pagina'") {
		t.Error("a página não guardou a versão com que nasceu")
	}
	if !strings.Contains(page, `id="new-version"`) {
		t.Error("a página não traz o aviso de versão nova")
	}

	srv := httptest.NewServer(f.s.WebRouter())
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+f.tableUrl()+"/fluxo", nil)
	if err != nil {
		t.Fatalf("montar pedido: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+f.token(t, f.gm))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("abrir o fluxo: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	reader := bufio.NewReader(resp.Body)
	for {
		line, readErr := reader.ReadString('\n')
		if strings.Contains(line, `"server_build":"versao-da-pagina"`) {
			return
		}
		if readErr != nil {
			t.Fatalf("o fluxo não disse a versão do servidor: %v", readErr)
		}
	}
}
