package engine

// Pure, catalog-free rules the breakdown layer needs: training bonus, spell save
// CD + caster attribute, and the class RD tables. Distinct from the MVP skill
// model in skills.go — the real derive uses a level-based training bonus.

// TrainingBonusForLevel é o bônus de treino do nível (p114): +2, +4 no 7º e
// +6 no 15º. Exportado para o Eclético, que o empresta num teste (p45).
//
//	TrainingBonusForLevel(7) // 4
func TrainingBonusForLevel(level int) int {
	return trainingBonusForLevel(level)
}

// trainingBonusForLevel: +2 / +4 (7º) / +6 (15º).
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

// arcanistaPathAttribute — Caminho do Arcanista, livro p37: "Seu
// atributo-chave para lançar magias é definido pelo seu Caminho".
var arcanistaPathAttribute = map[string]string{
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
// O Feiticeiro lança com CARISMA, e é o mesmo atributo pelo qual o catálogo lhe
// soma PM — as duas metades da mesma frase do livro. O exemplo trabalhado da
// p173 é uma feiticeira.
//
// @example spellcastingAttributeFor(samiraFeiticeira, "Arcanista") // "charisma"
func spellcastingAttributeFor(ch Character, className string) string {
	base := classSpellcastingAttribute[className]
	if className != "Arcanista" {
		return base
	}
	path := parseClassChoices(ch.ClassChoices)["Arcanista"].Path
	if attr, ok := arcanistaPathAttribute[path]; ok {
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
