package httpio

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// logLines lê a saída de um `slog.NewJSONHandler`, uma linha por registro.
func logLines(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var lines []map[string]any
	for _, raw := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if raw == "" {
			continue
		}
		var line map[string]any
		if err := json.Unmarshal([]byte(raw), &line); err != nil {
			t.Fatalf("o log não é JSON: %q (%v)", raw, err)
		}
		lines = append(lines, line)
	}
	return lines
}

func serveThroughTheLog(t *testing.T, handler http.HandlerFunc) []map[string]any {
	t.Helper()
	var buf bytes.Buffer
	logged := LogServerErrors(slog.New(slog.NewJSONHandler(&buf, nil)))(handler)
	logged.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/campanhas", nil))
	return logLines(t, &buf)
}

// Um 500 sem log é o que escondeu a trava de escrita do SQLite no e2e do
// tabuleiro: o caso reprovou em 5,0s, que é o `busy_timeout`, e nada dizia
// qual escrita segurava a trava.
func TestEveryServerErrorLeavesALogLine(t *testing.T) {
	lines := serveThroughTheLog(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "gravar a campanha: database is locked (5) (SQLITE_BUSY)", http.StatusInternalServerError)
	})
	if len(lines) != 1 {
		t.Fatalf("um 500 tinha de deixar UMA linha no log, e deixou %d: %v", len(lines), lines)
	}
	line := lines[0]
	if line["level"] != "ERROR" {
		t.Errorf("o 500 foi registrado como %v, e não como ERROR", line["level"])
	}
	if line["method"] != "POST" || line["path"] != "/api/campanhas" {
		t.Errorf("a linha não diz qual requisição falhou: method=%v path=%v", line["method"], line["path"])
	}
	if status, _ := line["status"].(float64); status != 500 {
		t.Errorf("a linha diz status %v, e a resposta foi 500", line["status"])
	}
	if body, _ := line["body"].(string); !strings.Contains(body, "SQLITE_BUSY") {
		t.Errorf("a linha não traz o começo do corpo, que é onde a causa está: body=%q", line["body"])
	}
	if _, has := line["ms"]; !has {
		t.Error("a linha não diz quanto a requisição levou, e uma espera de 5s pela trava é o que a denuncia")
	}
}

func TestASuccessfulResponseLeavesNoLogLine(t *testing.T) {
	lines := serveThroughTheLog(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":1}`))
	})
	if len(lines) != 0 {
		t.Fatalf("um 201 deixou %d linhas no log: %v", len(lines), lines)
	}
}

// O corpo vai cortado: um 500 que despeja uma página inteira encheria o log de
// HTML, e a causa está sempre no começo.
func TestTheLoggedBodyIsCapped(t *testing.T) {
	lines := serveThroughTheLog(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(strings.Repeat("x", 5000)))
	})
	if len(lines) != 1 {
		t.Fatalf("o 500 deixou %d linhas no log", len(lines))
	}
	if body, _ := lines[0]["body"].(string); len(body) > 512 {
		t.Errorf("o corpo registrado tem %d bytes, e o teto é 512", len(body))
	}
}

// O fluxo ao vivo da Mesa depende do `Flush` atravessar cada embrulho (ver o
// `gzipEnvelope`). Um embrulho que o engolisse prenderia o quadro no buffer.
func TestTheFlushCrossesTheLog(t *testing.T) {
	var buf bytes.Buffer
	logged := LogServerErrors(slog.New(slog.NewJSONHandler(&buf, nil)))(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			flusher, ok := w.(http.Flusher)
			if !ok {
				t.Fatal("o embrulho do log esconde o http.Flusher de quem está embaixo")
			}
			_, _ = w.Write([]byte("data: oi\n\n"))
			flusher.Flush()
		}))
	rec := httptest.NewRecorder()
	logged.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/fluxo", nil))
	if !rec.Flushed {
		t.Error("o Flush não chegou ao writer de baixo")
	}
}
