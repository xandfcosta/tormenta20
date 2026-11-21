package book

import (
	"testing"

	"t20engine/domain/engine"
)

// O ALIADO COMO ALVO (ALE-647, fatia 3, decisão do dono): a Cura pelas Mãos, a Coragem e a Libertação podem ter um
// aliado como alvo (p82-83), e é nelas que a Caridade corta. A Aura e a Égide alcançam aliados numa ÁREA, sem alvo,
// e ficam de fora; os julgamentos que marcam um inimigo também.
func TestTheAllyTargetedPaladinAbilitiesAreMarked(t *testing.T) {
	for id, want := range map[string]bool{
		"class.paladino.cura-pelas-maos-1d8-1-pv": true,
		"class.paladino.julgamento-coragem":       true,
		"class.paladino.julgamento-libertacao":    true,
		"class.paladino.aura-sagrada":             false,
		"class.paladino.julgamento-zelo":          false,
	} {
		spec := ActivationOf(id, "")
		if spec == nil {
			t.Fatalf("%s não está no catálogo de ativações", id)
		}
		if spec.AllyTarget != want {
			t.Errorf("%s marca aliado como alvo = %v; want %v", id, spec.AllyTarget, want)
		}
	}
}

// A DECISÃO de pagar os degraus confere o PM contra o preço com o corte: a Coragem de 2 PM com 1 PM sobrando sai
// num aliado (2 – 1) e não no próprio paladino.
func TestTheScaledDecisionWeighsTheAllyCut(t *testing.T) {
	courage := ActivationOf("class.paladino.julgamento-coragem", "")
	if can, _ := ScaledDecision(*courage, 0, 0, 1, engine.AbilityCostRules{}); can {
		t.Error("com 1 PM a Coragem de 2 PM saiu no próprio paladino")
	}
	if can, why := ScaledDecision(*courage, 0, 0, 1, engine.AbilityCostRules{}, -1); !can {
		t.Errorf("com 1 PM a Coragem num aliado (–1) foi recusada: %s", why)
	}
}

// A CLASSE DA HABILIDADE se acha pelo id do poder ou, quando o id da ativação é outro, pelo NOME — como a ficha liga
// a ativação ao dono. A Coragem é `class.paladino.julgamento-coragem` na ativação e
// `class.paladino.julgamento-divino-coragem` no poder; sem o nome, o corte da classe nunca a alcançaria.
func TestTheActivationClassFallsBackToTheName(t *testing.T) {
	courage := ActivationOf("class.paladino.julgamento-coragem", "")
	if got := ActivationClass(*courage); got != "Paladino" {
		t.Errorf("a Coragem é da classe %q; want Paladino", got)
	}
}
