package api

import "testing"

// A monster dropped into the tracker from the bestiary carries its PV — that is
// the whole point of tracking it there. `materializeNpcEntry` used to build the
// entry from label/initiative/type alone and DISCARD hpCurrent/hpMax, so the
// client sent them and the row arrived with no health bar (ALE-75).
func TestMaterializeNpcEntryKeepsHp(t *testing.T) {
	entry, err := materializeNpcEntry(map[string]any{
		"label":      "Goblin 1",
		"initiative": float64(13),
		"type":       "npc",
		"hpCurrent":  float64(4),
		"hpMax":      float64(4),
	})
	if err != nil {
		t.Fatalf("materializeNpcEntry: %v", err)
	}
	if entry.HpCurrent == nil || *entry.HpCurrent != 4 {
		t.Errorf("HpCurrent = %v, want 4", entry.HpCurrent)
	}
	if entry.HpMax == nil || *entry.HpMax != 4 {
		t.Errorf("HpMax = %v, want 4", entry.HpMax)
	}
}

// An entry without PV stays without PV — a bare NPC ("Voz na escuridão") has no
// health to track, and zeroing it would draw an empty bar that means nothing.
func TestMaterializeNpcEntryOmitsAbsentHp(t *testing.T) {
	entry, err := materializeNpcEntry(map[string]any{
		"label":      "Voz na escuridão",
		"initiative": float64(7),
	})
	if err != nil {
		t.Fatalf("materializeNpcEntry: %v", err)
	}
	if entry.HpCurrent != nil || entry.HpMax != nil {
		t.Errorf("PV = %v/%v, want nil/nil", entry.HpCurrent, entry.HpMax)
	}
}

func TestMaterializeNpcEntryRequiresLabel(t *testing.T) {
	if _, err := materializeNpcEntry(map[string]any{"initiative": float64(1)}); err == nil {
		t.Error("expected an error for a missing label")
	}
}
