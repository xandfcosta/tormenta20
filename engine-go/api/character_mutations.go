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
// Rule-free — pure DB write behind the
// owner/GM guard.
func (s *Server) handleUpdateVitals(w http.ResponseWriter, r *http.Request) {
	row, ok := s.characterFor(w, r)
	if !ok {
		return
	}
	var body vitalsBody
	if !decodeJSON(w, r, &body) {
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
		ID:        row.ID,
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

type applyDamageResult struct {
	HpCurrent       int           `json:"hpCurrent"`
	TempHpRemaining int           `json:"tempHpRemaining"`
	Drained         []damageDrain `json:"drained"`
}

// handleApplyDamage temp-first damage
// routing (drain pools, overflow to HP), persisting the drained/emptied effects
// and the new HP. Returns the {hpCurrent, tempHpRemaining, drained} delta.
func (s *Server) handleApplyDamage(w http.ResponseWriter, r *http.Request) {
	row, ok := s.characterFor(w, r)
	if !ok {
		return
	}
	var body struct {
		Amount *int64 `json:"amount"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	switch {
	case body.Amount == nil:
		writeValidationError(w, FieldErrorMap{"amount": {"amount must be an integer number"}})
		return
	case *body.Amount < 1:
		writeValidationError(w, FieldErrorMap{"amount": {"amount must not be less than 1"}})
		return
	case *body.Amount > 9999:
		writeValidationError(w, FieldErrorMap{"amount": {"amount must not be greater than 9999"}})
		return
	}

	effects, err := s.queries.ListActiveEffectsByCharacter(r.Context(), row.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not load effects")
		return
	}
	plan := planDamage(parseTempHpPools(effects), int(row.Hpcurrent), int(*body.Amount))

	for _, u := range plan.updates {
		if err := s.queries.UpdateEffectModifiers(r.Context(), sqlcgen.UpdateEffectModifiersParams{Modifiers: u.modifiers, ID: u.effectID}); err != nil {
			writeError(w, http.StatusInternalServerError, "Could not update effect")
			return
		}
	}
	for _, delID := range plan.deleteIDs {
		if err := s.queries.DeleteEffectByID(r.Context(), delID); err != nil {
			writeError(w, http.StatusInternalServerError, "Could not remove effect")
			return
		}
	}
	if plan.hpCurrent != int(row.Hpcurrent) {
		if err := s.queries.SetHpCurrent(r.Context(), sqlcgen.SetHpCurrentParams{
			HpCurrent: int64(plan.hpCurrent), UpdatedAt: nowISO(), ID: row.ID,
		}); err != nil {
			writeError(w, http.StatusInternalServerError, "Could not apply damage")
			return
		}
	}
	writeJSON(w, http.StatusOK, applyDamageResult{
		HpCurrent: plan.hpCurrent, TempHpRemaining: plan.tempHpRemaining, Drained: plan.drained,
	})
}
