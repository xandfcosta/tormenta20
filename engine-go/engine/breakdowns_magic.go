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
	return max(0, StatFor(e, ModifierTarget{K: "flySpeed"}).Total)
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
		if !isSpellcastingClass(entry.ClassName) {
			continue
		}
		found = true
		best = max(best, entry.Level)
	}
	if !found {
		return ch.Level
	}
	return best
}

// pmLimitBreakdown is the per-CHARACTER summary behind the HUD's "Limite PM"
// tile: the best level across the character's spellcasting classes. It is a
// SUMMARY, not the cap for any given spell — for that, use SpellPmLimit below,
// which asks which class grants THAT spell (ALE-92).
func pmLimitBreakdown(ch Character, e ItemEffects) ValueBreakdown {
	base := max(1, casterLevelForPmLimit(ch))
	stat := StatFor(e, ModifierTarget{K: "pmLimit"})
	return ValueBreakdown{
		Base:          base,
		ItemBonus:     stat.Total,
		Total:         base + stat.Total,
		Contributions: withNoteContribs(stat.Contributions),
	}
}

// bestBaseSpellCd is the best CD across the character's caster classes — p173:
// 10 + metade do nível DO PERSONAGEM + atributo-chave. Uses the FINAL attribute,
// so race and item bonuses count, and resolves the Arcanista's key attribute
// through its Caminho. Nil for non-casters.
func bestBaseSpellCd(ch Character, e ItemEffects) *int {
	var best *int
	for _, entry := range ch.Classes {
		attr := spellcastingAttributeFor(ch, entry.ClassName)
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

// characterDamageReduction aggregates the character's passive damage reduction.
//
// Book sources, all verified (ALE-111):
//   - Bárbaro, p42: passive, RD 2 at 5th, +2 every three levels, cap 10 at 17th.
//   - Cavaleiro "Bastião", p55: chosen path at 5th, RD 5 with heavy armour.
//   - "Especialização em Armadura": a CHOSEN power requiring 12th level in the
//     class, RD 5 with heavy armour — Cavaleiro p54, Guerreiro p65. Both texts
//     say it is cumulative with Bastião, hence it adds on top instead of
//     competing.
//
// General RD takes the max (two sources of the same kind don't add);
// Especialização stacks on top, which is the book's own wording.
func characterDamageReduction(ch Character, e ItemEffects) RdBreakdown {
	heavy := e.Flags["armadura-pesada"]
	chosen := parseChoiceSet(ch.ClassPowers)
	// Class-qualified: the power id is `class.<classe>.<poder>`, and matching by
	// bare suffix would let a Guerreiro's pick satisfy the Cavaleiro's branch in
	// a multiclasse.
	hasPower := func(class, power string) bool {
		want := "class." + class + "." + power
		for _, id := range chosen.list {
			if id == want || id == power {
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
				sources = append(sources, SourceAmount{"Bárbaro (p42)", rd})
			}
		case entry.ClassName == "Guerreiro" && heavy:
			// p65: poder escolhido, 12º nível. NÃO é passivo nem escala — o motor
			// dava a progressão do Bárbaro a todo Guerreiro desde o 5º (ALE-111).
			if entry.Level >= especializacaoArmaduraLevel && hasPower("guerreiro", "especializacao-em-armadura") {
				sources = append(sources, SourceAmount{"Especialização em Armadura", especializacaoArmaduraRd})
			}
		case entry.ClassName == "Cavaleiro" && heavy:
			if entry.Level >= 5 && hasPower("cavaleiro", "caminho-bastiao") {
				sources = append(sources, SourceAmount{"Bastião — armadura pesada", cavaleiroBastiaoRd})
			}
			// p54: também 12º nível, e o texto diz explicitamente que é
			// cumulativa com o Bastião.
			if entry.Level >= especializacaoArmaduraLevel && hasPower("cavaleiro", "especializacao-em-armadura") {
				sources = append(sources, SourceAmount{"Especialização em Armadura", especializacaoArmaduraRd})
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
		general = max(general, s.Amount)
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

// SpellPmLimit is the p224 ceiling for ONE spell: the character's level in the
// CLASS that grants it, or the character level when the source is not a class
// (a race, an origin, a general power). `itemBonus` must be the RESOLVED
// `pmLimit` total (`ComputedSheetV2.PmLimit.ItemBonus`) — never a raw sum over
// equipped modifiers, which double-counts two `item`-typed bonuses and honours a
// `wielded` condition on a merely vested item.
//
// This is the authority the cast gate hangs on, and it is NOT what
// `pmLimitBreakdown` above returns: that one is the per-CHARACTER HUD summary
// ("best caster level"), a defensible summary for a tile but the wrong number
// for a specific spell. The two used to be separate implementations that
// disagreed — the sheet offered a cap the server then refused (ALE-92).
//
// @example SpellPmLimit(bardo7Arcanista1, 0, []string{"Arcanista"}) // 1
func SpellPmLimit(ch Character, itemBonus int, spellClasses []string) int {
	grants := map[string]bool{}
	for _, name := range spellClasses {
		grants[name] = true
	}
	best := 0
	for _, entry := range ch.Classes {
		if grants[entry.ClassName] && entry.Level > best {
			best = entry.Level
		}
	}
	if best == 0 {
		best = ch.Level // race/origin/power grant: character level
	}
	return max(1, best) + itemBonus
}

// SpellPmLimitFor resolves the item bonus off the character's own sheet and
// applies the p224 rule — the one call a transport (HTTP handler, WASM export)
// should make, so no caller re-derives the bonus its own way.
func (c *Catalogs) SpellPmLimitFor(ch Character, spellClasses []string) int {
	sheet := c.ComputeSheetV2(ch, map[string]bool{})
	return SpellPmLimit(ch, sheet.PmLimit.ItemBonus, spellClasses)
}
