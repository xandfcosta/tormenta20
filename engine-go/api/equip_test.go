package api

import (
	"testing"

	"t20engine/engine"
)

func TestEquipLimitError(t *testing.T) {
	// 2 hands cap: wielded (1) + wielded2 (2) = 3 > 2.
	if equipLimitError([]string{"wielded"}, "wielded2") == "" {
		t.Error("expected hands-limit error")
	}
	// 4 vested cap: four already worn + a fifth.
	if equipLimitError([]string{"vested", "vested", "vested", "vested"}, "vested") == "" {
		t.Error("expected vested-limit error")
	}
	// Within limits.
	if msg := equipLimitError([]string{"vested", "vested"}, "vested"); msg != "" {
		t.Errorf("3 vested should pass, got %q", msg)
	}
	if msg := equipLimitError([]string{"wielded"}, "wielded"); msg != "" {
		t.Errorf("2 hands should pass, got %q", msg)
	}
}

func TestEquipAxisError(t *testing.T) {
	shield := &engine.CatalogItem{ID: "escudo", Name: "Escudo", Equip: "wielded"}
	if top, _ := equipAxisError(shield, "vested"); top == "" {
		t.Error("shield vested should be rejected")
	}
	if top, _ := equipAxisError(shield, "wielded"); top != "" {
		t.Error("shield wielded should be allowed")
	}
	armor := &engine.CatalogItem{ID: "arm", Name: "Armadura", Equip: "vested"}
	if top, _ := equipAxisError(armor, "wielded"); top == "" {
		t.Error("armor wielded should be rejected")
	}
	if top, _ := equipAxisError(nil, "wielded"); top != "" {
		t.Error("custom item (nil catalog) should skip the axis check")
	}
	homebrew := &engine.CatalogItem{ID: "medalhao-de-prata", Name: "Medalhão", Equip: "wielded"}
	if top, _ := equipAxisError(homebrew, "vested"); top != "" {
		t.Error("homebrew vested allowance should pass")
	}
}

func TestSlotsNotMultiple(t *testing.T) {
	for _, ok := range []float64{0, 0.5, 1, 1.5, 2} {
		if slotsNotMultiple(ok) {
			t.Errorf("%v should be a valid 0.5 multiple", ok)
		}
	}
	for _, bad := range []float64{0.3, 0.7, 1.1} {
		if !slotsNotMultiple(bad) {
			t.Errorf("%v should be rejected", bad)
		}
	}
}
