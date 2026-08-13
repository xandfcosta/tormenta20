package engine

import (
	"encoding/json"
	"math"
	"sort"
	"strconv"
)

// This file is the Go port of t20-data/src/items/engine.ts — the REAL front
// derivation engine (computeItemEffects), distinct from the MVP orchestrator
// (ComputeCharacterSheet) in compute.go. It is catalog-free: it resolves a
// pre-collected []ActiveItem into ItemEffects (non-stacking by bonusType, flags,
// conditional opt-ins). The catalog-reading collection layer (activeItemsFor)
// and the breakdown helpers land in later slices. See PORT-PLAN.md §2/§4.

// ─── Types (mirror items/types.ts + items/engine.ts) ──────────────────

// ModifierTarget is the TS discriminated union `{ k; ... }` flattened: only the
// fields a given `k` uses are populated (omitempty keeps the JSON shape 1:1).
type ModifierTarget struct {
	K         string `json:"k"`
	Name      string `json:"name,omitempty"`      // expertise, expertiseRemovePenalty, attribute, maneuver, flag
	Attribute string `json:"attribute,omitempty"` // expertiseByAttribute
	Scope     string `json:"scope,omitempty"`     // attack, damage ('this' | 'all')
	School    string `json:"school,omitempty"`    // catalyst
}

// ModifierCondition is the TS `ModifierCondition` union, flattened like above.
type ModifierCondition struct {
	C     string `json:"c"`
	Type  string `json:"type,omitempty"`  // terrain
	Trait string `json:"trait,omitempty"` // against
	Note  string `json:"note,omitempty"`  // context
	Flag  string `json:"flag,omitempty"`  // flagOn, flagOff
	Label string `json:"label,omitempty"` // flagOn, flagOff
}

// VitalScale mirrors items/types.ts VitalScale. Only the vitals collector reads
// it; the resolution engine ignores it. Carried on Modifier so the collection
// layer round-trips maxPv/maxPm mods byte-equal to the TS oracle.
type VitalScale struct {
	Per       string `json:"per"`
	Step      int    `json:"step,omitempty"`
	Round     string `json:"round,omitempty"`
	Attribute string `json:"attribute,omitempty"`
}

// Modifier mirrors items/types.ts Modifier. `scale` (maxPv/maxPm) is ignored by
// the resolution engine but preserved for the collection layer's parity dump.
type Modifier struct {
	Target    ModifierTarget     `json:"target"`
	Amount    int                `json:"amount"`
	BonusType string             `json:"bonusType"`
	Condition *ModifierCondition `json:"condition,omitempty"`
	Note      string             `json:"note,omitempty"`
	Scale     *VitalScale        `json:"scale,omitempty"`
}

// UnmarshalJSON rounds a modifier's amount to the nearest int. The engine is
// integer-modeled (see types.go), but the TS type is `number` and one catalog
// entry carries a fractional amount (botas-reforcadas, +1.5m displacement — not
// equipped by any seed). Rounding at the JSON boundary keeps catalog parsing from
// failing without widening every total to float. Integer amounts pass through
// unchanged, so parity is unaffected.
func (m *Modifier) UnmarshalJSON(b []byte) error {
	var shadow struct {
		Target    ModifierTarget     `json:"target"`
		Amount    float64            `json:"amount"`
		BonusType string             `json:"bonusType"`
		Condition *ModifierCondition `json:"condition"`
		Note      string             `json:"note"`
		Scale     *VitalScale        `json:"scale"`
	}
	if err := json.Unmarshal(b, &shadow); err != nil {
		return err
	}
	*m = Modifier{
		Target:    shadow.Target,
		Amount:    int(math.Round(shadow.Amount)),
		BonusType: shadow.BonusType,
		Condition: shadow.Condition,
		Note:      shadow.Note,
		Scale:     shadow.Scale,
	}
	return nil
}

// ActiveItem mirrors items/engine.ts ActiveItem. Equipped is a pointer so the
// null wear-state (item present but unequipped) is distinguishable from "vested".
type ActiveItem struct {
	Source    string     `json:"source"`
	Equipped  *string    `json:"equipped"`
	Modifiers []Modifier `json:"modifiers"`
}

