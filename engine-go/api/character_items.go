package api

import (
	"net/http"
	"t20engine/plataforma"
)

type createItemBody struct {
	CatalogID    *string   `json:"catalogId"`
	Name         *string   `json:"name"`
	Quantity     *int64    `json:"quantity"`
	Slots        *float64  `json:"slots"`
	Equipped     *string   `json:"equipped"`
	Improvements *[]string `json:"improvements"`
	Material     *string   `json:"material"`
}

// ptrEq reports whether two *string carry the same presence + value.
func ptrEq(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// writeAxisError emits the equip-axis BadRequest: a custom top message + the
// equipped field error (assertEquipAxisAllowed).
func writeAxisError(w http.ResponseWriter, top, field string) {
	plataforma.WriteFieldError(w, http.StatusBadRequest, top, plataforma.FieldErrorMap{"equipped": {field}})
}

// O `MarshalStrings` mora no `sheet` desde a ALE-278: a cena da ficha e o
// hospedeiro gravam a mesma coluna, e a normalização de nulo é forma do DADO.
