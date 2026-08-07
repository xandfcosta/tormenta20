package api

import (
	"encoding/json"
	"net/http"
)

// writeJSON serializes body as JSON with the given status. A nil body writes just
// the status line (204 responses).
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if body != nil {
		_ = json.NewEncoder(w).Encode(body)
	}
}

// writeError emits a Nest-style error envelope: {"statusCode","message"}. Kept
// minimal for B.0; the auth/validation slices extend it (fieldErrors, etc.).
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"statusCode": status, "message": message})
}
