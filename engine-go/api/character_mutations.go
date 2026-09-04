package api

import (
	"database/sql"
	"net/http"
	"t20engine/plataforma"
	"t20engine/sheet"

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
	if !plataforma.DecodeJSON(w, r, &body) {
		return
	}
	if body.HpCurrent == nil && body.MpCurrent == nil {
		plataforma.WriteError(w, http.StatusBadRequest, "No fields to update")
		return
	}

	fields := plataforma.FieldErrorMap{}
	if msg := vitalError("hpCurrent", body.HpCurrent, row.Hpmax, "HP"); msg != "" {
		fields["hpCurrent"] = []string{msg}
	}
	if msg := vitalError("mpCurrent", body.MpCurrent, row.Mpmax, "MP"); msg != "" {
		fields["mpCurrent"] = []string{msg}
	}
	if len(fields) > 0 {
		plataforma.WriteValidationError(w, fields)
		return
	}

	res, err := s.queries.UpdateVitals(r.Context(), sqlcgen.UpdateVitalsParams{
		HpCurrent: nullInt(body.HpCurrent),
		MpCurrent: nullInt(body.MpCurrent),
		UpdatedAt: plataforma.NowISO(),
		ID:        row.ID,
	})
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not update vitals")
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, vitalsResult{HpCurrent: res.Hpcurrent, MpCurrent: res.Mpcurrent})
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
	HpCurrent       int                 `json:"hpCurrent"`
	TempHpRemaining int                 `json:"tempHpRemaining"`
	Drained         []sheet.DamageDrain `json:"drained"`
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
	if !plataforma.DecodeJSON(w, r, &body) {
		return
	}
	switch {
	case body.Amount == nil:
		plataforma.WriteValidationError(w, plataforma.FieldErrorMap{"amount": {"amount must be an integer number"}})
		return
	case *body.Amount < 1:
		plataforma.WriteValidationError(w, plataforma.FieldErrorMap{"amount": {"amount must not be less than 1"}})
		return
	case *body.Amount > 9999:
		plataforma.WriteValidationError(w, plataforma.FieldErrorMap{"amount": {"amount must not be greater than 9999"}})
		return
	}

	// A MESMA função que o rastreador da sessão chama (ALE-122): duas cópias da
	// ordem de dano seriam duas regras, e foi exatamente isso que fez a pancada
	// da sessão ignorar PV temporários enquanto a da ficha os drenava.
	plan, err := applyDamagePlan(r.Context(), s.queries, row, int(*body.Amount))
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not apply damage")
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, applyDamageResult{
		HpCurrent: plan.HpCurrent, TempHpRemaining: plan.TempHpRemaining, Drained: plan.Drained,
	})
}

// handleUpdateTibar grava o dinheiro do personagem. O tibar é o MESMO campo que
// a Forja preenche com a Tabela 3-1 (p140) e o mesmo que a carga lê — não há um
// segundo lugar onde o dinheiro mora (ALE-215). Devolve o valor gravado para o
// cliente tomar a palavra do servidor.
func (s *Server) handleUpdateTibar(w http.ResponseWriter, r *http.Request) {
	row, ok := s.characterFor(w, r)
	if !ok {
		return
	}
	var body struct {
		Tibar *float64 `json:"tibar"`
	}
	if !plataforma.DecodeJSON(w, r, &body) {
		return
	}
	if msg := tibarError(body.Tibar); msg != "" {
		plataforma.WriteValidationError(w, plataforma.FieldErrorMap{"tibar": {msg}})
		return
	}
	err := s.queries.SetCharacterTibar(r.Context(), sqlcgen.SetCharacterTibarParams{
		Tibar: *body.Tibar, UpdatedAt: plataforma.NowISO(), ID: row.ID,
	})
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not update tibar")
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, map[string]float64{"tibar": *body.Tibar})
}

// tibarError recusa o que não é dinheiro. Não há checagem de finitude aqui de
// propósito: o JSON não escreve NaN nem Inf, e um `1e999` o próprio decodificador
// recusa antes ("cannot unmarshal number 1e999 into float64") — medido. Um
// `case math.IsInf` seria linha verde sobre caminho morto.
func tibarError(v *float64) string {
	switch {
	case v == nil:
		return "tibar must be a number"
	case *v < 0:
		return "tibar must not be less than 0"
	case *v > sheet.MaxTibar:
		return "tibar must not be greater than 1000000"
	}
	return ""
}
