package engine

// classVitals mirrors CLASS_VITALS (class-vitals.ts).
type classVitals struct {
	pvInicial  int
	pvPerLevel int
	mpPerLevel int
}

var classVitalsTable = map[string]classVitals{
	"Arcanista": {8, 2, 6},
	"Bárbaro":   {24, 6, 3},
	"Bardo":     {12, 3, 4},
	"Bucaneiro": {16, 4, 3},
	"Caçador":   {16, 4, 4},
	"Cavaleiro": {20, 5, 3},
	"Clérigo":   {16, 4, 5},
	"Druida":    {16, 4, 4},
	"Guerreiro": {20, 5, 3},
	"Inventor":  {12, 3, 4},
	"Ladino":    {12, 3, 4},
	"Lutador":   {20, 5, 3},
	"Nobre":     {16, 4, 4},
	"Paladino":  {20, 5, 3},
}

// pvPoolWithCon mirrors class-vitals.ts pvPoolWithCon (p34 min-1 floor).
func pvPoolWithCon(v classVitals, level, con int) int {
	perLevel := maxInt(1, v.pvPerLevel+con)
	return v.pvInicial + con + (level-1)*perLevel
}

// multiclassPvPool: only the first class seeds its PV inicial (p34-35).
func multiclassPvPool(classes []ClassEntry, con int) int {
	if len(classes) == 0 {
		return 0
	}
	seed, ok := classVitalsTable[classes[0].ClassName]
	if !ok {
		return 0
	}
	pv := pvPoolWithCon(seed, classes[0].Level, con)
	for _, c := range classes[1:] {
		entry, ok := classVitalsTable[c.ClassName]
		if !ok {
			continue
		}
		pv += c.Level * maxInt(1, entry.pvPerLevel+con)
	}
	return pv
}

// multiclassMpPool sums each class's mpPerLevel*level (p35).
func multiclassMpPool(classes []ClassEntry) int {
	mp := 0
	for _, c := range classes {
		if entry, ok := classVitalsTable[c.ClassName]; ok {
			mp += entry.mpPerLevel * c.Level
		}
	}
	return mp
}

// ─── Vital grants (maxPv / maxPm) ─────────────────────────────────────
// Only maxPv/maxPm modifiers reach the sheet, so the port carries just the
// ~19 catalog entries that bear one (see vital-grants.ts collectVitalGrants).

// vitalMod is a maxPv/maxPm modifier with a scale (items/types.ts Modifier).
type vitalMod struct {
	target    string // "maxPv" | "maxPm"
	amount    int
	perKind   string // "flat" | "level" | "levelStep" | "attribute"
	step      int
	roundUp   bool
	attribute string
}

// raceVitalMods keyed by race DISPLAY name (getRace uses the abilities-catalog
// name). Only Anão (Duro como Pedra) and Elfo (Sangue Mágico) grant vitals.
var raceVitalMods = map[string][]vitalMod{
	"Anão": {
		{target: "maxPv", amount: 2, perKind: "flat"},
		{target: "maxPv", amount: 1, perKind: "level"},
	},
	"Elfo": {
		{target: "maxPm", amount: 1, perKind: "level"},
	},
}

// classVitalPower is one class power bearing a vital modifier, plus its
// ownership rule (classes/*.ts). Exactly one of the ownership fields is set.
type classVitalPower struct {
	className    string
	grantedAtLvl int    // > 0 ⇒ auto-granted at that class level
	electiveID   string // non-empty ⇒ owned when present in powerIds
	choiceField  string // "caminho"/"devoto" ⇒ grantedByChoice
	choiceValue  string
	mods         []vitalMod
}

var classVitalPowers = []classVitalPower{
	// Arcanista — Poder Mágico (elective, +1 PM/level) + Caminhos (grantedByChoice).
	{className: "Arcanista", electiveID: "class.arcanista.poder-magico",
		mods: []vitalMod{{target: "maxPm", amount: 1, perKind: "level"}}},
	{className: "Arcanista", choiceField: "caminho", choiceValue: "bruxo",
		mods: []vitalMod{{target: "maxPm", amount: 1, perKind: "attribute", attribute: "intelligence"}}},
	{className: "Arcanista", choiceField: "caminho", choiceValue: "feiticeiro",
		mods: []vitalMod{{target: "maxPm", amount: 1, perKind: "attribute", attribute: "charisma"}}},
	{className: "Arcanista", choiceField: "caminho", choiceValue: "mago",
		mods: []vitalMod{{target: "maxPm", amount: 1, perKind: "attribute", attribute: "intelligence"}}},
	// Bárbaro — Totem Espiritual (elective, +Sab no PM).
	{className: "Bárbaro", electiveID: "class.barbaro.totem-espiritual",
		mods: []vitalMod{{target: "maxPm", amount: 1, perKind: "attribute", attribute: "wisdom"}}},
	// Bardo — Magias 1º círculo (auto L1, +Car no PM).
	{className: "Bardo", grantedAtLvl: 1,
		mods: []vitalMod{{target: "maxPm", amount: 1, perKind: "attribute", attribute: "charisma"}}},
	// Caçador — Elo com a Natureza (elective, +Sab no PM).
	{className: "Caçador", electiveID: "class.cacador.elo-com-a-natureza",
		mods: []vitalMod{{target: "maxPm", amount: 1, perKind: "attribute", attribute: "wisdom"}}},
	// Clérigo — Magias 1º círculo (auto L1, +Sab no PM).
	{className: "Clérigo", grantedAtLvl: 1,
		mods: []vitalMod{{target: "maxPm", amount: 1, perKind: "attribute", attribute: "wisdom"}}},
	// Druida — Magias 1º círculo (auto L1, +Sab no PM).
	{className: "Druida", grantedAtLvl: 1,
		mods: []vitalMod{{target: "maxPm", amount: 1, perKind: "attribute", attribute: "wisdom"}}},
	// Lutador — Sarado (elective, +For no PV).
	{className: "Lutador", electiveID: "class.lutador.sarado",
		mods: []vitalMod{{target: "maxPv", amount: 1, perKind: "attribute", attribute: "strength"}}},
	// Paladino — Abençoado (auto L1, +Car no PM).
	{className: "Paladino", grantedAtLvl: 1,
		mods: []vitalMod{{target: "maxPm", amount: 1, perKind: "attribute", attribute: "charisma"}}},
}

