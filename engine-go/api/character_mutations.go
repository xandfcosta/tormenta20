package api

import (
	"database/sql"
	"t20engine/sheet"
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
