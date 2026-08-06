package engine

import "testing"

// Port of t20-data/src/items/__tests__/engine*.test.ts — the resolution core.
// Catalog-free: every ActiveItem is built inline. See PORT-PLAN.md §3/§4.

func strp(s string) *string { return &s }

func vested(source string, mods ...Modifier) ActiveItem {
	return ActiveItem{Source: source, Equipped: strp("vested"), Modifiers: mods}
}
func wielded(source string, mods ...Modifier) ActiveItem {
	return ActiveItem{Source: source, Equipped: strp("wielded"), Modifiers: mods}
}

var defenseTarget = ModifierTarget{K: "defense"}
var damageThisTarget = ModifierTarget{K: "damage", Scope: "this"}

// ─── targetKey ────────────────────────────────────────────────────────

func TestTargetKeyDistinguishesAttackScopes(t *testing.T) {
	if targetKey(ModifierTarget{K: "attack", Scope: "this"}) == targetKey(ModifierTarget{K: "attack", Scope: "all"}) {
		t.Fatal("attack scopes must differ")
	}
}

func TestTargetKeyFullUnion(t *testing.T) {
	cases := []struct {
		target ModifierTarget
		want   string
	}{
		{ModifierTarget{K: "expertise", Name: "Atletismo"}, "expertise:Atletismo"},
		{ModifierTarget{K: "expertiseAll"}, "expertiseAll"},
		{ModifierTarget{K: "expertiseRemovePenalty", Name: "Furtividade"}, "expertiseRemovePenalty:Furtividade"},
		{ModifierTarget{K: "expertiseByAttribute", Attribute: "strength"}, "expertiseByAttribute:strength"},
		{ModifierTarget{K: "attribute", Name: "wisdom"}, "attribute:wisdom"},
		{ModifierTarget{K: "defense"}, "defense"},
		{ModifierTarget{K: "defenseDexCap"}, "defenseDexCap"},
		{ModifierTarget{K: "resistance"}, "resistance"},
		{ModifierTarget{K: "fearResistance"}, "fearResistance"},
		{ModifierTarget{K: "attack", Scope: "this"}, "attack:this"},
		{ModifierTarget{K: "damage", Scope: "all"}, "damage:all"},
		{ModifierTarget{K: "critRange"}, "critRange"},
		{ModifierTarget{K: "critMult"}, "critMult"},
		{ModifierTarget{K: "pmLimit"}, "pmLimit"},
		{ModifierTarget{K: "pmCost"}, "pmCost"},
		{ModifierTarget{K: "catalyst", School: "abjuracao"}, "catalyst:abjuracao"},
		{ModifierTarget{K: "spellDC"}, "spellDC"},
		{ModifierTarget{K: "inventorySlots"}, "inventorySlots"},
		{ModifierTarget{K: "displacement"}, "displacement"},
		{ModifierTarget{K: "flySpeed"}, "flySpeed"},
		{ModifierTarget{K: "armorPenalty"}, "armorPenalty"},
		{ModifierTarget{K: "armorPenaltyExpertises"}, "armorPenaltyExpertises"},
		{ModifierTarget{K: "tempHp"}, "tempHp"},
		{ModifierTarget{K: "tempMp"}, "tempMp"},
		{ModifierTarget{K: "maxPv"}, "maxPv"},
		{ModifierTarget{K: "maxPm"}, "maxPm"},
		{ModifierTarget{K: "maneuver", Name: "derrubar"}, "maneuver:derrubar"},
		{ModifierTarget{K: "flag", Name: "fatigue-on-sleep"}, "flag:fatigue-on-sleep"},
	}
	for _, c := range cases {
		if got := targetKey(c.target); got != c.want {
			t.Errorf("targetKey(%+v) = %q, want %q", c.target, got, c.want)
		}
	}
}

// ─── non-stacking by bonusType ────────────────────────────────────────

