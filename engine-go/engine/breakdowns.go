package engine

// This file ports the breakdown layer of derived.ts (the `*Total` helpers) into
// a single ComputedSheetV2 the future WASM boundary exposes. Each breakdown is a
// faithful port that reads the resolved ItemEffects (slice 1) over the collected
// ActiveItems (slice 2). Movement/defense/attribute/expertise live here; the
// magic/RD/tempHp breakdowns are in breakdowns_magic.go. See PORT-PLAN.md §3 (task
// #5). Parity oracle: engine-go/parity/<slug>.json `sheetV2`.

// BreakdownContribution is one {source, amount, note?} row — the display-shaped
// contribution the TS breakdowns emit (no bonusType, unlike the resolution
// Contribution). Note is omitted when empty, matching the TS `...(note?{note}:{})`.
type BreakdownContribution struct {
	Source string `json:"source"`
	Amount int    `json:"amount"`
	Note   string `json:"note,omitempty"`
}

// SourceAmount is a {source, amount} row (attribute/RD/tempHp contributions,
// which the TS drops the note from).
type SourceAmount struct {
	Source string `json:"source"`
	Amount int    `json:"amount"`
}

type DefenseBreakdown struct {
	Base          int                     `json:"base"`
	ItemBonus     int                     `json:"itemBonus"`
	Total         int                     `json:"total"`
	DexApplied    bool                    `json:"dexApplied"`
	Contributions []BreakdownContribution `json:"contributions"`
}

// ValueBreakdown is the shared {base, itemBonus, total, contributions} shape
// (displacement, pmLimit).
type ValueBreakdown struct {
	Base          int                     `json:"base"`
	ItemBonus     int                     `json:"itemBonus"`
	Total         int                     `json:"total"`
	Contributions []BreakdownContribution `json:"contributions"`
}

// TotalContribs is the {total, contributions} shape (spellDCBonus, pmCostMod).
type TotalContribs struct {
	Total         int                     `json:"total"`
	Contributions []BreakdownContribution `json:"contributions"`
}

type AttributeBreakdown struct {
	Total         int            `json:"total"`
	Contributions []SourceAmount `json:"contributions"`
}

type ExpertiseBreakdown struct {
	Name                string                  `json:"name"`
	Attribute           string                  `json:"attribute"`
	Base                int                     `json:"base"`
	ItemBonus           int                     `json:"itemBonus"`
	Total               int                     `json:"total"`
	HalfLevel           int                     `json:"halfLevel"`
	AttrValue           int                     `json:"attrValue"`
	Training            int                     `json:"training"`
	ItemContributions   []BreakdownContribution `json:"itemContributions"`
	ArmorPenaltyApplied int                     `json:"armorPenaltyApplied"`
}

// ComputedSheetV2 aggregates every breakdown — the rich sheet the endgame WASM
// boundary returns (replacing the front's derived.ts breakdown calls).
type ComputedSheetV2 struct {
	Defense         DefenseBreakdown              `json:"defense"`
	Displacement    ValueBreakdown                `json:"displacement"`
	FlySpeed        int                           `json:"flySpeed"`
	InventorySlots  int                           `json:"inventorySlots"`
	Attributes      map[string]AttributeBreakdown `json:"attributes"`
	PmLimit         ValueBreakdown                `json:"pmLimit"`
	BestBaseSpellCd *int                          `json:"bestBaseSpellCd"`
	SpellDCBonus    TotalContribs                 `json:"spellDCBonus"`
	PmCostMod       TotalContribs                 `json:"pmCostMod"`
	// AttackAll/DamageAll are the {k:attack|damage, scope:all} globals (Fúria,
	// Instinto Selvagem…) — the combat HUD adds them onto every weapon/attack.
	AttackAll       TotalContribs `json:"attackAll"`
	DamageAll       TotalContribs `json:"damageAll"`
	DamageReduction RdBreakdown   `json:"damageReduction"`
	// TempHpFuria is tempHpFromPowers with furia active — the interesting branch
	// (Alma de Bronze). The base sheet (furia off) is always {0, []}.
	TempHpFuria TempHpBreakdown      `json:"tempHpFuria"`
	Expertises  []ExpertiseBreakdown `json:"expertises"`
}

// ComputeSheetV2 resolves the full breakdown sheet for a raw Character under the
// given active conditionals — the collection → resolution → breakdown pipeline.
func (c *Catalogs) ComputeSheetV2(ch Character, activeConditionals map[string]bool) ComputedSheetV2 {
	effects := ApplyActiveConditionals(ComputeItemEffects(c.ActiveItemsFor(ch)), activeConditionals)

	attrs := make(map[string]AttributeBreakdown, len(AttributeKeys))
	for _, a := range AttributeKeys {
		attrs[a] = attributeBreakdown(ch, a, effects)
	}
	expertises := []ExpertiseBreakdown{}
	for _, ex := range ch.Expertises {
		expertises = append(expertises, expertiseBreakdown(ch, ex, effects))
	}

	return ComputedSheetV2{
		Defense:         defenseBreakdown(ch, effects),
		Displacement:    displacementBreakdown(ch, effects),
		FlySpeed:        flySpeedTotal(effects),
		InventorySlots:  inventorySlotsTotal(ch, effects),
		Attributes:      attrs,
		PmLimit:         pmLimitBreakdown(ch, effects),
		BestBaseSpellCd: bestBaseSpellCd(ch, effects),
		SpellDCBonus:    spellDCBonus(effects),
		PmCostMod:       pmCostMod(effects),
		AttackAll:       totalContribsFor(effects, ModifierTarget{K: "attack", Scope: "all"}),
		DamageAll:       totalContribsFor(effects, ModifierTarget{K: "damage", Scope: "all"}),
		DamageReduction: characterDamageReduction(ch, effects),
		TempHpFuria:     tempHpFromPowers(ch, effects, true),
		Expertises:      expertises,
	}
}

