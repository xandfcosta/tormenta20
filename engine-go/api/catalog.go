package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"t20engine/catalog"
)

// handleCatalogIndex serves GET /catalog — the accepted resource names.
func (s *Server) handleCatalogIndex(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, catalog.Resources())
}

// handleCatalogResource serves GET /catalog/:resource from the embedded JSON.
// Unknown name → 404 with the accepted set.
func (s *Server) handleCatalogResource(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "resource")
	body, ok := catalog.Resource(name)
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf(
			"unknown catalog resource: %q; expected one of %s", name, strings.Join(catalog.Resources(), ", ")))
		return
	}
	writeRawJSON(w, body)
}

// handleCharacterOptions serves GET /characters/options (public creation lists).
func (s *Server) handleCharacterOptions(w http.ResponseWriter, _ *http.Request) {
	body, err := catalog.Options()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not load options")
		return
	}
	writeRawJSON(w, body)
}

// writeRawJSON writes pre-serialized JSON bytes verbatim (no re-encode).
func writeRawJSON(w http.ResponseWriter, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}