func TestNonStacking(t *testing.T) {
	always := func(k string) *ModifierCondition { return &ModifierCondition{C: k} }
	tests := []struct {
		name       string
		items      []ActiveItem
		target     ModifierTarget
		wantTotal  int
		wantLenSet bool
		wantLen    int
	}{
		{
			name: "two armor keep highest",
			items: []ActiveItem{
				vested("cota-malha", Modifier{Target: defenseTarget, Amount: 6, BonusType: "armor", Condition: always("vested")}),
				vested("brunea", Modifier{Target: defenseTarget, Amount: 2, BonusType: "armor", Condition: always("vested")}),
			},
			target: defenseTarget, wantTotal: 6, wantLenSet: true, wantLen: 1,
		},
		{
			name: "two untyped stack",
			items: []ActiveItem{
				wielded("a", Modifier{Target: damageThisTarget, Amount: 1, BonusType: "untyped", Condition: always("wielded")}),
				wielded("b", Modifier{Target: damageThisTarget, Amount: 2, BonusType: "untyped", Condition: always("wielded")}),
			},
			target: damageThisTarget, wantTotal: 3, wantLenSet: true, wantLen: 2,
		},
		{
			name: "untyped stacks on typed",
			items: []ActiveItem{
				vested("armor", Modifier{Target: defenseTarget, Amount: 5, BonusType: "armor", Condition: always("vested")}),
				vested("untyped-bonus", Modifier{Target: defenseTarget, Amount: 1, BonusType: "untyped", Condition: always("vested")}),
			},
			target: defenseTarget, wantTotal: 6,
		},
		{
			name: "armor + shield stack (p226)",
			items: []ActiveItem{
				vested("armadura-completa", Modifier{Target: defenseTarget, Amount: 10, BonusType: "armor", Condition: always("vested")}),
				wielded("escudo-pesado", Modifier{Target: defenseTarget, Amount: 2, BonusType: "shield", Condition: always("wielded")}),
			},
			target: defenseTarget, wantTotal: 12, wantLenSet: true, wantLen: 2,
		},
		{
			name: "two shields do not stack",
			items: []ActiveItem{
				wielded("escudo-leve", Modifier{Target: defenseTarget, Amount: 1, BonusType: "shield", Condition: always("wielded")}),
				wielded("escudo-pesado", Modifier{Target: defenseTarget, Amount: 2, BonusType: "shield", Condition: always("wielded")}),
			},
			target: defenseTarget, wantTotal: 2,
		},
		{
			name: "most-negative kept",
			items: []ActiveItem{
				vested("couro-batido", Modifier{Target: ModifierTarget{K: "armorPenalty"}, Amount: -1, BonusType: "armor", Condition: always("vested")}),
				vested("cota-malha", Modifier{Target: ModifierTarget{K: "armorPenalty"}, Amount: -2, BonusType: "armor", Condition: always("vested")}),
			},
			target: ModifierTarget{K: "armorPenalty"}, wantTotal: -2,
		},
		{
			name: "three same-type keeps highest-abs",
			items: []ActiveItem{
				vested("a", Modifier{Target: defenseTarget, Amount: 2, BonusType: "enhancement", Condition: always("vested")}),
				vested("b", Modifier{Target: defenseTarget, Amount: 4, BonusType: "enhancement", Condition: always("vested")}),
				vested("c", Modifier{Target: defenseTarget, Amount: 3, BonusType: "enhancement", Condition: always("vested")}),
			},
			target: defenseTarget, wantTotal: 4, wantLenSet: true, wantLen: 1,
		},
		{
			name: "different typed both apply (sum)",
			items: []ActiveItem{
				vested("cota-malha", Modifier{Target: defenseTarget, Amount: 5, BonusType: "armor", Condition: always("vested")}),
				vested("aco-rubi", Modifier{Target: defenseTarget, Amount: 1, BonusType: "enhancement", Condition: always("vested")}),
			},
			target: defenseTarget, wantTotal: 6, wantLenSet: true, wantLen: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stat := StatFor(ComputeItemEffects(tt.items), tt.target)
			if stat.Total != tt.wantTotal {
				t.Errorf("total = %d, want %d", stat.Total, tt.wantTotal)
			}
			if tt.wantLenSet && len(stat.Contributions) != tt.wantLen {
				t.Errorf("contributions = %d, want %d", len(stat.Contributions), tt.wantLen)
			}
		})
	}
}

