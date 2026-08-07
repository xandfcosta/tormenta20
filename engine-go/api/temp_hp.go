package api

import (
	"encoding/json"
	"sort"

	"t20engine/db/sqlcgen"
)

// Temp-HP damage routing ported from backend temp-hp.helpers.ts: incoming damage
// drains the biggest temp-HP pool first, then HP. Pools live as `tempHp`
// modifiers on active-effect rows; a "pure" pool (only tempHp modifiers) is
// deleted when emptied, a mixed one is kept with its tempHp amount zeroed.

type tempHpPool struct {
	effectID int64
	amount   int
	pure     bool
	mods     []map[string]any // preserved verbatim so a rewrite drops no fields
}

type damageDrain struct {
	EffectID  int64 `json:"effectId"`
	NewAmount int   `json:"newAmount"`
	Removed   bool  `json:"removed"`
}

type damagePlan struct {
	drained         []damageDrain
	updates         []effectModifierWrite
	deleteIDs       []int64
	hpCurrent       int
	tempHpRemaining int
}

type effectModifierWrite struct {
	effectID  int64
	modifiers string
}

// parseTempHpPools mirrors temp-hp.helpers.ts parseTempHpPools.
func parseTempHpPools(rows []sqlcgen.ListActiveEffectsByCharacterRow) []tempHpPool {
	pools := []tempHpPool{}
	for _, row := range rows {
		var mods []map[string]any
		if json.Unmarshal([]byte(row.Modifiers), &mods) != nil {
			continue
		}
		amount, found, pure := 0, false, true
		for _, m := range mods {
			if isTempHpModifier(m) {
				if !found {
					amount = toInt(m["amount"])
					found = true
				}
			} else {
				pure = false
			}
		}
		if !found || amount <= 0 {
			continue
		}
		pools = append(pools, tempHpPool{effectID: row.ID, amount: amount, pure: pure, mods: mods})
	}
	return pools
}

// planDamage mirrors temp-hp.helpers.ts planDamage: drain temp pools (largest
// first), overflow to HP (floored at 0).
func planDamage(pools []tempHpPool, hpCurrent, amount int) damagePlan {
	plan := damagePlan{drained: []damageDrain{}, hpCurrent: hpCurrent}
	sort.SliceStable(pools, func(i, j int) bool { return pools[i].amount > pools[j].amount })
	left := amount
	for _, pool := range pools {
		drained := min(left, pool.amount)
		left -= drained
		newAmount := pool.amount - drained
		plan.tempHpRemaining += newAmount
		if newAmount == pool.amount {
			continue // untouched — not part of the delta
		}
		removed := newAmount == 0 && pool.pure
		plan.drained = append(plan.drained, damageDrain{EffectID: pool.effectID, NewAmount: newAmount, Removed: removed})
		if removed {
			plan.deleteIDs = append(plan.deleteIDs, pool.effectID)
			continue
		}
		plan.updates = append(plan.updates, effectModifierWrite{pool.effectID, withTempHpAmount(pool.mods, newAmount)})
	}
	plan.hpCurrent = max(0, hpCurrent-left)
	return plan
}

// withTempHpAmount rewrites the tempHp modifier's amount in place and re-encodes,
// preserving every other field (map round-trip, not a typed struct).
func withTempHpAmount(mods []map[string]any, amount int) string {
	for _, m := range mods {
		if isTempHpModifier(m) {
			m["amount"] = amount
		}
	}
	b, _ := json.Marshal(mods)
	return string(b)
}

func isTempHpModifier(m map[string]any) bool {
	t, ok := m["target"].(map[string]any)
	return ok && t["k"] == "tempHp"
}

func toInt(v any) int {
	if f, ok := v.(float64); ok {
		return int(f)
	}
	return 0
}
