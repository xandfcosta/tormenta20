package api

import (
	"strings"
	"testing"
)

// A PATA ARMADA ataca pelo mesmo extra das patas (p137): a arma na pata e a pata livre dividem os 2 PM e os
// dois ataques da rodada.
func TestTheArmedLegSharesThePaymentWithTheFreeOne(t *testing.T) {
	f, id, goblin := barbarianOnTurn(t)
	seedPowerChoices(t, f.s, id, `["membros-extras","estilo-de-duas-armas"]`, "{}")
	itemSemeia(t, f, id, "espada-curta", "Espada curta", "leg")
	sword := naturalIndex(t, f, "Espada curta (pata)")
	limbs := naturalIndex(t, f, "Patas insetoides")
	rollsFromTheActions(t, f, id, "atacar/0", goblin)
	gmRules(t, f, "errou")
	before := mpOf(t, f.s, id)

	if refusal := extraStrike(t, f, id, sword, goblin); refusal != "" {
		t.Fatalf("a pata armada: %q", refusal)
	}
	if pa := pendingOf(t, f); pa == nil || pa.Weapon != "Espada curta (pata)" {
		t.Errorf("o extra devia ser com a espada na pata: %+v", pa)
	}
	if got := mpOf(t, f.s, id); got != before-2 {
		t.Errorf("a pata armada paga os 2 PM: %d virou %d", before, got)
	}
	gmRules(t, f, "errou")
	if refusal := extraStrike(t, f, id, limbs, goblin); refusal != "" {
		t.Fatalf("a pata livre: %q", refusal)
	}
	if got := mpOf(t, f.s, id); got != before-2 {
		t.Errorf("a pata livre não paga de novo: %d virou %d", before, got)
	}
	gmRules(t, f, "errou")
	if refusal := extraStrike(t, f, id, sword, goblin); !strings.Contains(refusal, "uma vez") {
		t.Errorf("o terceiro ataque das patas devia ser recusado, e veio %q", refusal)
	}
}
