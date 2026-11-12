package engine

import (
	"strings"
	"testing"
)

// AS REDUÇÕES QUE O LIVRO DIZ CUMULATIVAS.
//
// "Reduções no custo de PM não são cumulativas" (p226), e o livro escreve as exceções no verbete: o
// cajado do poder é "cumulativo com Mestre em Escola" (p337), e o Foco em Magia é "cumulativo com outras
// reduções de custo" (p131). O livro não tem exemplo de três reduções juntas, e a leitura é do dono: as
// fontes ligadas somam e contam como UMA redução, que disputa com as outras; a que acumula com tudo entra
// por cima.

func schoolMaster(level int, school string) Character {
	hero := attributeSumHero("Arcanista", level,
		`["class.arcanista.caminho-mago","class.arcanista.especialista-em-escola","class.arcanista.mestre-em-escola"]`)
	hero.PowerChoices = `{"class.arcanista.especialista-em-escola":["` + school + `"],` +
		`"class.arcanista.mestre-em-escola":["` + school + `"]}`
	return hero
}

// O cajado do poder e o Mestre em Escola somam: −2 na evocação de quem dominou a evocação, e só o −1 do
// cajado nas outras escolas.
func TestThePowerStaffStacksWithTheSchoolMaster(t *testing.T) {
	world := BookRuleset(primeBook(t))
	mage := wielding(schoolMaster(8, "evocacao"), "cajado-do-poder")
	sheet := world.ComputeSheet(mage, nil)

	evocation := sheet.ForSpell("pmCost", SpellAt{Tradition: "arcana", School: "evocacao"})
	if evocation.Total != -2 {
		t.Errorf("na evocação, o cajado (−1) e o Mestre em Escola (−1) somam −2 (p337), e deu %d: %+v",
			evocation.Total, evocation.Contributions)
	}
	for _, c := range evocation.Contributions {
		if c.Dropped {
			t.Errorf("as duas reduções acumulam, e %s saiu riscada: %q", c.Source, c.Note)
		}
	}
	if got := sheet.ForSpell("pmCost", SpellAt{Tradition: "arcana", School: "ilusao"}).Total; got != -1 {
		t.Errorf("fora da escola do Mestre, só o cajado corta (−1), e deu %d", got)
	}
	if got := world.SpellPmCostFor(mage, 3, 0, nil, SpellAt{Tradition: "arcana", School: "evocacao"}); got != 1 {
		t.Errorf("a evocação de 3 PM custa %d na conjuração, e com −2 custa 1", got)
	}
}

// A dupla ligada conta como UMA redução, e disputa com as outras: contra um corte de −3, ela perde
// inteira, e as duas saem riscadas.
func TestTheLinkedCutsStillDisputeWithTheOthers(t *testing.T) {
	world := BookRuleset(primeBook(t))
	mage := wielding(schoolMaster(8, "evocacao"), "cajado-do-poder")
	mage.ActiveEffects = []ActiveEffectRow{{CatalogID: "corte-grande", Scope: "scene",
		Modifiers: `[{"target":{"k":"pmCost"},"amount":-3,"bonusType":"untyped"}]`}}

	reach := world.ComputeSheet(mage, nil).ForSpell("pmCost", SpellAt{Tradition: "arcana", School: "evocacao"})
	if reach.Total != -3 {
		t.Fatalf("a dupla (−2) perde para o corte de −3, e o total deu %d: %+v", reach.Total, reach.Contributions)
	}
	dropped := map[string]bool{}
	for _, c := range reach.Contributions {
		if c.Dropped {
			dropped[c.Source] = true
		}
	}
	if !dropped["cajado-do-poder"] || !dropped["Mestre em Escola"] {
		t.Errorf("o cajado e o Mestre perdem juntos, e as riscadas foram %v", dropped)
	}
}

// O Foco em Magia acumula com tudo: −1 na magia escolhida por cima da melhor redução, e ele aparece na
// conta da magia como parcela que entra.
func TestTheSpellFocusStacksOnTopOfTheBestCut(t *testing.T) {
	world := BookRuleset(primeBook(t))
	mage := attributeSumHero("Arcanista", 5, `["foco-em-magia"]`)
	mage.PowerChoices = `{"foco-em-magia":["bola-de-fogo"]}`
	mage.ActiveEffects = []ActiveEffectRow{{CatalogID: "corte", Scope: "scene",
		Modifiers: `[{"target":{"k":"pmCost"},"amount":-2,"bonusType":"untyped"}]`}}
	sheet := world.ComputeSheet(mage, nil)

	focused := sheet.ForSpell("pmCost", SpellAt{Tradition: "arcana", Spell: "bola-de-fogo"})
	if focused.Total != -3 {
		t.Errorf("a magia escolhida leva o corte (−2) e o Foco (−1), e deu %d: %+v", focused.Total, focused.Contributions)
	}
	var focus *BreakdownContribution
	for i, c := range focused.Contributions {
		if c.Source == "Foco em Magia" {
			focus = &focused.Contributions[i]
		}
	}
	if focus == nil || focus.Dropped || focus.Amount != -1 {
		t.Errorf("o Foco em Magia devia estar na conta da magia, −1, entrando: %+v", focused.Contributions)
	}
	if got := sheet.ForSpell("pmCost", SpellAt{Tradition: "arcana", Spell: "relampago"}).Total; got != -2 {
		t.Errorf("a magia que não foi escolhida leva só o corte (−2), e deu %d", got)
	}
}

// "A CD para resistir a suas magias dessa escola aumenta em +2" (Especialista em Escola, p38).
func TestTheSchoolSpecialistRaisesTheDCOfItsSchool(t *testing.T) {
	world := BookRuleset(primeBook(t))
	sheet := world.ComputeSheet(schoolMaster(8, "evocacao"), nil)
	inSchool := sheet.ForSpell("spellDC", SpellAt{Tradition: "arcana", School: "evocacao"}).Total
	outside := sheet.ForSpell("spellDC", SpellAt{Tradition: "arcana", School: "ilusao"}).Total
	if inSchool-outside != 2 {
		t.Errorf("a CD da evocação devia ficar 2 acima da ilusão, e ficou %+d", inSchool-outside)
	}
}

// A arma Harmonizada corta o custo de uma HABILIDADE de ataque escolhida, usada com ela (p165) — não o
// das magias. O motor ainda não conta custo de habilidade, e o corte não pode cair nas magias.
func TestTheHarmonizedWeaponDoesNotCutSpells(t *testing.T) {
	world := BookRuleset(primeBook(t))
	sword, wielded := "espada-longa", "wielded"
	mage := attributeSumHero("Arcanista", 5, `[]`)
	mage.Items = []CharacterItem{{CatalogID: &sword, Name: "Espada longa", Quantity: 1, Equipped: &wielded,
		Improvements: `["melhoria-certeira","melhoria-harmonizada-arma"]`}}
	reach := world.ComputeSheet(mage, nil).ForSpell("pmCost", SpellAt{Tradition: "arcana"})
	for _, c := range reach.Contributions {
		if strings.Contains(c.Source, "Harmoniz") || strings.Contains(c.Note, "habilidade escolhida") {
			t.Errorf("a Harmonizada cortou o custo das magias: %+v", c)
		}
	}
	if reach.Total != 0 {
		t.Errorf("sem redução de custo nenhuma, o custo das arcanas mudou em %d", reach.Total)
	}
}