func TestThreeSameTypeKeepsSourceB(t *testing.T) {
	always := &ModifierCondition{C: "vested"}
	stat := StatFor(ComputeItemEffects([]ActiveItem{
		vested("a", Modifier{Target: defenseTarget, Amount: 2, BonusType: "enhancement", Condition: always}),
		vested("b", Modifier{Target: defenseTarget, Amount: 4, BonusType: "enhancement", Condition: always}),
		vested("c", Modifier{Target: defenseTarget, Amount: 3, BonusType: "enhancement", Condition: always}),
	}), defenseTarget)
	if len(stat.Contributions) != 1 || stat.Contributions[0].Source != "b" {
		t.Fatalf("expected single contribution from source b, got %+v", stat.Contributions)
	}
}

// ─── equip gating ─────────────────────────────────────────────────────

func TestEquipGating(t *testing.T) {
	w := func() *ModifierCondition { return &ModifierCondition{C: "wielded"} }
	v := func() *ModifierCondition { return &ModifierCondition{C: "vested"} }

	// wielded mod on a vested item → ignored.
	if StatFor(ComputeItemEffects([]ActiveItem{
		vested("weapon-on-belt", Modifier{Target: damageThisTarget, Amount: 1, BonusType: "untyped", Condition: w()}),
	}), damageThisTarget).Total != 0 {
		t.Error("wielded mod applied on vested item")
	}

	// equipped=null → whole item ignored.
	nullItem := ActiveItem{Source: "pack", Equipped: nil, Modifiers: []Modifier{
		{Target: defenseTarget, Amount: 5, BonusType: "armor", Condition: v()},
	}}
	if StatFor(ComputeItemEffects([]ActiveItem{nullItem}), defenseTarget).Total != 0 {
		t.Error("null-equipped item contributed")
	}

	// wielded2 counts as wielded.
	offHand := ActiveItem{Source: "off-hand", Equipped: strp("wielded2"), Modifiers: []Modifier{
		{Target: damageThisTarget, Amount: 1, BonusType: "untyped", Condition: w()},
	}}
	if StatFor(ComputeItemEffects([]ActiveItem{offHand}), damageThisTarget).Total != 1 {
		t.Error("wielded2 did not count as wielded")
	}

	// vested mod on a wielded item → ignored (symmetric).
	if StatFor(ComputeItemEffects([]ActiveItem{
		wielded("vest-and-blade", Modifier{Target: defenseTarget, Amount: 2, BonusType: "armor", Condition: v()}),
	}), defenseTarget).Total != 0 {
		t.Error("vested mod applied on wielded item")
	}
}

// ─── always / no-condition ────────────────────────────────────────────

func TestAlwaysCondition(t *testing.T) {
	always := &ModifierCondition{C: "always"}
	if StatFor(ComputeItemEffects([]ActiveItem{
		vested("relic", Modifier{Target: defenseTarget, Amount: 1, BonusType: "untyped", Condition: always}),
	}), defenseTarget).Total != 1 {
		t.Error("always on vested item should apply")
	}
	if StatFor(ComputeItemEffects([]ActiveItem{
		wielded("amulet", Modifier{Target: defenseTarget, Amount: 1, BonusType: "untyped", Condition: always}),
	}), defenseTarget).Total != 1 {
		t.Error("always on wielded item should apply")
	}
	nullItem := ActiveItem{Source: "pack", Equipped: nil, Modifiers: []Modifier{
		{Target: defenseTarget, Amount: 1, BonusType: "untyped", Condition: always},
	}}
	if StatFor(ComputeItemEffects([]ActiveItem{nullItem}), defenseTarget).Total != 0 {
		t.Error("always on null-equipped item should be suppressed")
	}
	// no condition at all → falls back to always.
	if StatFor(ComputeItemEffects([]ActiveItem{
		vested("ring", Modifier{Target: defenseTarget, Amount: 1, BonusType: "untyped"}),
	}), defenseTarget).Total != 1 {
		t.Error("missing condition should default to always")
	}
}

