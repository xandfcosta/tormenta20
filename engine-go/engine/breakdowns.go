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
	Base       int  `json:"base"`
	ItemBonus  int  `json:"itemBonus"`
	Total      int  `json:"total"`
	DexApplied bool `json:"dexApplied"`
	// Defesa contra ataques corpo a corpo e à distância. Iguais ao Total na
	// maioria das fichas; separam-se quando algo é DIRECIONAL, hoje só o Caído
	// (p394: −5 contra corpo a corpo, +5 contra à distância).
	VsMelee       int                     `json:"vsMelee"`
	VsRanged      int                     `json:"vsRanged"`
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
	// SpellCdByAttribute is the spell save CD keyed by casting attribute (p173),
	// so a spell row can pick the CD for any of its applicable classes without
	// re-deriving (derived.ts computeBestCd).
	SpellCdByAttribute map[string]int `json:"spellCdByAttribute"`
	SpellDCBonus       TotalContribs  `json:"spellDCBonus"`
	PmCostMod          TotalContribs  `json:"pmCostMod"`
	// AttackAll/DamageAll are the {k:attack|damage, scope:all} globals (Fúria,
	// Instinto Selvagem…) — the combat HUD adds them onto every weapon/attack.
	AttackAll       TotalContribs `json:"attackAll"`
	DamageAll       TotalContribs `json:"damageAll"`
	DamageReduction RdBreakdown   `json:"damageReduction"`
	// TempHpFuria is tempHpFromPowers with furia active — the interesting branch
	// (Alma de Bronze). The base sheet (furia off) is always {0, []}.
	TempHpFuria TempHpBreakdown      `json:"tempHpFuria"`
	Expertises  []ExpertiseBreakdown `json:"expertises"`
	// Perícias em que o personagem FALHA AUTOMATICAMENTE — hoje só Reflexos, do
	// Indefeso (p394). É o motor quem responde isso, e não a UI reinterpretando
	// uma flag: a regra de quais condições implicam indefeso mora aqui.
	AutoFailExpertises []string `json:"autoFailExpertises"`
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
		Defense:            defenseBreakdown(ch, effects),
		Displacement:       displacementBreakdown(ch, effects),
		FlySpeed:           flySpeedTotal(effects),
		InventorySlots:     inventorySlotsTotal(ch, effects),
		Attributes:         attrs,
		PmLimit:            pmLimitBreakdown(ch, effects),
		BestBaseSpellCd:    bestBaseSpellCd(ch, effects),
		SpellCdByAttribute: spellCdByAttribute(ch, effects),
		SpellDCBonus:       spellDCBonus(effects),
		PmCostMod:          pmCostMod(effects),
		AttackAll:          totalContribsFor(effects, ModifierTarget{K: "attack", Scope: "all"}),
		DamageAll:          totalContribsFor(effects, ModifierTarget{K: "damage", Scope: "all"}),
		DamageReduction:    characterDamageReduction(ch, effects),
		TempHpFuria:        tempHpFromPowers(ch, effects, true),
		Expertises:         expertises,
		AutoFailExpertises: autoFailExpertises(effects),
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
	insolencia := insolenciaDefense(ch, e)
	melee := StatFor(e, ModifierTarget{K: "defense", Scope: "melee"})
	ranged := StatFor(e, ModifierTarget{K: "defense", Scope: "ranged"})
	total := base + stat.Total + insolencia
	contribs := stat.Contributions
	if insolencia > 0 {
		contribs = concatContribs(contribs, []Contribution{
			{Source: "Insolência (p47)", BonusType: "untyped", Amount: insolencia},
		})
	}
	return DefenseBreakdown{
		Base:          base,
		ItemBonus:     stat.Total + insolencia,
		Total:         total,
		DexApplied:    dexApplied,
		VsMelee:       total + melee.Total,
		VsRanged:      total + ranged.Total,
		Contributions: withNoteContribs(concatContribs(contribs, melee.Contributions, ranged.Contributions)),
	}
}

// insolenciaDefense — Bucaneiro p47: "Você soma seu Carisma na Defesa, limitado
// pelo seu nível. Esta habilidade exige liberdade de movimentos; você não pode
// usá-la se estiver de armadura pesada ou na condição imóvel."
//
// O teto é o nível NA CLASSE (p226, "Limites de Nível"), com exemplo trabalhado
// na mesma página: "um bucaneiro de 2º nível com Car 3 soma +2 na Defesa".
//
// Vive aqui, e não como modificador de catálogo, porque o motor de itens não
// avalia `scale` fora de PV/PM e não tem noção de TETO — a Insolência precisa
// das duas coisas. Estava no catálogo sem modificador nenhum: aparecia na ficha
// e não mexia na Defesa (ALE-115).
func insolenciaDefense(ch Character, e ItemEffects) int {
	if e.Flags["armadura-pesada"] || hasActiveCondition(ch, "imovel") {
		return 0
	}
	level := 0
	for _, entry := range ch.Classes {
		if entry.ClassName == "Bucaneiro" && entry.Level > level {
			level = entry.Level
		}
	}
	if level == 0 {
		return 0
	}
	return max(0, min(effectiveAttribute(ch, "charisma", e), level))
}

// hasActiveCondition reporta se a condição está ligada na ficha.
func hasActiveCondition(ch Character, id string) bool {
	for _, active := range parseStringArray(ch.ActiveConditions) {
		if active == id {
			return true
		}
	}
	return false
}

// displacementBreakdown ports derived.ts displacementTotal (floored at 0).
func displacementBreakdown(ch Character, e ItemEffects) ValueBreakdown {
	stat := StatFor(e, ModifierTarget{K: "displacement"})
	return ValueBreakdown{
		Base:          ch.Displacement,
		ItemBonus:     stat.Total,
		Total:         max(0, ch.Displacement+stat.Total),
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
	merged := resolveStack(concatContribs(stat.Contributions, allStat.Contributions, byAttrStat.Contributions))
	itemContribs := withNoteContribs(merged.Contributions)

	armorPenaltyApplied := 0
	if armorPenaltyExpertises[state.Name] {
		armorPenaltyApplied = StatFor(e, ModifierTarget{K: "armorPenalty"}).Total
		if armorPenaltyApplied != 0 {
			itemContribs = append(itemContribs, BreakdownContribution{Source: "Penalidade de armadura", Amount: armorPenaltyApplied})
		}
	}

	itemBonus := merged.Total
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

// autoFailExpertises lista as perícias em que o personagem falha
// AUTOMATICAMENTE. Hoje só Reflexos, pelo Indefeso (p394) e por tudo que o livro
// define COMO indefeso — paralisado, inconsciente, petrificado.
//
// Devolve sempre uma lista (nunca nil) para o JSON trazer `[]` em vez de `null`.
func autoFailExpertises(e ItemEffects) []string {
	out := []string{}
	if e.Flags[autoFailReflexosFlag] {
		out = append(out, "Reflexos")
	}
	return out
}
