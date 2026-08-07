package engine

import "fmt"

// Compra de pontos (book p17, Tabela 1-1) — ports t20-data/src/point-buy.ts.
// All six attributes start at 0 with a 10-point budget; costs 1→1, 2→2, 3→4,
// 4→7; exactly ONE attribute may drop to −1 to refund 1 point. Racial mods apply
// after, outside the budget.
const (
	PointBuyBudget = 10
	pointBuyMin    = -1
	pointBuyMax    = 4
)

var pointBuyCosts = map[int]int{-1: -1, 0: 0, 1: 1, 2: 2, 3: 4, 4: 7}

// PointBuyStatus is the front-facing point-buy result: total spent (nil when any
// value is out of range, mirroring pointBuySpent's throw) + advisory warnings.
type PointBuyStatus struct {
	Spent    *int     `json:"spent"`
	Warnings []string `json:"warnings"`
}

// PointBuySpent totals the points a BASE attribute spread costs. ok=false when
// any value is outside −1..4 (pointBuySpent throws there; the front shows "—").
func PointBuySpent(attrs map[string]int) (int, bool) {
	spent := 0
	for _, k := range AttributeKeys {
		cost, has := pointBuyCosts[attrs[k]]
		if !has {
			return 0, false
		}
		spent += cost
	}
	return spent, true
}

// PointBuyWarnings ports pointBuyWarnings: advisory p17 validation (empty = legal).
// Out-of-range values are reported, not thrown, so a live-editing UI stays up.
func PointBuyWarnings(attrs map[string]int) []string {
	warnings := []string{}
	spent, reduced := 0, 0
	for _, k := range AttributeKeys {
		v := attrs[k]
		if v < pointBuyMin || v > pointBuyMax {
			warnings = append(warnings, fmt.Sprintf(
				"compra de pontos: %s = %d fora do intervalo [%d, %d]", k, v, pointBuyMin, pointBuyMax))
			continue
		}
		if v == -1 {
			reduced++
		}
		spent += pointBuyCosts[v]
	}
	if reduced > 1 {
		warnings = append(warnings, fmt.Sprintf(
			"compra de pontos: apenas UM atributo pode ser reduzido a −1 (p17), há %d", reduced))
	}
	if spent > PointBuyBudget {
		warnings = append(warnings, fmt.Sprintf(
			"compra de pontos: %d pontos gastos excedem o limite de %d", spent, PointBuyBudget))
	}
	return warnings
}

// PointBuyStatusFor bundles spent + warnings for the WASM boundary.
func PointBuyStatusFor(attrs map[string]int) PointBuyStatus {
	warnings := PointBuyWarnings(attrs)
	if spent, ok := PointBuySpent(attrs); ok {
		return PointBuyStatus{Spent: &spent, Warnings: warnings}
	}
	return PointBuyStatus{Spent: nil, Warnings: warnings}
}