type Contribution struct {
	Source    string `json:"source"`
	BonusType string `json:"bonusType"`
	Amount    int    `json:"amount"`
	Note      string `json:"note,omitempty"`
}

type AggregatedStat struct {
	Total         int            `json:"total"`
	Contributions []Contribution `json:"contributions"`
}

type ConditionalEffect struct {
	Source    string         `json:"source"`
	BonusType string         `json:"bonusType"`
	Amount    int            `json:"amount"`
	Note      string         `json:"note"`
	Target    ModifierTarget `json:"target"`
	Flag      string         `json:"flag,omitempty"`
}

// ItemEffects mirrors items/engine.ts ItemEffects. Flags is a Set (map);
// MarshalJSON emits it as a sorted array so JSON parity with the TS oracle is
// order-independent.
type ItemEffects struct {
	ByTarget    map[string]AggregatedStat
	Flags       map[string]bool
	Conditional []ConditionalEffect
}

// ItemEffectsWire é a forma que o ItemEffects assume NO FIO: as flags viram
// array ordenado em vez do Set interno, e os nils viram vazios.
//
// Tem nome próprio (em vez de struct anônima) porque é o contrato que o gerador
// de tipos TS emite — refletir a struct em memória produziria
// `Flags: Record<string, boolean>`, que é mentira (ALE-108).
type ItemEffectsWire struct {
	ByTarget    map[string]AggregatedStat `json:"byTarget"`
	Flags       []string                  `json:"flags"`
	Conditional []ConditionalEffect       `json:"conditional"`
}

func (e ItemEffects) MarshalJSON() ([]byte, error) {
	byTarget := e.ByTarget
	if byTarget == nil {
		byTarget = map[string]AggregatedStat{}
	}
	conditional := e.Conditional
	if conditional == nil {
		conditional = []ConditionalEffect{}
	}
	return json.Marshal(ItemEffectsWire{byTarget, e.FlagList(), conditional})
}

// FlagList returns the active flag names sorted — the stable form for JSON and
// for value comparison against the TS Set.
func (e ItemEffects) FlagList() []string {
	out := make([]string, 0, len(e.Flags))
	for f := range e.Flags {
		out = append(out, f)
	}
	sort.Strings(out)
	return out
}

// ─── targetKey (stable identity per target shape) ─────────────────────

func targetKey(t ModifierTarget) string {
	switch t.K {
	case "expertise":
		return "expertise:" + t.Name
	case "expertiseAll":
		return "expertiseAll"
	case "expertiseRemovePenalty":
		return "expertiseRemovePenalty:" + t.Name
	case "expertiseByAttribute":
		return "expertiseByAttribute:" + t.Attribute
	case "attribute":
		return "attribute:" + t.Name
	case "defense":
		return "defense"
	case "defenseDexCap":
		return "defenseDexCap"
	case "resistance":
		return "resistance"
	case "fearResistance":
		return "fearResistance"
	case "attack":
		return "attack:" + t.Scope
	case "damage":
		return "damage:" + t.Scope
	case "critRange":
		return "critRange"
	case "critMult":
		return "critMult"
	case "pmLimit":
		return "pmLimit"
	case "pmCost":
		return "pmCost"
	case "catalyst":
		return "catalyst:" + t.School
	case "spellDC":
		return "spellDC"
	case "inventorySlots":
		return "inventorySlots"
	case "displacement":
		return "displacement"
	case "flySpeed":
		return "flySpeed"
	case "armorPenalty":
		return "armorPenalty"
	case "armorPenaltyExpertises":
		return "armorPenaltyExpertises"
	case "tempHp":
		return "tempHp"
	case "tempMp":
		return "tempMp"
	case "maxPv":
		return "maxPv"
	case "maxPm":
		return "maxPm"
	case "maneuver":
		return "maneuver:" + t.Name
	case "flag":
		return "flag:" + t.Name
	}
	return ""
}

// ─── condition helpers ────────────────────────────────────────────────

func isUnconditional(m Modifier) bool {
	if m.Condition == nil {
		return true
	}
	switch m.Condition.C {
	case "always", "wielded", "vested":
		return true
	case "terrain", "against", "context", "flagOn":
		return false
	case "flagOff":
		// Auto-evaluated against collected flags in the main pass — never a
		// user-toggled conditional.
		return true
	}
	return true
}

