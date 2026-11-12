package engine

import "testing"

// O orbe cristalino e o cajado arcano sobem o limite de PM das magias ARCANAS (p160). A magia do clérigo
// fica no limite dele.
func TestTheArcaneEsotericsRaiseOnlyTheArcanePmLimit(t *testing.T) {
	world := BookRuleset(primeBook(t))
	arcanist := wielding(attributeSumHero("Arcanista", 5, `[]`), "orbe-cristalino")
	plain := attributeSumHero("Arcanista", 5, `[]`)
	if got := world.SpellPmLimitFor(arcanist, nil, []string{"Arcanista"}, SpellAt{Tradition: "arcana"}) - world.SpellPmLimitFor(plain, nil, []string{"Arcanista"}, SpellAt{Tradition: "arcana"}); got != 1 {
		t.Errorf("o orbe mudou o limite do arcanista em %+d, e o livro dá +1", got)
	}
	cleric := wielding(attributeSumHero("Clérigo", 5, `[]`), "orbe-cristalino")
	plainCleric := attributeSumHero("Clérigo", 5, `[]`)
	if got := world.SpellPmLimitFor(cleric, nil, []string{"Clérigo"}, SpellAt{Tradition: "divina"}) - world.SpellPmLimitFor(plainCleric, nil, []string{"Clérigo"}, SpellAt{Tradition: "divina"}); got != 0 {
		t.Errorf("o orbe mudou o limite do clérigo em %+d, e ele é só das arcanas", got)
	}
}
