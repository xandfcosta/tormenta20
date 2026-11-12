package engine

import (
	"strings"
	"testing"
)

// A CONTA MOSTRA O QUE NÃO ENTROU, E POR QUÊ.
//
// A ficha responde "por que 15?", e a parcela que o motor descartou é metade da resposta: sem ela, quem
// veste dois itens de Defesa vê um deles sumir, e quem tem duas reduções de PM vê as duas e um total que
// não fecha com elas.

// "Efeitos de itens [...] acumulam com os de outras fontes, mas não entre si" (p226). A armadura que
// perde fica na conta, marcada, dizendo com quem não empilhou.
func TestTheStackingLoserReachesTheSheetWithItsReason(t *testing.T) {
	stat := StatFor(resolveInWorld([]ActiveItem{
		vested("Armadura acolchoada", Modifier{Target: defenseTarget, Amount: 1, BonusType: "armor"}),
		vested("Armadura de couro", Modifier{Target: defenseTarget, Amount: 2, BonusType: "armor"}),
	}, nil), defenseTarget)

	if stat.Total != 2 {
		t.Fatalf("a Defesa das duas armaduras deu %d, e vale só a maior (+2)", stat.Total)
	}
	if len(stat.Dropped) != 1 {
		t.Fatalf("a armadura que perdeu devia vir à parte, uma, e vieram %d: %+v", len(stat.Dropped), stat.Dropped)
	}
	lost := stat.Dropped[0]
	if lost.Source != "Armadura acolchoada" || lost.Amount != 1 {
		t.Errorf("a parcela descartada é a acolchoada (+1), e veio %s %+d", lost.Source, lost.Amount)
	}
	if !strings.Contains(lost.Note, "não empilha com Armadura de couro") {
		t.Errorf("o motivo devia dizer com quem a acolchoada não empilhou: %q", lost.Note)
	}
}

// "Reduções no custo de PM não são cumulativas" (p226). A conta do custo mostra a redução que perdeu,
// marcada, e as que contam fecham no total.
func TestTheCostCutThatLosesIsShownDropped(t *testing.T) {
	world := BookRuleset(primeBook(t))
	hero := attributeSumHero("Arcanista", 5, `[]`)
	hero.ActiveEffects = []ActiveEffectRow{
		{CatalogID: "corte-maior", Scope: "scene", Modifiers: `[{"target":{"k":"pmCost"},"amount":-2,"bonusType":"untyped"}]`},
		{CatalogID: "corte-menor", Scope: "scene", Modifiers: `[{"target":{"k":"pmCost"},"amount":-1,"bonusType":"untyped"}]`},
	}
	reach := world.ComputeSheet(hero, nil).ForSpell("pmCost", SpellAt{Tradition: "arcana"})
	if reach.Total != -2 {
		t.Fatalf("o custo deu %d, e vale a maior redução (−2)", reach.Total)
	}
	counted, dropped := 0, 0
	for _, c := range reach.Contributions {
		if c.Dropped {
			dropped++
			if c.Amount != -1 || !strings.Contains(c.Note, "não se acumulam") {
				t.Errorf("a redução descartada é a de −1, com o motivo da p226, e veio %+v", c)
			}
			continue
		}
		counted += c.Amount
	}
	if dropped != 1 {
		t.Errorf("devia haver uma redução descartada, e há %d: %+v", dropped, reach.Contributions)
	}
	if counted != reach.Total {
		t.Errorf("as parcelas que contam somam %d, e o total é %d: a conta não fecha", counted, reach.Total)
	}
}

// O FATOR DIZ QUEM O PÔS. A aba Magias avisa "Alta Arcana corta este custo pela metade (p39)", e o nome
// vinha de uma varredura do acervo da ficha que lia o alvo do poder por conta própria — e que, quando o
// alvo ganhou tradição própria, passou a prometer o corte em toda tradição. A resolução já sabe quem pôs
// o fator, e só nas arcanas.
func TestTheCostFactorNamesWhoCutsIt(t *testing.T) {
	world := BookRuleset(primeBook(t))
	sheet := world.ComputeSheet(attributeSumHero("Arcanista", 20, `[]`), nil)
	arcane := sheet.ForSpell("pmCost", SpellAt{Tradition: "arcana"})
	if !arcane.Factor.isSet() || arcane.FactorSource != "class.arcanista.alta-arcana" {
		t.Errorf("nas arcanas, quem corta o custo é a Alta Arcana (p39), e veio fator %+v de %q", arcane.Factor, arcane.FactorSource)
	}
	divine := sheet.ForSpell("pmCost", SpellAt{Tradition: "divina"})
	if divine.Factor.isSet() || divine.FactorSource != "" {
		t.Errorf("a Alta Arcana corta só as arcanas, e numa divina veio fator %+v de %q", divine.Factor, divine.FactorSource)
	}
}
