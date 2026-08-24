package plataforma

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// FieldErrorMap is the validation error shape the client reads: field →
// human messages. Emitted inside the {statusCode,error,message,fieldErrors} body.
type FieldErrorMap map[string][]string

// FieldError is a domain validation failure that also carries per-field messages.
// It lets a transport-agnostic domain rule signal a rich validation body without
// depending on http.ResponseWriter: the HTTP layer renders the full
// {statusCode,error,message,fieldErrors} envelope; other transports (the WS gateway)
// just read Error(). Foundation for the B.6 fase-0 extraction.
type FieldError struct {
	status  int
	message string
	fields  FieldErrorMap
}

// NewFieldError é a porta do `FieldError` para fora do pacote (ALE-254).
//
// Os três campos ficam não exportados de propósito: o construtor é o único
// lugar onde o envelope se monta, então status e corpo não podem divergir entre
// os pontos que o criam. Antes de a fronteira existir, cada chamador montava a
// struct à mão — e o compilador não tinha como impedir um envelope pela metade.
func NewFieldError(status int, message string, fields FieldErrorMap) *FieldError {
	return &FieldError{status: status, message: message, fields: fields}
}

func (e *FieldError) Error() string { return e.message }

// WriteFieldError emits the full validation envelope directly, for a handler
// that already knows the message and the per-field detail and has no domain
// error to wrap. Thirteen call sites used to build this map literal by hand —
// the rich shape existed only behind *FieldError, which is awkward to construct
// inline, so handlers wrote it out instead and the envelope drifted.
func WriteFieldError(w http.ResponseWriter, status int, message string, fields FieldErrorMap) {
	WriteJSON(w, status, map[string]any{
		"statusCode":  status,
		"error":       http.StatusText(status),
		"message":     message,
		"fieldErrors": fields,
	})
}

// WriteDomainError maps a domain error to the HTTP response: a *FieldError becomes the
// full validation envelope (preserving its custom message); anything else falls back to
// a plain {message} at the given status. The single seam every HTTP handler uses to
// translate a domain rule's (status, error) return.
func WriteDomainError(w http.ResponseWriter, status int, err error) {
	var fe *FieldError
	if errors.As(err, &fe) {
		WriteFieldError(w, fe.status, fe.message, fe.fields)
		return
	}
	WriteError(w, status, err.Error())
}

// DecodeJSON reads a JSON body into dst; on malformed input it writes a 400 and
// returns false so the handler can bail.
// maxBodyBytes é o teto de um corpo de requisição. Um megabyte é folgado para
// tudo que este app manda — a ficha inteira de nível 20 serializada dá ~40 KB —
// e o que ele impede é um corpo sem fim segurando memória e goroutine (ALE-157).
const maxBodyBytes = 1 << 20

func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		// Corpo grande demais tem resposta PRÓPRIA: dizer "JSON inválido" para
		// um JSON perfeitamente válido mandaria quem escreveu o cliente
		// procurar defeito onde não há.
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			WriteError(w, http.StatusRequestEntityTooLarge,
				fmt.Sprintf("O corpo da requisição passa de %d bytes", tooLarge.Limit))
			return false
		}
		WriteError(w, http.StatusBadRequest, "Invalid JSON body")
		return false
	}
	return true
}

// WriteValidationError emits the same envelope as validation-exception.factory.ts.
func WriteValidationError(w http.ResponseWriter, fields FieldErrorMap) {
	WriteJSON(w, http.StatusBadRequest, map[string]any{
		"statusCode":  http.StatusBadRequest,
		"error":       "Bad Request",
		"message":     "Validation failed",
		"fieldErrors": fields,
	})
}

// A pragmatic email shape check — class-validator's IsEmail is stricter, but the
// frontend pre-validates and the exact RFC is not worth reproducing here.
// NormalizeEmail is the single spelling of an account. Register and login both
// run it, so `Mestre@T20.local` and `mestre@t20.local` are ONE account and not
// two — which is what lets the admin check (Config.IsAdmin) ignore case without
// opening a door: a case variant can no longer become a second admin (ALE-120).
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

const IsoLayout = "2006-01-02T15:04:05.000Z"

func NowISO() string {
	return IsoAt(time.Now())
}

func IsoAt(t time.Time) string {
	return t.UTC().Format(IsoLayout)
}

func ParseInt(s string) (int, error) { return strconv.Atoi(strings.TrimSpace(s)) }
