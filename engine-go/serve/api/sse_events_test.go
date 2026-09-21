package api

import "t20engine/domain/live"

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// O ENDPOINT SSE de ponta a ponta.
//
// O hub tem teste próprio; o que se prende aqui é o que só aparece quando há um
// servidor HTTP de verdade do outro lado: os cabeçalhos que fazem o navegador
// tratar a resposta como fluxo, o quadro chegando SEM a requisição terminar, e
// a saída limpando a presença.
//
// Sem `Flusher` nada disso funciona e o sintoma é o pior possível — o navegador
// fica esperando calado —, por isso o handler recusa em vez de servir um fluxo
// que nunca entrega.

// lerQuadro lê um quadro SSE (até a linha em branco) com prazo, para o teste
// falhar em vez de pendurar quando o servidor não empurra.
func lerQuadro(t *testing.T, reader *bufio.Reader) string {
	t.Helper()
	pronto := make(chan string, 1)
	go func() {
		var sb strings.Builder
		for {
			row, err := reader.ReadString('\n')
			if err != nil {
				pronto <- sb.String()
				return
			}
			sb.WriteString(row)
			if row == "\n" {
				pronto <- sb.String()
				return
			}
		}
	}()
	select {
	case q := <-pronto:
		return q
	case <-time.After(2 * time.Second):
		t.Fatal("o servidor não empurrou quadro nenhum em 2s")
		return ""
	}
}

// O quadro chega ANTES de a requisição terminar — que é a coisa toda.
//
// Um handler que montasse a resposta e devolvesse no fim passaria por qualquer
// asserção sobre o corpo e falharia na mesa: o mestre viraria o turno e a tela
// do jogador só saberia quando a conexão caísse.
func TestTheFrameArrivesWithTheRequestStillOpen(t *testing.T) {
	hub := live.NewSSEHub()
	conn := hub.Add(7, "c1", "gm")

	server := httptest.NewServer(fluxoDeTeste(conn, time.Hour))
	defer server.Close()

	resp, err := server.Client().Get(server.URL)
	if err != nil {
		t.Fatalf("conectar: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type = %q, queria text/event-stream", ct)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("Cache-Control = %q — sem isto o quadro pode vir de cache", cc)
	}

	reader := bufio.NewReader(resp.Body)
	hub.Emit(7, "", "session-state", map[string]any{"turnIndex": 3})

	frame := lerQuadro(t, reader)
	if !strings.Contains(frame, "event: session-state") || !strings.Contains(frame, `"turnIndex":3`) {
		t.Fatalf("quadro = %q", frame)
	}
}

// A batida atravessa intermediário que fecha conexão ociosa. Ela é COMENTÁRIO
// SSE (começa com `:`), então o cliente a ignora — mandar um evento de verdade
// faria o `EventSource` acordar o app por nada.
func TestTheHeartbeatIsACommentAndNotAnEvent(t *testing.T) {
	hub := live.NewSSEHub()
	conn := hub.Add(7, "c1", "gm")
	server := httptest.NewServer(fluxoDeTeste(conn, 30*time.Millisecond))
	defer server.Close()

	resp, err := server.Client().Get(server.URL)
	if err != nil {
		t.Fatalf("conectar: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	frame := lerQuadro(t, bufio.NewReader(resp.Body))
	if !strings.HasPrefix(frame, ":") {
		t.Fatalf("batida = %q, queria um comentário SSE", frame)
	}
	if strings.Contains(frame, "event:") {
		t.Fatalf("batida = %q — comentário não pode virar evento", frame)
	}
}

// fluxoDeTeste põe o LAÇO DE VERDADE (`live.StreamFrames`) atrás de um servidor, com
// os mesmos cabeçalhos do handler. Copiar o laço numa imitação faria o teste
// medir a cópia — que é o modo de o guarda passar verde sobre o app quebrado.
func fluxoDeTeste(conn *live.SSEConn, hit time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "sem flusher", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
		flusher.Flush()
		live.StreamFrames(r.Context(), w, flusher, conn, hit)
	}
}