func isWielded(equipped *string) bool {
	return equipped != nil && (*equipped == "wielded" || *equipped == "wielded2")
}

func conditionMet(m Modifier, equipped *string) bool {
	if m.Condition == nil || m.Condition.C == "always" {
		return true
	}
	switch m.Condition.C {
	case "wielded":
		return isWielded(equipped)
	case "vested":
		return equipped != nil && *equipped == "vested"
	case "flagOff":
		// flagOff passed the flags gate in the main loop — treat as met here.
		return true
	}
	return false
}

func describeCondition(m Modifier) string {
	if m.Condition == nil {
		return ""
	}
	switch m.Condition.C {
	case "terrain":
		return "terreno: " + m.Condition.Type
	case "against":
		return "contra: " + m.Condition.Trait
	case "context":
		return m.Condition.Note
	case "flagOn", "flagOff":
		return m.Condition.Label
	}
	return ""
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// ─── non-stacking resolution ──────────────────────────────────────────

// resolveStack applies the T20 non-stacking rule: within a target, entries of
// the same bonusType keep only the highest-abs; 'untyped' stack freely. The
// contribution order follows the first-seen order of each bonusType (matching
// the TS Map iteration order) so JSON parity holds.
func resolveStack(contribs []Contribution) AggregatedStat {
	order := []string{}
	byType := map[string][]Contribution{}
	for _, c := range contribs {
		if _, ok := byType[c.BonusType]; !ok {
			order = append(order, c.BonusType)
		}
		byType[c.BonusType] = append(byType[c.BonusType], c)
	}

	kept := []Contribution{}
	for _, bt := range order {
		list := byType[bt]
		if bt == "untyped" {
			kept = append(kept, list...)
			continue
		}
		best := list[0]
		for _, e := range list {
			if absInt(e.Amount) > absInt(best.Amount) {
				best = e
			}
		}
		kept = append(kept, best)
	}

	total := 0
	for _, c := range kept {
		total += c.Amount
	}
	return AggregatedStat{Total: total, Contributions: kept}
}

// ConditionalDisplayInput is one conditional-effect row fed to
// ResolveConditionalDisplay (an active stance's rows).
type ConditionalDisplayInput struct {
	Target    ModifierTarget `json:"target"`
	BonusType string         `json:"bonusType"`
	Amount    int            `json:"amount"`
}

type ConditionalDisplayRow struct {
	Target ModifierTarget `json:"target"`
	Amount int            `json:"amount"`
}

// ResolveConditionalDisplay resolves a set of conditional-effect rows (an active
// stance) for display: buckets by target identity, runs the same per-bonusType
// resolution, and returns only the surviving {target, amount} rows.
func ResolveConditionalDisplay(effects []ConditionalDisplayInput) []ConditionalDisplayRow {
	order := []string{}
	targets := map[string]ModifierTarget{}
	byKey := map[string][]Contribution{}
	for _, e := range effects {
		key := targetKey(e.Target)
		if _, ok := byKey[key]; !ok {
			order = append(order, key)
			targets[key] = e.Target
		}
		byKey[key] = append(byKey[key], Contribution{BonusType: e.BonusType, Amount: e.Amount})
	}
	kept := []ConditionalDisplayRow{}
	for _, key := range order {
		for _, c := range resolveStack(byKey[key]).Contributions {
			kept = append(kept, ConditionalDisplayRow{Target: targets[key], Amount: c.Amount})
		}
	}
	return kept
}

// ─── computeItemEffects ───────────────────────────────────────────────

// ComputeItemEffects folds a set of ActiveItems into resolved ItemEffects. A
// pre-pass collects flags first (so flagOff conditions don't depend on item
// order), then the main pass buckets unconditional modifiers by target and
// defers conditional opt-ins to the conditional list.
func ComputeItemEffects(items []ActiveItem) ItemEffects {
	order := []string{}
	buckets := map[string][]Contribution{}
	flags := map[string]bool{}
	conditional := []ConditionalEffect{}

	// Pre-pass: collect flags from every equipped item first.
	for i := range items {
		item := items[i]
		if item.Equipped == nil {
			continue
		}
		for _, m := range item.Modifiers {
			if m.Target.K == "flag" && conditionMet(m, item.Equipped) {
				flags[m.Target.Name] = true
			}
		}
	}

	for i := range items {
		item := items[i]
		if item.Equipped == nil {
			continue
		}
		for _, m := range item.Modifiers {
			// flagOff: book-passive that switches off while the flag is set.
			if m.Condition != nil && m.Condition.C == "flagOff" && flags[m.Condition.Flag] {
				continue
			}
			if !isUnconditional(m) {
				ce := ConditionalEffect{
					Source:    item.Source,
					BonusType: m.BonusType,
					Amount:    m.Amount,
					Note:      firstNonEmpty(describeCondition(m), m.Note),
					Target:    m.Target,
				}
				if m.Condition != nil && m.Condition.C == "flagOn" {
					ce.Flag = m.Condition.Flag
				}
				conditional = append(conditional, ce)
				continue
			}
			if !conditionMet(m, item.Equipped) {
				continue
			}
			if m.Target.K == "flag" {
				flags[m.Target.Name] = true
				continue
			}
			key := targetKey(m.Target)
			if _, ok := buckets[key]; !ok {
				order = append(order, key)
			}
			c := Contribution{Source: item.Source, BonusType: m.BonusType, Amount: m.Amount}
			if m.Note != "" {
				c.Note = m.Note
			}
			buckets[key] = append(buckets[key], c)
		}
	}

	byTarget := map[string]AggregatedStat{}
	for _, key := range order {
		byTarget[key] = resolveStack(buckets[key])
	}
	return ItemEffects{ByTarget: byTarget, Flags: flags, Conditional: conditional}
}

// firstNonEmpty mirrors the TS `describeCondition(m) || (m.note ?? ”)`.
func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// StatFor looks up a target's aggregated stat, defaulting to zeroed.
func StatFor(effects ItemEffects, target ModifierTarget) AggregatedStat {
	if stat, ok := effects.ByTarget[targetKey(target)]; ok {
		return stat
	}
	return AggregatedStat{Total: 0, Contributions: []Contribution{}}
}

// ConditionalID is the stable identifier for a conditional effect, used to
// persist which opt-ins are toggled on. Mirrors the TS join('::').
func ConditionalID(c ConditionalEffect) string {
	return c.Source + "::" +
		targetKey(c.Target) + "::" +
		c.Note + "::" +
		strconv.Itoa(c.Amount) + "::" +
		c.BonusType
}

// ApplyActiveConditionals folds the conditional effects whose ids are in
// activeIds back into byTarget, re-running non-stacking resolution per target.
// Flag conditionals are ignored (no UI for opt-in flags yet).
func ApplyActiveConditionals(effects ItemEffects, activeIds map[string]bool) ItemEffects {
	if len(activeIds) == 0 {
		return effects
	}
	buckets := map[string][]Contribution{}
	for key, agg := range effects.ByTarget {
		buckets[key] = append([]Contribution{}, agg.Contributions...)
	}
	// Unlike ComputeItemEffects above, this one needs no key ordering: it emits a
	// map, and per-key contribution order is already stable because each bucket
	// is a slice appended in source order. The tracking slice here was built and
	// then discarded (`_ = order`), which read as if ordering mattered.
	remaining := []ConditionalEffect{}
	for _, c := range effects.Conditional {
		if !activeIds[ConditionalID(c)] {
			remaining = append(remaining, c)
			continue
		}
		if c.Target.K == "flag" {
			continue
		}
		key := targetKey(c.Target)
		fold := Contribution{Source: c.Source + " (cond.)", BonusType: c.BonusType, Amount: c.Amount}
		if c.Note != "" {
			fold.Note = c.Note
		}
		buckets[key] = append(buckets[key], fold)
	}
	byTarget := map[string]AggregatedStat{}
	for key := range buckets {
		byTarget[key] = resolveStack(buckets[key])
	}
	return ItemEffects{ByTarget: byTarget, Flags: effects.Flags, Conditional: remaining}
}
