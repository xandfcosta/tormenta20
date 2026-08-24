package api

import (
	"context"
	"io"
	"net/http"
	"time"
)

// O fluxo de eventos da sessão ao vivo (ALE-253).
//
// Substitui o handshake do socket.io. Três coisas que a troca ganhou de graça,
// e vale saber por quê:
//
//  1. AUTENTICAÇÃO. O `extractToken` lê o COOKIE antes do header, e o
//     `EventSource` manda cookie sozinho. Então este endpoint entra debaixo do
//     `requireAuth` como qualquer outra rota — sem token na query string, que é
//     a saída feia que SSE costuma exigir quando a casa autentica por header.
//  2. ORIGEM. Some o `guardSocketOrigin` inteiro: isto é HTTP, então vale a
//     política de CORS que o `Router()` já aplica. O comentário da ALE-158
//     dizia "a política é uma só ou não é política" — agora é.
//  3. QUEDA. O `r.Context()` é cancelado quando o cliente vai embora, então a
//     saída é detectada pelo próprio servidor HTTP em vez de por heartbeat da
//     biblioteca.

// O intervalo do comentário-batida. Ele NÃO é para detectar queda — disso o
// contexto cuida —, é para atravessar intermediário que fecha conexão ociosa: o
// proxy do Vite em desenvolvimento e qualquer coisa entre o mestre e a mesa
// numa rede doméstica. 25s fica confortavelmente abaixo dos 60s que é o padrão
// mais comum.
const sseHeartbeat = 25 * time.Second

// handleSessionEvents mantém o fluxo aberto enquanto o cliente estiver lá.
func (s *Server) handleSessionEvents(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := intParam(w, r, "campaignId")
	if !ok {
		return
	}
	sessionID, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	user := currentUser(r)
	_, role, status, err := s.sessionForCaller(r.Context(), user, campaignID, sessionID)
	if err != nil {
		writeError(w, status, err.Error())
		return
	}
	// Sem `Flusher` não há SSE: o quadro ficaria no buffer do net/http e o
	// navegador esperaria para sempre. Falhar aqui é melhor que servir um fluxo
	// que nunca entrega.
	flusher, podeEmpurrar := w.(http.Flusher)
	if !podeEmpurrar {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// Desliga o buffer de proxy reverso que não conhece SSE — sem isto um nginx
	// na frente segura os quadros até encher o buffer dele.
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	connID := newUUID()
	conn := s.sse.add(sessionID, connID, role)
	defer s.sse.remove(sessionID, connID)

	s.announcePresence(sessionID, connID, user, role)
	defer s.dropPresence(sessionID, connID)

	streamFrames(r.Context(), w, flusher, conn, sseHeartbeat)
}

// streamFrames é o laço do fluxo, separado do handler para o teste exercitar o
// código DE VERDADE em vez de uma imitação com a mesma forma.
//
// Sai por três caminhos, e os três são normais: a fila fechou (o hub tirou a
// conexão), a escrita falhou (o cliente sumiu no meio de um quadro), ou o
// contexto foi cancelado (ele foi embora). Nenhum é erro a registrar.
func streamFrames(
	ctx context.Context,
	w io.Writer,
	flusher http.Flusher,
	conn *sseConn,
	batidaCada time.Duration,
) {
	batida := time.NewTicker(batidaCada)
	defer batida.Stop()

	for {
		select {
		case frame, aberta := <-conn.frames:
			if !aberta {
				return
			}
			if _, err := w.Write(frame); err != nil {
				return
			}
			flusher.Flush()
		case <-batida.C:
			// Comentário SSE: o cliente ignora, o intermediário vê tráfego.
			if _, err := w.Write([]byte(": ping\n\n")); err != nil {
				return
			}
			flusher.Flush()
		case <-ctx.Done():
			return
		}
	}
}

// announcePresence põe o leitor no elenco e conta à mesa.
func (s *Server) announcePresence(sessionID int64, connID string, user AuthUser, role string) {
	nome := user.Email
	if user.Name != nil && *user.Name != "" {
		nome = *user.Name
	}
	if role == "" {
		role = "player"
	}
	elenco := s.presence.join(sessionID, connID, PresenceUser{UserID: user.ID, Name: nome, Role: role})
	s.emitPresence(sessionID, elenco)
}

func (s *Server) dropPresence(sessionID int64, connID string) {
	elenco, mudou := s.presence.leave(sessionID, connID)
	if mudou {
		s.emitPresence(sessionID, elenco)
	}
}

func (s *Server) emitPresence(sessionID int64, elenco []PresenceUser) {
	s.sse.emit(sessionID, "", "presence", map[string]any{
		"sessionId": sessionID, "users": elenco,
	})
}
