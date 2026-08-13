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

// classSpellcastingAttribute is the key attribute FIXED by the class entry
// ("" for non-casters): Bardo p44, Clérigo p57, Druida p61, and the Paladino's
// Orar power p83 ("Seu atributo-chave para esta magia é Sabedoria").
//
// The Arcanista's entry is only the FALLBACK — p37 says its key attribute is
// defined by the Caminho, so read it through `spellcastingAttributeFor`.
// Membership here doubles as the "is a spellcasting class" predicate, which is
// why the Arcanista still has a row.
var classSpellcastingAttribute = map[string]string{
	"Arcanista": "intelligence",
	"Bardo":     "charisma",
	"Clérigo":   "wisdom",
	"Druida":    "wisdom",
	"Paladino":  "wisdom",
}

// arcanistaCaminhoAttribute — Caminho do Arcanista, livro p37: "Seu
// atributo-chave para lançar magias é definido pelo seu Caminho".
var arcanistaCaminhoAttribute = map[string]string{
	"bruxo":      "intelligence",
	"feiticeiro": "charisma",
	"mago":       "intelligence",
}

// isSpellcastingClass reports whether the class casts magias at all — the
// predicate behind the per-character "Limite PM" summary.
func isSpellcastingClass(className string) bool {
	return classSpellcastingAttribute[className] != ""
}

// spellcastingAttributeFor resolves the key attribute ONE of the character's
// classes casts with, honouring the Caminho do Arcanista (p37). Returns "" for a
// class that does not cast.
//
// The Feiticeiro lançava com Inteligência na ficha enquanto o catálogo já lhe
// somava PM por Carisma — as duas metades da mesma frase do livro, uma certa e
// uma errada (ALE-113). O exemplo trabalhado da p173 é uma feiticeira.
//
// @example spellcastingAttributeFor(samiraFeiticeira, "Arcanista") // "charisma"
func spellcastingAttributeFor(ch Character, className string) string {
	base := classSpellcastingAttribute[className]
	if className != "Arcanista" {
		return base
	}
	caminho := parseClassChoices(ch.ClassChoices)["Arcanista"].Caminho
	if attr, ok := arcanistaCaminhoAttribute[caminho]; ok {
		return attr
	}
	// Ficha ainda sem a escolha obrigatória do 1º nível: Inteligência, que é o
	// atributo de dois dos três caminhos.
	return base
}

// spellSaveDc is the save CD against any spell — p173: "10 + metade do nível
// do personagem + atributo-chave da magia" (a p227 repete a fórmula para
// qualquer efeito de personagem).
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

// especializacaoArmaduraRd — "Especialização em Armadura": RD 5 com armadura
// pesada, poder ESCOLHIDO com pré-requisito de 12º nível na classe. Existe igual
// para Cavaleiro (p54) e Guerreiro (p65), e as duas descrições dizem que é
// cumulativa com o Bastião.
const especializacaoArmaduraRd = 5
const especializacaoArmaduraLevel = 12

// cavaleiroBastiaoRd — Caminho do Cavaleiro, livro p55: "Se estiver usando
// armadura pesada, você recebe redução de dano 5". Escolhido no 5º nível, valor
// fixo. Confirmado no livro; o Cavaleiro tem ainda outras duas fontes de RD 5
// que o motor NÃO modela (Especialização em Armadura, p54, cumulativa com esta;
// e Desprezar os Covardes, p54) — ver ALE-111.
const cavaleiroBastiaoRd = 5
