package engine

import "fmt"

// defenseBase is the fixed 10 from the book (p106).
const defenseBase = 10

var zeroAttrMods = map[string]int{
	"strength": 0, "dexterity": 0, "constitution": 0,
	"intelligence": 0, "wisdom": 0, "charisma": 0,
}

// ComputeCharacterSheet is the orchestrator — a faithful port of
// character-sheet.ts computeCharacterSheet. It never panics: bad input turns
// into warnings + conservative defaults.
func ComputeCharacterSheet(in *CharacterInput) *ComputedSheet {
	warnings := []string{}

	if in.Level < 1 || in.Level > 20 {
		warnings = append(warnings, fmt.Sprintf("nível fora do range T20 (1-20): %d", in.Level))
	}

	raceMods := resolveRaceMods(in, &warnings)
	attributes := map[string]AttributeComputed{}
	for _, key := range AttributeKeys {
		base := in.BaseAttributes[key]
		rm := raceMods[key]
		attributes[key] = AttributeComputed{Base: base, RaceMod: rm, Total: base + rm}
	}

	if loss := tormentaCarismaLoss(in); loss > 0 {
		car := attributes["charisma"]
		neg := -loss
		car.TormentaMod = &neg
		car.Total -= loss
		attributes["charisma"] = car
	}

	buffs := resolveBuffs(in.ActiveEffects)

	// Attribute buffs first — they feed every later derivation.
	for _, key := range AttributeKeys {
		if delta, ok := buffs.totals["attribute:"+key]; ok {
			a := attributes[key]
			a.Total += delta
			attributes[key] = a
		}
	}

	classEntries := in.Classes
	if len(classEntries) == 0 {
		classEntries = []ClassEntry{{ClassName: in.ClassName, Level: in.Level}}
	}

	vitals := computeVitals(in, attributes, classEntries, &warnings)

	defenseAttr := attributes["dexterity"].Total
	armorBonus := 0
	shieldBonus := 0
	if in.Equipment != nil {
		if in.Equipment.Armor != nil {
			armorBonus = in.Equipment.Armor.Defense
		}
		if in.Equipment.Shield != nil {
			shieldBonus = in.Equipment.Shield.Defense
		}
	}
	defenseBuff := buffs.totals["defense"]
	defense := Defense{
		Base:      defenseBase,
		Attribute: defenseAttr,
		Armor:     armorBonus,
		Shield:    shieldBonus,
		Total:     defenseBase + defenseAttr + armorBonus + shieldBonus + defenseBuff,
	}

	half := in.Level / 2
	saves := Saves{
		Fortitude: half + attributes["constitution"].Total + buffs.totals["save:fortitude"],
		Reflexos:  half + attributes["dexterity"].Total + buffs.totals["save:reflexos"],
		Vontade:   half + attributes["wisdom"].Total + buffs.totals["save:vontade"],
	}

	deslocamento, tamanho := resolveRaceMovement(in, &warnings)
	skills := computeSkills(in, attributes, &warnings, buffs.totals)
	attacks := computeAttacks(in, skills, attributes, buffs.totals)
	conditions := resolveConditions(in, &warnings)

	contribs := buffs.contributions
	if contribs == nil {
		contribs = []BuffContribution{}
	}
	return &ComputedSheet{
		Level:        in.Level,
		ClassName:    in.ClassName,
		Attributes:   attributes,
		Vitals:       vitals,
		Defense:      defense,
		Saves:        saves,
		Skills:       skills,
		Attacks:      attacks,
		Conditions:   conditions,
		Buffs:        BuffsSummary{Totals: buffs.totals, Contributions: contribs},
		Deslocamento: deslocamento,
		Tamanho:      tamanho,
		Warnings:     warnings,
	}
}

// raceAttrMods resolves one race's attribute mods; errors become warnings + 0.
func raceAttrMods(raceID string, floatingPicks []string, ascendencia string, warnings *[]string) map[string]int {
	if raceID == "" {
		return cloneAttrMods(zeroAttrMods)
	}
	r, ok := racas[raceID]
	if !ok {
		*warnings = append(*warnings, "raça desconhecida: "+raceID)
		return cloneAttrMods(zeroAttrMods)
	}
	mods, err := resolveAtributoMod(r, floatingPicks, ascendencia)
	if err != nil {
		*warnings = append(*warnings, fmt.Sprintf("mod racial inválido para %s: %s", raceID, err.Error()))
		return cloneAttrMods(zeroAttrMods)
	}
	out := cloneAttrMods(zeroAttrMods)
	for k, v := range mods {
		out[k] = v
	}
	return out
}

// resolveRaceMods sums primary + additional (homebrew) race attribute mods.
func resolveRaceMods(in *CharacterInput, warnings *[]string) map[string]int {
	primary := raceAttrMods(in.RaceID, in.RaceFloatingPicks, in.RaceAscendencia, warnings)
	if len(in.AdditionalRaces) == 0 {
		return primary
	}
	total := cloneAttrMods(primary)
	for _, add := range in.AdditionalRaces {
		m := raceAttrMods(add.RaceID, add.FloatingPicks, add.Ascendencia, warnings)
		for _, key := range AttributeKeys {
			total[key] += m[key]
		}
	}
	return total
}

