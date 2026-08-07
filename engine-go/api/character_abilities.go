package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"t20engine/db/sqlcgen"
)

// proficiencyCategories mirrors t20-data PROFICIENCY_CATEGORIES.
var proficiencyCategories = toStringSet([]string{
	"armas-simples", "armas-marciais", "armas-exoticas", "armas-de-fogo",
	"armaduras-leves", "armaduras-pesadas", "escudos",
})

// handleUpdateAbilities ports updateAbilityChoices: patch any subset of the
// character's ability-choice JSON blobs, echoing back only the fields written.
// NOTE: classChoices sanitization (devoto/caminho validation vs the DEUS/CAMINHOS
// catalogs) is deferred — the frontend pre-validates; stored as sent.
func (s *Server) handleUpdateAbilities(w http.ResponseWriter, r *http.Request) {
	id, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	var body struct {
		RaceAbilityChoices *[]string        `json:"raceAbilityChoices"`
		OriginChoices      *[]string        `json:"originChoices"`
		ClassPowers        *[]string        `json:"classPowers"`
		ClassChoices       *json.RawMessage `json:"classChoices"`
		PowerChoices       *json.RawMessage `json:"powerChoices"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if _, status, err := s.authorizedCharacter(r.Context(), currentUser(r), id); err != nil {
		writeError(w, status, err.Error())
		return
	}

	sets := []string{}
	args := []any{}
	resp := map[string]string{}
	add := func(column string, value string) {
		sets = append(sets, column+" = ?")
		args = append(args, value)
		resp[column] = value
	}
	if body.RaceAbilityChoices != nil {
		add("raceAbilityChoices", marshalStrings(body.RaceAbilityChoices))
	}
	if body.OriginChoices != nil {
		add("originChoices", marshalStrings(body.OriginChoices))
	}
	if body.ClassPowers != nil {
		add("classPowers", marshalStrings(body.ClassPowers))
	}
	if body.ClassChoices != nil {
		add("classChoices", compactJSON(*body.ClassChoices))
	}
	if body.PowerChoices != nil {
		add("powerChoices", compactJSON(*body.PowerChoices))
	}
	if len(sets) == 0 {
		writeError(w, http.StatusBadRequest, "No fields to update")
		return
	}

	sets = append(sets, "updatedAt = ?")
	args = append(args, nowISO(), id)
	//nolint:gosec // SET clause is a fixed column allowlist, not input.
	if _, err := s.db.ExecContext(r.Context(),
		"UPDATE characters SET "+strings.Join(sets, ", ")+" WHERE id = ?", args...); err != nil {
		writeError(w, http.StatusInternalServerError, "Could not update abilities")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleUpdateProficiencies ports updateProficiencies: validate every category
// against the catalog, dedup, store, return {proficiencies}.
func (s *Server) handleUpdateProficiencies(w http.ResponseWriter, r *http.Request) {
	id, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	var body struct {
		Proficiencies []string `json:"proficiencies"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if _, status, err := s.authorizedCharacter(r.Context(), currentUser(r), id); err != nil {
		writeError(w, status, err.Error())
		return
	}
	if body.Proficiencies == nil {
		writeValidationError(w, FieldErrorMap{"proficiencies": {"proficiencies must be an array"}})
		return
	}
	var unknown []string
	seen := map[string]bool{}
	dedup := []string{}
	for _, cat := range body.Proficiencies {
		if !proficiencyCategories[cat] {
			unknown = append(unknown, fmt.Sprintf("Unknown category %q", cat))
		}
		if !seen[cat] {
			seen[cat] = true
			dedup = append(dedup, cat)
		}
	}
	if len(unknown) > 0 {
		writeValidationError(w, FieldErrorMap{"proficiencies": unknown})
		return
	}
	proficiencies := marshalStrings(&dedup)
	if err := s.queries.SetProficiencies(r.Context(), sqlcgen.SetProficienciesParams{
		Proficiencies: proficiencies, UpdatedAt: nowISO(), ID: id,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "Could not update proficiencies")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"proficiencies": proficiencies})
}

// compactJSON normalizes an object blob to compact JSON (matching JSON.stringify
// of the sanitized value, minus the deferred sanitization).
func compactJSON(raw json.RawMessage) string {
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return "{}"
	}
	b, _ := json.Marshal(v)
	return string(b)
}