// effectiveAttribute ports derived.ts attributeTotal: raw attribute + summed
// `attribute` modifiers.
func effectiveAttribute(ch Character, attr string, e ItemEffects) int {
	return ch.attributeValue(attr) + StatFor(e, ModifierTarget{K: "attribute", Name: attr}).Total
}

// defenseBreakdown ports derived.ts defenseTotal: 10 + Dex (unless blocked) + mods.
func defenseBreakdown(ch Character, e ItemEffects) DefenseBreakdown {
	stat := StatFor(e, ModifierTarget{K: "defense"})
	dexApplied := !e.Flags["cannot-apply-dex-to-defense"]
	base := 10
	if dexApplied {
		base += effectiveAttribute(ch, "dexterity", e)
	}
	return DefenseBreakdown{
		Base:          base,
		ItemBonus:     stat.Total,
		Total:         base + stat.Total,
		DexApplied:    dexApplied,
		Contributions: withNoteContribs(stat.Contributions),
	}
}

// displacementBreakdown ports derived.ts displacementTotal (floored at 0).
func displacementBreakdown(ch Character, e ItemEffects) ValueBreakdown {
	stat := StatFor(e, ModifierTarget{K: "displacement"})
	return ValueBreakdown{
		Base:          ch.Displacement,
		ItemBonus:     stat.Total,
		Total:         maxInt(0, ch.Displacement+stat.Total),
		Contributions: withNoteContribs(stat.Contributions),
	}
}

// attributeBreakdown ports derived.ts attributeTotal + attributeContributions
// ({source, amount}, note dropped).
func attributeBreakdown(ch Character, attr string, e ItemEffects) AttributeBreakdown {
	stat := StatFor(e, ModifierTarget{K: "attribute", Name: attr})
	return AttributeBreakdown{
		Total:         ch.attributeValue(attr) + stat.Total,
		Contributions: sourceAmountContribs(stat.Contributions),
	}
}

// armorPenaltyExpertises mirrors derived.ts ARMOR_PENALTY_EXPERTISES.
var armorPenaltyExpertises = map[string]bool{"Acrobacia": true, "Furtividade": true, "Ladinagem": true}

// expertiseBreakdown ports derived.ts expertiseTotalWithItems: ½ level + attr +
// training + item mods (expertise/expertiseAll/expertiseByAttribute) + armor penalty.
func expertiseBreakdown(ch Character, state CharacterExpertise, e ItemEffects) ExpertiseBreakdown {
	halfLevel := ch.Level / 2
	attrValue := effectiveAttribute(ch, state.Attribute, e)
	training := 0
	if state.Trained {
		training = trainingBonusForLevel(ch.Level)
	}
	base := halfLevel + attrValue + training

	stat := StatFor(e, ModifierTarget{K: "expertise", Name: state.Name})
	allStat := StatFor(e, ModifierTarget{K: "expertiseAll"})
	byAttrStat := StatFor(e, ModifierTarget{K: "expertiseByAttribute", Attribute: state.Attribute})
	itemContribs := withNoteContribs(concatContribs(stat.Contributions, allStat.Contributions, byAttrStat.Contributions))

	armorPenaltyApplied := 0
	if armorPenaltyExpertises[state.Name] {
		armorPenaltyApplied = StatFor(e, ModifierTarget{K: "armorPenalty"}).Total
		if armorPenaltyApplied != 0 {
			itemContribs = append(itemContribs, BreakdownContribution{Source: "Penalidade de armadura", Amount: armorPenaltyApplied})
		}
	}

	itemBonus := stat.Total + allStat.Total + byAttrStat.Total
	return ExpertiseBreakdown{
		Name:                state.Name,
		Attribute:           state.Attribute,
		Base:                base,
		ItemBonus:           itemBonus + armorPenaltyApplied,
		Total:               base + itemBonus + armorPenaltyApplied,
		HalfLevel:           halfLevel,
		AttrValue:           attrValue,
		Training:            training,
		ItemContributions:   itemContribs,
		ArmorPenaltyApplied: armorPenaltyApplied,
	}
}

// totalContribsFor is the {total, contributions} shape for a single target
// (spellDC, pmCost, attack/damage globals).
func totalContribsFor(e ItemEffects, target ModifierTarget) TotalContribs {
	stat := StatFor(e, target)
	return TotalContribs{Total: stat.Total, Contributions: withNoteContribs(stat.Contributions)}
}

// withNoteContribs maps resolution Contributions to display rows, keeping note.
func withNoteContribs(cs []Contribution) []BreakdownContribution {
	out := []BreakdownContribution{}
	for _, c := range cs {
		bc := BreakdownContribution{Source: c.Source, Amount: c.Amount}
		if c.Note != "" {
			bc.Note = c.Note
		}
		out = append(out, bc)
	}
	return out
}

// sourceAmountContribs maps to {source, amount}, dropping note (attributes).
func sourceAmountContribs(cs []Contribution) []SourceAmount {
	out := []SourceAmount{}
	for _, c := range cs {
		out = append(out, SourceAmount{Source: c.Source, Amount: c.Amount})
	}
	return out
}

func concatContribs(lists ...[]Contribution) []Contribution {
	out := []Contribution{}
	for _, l := range lists {
		out = append(out, l...)
	}
	return out
}
