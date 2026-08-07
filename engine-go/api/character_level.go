package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"t20engine/db/sqlcgen"
	"t20engine/engine"
)

type storedVitals struct {
	HpMax     int64 `json:"hpMax"`
	HpCurrent int64 `json:"hpCurrent"`
	MpMax     int64 `json:"mpMax"`
	MpCurrent int64 `json:"mpCurrent"`
}

type levelResult struct {
	Level  int64        `json:"level"`
	Vitals storedVitals `json:"vitals"`
}

type classLevelResult struct {
	Level   int64        `json:"level"`
	Classes []ClassDTO   `json:"classes"`
	Vitals  storedVitals `json:"vitals"`
}

// levelVitalsNext ports vitals-sync.helpers levelVitalsPatch: new maxes from the
// engine, with currents FOLLOWING the max delta (level up heals the gain, down
// walks it back), clamped to [0, newMax]. Returns the next pools + changed flag.
func levelVitalsNext(stored storedVitals, pvMax, pmMax int) (storedVitals, bool) {
	next := storedVitals{
		HpMax:     int64(pvMax),
		HpCurrent: clampCurrent(stored.HpCurrent+(int64(pvMax)-stored.HpMax), int64(pvMax)),
		MpMax:     int64(pmMax),
		MpCurrent: clampCurrent(stored.MpCurrent+(int64(pmMax)-stored.MpMax), int64(pmMax)),
	}
	return next, next != stored
}

func clampCurrent(c, hi int64) int64 { return min(max(int64(0), c), hi) }

// engineCharacterFrom bridges the API aggregate to engine.Character via JSON —
// both mirror the frontend Character contract, so the round-trip is lossless.
func engineCharacterFrom(dto CharacterDTO) (engine.Character, error) {
	var ec engine.Character
	b, err := json.Marshal(dto)
	if err != nil {
		return ec, err
	}
	return ec, json.Unmarshal(b, &ec)
}

// syncLevelVitals recomputes the pools for the (already mutated) aggregate and
// persists the level-shifted currents — the server-side syncVitalsForProjection.
func (s *Server) syncLevelVitals(r *http.Request, id int64, dto CharacterDTO) (storedVitals, error) {
	stored := storedVitals{HpMax: dto.HpMax, HpCurrent: dto.HpCurrent, MpMax: dto.MpMax, MpCurrent: dto.MpCurrent}
	if s.catalogs == nil || len(dto.Classes) == 0 {
		return stored, nil // no engine pools (0/0) → keep stored, matching Nest
	}
	ec, err := engineCharacterFrom(dto)
	if err != nil {
		return stored, err
	}
	pools := s.catalogs.VitalsForCharacter(ec)
	next, changed := levelVitalsNext(stored, pools.PvMax, pools.PmMax)
	if changed {
		if err := s.queries.SetCharacterVitals(r.Context(), sqlcgen.SetCharacterVitalsParams{
			HpMax: next.HpMax, HpCurrent: next.HpCurrent, MpMax: next.MpMax, MpCurrent: next.MpCurrent,
			UpdatedAt: nowISO(), ID: id,
		}); err != nil {
			return stored, err
		}
	}
	return next, nil
}

// handleUpdateLevel ports updateLevel: set the total level, resync the derived
// pools. (Pools track class levels, so setting the total alone usually leaves
// them unchanged.)
func (s *Server) handleUpdateLevel(w http.ResponseWriter, r *http.Request) {
	id, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	var body struct {
		Level *int64 `json:"level"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	row, status, err := s.authorizedCharacter(r.Context(), currentUser(r), id)
	if err != nil {
		writeError(w, status, err.Error())
		return
	}
	if msg := levelRangeError(body.Level); msg != "" {
		writeValidationError(w, FieldErrorMap{"level": {msg}})
		return
	}
	if err := s.queries.SetCharacterLevel(r.Context(), sqlcgen.SetCharacterLevelParams{
		Level: *body.Level, UpdatedAt: nowISO(), ID: id,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "Could not update level")
		return
	}
	dto, err := s.loadCharacter(r.Context(), row)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not load character")
		return
	}
	dto.Level = *body.Level
	vitals, err := s.syncLevelVitals(r, id, dto)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not sync vitals")
		return
	}
	writeJSON(w, http.StatusOK, levelResult{Level: *body.Level, Vitals: vitals})
}

// handleUpdateClassLevel ports updateClassLevel: bump one class, recompute the
// total (≤ 20 cap) and the derived pools with the level-shift.
func (s *Server) handleUpdateClassLevel(w http.ResponseWriter, r *http.Request) {
	id, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	var body struct {
		ClassName string `json:"className"`
		Level     *int64 `json:"level"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	row, status, err := s.authorizedCharacter(r.Context(), currentUser(r), id)
	if err != nil {
		writeError(w, status, err.Error())
		return
	}
	if msg := levelRangeError(body.Level); msg != "" {
		writeValidationError(w, FieldErrorMap{"level": {msg}})
		return
	}
	dto, err := s.loadCharacter(r.Context(), row)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not load character")
		return
	}
	found := false
	var total int64
	for i := range dto.Classes {
		if dto.Classes[i].ClassName == body.ClassName {
			dto.Classes[i].Level = *body.Level
			found = true
		}
		total += dto.Classes[i].Level
	}
	if !found {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"statusCode":  http.StatusBadRequest,
			"error":       "Bad Request",
			"message":     fmt.Sprintf("Character does not have class %q", body.ClassName),
			"fieldErrors": FieldErrorMap{"className": {"Class not on character"}},
		})
		return
	}
	if total > 20 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"statusCode":  http.StatusBadRequest,
			"error":       "Bad Request",
			"message":     fmt.Sprintf("Total level %d exceeds 20", total),
			"fieldErrors": FieldErrorMap{"level": {"Sum of class levels capped at 20"}},
		})
		return
	}
	if _, err := s.queries.SetCharacterClassLevel(r.Context(), sqlcgen.SetCharacterClassLevelParams{
		Level: *body.Level, CharacterId: id, ClassName: body.ClassName,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "Could not update class level")
		return
	}
	if err := s.queries.SetCharacterLevel(r.Context(), sqlcgen.SetCharacterLevelParams{
		Level: total, UpdatedAt: nowISO(), ID: id,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "Could not update level")
		return
	}
	dto.Level = total
	vitals, err := s.syncLevelVitals(r, id, dto)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not sync vitals")
		return
	}
	writeJSON(w, http.StatusOK, classLevelResult{Level: total, Classes: dto.Classes, Vitals: vitals})
}

// levelRangeError applies the UpdateLevelDto range (@IsInt @Min(1) @Max(20)).
func levelRangeError(level *int64) string {
	switch {
	case level == nil:
		return "level must be an integer number"
	case *level < 1:
		return "level must not be less than 1"
	case *level > 20:
		return "level must not be greater than 20"
	}
	return ""
}
