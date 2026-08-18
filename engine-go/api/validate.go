package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// FieldErrorMap is the validation error shape the client reads: field →
// human messages. Emitted inside the {statusCode,error,message,fieldErrors} body.
type FieldErrorMap map[string][]string

// fieldError is a domain validation failure that also carries per-field messages.
// It lets a transport-agnostic domain rule signal a rich validation body without
// depending on http.ResponseWriter: the HTTP layer renders the full
// {statusCode,error,message,fieldErrors} envelope; other transports (the WS gateway)
// just read Error(). Foundation for the B.6 fase-0 extraction.
type fieldError struct {
	status  int
	message string
	fields  FieldErrorMap
}

func (e *fieldError) Error() string { return e.message }

// writeFieldError emits the full validation envelope directly, for a handler
// that already knows the message and the per-field detail and has no domain
// error to wrap. Thirteen call sites used to build this map literal by hand —
// the rich shape existed only behind *fieldError, which is awkward to construct
// inline, so handlers wrote it out instead and the envelope drifted.
func writeFieldError(w http.ResponseWriter, status int, message string, fields FieldErrorMap) {
	writeJSON(w, status, map[string]any{
		"statusCode":  status,
		"error":       http.StatusText(status),
		"message":     message,
		"fieldErrors": fields,
	})
}

// writeDomainError maps a domain error to the HTTP response: a *fieldError becomes the
// full validation envelope (preserving its custom message); anything else falls back to
// a plain {message} at the given status. The single seam every HTTP handler uses to
// translate a domain rule's (status, error) return.
func writeDomainError(w http.ResponseWriter, status int, err error) {
	var fe *fieldError
	if errors.As(err, &fe) {
		writeFieldError(w, fe.status, fe.message, fe.fields)
		return
	}
	writeError(w, status, err.Error())
}

// decodeJSON reads a JSON body into dst; on malformed input it writes a 400 and
// returns false so the handler can bail.
// maxBodyBytes é o teto de um corpo de requisição. Um megabyte é folgado para
// tudo que este app manda — a ficha inteira de nível 20 serializada dá ~40 KB —
// e o que ele impede é um corpo sem fim segurando memória e goroutine (ALE-157).
const maxBodyBytes = 1 << 20

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		// Corpo grande demais tem resposta PRÓPRIA: dizer "JSON inválido" para
		// um JSON perfeitamente válido mandaria quem escreveu o cliente
		// procurar defeito onde não há.
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge,
				fmt.Sprintf("O corpo da requisição passa de %d bytes", tooLarge.Limit))
			return false
		}
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return false
	}
	return true
}

// writeValidationError emits the same envelope as validation-exception.factory.ts.
func writeValidationError(w http.ResponseWriter, fields FieldErrorMap) {
	writeJSON(w, http.StatusBadRequest, map[string]any{
		"statusCode":  http.StatusBadRequest,
		"error":       "Bad Request",
		"message":     "Validation failed",
		"fieldErrors": fields,
	})
}

// A pragmatic email shape check — class-validator's IsEmail is stricter, but the
// frontend pre-validates and the exact RFC is not worth reproducing here.
var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

func isEmail(s string) bool { return emailRe.MatchString(s) }

// normalizeEmail is the single spelling of an account. Register and login both
// run it, so `Mestre@T20.local` and `mestre@t20.local` are ONE account and not
// two — which is what lets the admin check (Config.IsAdmin) ignore case without
// opening a door: a case variant can no longer become a second admin (ALE-120).
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validateRegister(b registerBody) FieldErrorMap {
	f := FieldErrorMap{}
	if !isEmail(b.Email) {
		f["email"] = []string{"email must be an email"}
	}
	for field, messages := range validatePassword(b.Password) {
		f[field] = messages
	}
	if b.Name != nil && utf8.RuneCountInString(*b.Name) > 80 {
		f["name"] = []string{"name must be shorter than or equal to 80 characters"}
	}
	return f
}

// validatePassword is the ONE password rule, shared by registration and by the
// reset link (ALE-120) — two spellings of "at least 8" would drift, and the
// screen that ended up laxer would be the one that matters.
func validatePassword(password string) FieldErrorMap {
	f := FieldErrorMap{}
	length := utf8.RuneCountInString(password)
	if length < 8 {
		f["password"] = append(f["password"], "Password must be at least 8 characters")
	}
	if length > 128 {
		f["password"] = append(f["password"], "password must be shorter than or equal to 128 characters")
	}
	return f
}

func validateLogin(b loginBody) FieldErrorMap {
	f := FieldErrorMap{}
	if !isEmail(b.Email) {
		f["email"] = []string{"email must be an email"}
	}
	if b.Password == "" {
		f["password"] = []string{"password must be longer than or equal to 1 characters"}
	}
	return f
}

// nowISO is the timestamp format stored in the TEXT DateTime columns (ISO-8601,
// UTC, millisecond precision — what Prisma serialized).
// isoLayout is the one spelling of a timestamp in this database. Shared so a
// stored value parses back with the layout that wrote it (ALE-120).
const isoLayout = "2006-01-02T15:04:05.000Z"

func nowISO() string {
	return isoAt(time.Now())
}

func isoAt(t time.Time) string {
	return t.UTC().Format(isoLayout)
}

func parseInt(s string) (int, error) { return strconv.Atoi(strings.TrimSpace(s)) }
