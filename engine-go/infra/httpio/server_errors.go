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
// nascer amanhã. O handler que responde uma frase genérica ANEXA o próprio erro
// (`RecordCause`, `WriteServerError`), porque o corpo dele não tem a causa para
// este registro copiar — e assim rota, tempo e causa saem numa linha só.
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
			attrs := []any{
				"method", r.Method,
				"path", r.URL.Path,
				"status", watch.status,
				"ms", time.Since(start).Milliseconds(),
				"body", string(watch.body),
			}
			if watch.cause != nil {
				attrs = append(attrs, "cause", watch.cause.Error())
			}
			log.Error("a requisição respondeu erro do servidor", attrs...)
		})
	}
}

// statusWatch guarda o status e, só quando ele é 5xx, o começo do corpo.
type statusWatch struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
	body        []byte
	cause       error // ver `RecordCause`
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

// RecordCause anexa o erro à linha que o `LogServerErrors` escreve se esta
// resposta acabar 5xx. Numa resposta 4xx ele não aparece, e por isso quem chama
// um mapeador de status pode anexar sempre.
//
// O observador é achado pela cadeia de `Unwrap`, que é o contrato do
// `http.ResponseController`: o handler pode estar recebendo o gzip, ou outro
// embrulho, por cima dele. Fora do middleware (um teste que chama o handler
// direto) não há observador, e a causa simplesmente não vai a lugar nenhum.
//
// @example httpio.RecordCause(w, err); httpio.WriteError(w, statusForAccess(err), "Could not delete campaign")
func RecordCause(w http.ResponseWriter, cause error) {
	for w != nil {
		if watch, ok := w.(*statusWatch); ok {
			watch.cause = cause
			return
		}
		unwrapper, ok := w.(interface{ Unwrap() http.ResponseWriter })
		if !ok {
			return
		}
		w = unwrapper.Unwrap()
	}
}

// WriteServerError responde o 500 com a frase para o cliente e anexa a causa
// para o log, que é onde ela serve: a frase genérica não diz o que quebrou.
//
// @example httpio.WriteServerError(w, "Could not create campaign", err)
func WriteServerError(w http.ResponseWriter, message string, cause error) {
	RecordCause(w, cause)
	WriteError(w, http.StatusInternalServerError, message)
}
