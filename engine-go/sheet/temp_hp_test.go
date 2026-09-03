package sheet

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

func pool(EffectID, Amount int, Pure bool) TempHpPool {
	return TempHpPool{
		EffectID: int64(EffectID), Amount: Amount, Pure: Pure,
		Mods: []map[string]any{{"target": map[string]any{"k": "tempHp"}, "amount": float64(Amount)}},
	}
}

func TestPlanDamage(t *testing.T) {
	// No temp HP: straight to HP, floored at 0.
	if p := PlanDamage(nil, 20, 8); p.HpCurrent != 12 || p.TempHpRemaining != 0 {
		t.Fatalf("plain damage: hp=%d temp=%d, want 12/0", p.HpCurrent, p.TempHpRemaining)
	}
	if p := PlanDamage(nil, 5, 8); p.HpCurrent != 0 {
		t.Fatalf("overkill floors at 0, got %d", p.HpCurrent)
	}

	// Biggest pool drains first; overflow hits HP. Pools 5 + 3, dmg 7:
	// drains 5 (empty→removed if Pure) then 2 of 3 (→1 left), HP untouched.
	p := PlanDamage([]TempHpPool{pool(1, 3, true), pool(2, 5, true)}, 20, 7)
	if p.HpCurrent != 20 {
		t.Errorf("HP should be untouched (temp absorbed all), got %d", p.HpCurrent)
	}
	if p.TempHpRemaining != 1 {
		t.Errorf("tempHpRemaining = %d, want 1", p.TempHpRemaining)
	}
	// Pool 2 (5) emptied + Pure → deleted; pool 1 (3→1) → updated.
	if len(p.DeleteIDs) != 1 || p.DeleteIDs[0] != 2 {
		t.Errorf("expected pool 2 deleted, got %v", p.DeleteIDs)
	}
	if len(p.Updates) != 1 || p.Updates[0].EffectID != 1 {
		t.Errorf("expected pool 1 updated, got %v", p.Updates)
	}

	// Mixed pool (not Pure) emptied → kept with Amount 0, not deleted.
	mp := PlanDamage([]TempHpPool{pool(9, 4, false)}, 20, 10)
	if len(mp.DeleteIDs) != 0 || len(mp.Updates) != 1 {
		t.Errorf("mixed emptied pool should be updated not deleted: del=%v upd=%v", mp.DeleteIDs, mp.Updates)
	}
	if mp.HpCurrent != 14 { // 10 - 4 temp = 6 to HP → 20-6
		t.Errorf("overflow to HP = %d, want 14", mp.HpCurrent)
	}
}

func TestParseTempHpPools(t *testing.T) {
	rows := listEffectRows(
		`[{"target":{"k":"tempHp"},"amount":5,"bonusType":"untyped"}]`,                 // Pure pool
		`[{"target":{"k":"defense"},"amount":2}]`,                                      // no tempHp → skipped
		`[{"target":{"k":"tempHp"},"amount":0}]`,                                       // Amount 0 → skipped
		`[{"target":{"k":"tempHp"},"amount":3},{"target":{"k":"defense"},"amount":1}]`, // mixed
	)
	pools := ParseTempHpPools(rows)
	if len(pools) != 2 {
		t.Fatalf("expected 2 pools, got %d", len(pools))
	}
	if !pools[0].Pure || pools[0].Amount != 5 {
		t.Errorf("pool 0 should be Pure Amount 5, got Pure=%v Amount=%d", pools[0].Pure, pools[0].Amount)
	}
	if pools[1].Pure {
		t.Errorf("mixed pool should not be Pure")
	}
}
