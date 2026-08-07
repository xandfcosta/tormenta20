package api

import (
	"fmt"

	"t20engine/engine"
)

// Equip rules ported from t20-data + backend equip validators: the equip axis
// (equip-axis.helpers.ts), the 4-vested / 2-hands caps (rules/equip.ts), and the
// carga slot-multiple check (characters.helpers.ts).

const (
	handsLimit  = 2
	vestedLimit = 4
)

// homebrewVestedOK mirrors t20-data items/homebrew.ts — esotéricos that may be
// WORN despite a wielded-only raw axis.
var homebrewVestedOK = map[string]bool{"medalhao-de-prata": true}

func handsFor(slot string) int {
	switch slot {
	case "wielded":
		return 1
	case "wielded2":
		return 2
	default:
		return 0
	}
}

// allowedEquipStates maps an item's equip axis to the states it may occupy.
func allowedEquipStates(equip string) []string {
	switch equip {
	case "vested":
		return []string{"vested"}
	case "wielded":
		return []string{"wielded", "wielded2"}
	default:
		return nil
	}
}

// equipAxisError enforces that `equipped` sits on the catalog item's equip axis
// (assertEquipAxisAllowed). Returns the top-level + field message, both "" when
// valid, the item is custom/unknown, or a homebrew vested allowance applies.
func equipAxisError(catalog *engine.CatalogItem, equipped string) (topMsg, fieldMsg string) {
	if catalog == nil {
		return "", ""
	}
	if equipped == "vested" && homebrewVestedOK[catalog.ID] {
		return "", ""
	}
	allowed := allowedEquipStates(catalog.Equip)
	if contains(allowed, equipped) {
		return "", ""
	}
	expected := "null (item is not equippable)"
	field := fmt.Sprintf("%q não é equipável", catalog.Name)
	if len(allowed) > 0 {
		expected = "null | " + quoteJoin(allowed)
		field = fmt.Sprintf("%q só aceita %s", catalog.Name, join(allowed, " ou "))
	}
	top := fmt.Sprintf("equipped '%s' is invalid for %q (equip axis '%s') — expected %s",
		equipped, catalog.Name, catalog.Equip, expected)
	return top, field
}

// equipLimitError ports validateEquipChange + assertEquipLimits: the 4-vested /
// 2-hands caps over the OTHER equipped items. Returns "" when within limits.
func equipLimitError(otherEquipped []string, incoming string) string {
	vested, hands := 0, 0
	for _, s := range otherEquipped {
		if s == "vested" {
			vested++
		}
		hands += handsFor(s)
	}
	if incoming == "vested" {
		vested++
	}
	if vested > vestedLimit {
		return fmt.Sprintf("Limite de %d itens vestidos atingido", vestedLimit)
	}
	if hands+handsFor(incoming) > handsLimit {
		return fmt.Sprintf("Limite de %d mãos atingido", handsLimit)
	}
	return ""
}

// slotsNotMultiple reports whether slots is not a finite multiple of 0.5
// (assertSlotsMultiple).
func slotsNotMultiple(slots float64) bool {
	doubled := slots * 2
	return doubled != float64(int64(doubled))
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

func join(xs []string, sep string) string {
	out := ""
	for i, x := range xs {
		if i > 0 {
			out += sep
		}
		out += x
	}
	return out
}

func quoteJoin(xs []string) string {
	q := make([]string, len(xs))
	for i, x := range xs {
		q[i] = "'" + x + "'"
	}
	return join(q, " | ")
}