func cloneAttrMods(src map[string]int) map[string]int {
	out := make(map[string]int, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func computeVitals(in *CharacterInput, attributes map[string]AttributeComputed, classEntries []ClassEntry, warnings *[]string) Vitals {
	if _, ok := classVitalsTable[in.ClassName]; !ok {
		*warnings = append(*warnings, "classe desconhecida: "+in.ClassName)
		return Vitals{}
	}
	con := attributes["constitution"].Total
	pvBase := multiclassPvPool(classEntries, con)
	pmBase := multiclassMpPool(classEntries)

	attrTotals := map[string]int{}
	for _, key := range AttributeKeys {
		attrTotals[key] = attributes[key].Total
	}
	grantPv, grantPm := collectVitalGrants(in, classEntries, in.Level, attrTotals)

	pvMax := maxInt(0, pvBase+grantPv)
	pmMax := maxInt(0, pmBase+grantPm)
	pvCurrent := pvMax
	if in.CurrentPv != nil {
		pvCurrent = *in.CurrentPv
	}
	pmCurrent := pmMax
	if in.CurrentPm != nil {
		pmCurrent = *in.CurrentPm
	}
	return Vitals{
		PvMax:     pvMax,
		PmMax:     pmMax,
		PvCurrent: minInt(pvCurrent, pvMax),
		PmCurrent: minInt(pmCurrent, pmMax),
	}
}

// deriveArmorPenalty mirrors character-sheet.ts deriveArmorPenalty.
func deriveArmorPenalty(in *CharacterInput, warnings *[]string) int {
	eq := in.Equipment
	if eq != nil && (eq.Armor != nil || eq.Shield != nil) {
		p := 0
		if eq.Armor != nil {
			p += eq.Armor.Penalty
		}
		if eq.Shield != nil {
			p += eq.Shield.Penalty
		}
		return p
	}
	ap := 0
	if in.ArmorPenalty != nil {
		ap = *in.ArmorPenalty
	}
	if ap < 0 {
		*warnings = append(*warnings, fmt.Sprintf("armorPenalty deve ser não-negativa, got %d", ap))
	}
	return maxInt(0, ap)
}

func computeSkills(in *CharacterInput, attributes map[string]AttributeComputed, warnings *[]string, buffTotals map[string]int) map[string]SkillComputed {
	trained := toSet(in.TrainedSkills)
	penalty := deriveArmorPenalty(in, warnings)
	deformSkills := toSet(deformidadeSkillIDs(in.Deformidade))

	for id := range trained {
		if _, ok := skillIndex[id]; !ok {
			*warnings = append(*warnings, "perícia treinada desconhecida: "+id)
		}
	}

	out := map[string]SkillComputed{}
	for _, id := range skillIDs {
		meta := skillIndex[id]
		isTrained := trained[id]
		attrValue := attributes[meta.keyAttribute].Total
		applies := meta.armorPenalty && penalty > 0
		baseTotal := skillValue(in.Level, attrValue, isTrained, applies, penalty)
		deformBonus := 0
		if deformSkills[id] {
			deformBonus = 2
		}
		appliedPenalty := 0
		if meta.armorPenalty {
			appliedPenalty = penalty
		}
		out[id] = SkillComputed{
			Total:               baseTotal + buffTotals["skill:"+id] + deformBonus,
			Trained:             isTrained,
			KeyAttribute:        meta.keyAttribute,
			CannotUse:           meta.trainedOnly && !isTrained,
			ArmorPenaltyApplied: appliedPenalty,
		}
	}
	return out
}

func computeAttacks(in *CharacterInput, skills map[string]SkillComputed, attributes map[string]AttributeComputed, buffTotals map[string]int) Attacks {
	if in.Equipment == nil {
		return Attacks{}
	}
	attackBuff := buffTotals["attack"]
	damageBuff := buffTotals["damage"]
	return Attacks{
		MainHand: computeAttackFor(in.Equipment.MainHand, skills, attributes, attackBuff, damageBuff),
		OffHand:  computeAttackFor(in.Equipment.OffHand, skills, attributes, attackBuff, damageBuff),
	}
}

func computeAttackFor(w *EquippedWeapon, skills map[string]SkillComputed, attributes map[string]AttributeComputed, attackBuff, damageBuff int) *ComputedAttack {
	if w == nil {
		return nil
	}
	skillID := "luta"
	if w.Purpose == "ranged" {
		skillID = "pontaria"
	}
	dmgAttr := attributes["strength"].Total
	if w.Purpose == "ranged" {
		dmgAttr = 0
	}
	return &ComputedAttack{
		WeaponName:           w.Name,
		Skill:                skillID,
		AttackTotal:          skills[skillID].Total + attackBuff,
		DamageDice:           w.Damage,
		DamageAttributeBonus: dmgAttr + damageBuff,
		DamageType:           w.DamageType,
		CritRange:            w.CritRange,
		CritMult:             w.CritMult,
		Hand:                 w.Hand,
		Purpose:              w.Purpose,
	}
}

// conditionsCatalog is only a summary lookup (conditions.ts). No benchmark
// payload exercises it, so unknown ids simply warn (matching the TS path).
var conditionsCatalog = map[string]ConditionSummary{}

func resolveConditions(in *CharacterInput, warnings *[]string) []ConditionSummary {
	out := []ConditionSummary{}
	for _, id := range in.ActiveConditions {
		cond, ok := conditionsCatalog[id]
		if !ok {
			*warnings = append(*warnings, "condição desconhecida: "+id)
			continue
		}
		out = append(out, cond)
	}
	return out
}

func resolveRaceMovement(in *CharacterInput, warnings *[]string) (int, string) {
	if in.RaceID == "" {
		return 9, "Médio"
	}
	r, ok := racas[in.RaceID]
	if !ok {
		*warnings = append(*warnings, "raça desconhecida ao resolver deslocamento: "+in.RaceID)
		return 9, "Médio"
	}
	return r.deslocamento, r.tamanho
}
