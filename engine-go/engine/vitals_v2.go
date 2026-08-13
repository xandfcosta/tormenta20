package engine

// Catalog-driven vitals (PV/PM máximos) for the REAL engine (Inc.3) — the Go
// port of the front's TS vitals pipeline (t20-data `collectVitalGrants` +
// `frontVitalResolver` + `multiclassPvPool/MpPool`). Distinct from the MVP
// vitals.go, which is catalog-FREE (hardcoded table) for the server /sheet: this
// reuses the pool math there but collects the maxPv/maxPm grants from the PRIMED
// catalogs (same lookups as Inc.2's collection) and takes the REAL attrTotals the
// front computes over engine effects. See INC3-VITALS-PLAN.md.

// VitalContext is the normalized input both front consumers (optimisticLevelVitals,
// deriveDraftVitals) build — mirrors t20-data VitalGrantContext (the fields the
// grant collection reads). `RaceID` is the race NAME (getRace resolves by it, as
// Inc.2's collection does). AttrTotals are the FINAL totals (post item/race mods).
type VitalContext struct {
	Level              int                              `json:"level"`
	Classes            []ClassEntry                     `json:"classes"`
	RaceID             string                           `json:"raceId"`
	RaceAbilityChoices []string                         `json:"raceAbilityChoices"`
	PowerIDs           []string                         `json:"powerIds"`
	ClassChoices       map[string]ClassChoiceSelections `json:"classChoices"`
	GodPower           string                           `json:"godPower"`
	Origin             string                           `json:"origin"`
	OriginChoices      []string                         `json:"originChoices"`
	AttrTotals         map[string]int                   `json:"attrTotals"`
}

type VitalPools struct {
	PvMax int `json:"pvMax"`
	PmMax int `json:"pmMax"`
}

// ComputeVitals ports the front vitals pipeline: multiclass pools (p34-35) + the
// summed maxPv/maxPm grants, floored at 0. The pool helpers are shared with
// vitals.go (classVitalsTable/multiclass*).
func (c *Catalogs) ComputeVitals(ctx VitalContext) VitalPools {
	con := ctx.AttrTotals["constitution"]
	gpv, gpm := c.sumVitalGrants(ctx)
	return VitalPools{
		PvMax: max(0, multiclassPvPool(ctx.Classes, con)+gpv),
		PmMax: max(0, multiclassMpPool(ctx.Classes)+gpm),
	}
}

// sumVitalGrants ports collectVitalGrants: sum maxPv/maxPm over owned abilities,
// evaluating each scale, with the p225 dedupe (one attribute grant per
// target+attribute — no somar duas vezes a Sabedoria no PM).
func (c *Catalogs) sumVitalGrants(ctx VitalContext) (pv, pm int) {
	seen := map[string]bool{}
	for _, m := range c.vitalGrantMods(ctx) {
		if m.Target.K != "maxPv" && m.Target.K != "maxPm" {
			continue
		}
		if m.Scale != nil && m.Scale.Per == "attribute" {
			key := m.Target.K + ":" + m.Scale.Attribute
			if seen[key] {
				continue
			}
			seen[key] = true
		}
		amount := evalModifierScale(m.Amount, m.Scale, ctx.Level, ctx.AttrTotals)
		if m.Target.K == "maxPv" {
			pv += amount
		} else {
			pm += amount
		}
	}
	return pv, pm
}

// vitalGrantMods ports t20-data ownedModifiers: every modifier from the abilities
// the character owns — race, class powers (per class at ITS level), general
// powers, the god power, and origin benefits. NOT items/activeEffects (those
// aren't vital-grant sources). Filtered to vital targets by the caller.
func (c *Catalogs) vitalGrantMods(ctx VitalContext) []Modifier {
	out := []Modifier{}
	powers := toSet(ctx.PowerIDs)

	if ctx.RaceID != "" {
		if race := c.getRace(ctx.RaceID); race != nil {
			out = append(out, raceModifiers(race, toSet(ctx.RaceAbilityChoices))...)
		}
	}
	for _, ce := range ctx.Classes {
		for _, power := range c.ownedClassPowers(ce.ClassName, ce.Level, powers, ctx.ClassChoices[ce.ClassName]) {
			out = append(out, power.Modifiers...)
		}
	}
	for _, id := range ctx.PowerIDs {
		if gp := c.getGeneralPower(id); gp != nil {
			out = append(out, gp.Modifiers...)
		}
	}
	if ctx.GodPower != "" {
		if gp := c.grantedPowerByName(ctx.GodPower); gp != nil {
			out = append(out, gp.Modifiers...)
		}
	}
	if ctx.Origin != "" {
		if origin := c.getOrigin(ctx.Origin); origin != nil {
			out = append(out, originModifiers(origin, toSet(ctx.OriginChoices))...)
		}
	}
	return out
}

// evalModifierScale ports t20-data evalVitalScale for a Modifier's VitalScale
// (distinct from vitals.go's evalVitalScale over the MVP's vitalMod struct).
func evalModifierScale(amount int, scale *VitalScale, level int, attrTotals map[string]int) int {
	if scale == nil || scale.Per == "" || scale.Per == "flat" {
		return amount
	}
	switch scale.Per {
	case "level":
		return amount * level
	case "levelStep":
		steps := level / scale.Step
		if scale.Round == "up" && level%scale.Step != 0 {
			steps++
		}
		return amount * steps
	default: // attribute
		return amount * attrTotals[scale.Attribute]
	}
}