// ─── flags ────────────────────────────────────────────────────────────

func TestFlagModifiersPopulateFlagsSet(t *testing.T) {
	effects := ComputeItemEffects([]ActiveItem{
		vested("cota-malha", Modifier{Target: ModifierTarget{K: "flag", Name: "fatigue-on-sleep"}, Amount: 1, BonusType: "untyped", Condition: &ModifierCondition{C: "vested"}}),
	})
	if !effects.Flags["fatigue-on-sleep"] {
		t.Error("flag not set")
	}
	if _, ok := effects.ByTarget["flag:fatigue-on-sleep"]; ok {
		t.Error("flag should not bucket into byTarget")
	}
}

// ─── flagOff (Pele de Ferro p42) ──────────────────────────────────────

func peleDeFerro() ActiveItem {
	return ActiveItem{Source: "Classe: Bárbaro", Equipped: strp("vested"), Modifiers: []Modifier{
		{Target: defenseTarget, Amount: 4, BonusType: "untyped", Condition: &ModifierCondition{C: "flagOff", Flag: "armadura-pesada", Label: "sem armadura pesada"}},
	}}
}
func brunea() ActiveItem {
	return ActiveItem{Source: "Brunea", Equipped: strp("vested"), Modifiers: []Modifier{
		{Target: defenseTarget, Amount: 5, BonusType: "armor", Condition: &ModifierCondition{C: "vested"}},
		{Target: ModifierTarget{K: "flag", Name: "armadura-pesada"}, Amount: 1, BonusType: "untyped", Condition: &ModifierCondition{C: "vested"}},
	}}
}

func TestFlagOff(t *testing.T) {
	if StatFor(ComputeItemEffects([]ActiveItem{peleDeFerro()}), defenseTarget).Total != 4 {
		t.Error("flagOff should apply when flag absent")
	}
	if StatFor(ComputeItemEffects([]ActiveItem{peleDeFerro(), brunea()}), defenseTarget).Total != 5 {
		t.Error("flagOff should switch off with heavy armor (only brunea)")
	}
	// order-independent (pre-pass collects flags first).
	if StatFor(ComputeItemEffects([]ActiveItem{brunea(), peleDeFerro()}), defenseTarget).Total != 5 {
		t.Error("flagOff result should be order-independent")
	}
	if len(ComputeItemEffects([]ActiveItem{peleDeFerro()}).Conditional) != 0 {
		t.Error("flagOff must not become a togglable conditional")
	}
}

// ─── conditional opt-ins ──────────────────────────────────────────────

func TestConditionalDeferral(t *testing.T) {
	// against → deferred, no numeric total.
	effA := ComputeItemEffects([]ActiveItem{
		wielded("material-aco-rubi", Modifier{Target: damageThisTarget, Amount: 2, BonusType: "enhancement", Condition: &ModifierCondition{C: "against", Trait: "vivos"}, Note: "+2 dano vs vivos"}),
	})
	if len(effA.Conditional) != 1 || StatFor(effA, damageThisTarget).Total != 0 {
		t.Error("against condition should defer to conditional list")
	}

	// terrain → note "terreno: floresta".
	effT := ComputeItemEffects([]ActiveItem{
		vested("explorador-boots", Modifier{Target: ModifierTarget{K: "expertise", Name: "Sobrevivência"}, Amount: 2, BonusType: "item", Condition: &ModifierCondition{C: "terrain", Type: "floresta"}}),
	})
	if len(effT.Conditional) != 1 || effT.Conditional[0].Note != "terreno: floresta" {
		t.Errorf("terrain note = %q", effT.Conditional[0].Note)
	}

	// context → note from payload.
	effC := ComputeItemEffects([]ActiveItem{
		vested("item", Modifier{Target: defenseTarget, Amount: 1, BonusType: "untyped", Condition: &ModifierCondition{C: "context", Note: "ao usar manobra"}}),
	})
	if effC.Conditional[0].Note != "ao usar manobra" {
		t.Errorf("context note = %q", effC.Conditional[0].Note)
	}

	// flagOn → carries flag + label note.
	effF := ComputeItemEffects([]ActiveItem{
		wielded("amulet", Modifier{Target: damageThisTarget, Amount: 2, BonusType: "untyped", Condition: &ModifierCondition{C: "flagOn", Flag: "enraged", Label: "Enfurecido"}}),
	})
	if effF.Conditional[0].Flag != "enraged" || effF.Conditional[0].Note != "Enfurecido" {
		t.Errorf("flagOn conditional = %+v", effF.Conditional[0])
	}

	// describeCondition non-empty wins over modifier.note.
	effFb := ComputeItemEffects([]ActiveItem{
		vested("item", Modifier{Target: defenseTarget, Amount: 1, BonusType: "untyped", Condition: &ModifierCondition{C: "against", Trait: ""}, Note: "fallback"}),
	})
	if effFb.Conditional[0].Note != "contra: " {
		t.Errorf("expected 'contra: ', got %q", effFb.Conditional[0].Note)
	}
}