// generalVitalMods keyed by general-power id (owned via powerIds).
var generalVitalMods = map[string][]vitalMod{
	"vitalidade":       {{target: "maxPv", amount: 1, perKind: "level"}},
	"vontade-de-ferro": {{target: "maxPm", amount: 1, perKind: "levelStep", step: 2, roundUp: false}},
}

// grantedVitalMods keyed by granted-power NAME (owned via godPower).
var grantedVitalMods = map[string][]vitalMod{
	"Bênção do Mana": {{target: "maxPm", amount: 1, perKind: "levelStep", step: 2, roundUp: true}},
}

// originVitalMods keyed by origin-benefit id (owned via originChoices).
var originVitalMods = map[string][]vitalMod{
	"poder-vontade-de-ferro": {{target: "maxPm", amount: 1, perKind: "levelStep", step: 2, roundUp: false}},
	"poder-vitalidade":       {{target: "maxPv", amount: 1, perKind: "level"}},
}

// evalVitalScale mirrors vital-grants.ts evalVitalScale.
func evalVitalScale(m vitalMod, level int, attrTotals map[string]int) int {
	switch m.perKind {
	case "", "flat":
		return m.amount
	case "level":
		return m.amount * level
	case "levelStep":
		steps := level / m.step
		if m.roundUp && level%m.step != 0 {
			steps++
		}
		return m.amount * steps
	default: // attribute
		return m.amount * attrTotals[m.attribute]
	}
}

// ownedVitalMods collects every maxPv/maxPm modifier the character owns —
// mirrors vital-grants.ts ownedModifiers, filtered to vital targets.
func ownedVitalMods(in *CharacterInput, classes []ClassEntry) []vitalMod {
	var out []vitalMod
	powers := toSet(in.PowerIDs)

	if in.RaceID != "" {
		if r, ok := racas[in.RaceID]; ok {
			out = append(out, raceVitalMods[r.name]...)
		}
	}
	for _, c := range classes {
		var choice *ClassChoice
		if in.ClassChoices != nil {
			if cc, ok := in.ClassChoices[c.ClassName]; ok {
				choice = &cc
			}
		}
		for _, p := range classVitalPowers {
			if p.className != c.ClassName {
				continue
			}
			if ownsClassVitalPower(p, c.Level, powers, choice) {
				out = append(out, p.mods...)
			}
		}
	}
	for id := range powers {
		out = append(out, generalVitalMods[id]...)
	}
	if in.GodPower != "" {
		out = append(out, grantedVitalMods[in.GodPower]...)
	}
	if in.Origin != "" {
		for _, id := range in.OriginChoices {
			out = append(out, originVitalMods[id]...)
		}
	}
	return out
}

func ownsClassVitalPower(p classVitalPower, classLevel int, powers map[string]bool, choice *ClassChoice) bool {
	if p.grantedAtLvl > 0 && p.grantedAtLvl <= classLevel {
		return true
	}
	if p.electiveID != "" && powers[p.electiveID] {
		return true
	}
	if p.choiceField != "" && choice != nil {
		if p.choiceField == "caminho" && choice.Caminho == p.choiceValue {
			return true
		}
		if p.choiceField == "devoto" && choice.Devoto == p.choiceValue {
			return true
		}
	}
	return false
}

// collectVitalGrants mirrors vital-grants.ts collectVitalGrants, including the
// p225 attribute-grant dedupe (one Sabedoria in PM, not two).
func collectVitalGrants(in *CharacterInput, classes []ClassEntry, level int, attrTotals map[string]int) (pv, pm int) {
	seen := map[string]bool{}
	for _, m := range ownedVitalMods(in, classes) {
		if m.perKind == "attribute" {
			key := m.target + ":" + m.attribute
			if seen[key] {
				continue
			}
			seen[key] = true
		}
		amount := evalVitalScale(m, level, attrTotals)
		if m.target == "maxPv" {
			pv += amount
		} else {
			pm += amount
		}
	}
	return pv, pm
}
