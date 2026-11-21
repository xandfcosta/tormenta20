package sheetui

import (
	"testing"

	"t20engine/domain/book"
	"t20engine/domain/engine"
)

// O RÓTULO DO PREÇO na aba Poderes diz o que se cobra: a Cabeçada de 2 PM, Alquebrado, custa 3 (p394, ALE-647).
func TestTheCostLabelCarriesTheBrokenIncrease(t *testing.T) {
	headbutt := book.ActivationOf("class.lutador.cabecada", "")
	if headbutt == nil {
		t.Fatal("a Cabeçada não está no catálogo")
	}
	if got := writtenCost(*headbutt, nil, engine.AbilityCostRules{}); got != "LIVRE · 2 PM" {
		t.Fatalf("sem condição, o rótulo diz %q", got)
	}
	if got := writtenCost(*headbutt, nil, engine.AbilityCostRules{Increase: 1}); got != "LIVRE · 3 PM" {
		t.Errorf("Alquebrado, o rótulo diz %q; want LIVRE · 3 PM", got)
	}
}
