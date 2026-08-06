package engine

// skillMeta mirrors SKILL_INDEX entries (skill-index.ts).
type skillMeta struct {
	keyAttribute string
	trainedOnly  bool
	armorPenalty bool
}

// skillIDs is the canonical order of the 29 skills (SKILL_IDS).
var skillIDs = []string{
	"acrobacia", "adestramento", "atletismo", "atuacao", "cavalgar",
	"conhecimento", "cura", "diplomacia", "enganacao", "fortitude",
	"furtividade", "guerra", "iniciativa", "intimidacao", "intuicao",
	"investigacao", "jogatina", "ladinagem", "luta", "misticismo",
	"nobreza", "oficio", "percepcao", "pilotagem", "pontaria",
	"reflexos", "religiao", "sobrevivencia", "vontade",
}

var skillIndex = map[string]skillMeta{
	"acrobacia":     {"dexterity", false, true},
	"adestramento":  {"charisma", false, false},
	"atletismo":     {"strength", false, false},
	"atuacao":       {"charisma", false, false},
	"cavalgar":      {"dexterity", false, false},
	"conhecimento":  {"intelligence", true, false},
	"cura":          {"wisdom", false, false},
	"diplomacia":    {"charisma", false, false},
	"enganacao":     {"charisma", false, false},
	"fortitude":     {"constitution", false, false},
	"furtividade":   {"dexterity", false, true},
	"guerra":        {"intelligence", true, false},
	"iniciativa":    {"dexterity", false, false},
	"intimidacao":   {"charisma", false, false},
	"intuicao":      {"wisdom", false, false},
	"investigacao":  {"intelligence", false, false},
	"jogatina":      {"charisma", true, false},
	"ladinagem":     {"dexterity", true, true},
	"luta":          {"strength", false, false},
	"misticismo":    {"intelligence", true, false},
	"nobreza":       {"intelligence", true, false},
	"oficio":        {"intelligence", true, false},
	"percepcao":     {"wisdom", false, false},
	"pilotagem":     {"dexterity", true, false},
	"pontaria":      {"dexterity", false, false},
	"reflexos":      {"dexterity", false, false},
	"religiao":      {"wisdom", true, false},
	"sobrevivencia": {"wisdom", false, false},
	"vontade":       {"wisdom", false, false},
}

const trainedBonusBase = 2
const trainedHalfLevelMin = 1

// trainedComponent = ½ level (min +1) + 2 when trained, else 0 (skill-index.ts).
func trainedComponent(level int, trained bool) int {
	if !trained {
		return 0
	}
	return maxInt(trainedHalfLevelMin, level/2) + trainedBonusBase
}

// skillValue reproduces skillValue() in skill-index.ts.
func skillValue(level, attributeValue int, trained, armorPenaltyApplies bool, armorPenalty int) int {
	penalty := 0
	if armorPenaltyApplies {
		penalty = armorPenalty
	}
	return attributeValue + trainedComponent(level, trained) - penalty
}
