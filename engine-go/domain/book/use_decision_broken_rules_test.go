package book

import (
	"testing"

	"t20engine/domain/engine"
)

// A DECISÃO DE USAR confere o PM contra o preço que se cobra, e o Alquebrado sobe esse preço (p394, ALE-647): com
// 2 PM sobrando, a Cabeçada de 2 PM sai, e Alquebrado (3 PM) não.
func TestTheUseDecisionWeighsTheBrokenIncrease(t *testing.T) {
	headbutt := ActivationOf("class.lutador.cabecada", "")
	if headbutt == nil || ActivationPm(*headbutt) != 2 {
		t.Fatalf("a Cabeçada do catálogo não custa 2 PM: %+v", headbutt)
	}
	if _, why := UseDecision(*headbutt, UseContext{CurrentPM: 2}, Use{}); why != "" {
		t.Fatalf("com 2 PM a Cabeçada foi recusada: %s", why)
	}
	if _, why := UseDecision(*headbutt, UseContext{CurrentPM: 2, Cost: engine.AbilityCostRules{Increase: 1}}, Use{}); why != "PM insuficiente" {
		t.Errorf("Alquebrado com 2 PM a Cabeçada saiu (recusa %q); want recusada por PM insuficiente", why)
	}
}
