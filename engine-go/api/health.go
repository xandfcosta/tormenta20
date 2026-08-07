package api

import "net/http"

// handleHealth is the liveness probe (mirrors the Nest GET /health). Extend with
// a DB ping if we ever need readiness vs liveness split.
func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
