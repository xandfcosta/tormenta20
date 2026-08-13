package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

// nullToPtr converts a nullable TEXT column into a *string (nil → JSON null).
func nullToPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

// writeJSON serializes body as JSON with the given status. A nil body writes just
// the status line (204 responses). HTML escaping is off so `<`, `>`, `&` inside
// the stored JSON-string columns (modifiers, choices) reach the client verbatim
// instead of as \u003c escapes.
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if body != nil {
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(body)
	}
}

// writeError emits the plain error envelope: {"statusCode","message"}. When a
// failure also carries per-field detail, use writeFieldError (validate.go) —
// this one deliberately stays the minimal shape.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"statusCode": status, "message": message})
}