// ─── applyActiveConditionals ──────────────────────────────────────────

func TestApplyActiveConditionals(t *testing.T) {
	base := ComputeItemEffects([]ActiveItem{
		wielded("material-aco-rubi", Modifier{Target: damageThisTarget, Amount: 2, BonusType: "enhancement", Condition: &ModifierCondition{C: "against", Trait: "vivos"}, Note: "+2 dano vs vivos"}),
	})
	id := ConditionalID(base.Conditional[0])
	next := ApplyActiveConditionals(base, map[string]bool{id: true})
	if StatFor(next, damageThisTarget).Total != 2 || len(next.Conditional) != 0 {
		t.Errorf("fold failed: total=%d remaining=%d", StatFor(next, damageThisTarget).Total, len(next.Conditional))
	}

	// no active ids → original returned.
	same := ApplyActiveConditionals(base, map[string]bool{})
	if len(same.Conditional) != len(base.Conditional) {
		t.Error("empty active set should return original")
	}

	// unmatched conditional stays in remaining.
	multi := ComputeItemEffects([]ActiveItem{
		wielded("a",
			Modifier{Target: damageThisTarget, Amount: 2, BonusType: "enhancement", Condition: &ModifierCondition{C: "against", Trait: "vivos"}},
			Modifier{Target: damageThisTarget, Amount: 1, BonusType: "untyped", Condition: &ModifierCondition{C: "terrain", Type: "urbano"}},
		),
	})
	first := ConditionalID(multi.Conditional[0])
	if len(ApplyActiveConditionals(multi, map[string]bool{first: true}).Conditional) != 1 {
		t.Error("unmatched conditional should remain")
	}
}

func TestApplyActiveConditionalsReResolves(t *testing.T) {
	base := ComputeItemEffects([]ActiveItem{
		vested("base", Modifier{Target: defenseTarget, Amount: 1, BonusType: "enhancement", Condition: &ModifierCondition{C: "vested"}}),
		wielded("blade", Modifier{Target: defenseTarget, Amount: 3, BonusType: "enhancement", Condition: &ModifierCondition{C: "against", Trait: "mortos-vivos"}}),
	})
	if StatFor(base, defenseTarget).Total != 1 {
		t.Fatalf("base total = %d, want 1", StatFor(base, defenseTarget).Total)
	}
	next := ApplyActiveConditionals(base, map[string]bool{ConditionalID(base.Conditional[0]): true})
	def := StatFor(next, defenseTarget)
	if def.Total != 3 || len(def.Contributions) != 1 || def.Contributions[0].Amount != 3 {
		t.Errorf("conditional +3 should displace base +1, got %+v", def)
	}
}

