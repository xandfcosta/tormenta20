package api

import (
	"testing"

	"t20engine/db/sqlcgen"
)

func listEffectRows(modifiers ...string) []sqlcgen.ListActiveEffectsByCharacterRow {
	rows := make([]sqlcgen.ListActiveEffectsByCharacterRow, len(modifiers))
	for i, m := range modifiers {
		rows[i] = sqlcgen.ListActiveEffectsByCharacterRow{ID: int64(i + 1), Modifiers: m}
	}
	return rows
}

func pool(effectID, amount int, pure bool) tempHpPool {
	return tempHpPool{
		effectID: int64(effectID), amount: amount, pure: pure,
		mods: []map[string]any{{"target": map[string]any{"k": "tempHp"}, "amount": float64(amount)}},
	}
}

func TestPlanDamage(t *testing.T) {
	// No temp HP: straight to HP, floored at 0.
	if p := planDamage(nil, 20, 8); p.hpCurrent != 12 || p.tempHpRemaining != 0 {
		t.Fatalf("plain damage: hp=%d temp=%d, want 12/0", p.hpCurrent, p.tempHpRemaining)
	}
	if p := planDamage(nil, 5, 8); p.hpCurrent != 0 {
		t.Fatalf("overkill floors at 0, got %d", p.hpCurrent)
	}

	// Biggest pool drains first; overflow hits HP. Pools 5 + 3, dmg 7:
	// drains 5 (empty→removed if pure) then 2 of 3 (→1 left), HP untouched.
	p := planDamage([]tempHpPool{pool(1, 3, true), pool(2, 5, true)}, 20, 7)
	if p.hpCurrent != 20 {
		t.Errorf("HP should be untouched (temp absorbed all), got %d", p.hpCurrent)
	}
	if p.tempHpRemaining != 1 {
		t.Errorf("tempHpRemaining = %d, want 1", p.tempHpRemaining)
	}
	// Pool 2 (5) emptied + pure → deleted; pool 1 (3→1) → updated.
	if len(p.deleteIDs) != 1 || p.deleteIDs[0] != 2 {
		t.Errorf("expected pool 2 deleted, got %v", p.deleteIDs)
	}
	if len(p.updates) != 1 || p.updates[0].effectID != 1 {
		t.Errorf("expected pool 1 updated, got %v", p.updates)
	}

	// Mixed pool (not pure) emptied → kept with amount 0, not deleted.
	mp := planDamage([]tempHpPool{pool(9, 4, false)}, 20, 10)
	if len(mp.deleteIDs) != 0 || len(mp.updates) != 1 {
		t.Errorf("mixed emptied pool should be updated not deleted: del=%v upd=%v", mp.deleteIDs, mp.updates)
	}
	if mp.hpCurrent != 14 { // 10 - 4 temp = 6 to HP → 20-6
		t.Errorf("overflow to HP = %d, want 14", mp.hpCurrent)
	}
}

func TestParseTempHpPools(t *testing.T) {
	rows := listEffectRows(
		`[{"target":{"k":"tempHp"},"amount":5,"bonusType":"untyped"}]`,                 // pure pool
		`[{"target":{"k":"defense"},"amount":2}]`,                                      // no tempHp → skipped
		`[{"target":{"k":"tempHp"},"amount":0}]`,                                       // amount 0 → skipped
		`[{"target":{"k":"tempHp"},"amount":3},{"target":{"k":"defense"},"amount":1}]`, // mixed
	)
	pools := parseTempHpPools(rows)
	if len(pools) != 2 {
		t.Fatalf("expected 2 pools, got %d", len(pools))
	}
	if !pools[0].pure || pools[0].amount != 5 {
		t.Errorf("pool 0 should be pure amount 5, got pure=%v amount=%d", pools[0].pure, pools[0].amount)
	}
	if pools[1].pure {
		t.Errorf("mixed pool should not be pure")
	}
}
