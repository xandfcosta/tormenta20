package api

import (
	"bufio"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// O desligamento ABORTA o fluxo da mesa. Fechá-lo limpo faria o `Shutdown` terminar
// do mesmo jeito, mas para o Datastar é uma resposta completa: ele não reconecta, e a
// mesa de quem estava jogando ficaria parada depois do deploy, sem aviso nenhum.
func TestClosingLiveStreamsAbortsTheConnectionSoTheBrowserRetries(t *testing.T) {
	f := newSceneFixture(t)
	srv := httptest.NewServer(f.s.WebRouter())
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL+f.tableUrl()+"/fluxo", nil)
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
	if frame := lerQuadro(t, reader); frame == "" {
		t.Fatal("o fluxo não mandou o primeiro quadro: o caso mediria uma conexão morta")
	}

	f.s.CloseLiveStreams()

	ended := make(chan error, 1)
	go func() {
		_, readErr := io.Copy(io.Discard, reader)
		ended <- readErr
	}()
	select {
	case readErr := <-ended:
		if readErr == nil {
			t.Error("o fluxo fechou limpo: o Datastar não reconectaria depois do deploy")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("o fluxo continuou aberto depois de CloseLiveStreams: o desligamento gastaria o prazo inteiro")
	}
}
