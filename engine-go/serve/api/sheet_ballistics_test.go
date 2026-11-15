package api

import (
	"strings"
	"testing"
)

// A BALÍSTICA ESCOLHE UMA PROFICIÊNCIA: "armas marciais de ataque à distância ou com armas de fogo" (p69), escolhida
// no poder (decisão do dono). Sem escolha, ela é uma pendência; escolhida, o painel diz de onde vem só a dela.
func TestTheBallisticsPicksOneProficiency(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Balista", "Inventor", 5, 0, 0)
	seedMember(t, f.s, f.campaignID, id)
	saveHand(t, f, id, `["armas-simples"]`)
	seedPowerChoices(t, f.s, id, `["class.inventor.balistica"]`, `{}`)

	if step := choiceStepTitled(t, powerScreen(t, f, id), "Inventor · Balística"); step.Tag != "0 de 1" {
		t.Errorf("a Balística sem escolha não é pendência de uma: o passo diz %q", step.Tag)
	}
	if refusal := powerCommand(t, f, id, "escolha/class.inventor.balistica/armas-de-fogo", ""); refusal != "" {
		t.Fatalf("escolher as armas de fogo: %q", refusal)
	}
	panel := proficiencyPanel(t, f, id)
	if row := proficiencyRow(t, panel, "Armas de fogo"); !strings.Contains(row, "vem de Balística") {
		t.Errorf("a linha das armas de fogo não diz que vem da Balística:\n%s", row)
	}
	if row := proficiencyRow(t, panel, "Armas marciais"); strings.Contains(row, "Balística") {
		t.Errorf("a linha das marciais fala da Balística, que escolheu as de fogo:\n%s", row)
	}
}
