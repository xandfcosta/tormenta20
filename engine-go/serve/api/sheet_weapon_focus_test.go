package api

import (
	"strings"
	"testing"

	"t20engine/domain/sheet"
)

// p128 Foco em Arma: +2 no ataque com a arma escolhida, e só com ela.
func TestWeaponFocusAddsTwoToTheChosenWeaponAttack(t *testing.T) {
	f, id := fighterFixture(t)
	saveHand(t, f, id, `["armas-simples","armas-marciais"]`)
	itemSemeia(t, f, id, "espada-longa", "Espada longa", "wielded")
	itemSemeia(t, f, id, "adaga", "Adaga", "wielded2")
	before := attackByWeapon(t, f, id)
	if refusal := powerCommand(t, f, id, "escolha/foco-em-arma/espada-longa", ""); refusal != "" {
		t.Fatalf("escolher a espada longa no Foco em Arma foi recusado: %q", refusal)
	}
	after := attackByWeapon(t, f, id)
	if after["Espada longa"]-before["Espada longa"] != 2 {
		t.Errorf("a espada longa foi de %+d a %+d; o Foco em Arma dá +2", before["Espada longa"], after["Espada longa"])
	}
	if after["Adaga"] != before["Adaga"] {
		t.Errorf("a adaga foi de %+d a %+d, e não foi escolhida", before["Adaga"], after["Adaga"])
	}
}

func attackByWeapon(t *testing.T, f sceneFixture, id int64) map[string]int {
	t.Helper()
	dto, ec := loadedCharacter(t, f, id)
	out := map[string]int{}
	for _, card := range dto.Ruleset.ComputeWeaponCards(ec, sheet.ToStringSet(dto.Conditionals)) {
		out[card.Name] = card.Attack
	}
	return out
}

// "Pré-requisito: proficiência com a arma" (p128).
func TestWeaponFocusAsksForTheWeaponProficiency(t *testing.T) {
	f, id := fighterFixture(t)
	saveHand(t, f, id, `["armas-simples","armas-marciais"]`)
	if refusal := powerCommand(t, f, id, "escolha/foco-em-arma/machado-taurico", ""); !strings.Contains(refusal, "armas exóticas") {
		t.Errorf("o Foco no machado táurico, sem armas exóticas, deu %q; want a recusa que cita a proficiência", refusal)
	}
	if refusal := powerCommand(t, f, id, "escolha/foco-em-arma/espada-longa", ""); refusal != "" {
		t.Errorf("o Foco na espada longa, com armas marciais, foi recusado: %q", refusal)
	}
}

// O pré-requisito vale enquanto o poder fica: a proficiência marcada na mão
// que o Foco pede não se desmarca.
func TestWeaponFocusKeepsTheProficiencyItNeeds(t *testing.T) {
	f, id := fighterFixture(t)
	saveHand(t, f, id, `["armas-simples","armas-exoticas"]`)
	if refusal := powerCommand(t, f, id, "escolha/foco-em-arma/machado-taurico", ""); refusal != "" {
		t.Fatalf("o Foco no machado táurico, com armas exóticas, foi recusado: %q", refusal)
	}
	if refusal := toggleProficiencyOf(t, f, id, "armas-exoticas"); !strings.Contains(refusal, "Foco em Arma") {
		t.Errorf("desmarcar armas exóticas, que o Foco em Arma pede, deu %q; want a recusa que nomeia o poder", refusal)
	}
}
