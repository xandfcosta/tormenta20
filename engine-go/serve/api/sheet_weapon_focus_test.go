package api

import (
	"testing"

	"t20engine/domain/sheet"
)

// p128 Foco em Arma: +2 no ataque com a arma escolhida, e só com ela.
func TestWeaponFocusAddsTwoToTheChosenWeaponAttack(t *testing.T) {
	f, id := fighterFixture(t)
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