func TestApplyActiveConditionalsIgnoresFlagTargets(t *testing.T) {
	base := ComputeItemEffects([]ActiveItem{
		vested("weird", Modifier{Target: ModifierTarget{K: "flag", Name: "fatigue-on-sleep"}, Amount: 1, BonusType: "untyped", Condition: &ModifierCondition{C: "context", Note: "durante a noite"}}),
	})
	next := ApplyActiveConditionals(base, map[string]bool{ConditionalID(base.Conditional[0]): true})
	if next.Flags["fatigue-on-sleep"] {
		t.Error("flag-target conditional should not set the flag on fold")
	}
	if _, ok := next.ByTarget["flag:fatigue-on-sleep"]; ok {
		t.Error("flag-target conditional should not bucket on fold")
	}
}

// ─── resolveConditionalDisplay ────────────────────────────────────────

func TestResolveConditionalDisplayTierDedupe(t *testing.T) {
	furiaTargets := []ModifierTarget{
		{K: "attack", Scope: "all"},
		{K: "damage", Scope: "all"},
		{K: "expertise", Name: "Fortitude"},
		{K: "expertise", Name: "Vontade"},
	}
	tier := func(amount int) []ConditionalDisplayInput {
		out := []ConditionalDisplayInput{}
		for _, tg := range furiaTargets {
			out = append(out, ConditionalDisplayInput{Target: tg, BonusType: "morale", Amount: amount})
		}
		return out
	}
	kept := ResolveConditionalDisplay(append(tier(2), tier(3)...))
	if len(kept) != 4 {
		t.Fatalf("kept = %d, want 4", len(kept))
	}
	for _, row := range kept {
		if row.Amount != 3 {
			t.Errorf("row amount = %d, want 3 (only +3 tier survives)", row.Amount)
		}
	}
}

func TestResolveConditionalDisplayUntypedStacks(t *testing.T) {
	kept := ResolveConditionalDisplay([]ConditionalDisplayInput{
		{Target: defenseTarget, BonusType: "morale", Amount: 2},
		{Target: defenseTarget, BonusType: "untyped", Amount: 1},
		{Target: defenseTarget, BonusType: "untyped", Amount: 1},
	})
	sum := 0
	for _, r := range kept {
		sum += r.Amount
	}
	if len(kept) != 3 || sum != 4 {
		t.Errorf("kept=%d sum=%d, want 3 rows summing 4", len(kept), sum)
	}
}

func TestResolveConditionalDisplayEmpty(t *testing.T) {
	if len(ResolveConditionalDisplay(nil)) != 0 {
		t.Error("empty input should yield empty output")
	}
}

// ─── statFor / conditionalId ──────────────────────────────────────────

func TestStatForAbsentTarget(t *testing.T) {
	res := StatFor(ComputeItemEffects(nil), defenseTarget)
	if res.Total != 0 || len(res.Contributions) != 0 {
		t.Errorf("absent target should be zeroed, got %+v", res)
	}
}

func TestConditionalIDDivergence(t *testing.T) {
	a := ConditionalEffect{Source: "a", BonusType: "untyped", Amount: 2, Note: "n", Target: defenseTarget}
	if ConditionalID(a) != ConditionalID(a) {
		t.Error("conditionalId must be stable")
	}
	diffs := []ConditionalEffect{
		{Source: "b", BonusType: "untyped", Amount: 2, Note: "n", Target: defenseTarget},
		{Source: "a", BonusType: "untyped", Amount: 2, Note: "n", Target: damageThisTarget},
		{Source: "a", BonusType: "untyped", Amount: 3, Note: "n", Target: defenseTarget},
		{Source: "a", BonusType: "enhancement", Amount: 2, Note: "n", Target: defenseTarget},
	}
	for _, d := range diffs {
		if ConditionalID(a) == ConditionalID(d) {
			t.Errorf("conditionalId should differ for %+v", d)
		}
	}
}

// ─── empty / trivial ──────────────────────────────────────────────────

func TestEmptyInputs(t *testing.T) {
	for _, items := range [][]ActiveItem{nil, {vested("empty")}} {
		e := ComputeItemEffects(items)
		if len(e.ByTarget) != 0 || len(e.Flags) != 0 || len(e.Conditional) != 0 {
			t.Errorf("expected wholly-empty ItemEffects, got %+v", e)
		}
	}
}
