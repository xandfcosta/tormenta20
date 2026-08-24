package plataforma

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

// NullToPtr converts a nullable TEXT column into a *string (nil → JSON null).
func NullToPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

// WriteJSON serializes body as JSON with the given status. A nil body writes just
// the status line (204 responses). HTML escaping is off so `<`, `>`, `&` inside
// the stored JSON-string columns (modifiers, choices) reach the client verbatim
// instead of as \u003c escapes.
func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if body != nil {
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(body)
	}
}

// WriteError emits the plain error envelope: {"statusCode","message"}. When a
// failure also carries per-field detail, use WriteFieldError (validate.go) —
// this one deliberately stays the minimal shape.
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]any{"statusCode": status, "message": message})
}
