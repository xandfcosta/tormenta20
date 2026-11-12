package httpio

import (
	"log/slog"
	"net/http"
	"time"
)

// loggedBodyCeiling é quanto do corpo de um 5xx entra no log. A causa está no
// começo — `http.Error(w, err.Error(), 500)` escreve só ela —, e uma página
// inteira de HTML encheria o log sem dizer mais nada.
const loggedBodyCeiling = 512

// LogServerErrors registra toda resposta 5xx: método, rota, status, quanto ela
// levou e o começo do corpo.
//
// # Por que um middleware, e não uma linha em cada handler
//
// Os 500 nascem em dezenas de lugares, e metade escreve `err.Error()` no corpo:
// a causa existe, mas só o cliente a via. Aqui ela passa pelo log de uma vez,
// junto com o 500 que o `Recoverer` escreve depois de um pânico e com o que
// nascer amanhã. O handler que responde uma frase genérica ainda registra o
// próprio erro — o corpo dele não tem a causa para este registro copiar.
//
// O tempo entra porque foi ele que denunciou o caso que produziu este
// middleware: um 500 em 5,0s é uma espera inteira pelo `busy_timeout` do
// SQLite, e não uma falha da consulta.
//
// @example r.Use(httpio.LogServerErrors(slog.Default()))
func LogServerErrors(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			watch := &statusWatch{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(watch, r)
			if watch.status < http.StatusInternalServerError {
				return
			}
			log.Error("a requisição respondeu erro do servidor",
				"method", r.Method,
				"path", r.URL.Path,
				"status", watch.status,
				"ms", time.Since(start).Milliseconds(),
				"body", string(watch.body),
			)
		})
	}
}

// statusWatch guarda o status e, só quando ele é 5xx, o começo do corpo.
type statusWatch struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
	body        []byte
}

func (s *statusWatch) WriteHeader(status int) {
	if !s.wroteHeader {
		s.status, s.wroteHeader = status, true
	}
	s.ResponseWriter.WriteHeader(status)
}

func (s *statusWatch) Write(p []byte) (int, error) {
	s.wroteHeader = true
	if s.status >= http.StatusInternalServerError && len(s.body) < loggedBodyCeiling {
		room := loggedBodyCeiling - len(s.body)
		s.body = append(s.body, p[:min(room, len(p))]...)
	}
	return s.ResponseWriter.Write(p)
}

// Flush repassa, porque o fluxo ao vivo da Mesa morre sem ele (ver o
// `gzipEnvelope`).
func (s *statusWatch) Flush() {
	if flusher, ok := s.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Unwrap é o contrato do `http.ResponseController`, pelo mesmo motivo do
// `gzipEnvelope.Unwrap`.
func (s *statusWatch) Unwrap() http.ResponseWriter { return s.ResponseWriter }
