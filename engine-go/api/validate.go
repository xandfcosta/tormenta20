package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// FieldErrorMap mirrors the Nest validationExceptionFactory output: field →
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

// writeDomainError maps a domain error to the HTTP response: a *fieldError becomes the
// full validation envelope (preserving its custom message); anything else falls back to
// a plain {message} at the given status. The single seam every HTTP handler uses to
// translate a domain rule's (status, error) return.
func writeDomainError(w http.ResponseWriter, status int, err error) {
	var fe *fieldError
	if errors.As(err, &fe) {
		writeJSON(w, fe.status, map[string]any{
			"statusCode":  fe.status,
			"error":       http.StatusText(fe.status),
			"message":     fe.message,
			"fieldErrors": fe.fields,
		})
		return
	}
	writeError(w, status, err.Error())
}

// decodeJSON reads a JSON body into dst; on malformed input it writes a 400 and
// returns false so the handler can bail.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
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

func validateRegister(b registerBody) FieldErrorMap {
	f := FieldErrorMap{}
	if !isEmail(b.Email) {
		f["email"] = []string{"email must be an email"}
	}
	pw := utf8.RuneCountInString(b.Password)
	if pw < 8 {
		f["password"] = append(f["password"], "Password must be at least 8 characters")
	}
	if pw > 128 {
		f["password"] = append(f["password"], "password must be shorter than or equal to 128 characters")
	}
	if b.Name != nil && utf8.RuneCountInString(*b.Name) > 80 {
		f["name"] = []string{"name must be shorter than or equal to 80 characters"}
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
func nowISO() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
}

func parseInt(s string) (int, error) { return strconv.Atoi(strings.TrimSpace(s)) }
