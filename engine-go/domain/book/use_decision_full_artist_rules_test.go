package book

import (
	"testing"

	"t20engine/domain/engine"
)

// A METADE DO ARTISTA COMPLETO NA DECISÃO DE USAR (p45, ALE-647): sob a Inspiração, o Fascinar em Massa de 2 PM
// custa 1, e o bardo com 1 PM o usa. A metade é da CLASSE: a mesma regra não alcança uma habilidade de outra.
func TestTheUseDecisionHalvesTheBardAbilityUnderInspiration(t *testing.T) {
	mass := ActivationOf("class.bardo.fascinar-em-massa", "")
	if mass == nil || ActivationPm(*mass) != 2 {
		t.Fatalf("o Fascinar em Massa do catálogo não é uma habilidade de 2 PM: %+v", mass)
	}
	if entry, _ := entryOf(*mass); entry.class != "Bardo" {
		t.Fatalf("o Fascinar em Massa do catálogo não é uma habilidade de bardo de 2 PM: %+v", mass)
	}
	halved := engine.AbilityCostRules{Factors: map[string]engine.Ratio{"Bardo": {Num: 1, Den: 2}}}
	if _, why := UseDecision(*mass, UseContext{CurrentPM: 1, Cost: halved}, Use{}); why != "" {
		t.Errorf("sob a Inspiração, com 1 PM o Fascinar em Massa foi recusado: %s", why)
	}
	if _, why := UseDecision(*mass, UseContext{CurrentPM: 1}, Use{}); why == "" {
		t.Error("sem a metade, com 1 PM o Fascinar em Massa de 2 PM saiu")
	}
	headbutt := ActivationOf("class.lutador.cabecada", "")
	if _, why := UseDecision(*headbutt, UseContext{CurrentPM: 1, Cost: halved}, Use{}); why == "" {
		t.Error("a metade do bardo alcançou a Cabeçada do lutador")
	}
}
