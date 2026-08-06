package engine

// nonSelfStackingSources — item/magia/parceiro/ambiente take best-only within
// the source (modifier-stacking.ts NON_SELF_STACKING_SOURCES).
var nonSelfStackingSources = map[string]bool{
	"item": true, "magia": true, "parceiro": true, "ambiente": true,
}

// effectTargetKey mirrors character-sheet.ts effectTargetKey.
func effectTargetKey(t EffectTarget) string {
	switch t.K {
	case "attribute":
		return "attribute:" + t.Attribute
	case "save":
		return "save:" + t.Save
	case "skill":
		return "skill:" + t.Skill
	default: // defense | attack | damage
		return t.K
	}
}

type flatModifier struct {
	effectID   string
	effectName string
	source     string
	amount     int
}

// buffsResult is the aggregated buff totals + full contribution list.
type buffsResult struct {
	totals        map[string]int
	contributions []BuffContribution
}

// resolveBuffs processes every ActiveEffect, grouping by target key and
// applying the p226 stacking rules (character-sheet.ts resolveBuffs).
func resolveBuffs(effects []ActiveEffect) buffsResult {
	// Preserve first-seen target-key order for deterministic contributions.
	var order []string
	byTarget := map[string][]flatModifier{}
	for _, eff := range effects {
		for _, mod := range eff.Modifiers {
			key := effectTargetKey(mod.Target)
			if _, ok := byTarget[key]; !ok {
				order = append(order, key)
			}
			byTarget[key] = append(byTarget[key], flatModifier{
				effectID: eff.ID, effectName: eff.Name, source: eff.Source, amount: mod.Amount,
			})
		}
	}

	totals := map[string]int{}
	contributions := []BuffContribution{}
	for _, key := range order {
		total, contribs := stackModifiersForTarget(byTarget[key])
		totals[key] = total
		for _, c := range contribs {
			c.TargetKey = key
			contributions = append(contributions, c)
		}
	}
	return buffsResult{totals: totals, contributions: contributions}
}

// stackModifiersForTarget mirrors character-sheet.ts stackModifiersForTarget:
// collapse same-effectId duplicates (best-only), group by source, then sum
// (self-stacking) or take best-only (non-self-stacking) per source.
func stackModifiersForTarget(mods []flatModifier) (int, []BuffContribution) {
	// Collapse duplicate effectIds within a source (best-only).
	type keyT struct{ source, effectID string }
	var keyOrder []keyT
	perEffect := map[keyT]flatModifier{}
	for _, m := range mods {
		k := keyT{m.source, m.effectID}
		if existing, ok := perEffect[k]; !ok {
			keyOrder = append(keyOrder, k)
			perEffect[k] = m
		} else if m.amount > existing.amount {
			perEffect[k] = m
		}
	}

	// Group by source, preserving first-seen order.
	var srcOrder []string
	bySource := map[string][]flatModifier{}
	for _, k := range keyOrder {
		if _, ok := bySource[k.source]; !ok {
			srcOrder = append(srcOrder, k.source)
		}
		bySource[k.source] = append(bySource[k.source], perEffect[k])
	}

	total := 0
	var contributions []BuffContribution
	for _, source := range srcOrder {
		arr := bySource[source]
		nonStacking := nonSelfStackingSources[source]
		var applied int
		if nonStacking {
			applied = bestBonusOnly(arr)
		} else {
			applied = sumAmounts(arr)
		}
		total += applied
		for _, m := range arr {
			isApplied := true
			if nonStacking {
				isApplied = m.amount == applied
			}
			contributions = append(contributions, BuffContribution{
				EffectID: m.effectID, EffectName: m.effectName, Source: m.source,
				Amount: m.amount, Applied: isApplied,
			})
		}
	}
	return total, contributions
}

func bestBonusOnly(mods []flatModifier) int {
	best := 0
	for i, m := range mods {
		if i == 0 || m.amount > best {
			best = m.amount
		}
	}
	return best
}

func sumAmounts(mods []flatModifier) int {
	sum := 0
	for _, m := range mods {
		sum += m.amount
	}
	return sum
}
