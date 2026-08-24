package api

import "t20engine/aovivo"

import (
	"net/http"
	"t20engine/plataforma"
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
		plataforma.WriteError(w, status, err.Error())
		return
	}
	// Sem `Flusher` não há SSE: o quadro ficaria no buffer do net/http e o
	// navegador esperaria para sempre. Falhar aqui é melhor que servir um fluxo
	// que nunca entrega.
	flusher, podeEmpurrar := w.(http.Flusher)
	if !podeEmpurrar {
		plataforma.WriteError(w, http.StatusInternalServerError, "streaming unsupported")
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

	connID := aovivo.NewUUID()
	conn := s.sse.Add(sessionID, connID, role)
	defer s.sse.Remove(sessionID, connID)

	s.announcePresence(sessionID, connID, user, role)
	defer s.dropPresence(sessionID, connID)

	aovivo.StreamFrames(r.Context(), w, flusher, conn, aovivo.Heartbeat)
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
	elenco := s.presence.Join(sessionID, connID, aovivo.PresenceUser{UserID: user.ID, Name: nome, Role: role})
	s.emitPresence(sessionID, elenco)
}

func (s *Server) dropPresence(sessionID int64, connID string) {
	elenco, mudou := s.presence.Leave(sessionID, connID)
	if mudou {
		s.emitPresence(sessionID, elenco)
	}
}

func (s *Server) emitPresence(sessionID int64, elenco []aovivo.PresenceUser) {
	s.sse.Emit(sessionID, "", "presence", map[string]any{
		"sessionId": sessionID, "users": elenco,
	})
}
