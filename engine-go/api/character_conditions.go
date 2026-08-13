package api

import (
	"fmt"
	"net/http"
	"strings"

	"t20engine/db/sqlcgen"
)

// conditionIDs mirrors t20-data conditions.ts CONDITION_IDS (PDF p394-395) — the
// valid book conditions updateConditions accepts.
var conditionIDs = toStringSet([]string{
	"abalado", "agarrado", "alquebrado", "apavorado", "atordoado", "caido",
	"cego", "confuso", "debilitado", "desprevenido", "doente", "em-chamas",
	"enjoado", "enredado", "envenenado", "esmorecido", "exausto", "fascinado",
	"fatigado", "fraco", "frustrado", "imovel", "inconsciente", "indefeso",
	"lento", "ofuscado", "paralisado", "pasmo", "petrificado", "sangrando",
	"sobrecarregado", "surdo", "surpreendido", "vulneravel",
})

// handleUpdateConditions ports CharactersService.updateConditions: replace the
// active book conditions after validating every id against the catalog.
func (s *Server) handleUpdateConditions(w http.ResponseWriter, r *http.Request) {
	row, ok := s.characterFor(w, r)
	if !ok {
		return
	}
	var body struct {
		ActiveConditions []string `json:"activeConditions"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.ActiveConditions == nil {
		writeValidationError(w, FieldErrorMap{"activeConditions": {"activeConditions must be an array"}})
		return
	}
	var unknown []string
	for _, c := range body.ActiveConditions {
		if !conditionIDs[c] {
			unknown = append(unknown, c)
		}
	}
	if len(unknown) > 0 {
		writeError(w, http.StatusBadRequest, fmt.Sprintf(
			"Unknown condition ids: %s — expected ids from the CONDITIONS catalog", strings.Join(unknown, ", ")))
		return
	}
	activeConditions := marshalStrings(&body.ActiveConditions)
	if err := s.queries.UpdateConditions(r.Context(), sqlcgen.UpdateConditionsParams{
		ActiveConditions: activeConditions, UpdatedAt: nowISO(), ID: row.ID,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "Could not update conditions")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"activeConditions": activeConditions})
}

func toStringSet(xs []string) map[string]bool {
	m := make(map[string]bool, len(xs))
	for _, x := range xs {
		m[x] = true
	}
	return m
}
