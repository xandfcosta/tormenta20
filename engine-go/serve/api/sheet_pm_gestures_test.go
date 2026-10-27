package api

import (
	"strings"
	"testing"
)

// O GESTO DE GASTAR PM (ALE-492): a regra "gaste N PM para…" vira gesto na
// aba Ações — cobra o PM, e o efeito é anúncio que o mestre aplica.

// p66 Durão: "Sempre que sofre dano, você pode gastar 3 PM para reduzir esse
// dano à metade."
func TestTheToughWarriorPaysThreePmToHalveTheDamage(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Durona", "Guerreiro", 3, 0, 0)

	if !strings.Contains(powerScreen(t, f, id), `aria-label="Usar Durão"`) {
		t.Fatal("a guerreira de nível 3 não tem o gesto do Durão")
	}
	before := pm(t, f, id)
	if refusal := powerCommand(t, f, id, "usa/class.guerreiro.durao", ""); refusal != "" {
		t.Fatalf("usar o Durão: %q", refusal)
	}
	if spent := before - pm(t, f, id); spent != 3 {
		t.Errorf("o Durão cobrou %d PM, e a p66 diz 3", spent)
	}
}

// p129 Trespassar: o poder GERAL também ganha gesto — 1 PM.
func TestAGeneralPowerThatSpendsPmHasAGesture(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Lâmina", "Guerreiro", 3, 0, 0)
	choiceCom(t, f, id, `["trespassar"]`, "[]")

	before := pm(t, f, id)
	if refusal := powerCommand(t, f, id, "usa/trespassar", ""); refusal != "" {
		t.Fatalf("usar Trespassar: %q", refusal)
	}
	if spent := before - pm(t, f, id); spent != 1 {
		t.Errorf("Trespassar cobrou %d PM, e a p129 diz 1", spent)
	}
}

// p80 Gritar Ordens: "uma quantidade de PM a sua escolha (limitado pelo seu
// Carisma)". O custo é VARIÁVEL: o gesto aparece, e não cobra um número que
// ninguém escolheu.
func TestAVariableCostGestureIsShownAndNotCharged(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Generala", "Nobre", 4, 0, 0)

	before := pm(t, f, id)
	if refusal := powerCommand(t, f, id, "usa/class.nobre.gritar-ordens", ""); !strings.Contains(refusal, "variável") {
		t.Errorf("usar Gritar Ordens não disse que o custo é variável: %q", refusal)
	}
	if pm(t, f, id) != before {
		t.Error("o gesto de custo variável cobrou PM")
	}
}
