package book

import (
	"slices"
	"testing"

	"t20engine/domain/engine"
)

// O PREÇO DO USO é um lugar só: o custo do catálogo com os degraus, pelas regras de custo da ficha, com o corte da
// Caridade quando o alvo é um aliado, e o piso de 1 PM. A decisão, o gesto e a tela pedem o número aqui.
func TestTheUsePriceFollowsTheSheetRules(t *testing.T) {
	broken := engine.AbilityCostRules{Increase: 1, AllyCuts: map[string]int{"Paladino": -1}}
	charity := engine.AbilityCostRules{AllyCuts: map[string]int{"Paladino": -1}}
	inspired := engine.AbilityCostRules{Factors: map[string]engine.Ratio{"Bardo": {Num: 1, Den: 2}}}
	courage := *ActivationOf("class.paladino.julgamento-coragem", "")
	zeal := *ActivationOf("class.paladino.julgamento-zelo", "")
	healing := *ActivationOf("class.paladino.cura-pelas-maos-1d8-1-pv", "")
	massCharm := *ActivationOf("class.bardo.fascinar-em-massa", "")
	for _, c := range []struct {
		name  string
		spec  Activation
		rules engine.AbilityCostRules
		use   Use
		want  int
	}{
		{"a Coragem sob o Alquebrado, no próprio paladino", courage, broken, Use{}, 3},
		{"a Coragem sob o Alquebrado, num aliado com a Caridade", courage, broken, Use{OnAlly: true}, 2},
		{"o Zelo não tem aliado como alvo, e o pedido não corta", zeal, charity, Use{OnAlly: true}, ActivationPm(zeal)},
		{"a Cura com 1 degrau num aliado", healing, charity, Use{Steps: 1, OnAlly: true}, 1},
		{"a Cura sem degrau num aliado fica no piso", healing, charity, Use{OnAlly: true}, 1},
		{"o Fascinar em Massa sob a Inspiração do Artista Completo", massCharm, inspired, Use{}, 1},
	} {
		if got := UsePrice(c.spec, c.rules, c.use); got != c.want {
			t.Errorf("%s: o preço é %d PM; want %d", c.name, got, c.want)
		}
	}
	if got := UsePrices(healing, charity, 2, true); !slices.Equal(got, []int{1, 1, 2}) {
		t.Errorf("os preços da Cura num aliado, de 0 a 2 degraus, são %v; want [1 1 2]", got)
	}
	if got := UsePrices(healing, engine.AbilityCostRules{}, 2, false); !slices.Equal(got, []int{1, 2, 3}) {
		t.Errorf("os preços da Cura no próprio paladino, de 0 a 2 degraus, são %v; want [1 2 3]", got)
	}
}

// A DECISÃO recebe o ALVO e não o corte: quem a chama não precisa saber que existe a Caridade. O pedido "num
// aliado" numa habilidade que não pode tê-lo é recusado aqui, e com 1 PM a Coragem sai num aliado.
func TestTheUseDecisionTakesTheTarget(t *testing.T) {
	charity := engine.AbilityCostRules{AllyCuts: map[string]int{"Paladino": -1}}
	courage := *ActivationOf("class.paladino.julgamento-coragem", "")
	if can, _ := UseDecision(courage, UseContext{CurrentPM: 1, Cost: charity}); can {
		t.Error("com 1 PM, a Coragem de 2 PM saiu no próprio paladino")
	}
	if can, why := UseDecision(courage, UseContext{CurrentPM: 1, Cost: charity, OnAlly: true}); !can {
		t.Errorf("com 1 PM, a Coragem num aliado foi recusada: %s", why)
	}
	zeal := *ActivationOf("class.paladino.julgamento-zelo", "")
	if can, why := UseDecision(zeal, UseContext{CurrentPM: 10, Cost: charity, OnAlly: true}); can || why != "não tem um aliado como alvo" {
		t.Errorf("o Zelo pedido num aliado: pode %v, motivo %q; want recusado por não ter um aliado como alvo", can, why)
	}
}
