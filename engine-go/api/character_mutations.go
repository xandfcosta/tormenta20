package api

import (
	"database/sql"
	"net/http"

	"t20engine/db/sqlcgen"
)

type vitalsBody struct {
	HpCurrent *int64 `json:"hpCurrent"`
	MpCurrent *int64 `json:"mpCurrent"`
}

// vitalsResult is the delta the client merges onto its cached character — both
// current values (the touched one + the unchanged stored one).
type vitalsResult struct {
	HpCurrent int64 `json:"hpCurrent"`
	MpCurrent int64 `json:"mpCurrent"`
}

// handleUpdateVitals clamps hp/mp current against the stored maxes
// (CharactersService.updateVitals). Rule-free — pure DB write behind the
// owner/GM guard.
func (s *Server) handleUpdateVitals(w http.ResponseWriter, r *http.Request) {
	id, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	var body vitalsBody
	if !decodeJSON(w, r, &body) {
		return
	}
	row, status, err := s.authorizedCharacter(r.Context(), currentUser(r), id)
	if err != nil {
		writeError(w, status, err.Error())
		return
	}
	if body.HpCurrent == nil && body.MpCurrent == nil {
		writeError(w, http.StatusBadRequest, "No fields to update")
		return
	}

	fields := FieldErrorMap{}
	if msg := vitalError("hpCurrent", body.HpCurrent, row.Hpmax, "HP"); msg != "" {
		fields["hpCurrent"] = []string{msg}
	}
	if msg := vitalError("mpCurrent", body.MpCurrent, row.Mpmax, "MP"); msg != "" {
		fields["mpCurrent"] = []string{msg}
	}
	if len(fields) > 0 {
		writeValidationError(w, fields)
		return
	}

	res, err := s.queries.UpdateVitals(r.Context(), sqlcgen.UpdateVitalsParams{
		HpCurrent: nullInt(body.HpCurrent),
		MpCurrent: nullInt(body.MpCurrent),
		UpdatedAt: nowISO(),
		ID:        id,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not update vitals")
		return
	}
	writeJSON(w, http.StatusOK, vitalsResult{HpCurrent: res.Hpcurrent, MpCurrent: res.Mpcurrent})
}

// vitalError applies the DTO range (0..9999, class-validator messages) then the
// service-level "cannot exceed max" check. Empty string = valid / absent.
func vitalError(field string, v *int64, max int64, label string) string {
	if v == nil {
		return ""
	}
	switch {
	case *v < 0:
		return field + " must not be less than 0"
	case *v > 9999:
		return field + " must not be greater than 9999"
	case *v > max:
		return label + " current cannot exceed " + label + " max"
	}
	return ""
}

func nullInt(p *int64) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *p, Valid: true}
}
