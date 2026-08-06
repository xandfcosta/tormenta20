package engine

import "strings"

// The magic / carrying / damage-reduction / temp-HP breakdowns of derived.ts,
// ported 1:1. Split from breakdowns.go by responsibility (spell + PM economy,
// carga, RD, PV temporário). See PORT-PLAN.md §3 (task #5).

type RdBreakdown struct {
	Total   int            `json:"total"`
	Sources []SourceAmount `json:"sources"`
}

type TempHpBreakdown struct {
	Total   int            `json:"total"`
	Sources []SourceAmount `json:"sources"`
}

// flySpeedTotal ports derived.ts: fly speed from effects (0 = can't fly).
func flySpeedTotal(e ItemEffects) int {
	return maxInt(0, StatFor(e, ModifierTarget{K: "flySpeed"}).Total)
}

// inventorySlotsTotal ports derived.ts (p141): 10 +2/Força (or −1/Força negativa)
// + item slot mods.
func inventorySlotsTotal(ch Character, e ItemEffects) int {
	effStr := effectiveAttribute(ch, "strength", e)
	base := 10 + 2*effStr
	if effStr < 0 {
		base = 10 + effStr
	}
	return base + StatFor(e, ModifierTarget{K: "inventorySlots"}).Total
}

// casterLevelForPmLimit ports derived.ts (p224): best spellcasting-class level,
// or character level for non-casters.
func casterLevelForPmLimit(ch Character) int {
	best, found := 0, false
	for _, entry := range ch.Classes {
		if classSpellcastingAttribute[entry.ClassName] == "" {
			continue
		}
		found = true
		best = maxInt(best, entry.Level)
	}
	if !found {
		return ch.Level
	}
	return best
}

// pmLimitBreakdown ports derived.ts pmLimitTotal: max(1, caster level) + mods.
func pmLimitBreakdown(ch Character, e ItemEffects) ValueBreakdown {
	base := maxInt(1, casterLevelForPmLimit(ch))
	stat := StatFor(e, ModifierTarget{K: "pmLimit"})
	return ValueBreakdown{
		Base:          base,
		ItemBonus:     stat.Total,
		Total:         base + stat.Total,
		Contributions: withNoteContribs(stat.Contributions),
	}
}

// bestBaseSpellCd ports derived.ts (p171): best CD over caster classes, using the
// FINAL key attribute. Nil for non-casters.
func bestBaseSpellCd(ch Character, e ItemEffects) *int {
	var best *int
	for _, entry := range ch.Classes {
		attr := classSpellcastingAttribute[entry.ClassName]
		if attr == "" {
			continue
		}
		dc := spellSaveDc(ch.Level, effectiveAttribute(ch, attr, e))
		if best == nil || dc > *best {
			d := dc
			best = &d
		}
	}
	return best
}

// spellCdByAttribute ports derived.ts computeBestCd's per-class CD: the spell
// save CD keyed by casting attribute (10 + ½ nível + FINAL attr mod), so a spell
// row can pick the CD for any of its applicable classes without re-deriving.
func spellCdByAttribute(ch Character, e ItemEffects) map[string]int {
	out := make(map[string]int, len(AttributeKeys))
	for _, a := range AttributeKeys {
		out[a] = spellSaveDc(ch.Level, effectiveAttribute(ch, a, e))
	}
	return out
}

// spellDCBonus ports derived.ts: item spellDC bonus + contributions.
func spellDCBonus(e ItemEffects) TotalContribs {
	return totalContribsFor(e, ModifierTarget{K: "spellDC"})
}

// pmCostMod ports derived.ts: item pmCost modifier + contributions.
func pmCostMod(e ItemEffects) TotalContribs {
	return totalContribsFor(e, ModifierTarget{K: "pmCost"})
}

// characterDamageReduction ports derived.ts: aggregate passive RD (Bárbaro p47,
// Guerreiro heavy, Cavaleiro Bastião + Especialização). General RD takes the max;
// Especialização stacks on top (explicit rule text).
func characterDamageReduction(ch Character, e ItemEffects) RdBreakdown {
	heavy := e.Flags["armadura-pesada"]
	chosen := parseChoiceSet(ch.ClassPowers)
	has := func(suffix string) bool {
		for _, id := range chosen.list {
			if id == suffix || strings.HasSuffix(id, "."+suffix) {
				return true
			}
		}
		return false
	}

	sources := []SourceAmount{}
	for _, entry := range ch.Classes {
		switch {
		case entry.ClassName == "Bárbaro":
			if rd := barbaroRdForLevel(entry.Level); rd > 0 {
				sources = append(sources, SourceAmount{"Bárbaro (p47)", rd})
			}
		case entry.ClassName == "Guerreiro" && heavy:
			if rd := guerreiroRdForLevel(entry.Level, heavy); rd > 0 {
				sources = append(sources, SourceAmount{"Guerreiro — armadura pesada", rd})
			}
		case entry.ClassName == "Cavaleiro" && heavy:
			if entry.Level >= 5 && has("caminho-bastiao") {
				sources = append(sources, SourceAmount{"Bastião — armadura pesada", cavaleiroBastiaoRd})
			}
			if has("especializacao-em-armadura") {
				sources = append(sources, SourceAmount{"Especialização em Armadura", 5})
			}
		}
	}
	if len(sources) == 0 {
		return RdBreakdown{Total: 0, Sources: sources}
	}

	especializacao, general := 0, 0
	for _, s := range sources {
		if s.Source == "Especialização em Armadura" {
			especializacao += s.Amount
			continue
		}
		general = maxInt(general, s.Amount)
	}
	return RdBreakdown{Total: general + especializacao, Sources: sources}
}

// tempHpFromPowers ports derived.ts: Alma de Bronze (Bárbaro p41) grants
// nível + Força temp PV while furia is active.
func tempHpFromPowers(ch Character, e ItemEffects, furiaActive bool) TempHpBreakdown {
	empty := TempHpBreakdown{Total: 0, Sources: []SourceAmount{}}
	if !furiaActive {
		return empty
	}
	chosen := parseChoiceSet(ch.ClassPowers)
	owns := false
	for _, id := range chosen.list {
		if id == "alma-de-bronze" || strings.HasSuffix(id, ".alma-de-bronze") {
			owns = true
			break
		}
	}
	if !owns {
		return empty
	}
	amount := ch.Level + effectiveAttribute(ch, "strength", e)
	return TempHpBreakdown{Total: amount, Sources: []SourceAmount{{"Alma de Bronze (Fúria, p41)", amount}}}
}
