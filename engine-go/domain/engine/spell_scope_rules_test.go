package engine

import "testing"

// O QUE VALE PARA UMA MAGIA: cada modificador de magia diz o que alcança (a tradição, a escola, o alcance),
// e a ficha e a conjuração perguntam a mesma coisa.

// "Reduções no custo de PM não são cumulativas" (p226). O mosaico "Custo PM" somava as duas e mostrava
// −3, enquanto a conjuração cobrava pela melhor (−2): a ficha prometia um preço que o servidor não dava.
func TestTheSheetAndTheCastAgreeThatCostCutsDoNotStack(t *testing.T) {
	world := BookRuleset(primeBook(t))
	hero := attributeSumHero("Arcanista", 5, `[]`)
	hero.ActiveEffects = []ActiveEffectRow{
		{CatalogID: "corte-maior", Scope: "scene", Modifiers: `[{"target":{"k":"pmCost"},"amount":-2,"bonusType":"untyped"}]`},
		{CatalogID: "corte-menor", Scope: "scene", Modifiers: `[{"target":{"k":"pmCost"},"amount":-1,"bonusType":"untyped"}]`},
	}
	if got := world.ComputeSheet(hero, nil).PmCostMod.Total; got != -2 {
		t.Errorf("o mosaico mostra %d, e as reduções não se acumulam: vale a melhor, −2", got)
	}
	if got := world.SpellPmCostFor(hero, 3, 0, nil, SpellAt{Tradition: "arcana"}); got != 1 {
		t.Errorf("a magia de 3 PM custa %d, e com a melhor redução (−2) custa 1", got)
	}
}

// O cajado do poder: "o custo de suas magias arcanas diminui em –1 PM [...] e a CD para resistir a elas
// aumenta em +2" (p337). Os dois valem só nas arcanas: o −1 não chegava a magia nenhuma, e o +2 subia
// também a CD do clérigo.
func TestThePowerStaffReachesOnlyTheArcaneSpells(t *testing.T) {
	world := BookRuleset(primeBook(t))
	arcanist := wielding(attributeSumHero("Arcanista", 5, `[]`), "cajado-do-poder")
	if got := world.SpellPmCostFor(arcanist, 3, 0, nil, SpellAt{Tradition: "arcana"}); got != 2 {
		t.Errorf("a arcana de 3 PM custa %d com o cajado do poder, e o livro tira 1", got)
	}
	cleric := wielding(attributeSumHero("Clérigo", 5, `[]`), "cajado-do-poder")
	plainCleric := attributeSumHero("Clérigo", 5, `[]`)
	if got := world.SpellPmCostFor(cleric, 3, 0, nil, SpellAt{Tradition: "divina"}); got != 3 {
		t.Errorf("a divina de 3 PM custa %d na mão do clérigo, e o cajado é só das arcanas", got)
	}
	withStaff := world.ComputeSheet(cleric, nil).ForSpell("spellDC", SpellAt{Tradition: "divina"}).Total
	without := world.ComputeSheet(plainCleric, nil).ForSpell("spellDC", SpellAt{Tradition: "divina"}).Total
	if withStaff != without {
		t.Errorf("o cajado do poder mudou a CD divina do clérigo em %+d, e ele é só das arcanas", withStaff-without)
	}
}

// O símbolo abençoado: "o custo de suas magias divinas diminui em –1 PM" (p345). Ninguém o lia.
func TestTheBlessedSymbolCutsTheDivineSpells(t *testing.T) {
	world := BookRuleset(primeBook(t))
	cleric := wearing(attributeSumHero("Clérigo", 5, `[]`), "simbolo-abencoado")
	if got := world.SpellPmCostFor(cleric, 3, 0, nil, SpellAt{Tradition: "divina"}); got != 2 {
		t.Errorf("a divina de 3 PM custa %d com o símbolo abençoado, e o livro tira 1", got)
	}
	arcanist := wearing(attributeSumHero("Arcanista", 5, `[]`), "simbolo-abencoado")
	if got := world.SpellPmCostFor(arcanist, 3, 0, nil, SpellAt{Tradition: "arcana"}); got != 3 {
		t.Errorf("a arcana de 3 PM custa %d com o símbolo, e ele é só das divinas", got)
	}
}
