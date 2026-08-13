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

// barbaroRdForLevel is the Bárbaro's Redução de Dano — livro p42: RD 2 no 5º
// nível, +2 a cada três níveis, teto de 10 no 17º. (A citação anterior dizia
// p47, que é a página do Bucaneiro.)
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

// guerreiroRdForLevel NÃO CORRESPONDE AO LIVRO — ver ALE-111.
//
// Ele dá ao Guerreiro a progressão do Bárbaro (RD 2 no 5º, +2 a cada três) com
// armadura pesada. O livro p65 tem outra coisa: "Especialização em Armadura.
// Você recebe redução de dano 5 se estiver usando uma armadura pesada.
// Pré-requisito: 12º nível de guerreiro" — um poder ESCOLHIDO, de 12º nível,
// valor fixo. Um Guerreiro de 5º a 11º não deveria ter RD nenhuma.
//
// Mantido como está porque corrigir muda o número de personagens já criados; a
// decisão está registrada. O teste fixa o comportamento atual de propósito.
func guerreiroRdForLevel(level int, heavyArmor bool) int {
	if !heavyArmor {
		return 0
	}
	return barbaroRdForLevel(level)
}

// cavaleiroBastiaoRd — Caminho do Cavaleiro, livro p55: "Se estiver usando
// armadura pesada, você recebe redução de dano 5". Escolhido no 5º nível, valor
// fixo. Confirmado no livro; o Cavaleiro tem ainda outras duas fontes de RD 5
// que o motor NÃO modela (Especialização em Armadura, p54, cumulativa com esta;
// e Desprezar os Covardes, p54) — ver ALE-111.
const cavaleiroBastiaoRd = 5
