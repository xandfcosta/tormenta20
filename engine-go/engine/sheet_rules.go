package engine

// Pure, catalog-free rules the breakdown layer needs, ported 1:1 from t20-data:
// training bonus (expertises.ts), spell save CD + caster attribute (spells.ts),
// and the class RD tables (damage-reduction.ts). Distinct from the MVP skill
// model in skills.go — the real derive uses a level-based training bonus.

// trainingBonusForLevel ports expertises.ts: +2 / +4 (7º) / +6 (15º).
func trainingBonusForLevel(level int) int {
	if level >= 15 {
		return 6
	}
	if level >= 7 {
		return 4
	}
	return 2
}

// classSpellcastingAttribute ports spells.ts CLASS_SPELLCASTING_ATTRIBUTE — the
// key attribute per caster class ("" for non-casters).
var classSpellcastingAttribute = map[string]string{
	"Arcanista": "intelligence",
	"Bardo":     "charisma",
	"Clérigo":   "wisdom",
	"Druida":    "wisdom",
	"Paladino":  "wisdom",
}

// spellSaveDc ports spells.ts (p171): CD = 10 + ½ nível + mod do atributo-chave.
func spellSaveDc(casterLevel, keyAttributeMod int) int {
	return 10 + casterLevel/2 + keyAttributeMod
}

// barbaroRdForLevel ports damage-reduction.ts (p47): 0/2/4/6/8/10 at 5/8/11/14/17.
func barbaroRdForLevel(level int) int {
	switch {
	case level >= 17:
		return 10
	case level >= 14:
		return 8
	case level >= 11:
		return 6
	case level >= 8:
		return 4
	case level >= 5:
		return 2
	}
	return 0
}

// guerreiroRdForLevel ports damage-reduction.ts: same progression, heavy armor only.
func guerreiroRdForLevel(level int, heavyArmor bool) int {
	if !heavyArmor {
		return 0
	}
	return barbaroRdForLevel(level)
}

// cavaleiroBastiaoRd ports damage-reduction.ts CAVALEIRO_BASTIAO_RD.
const cavaleiroBastiaoRd = 5
