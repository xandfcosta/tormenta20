package api

import (
	"fmt"
	"strings"
	"testing"
)

// AS PATAS ARMADAS na Mochila (p137): com a Membros Extras e a Ambidestria ou o Estilo de Duas Armas, a
// arma leve se empunha "na pata", até duas.

func TestTheLegIsOfferedAndTakenOnlyWithTheStyle(t *testing.T) {
	f, id := fighterFixture(t)
	sword := itemSemeia(t, f, id, "espada-curta", "Espada curta", "")
	seedPowerChoices(t, f.s, id, `["membros-extras"]`, "{}")

	if strings.Contains(bagScreen(t, f, id), "Empunhar na pata") {
		t.Error("sem Ambidestria nem Estilo de Duas Armas, a folha não oferece a pata")
	}
	if refusal := bagCommand(t, f, id, fmt.Sprintf("itens/%d/equipa/leg", sword)); refusal == "" {
		t.Error("sem o estilo, a pata devia recusar a arma")
	}

	seedPowerChoices(t, f.s, id, `["membros-extras","estilo-de-duas-armas"]`, "{}")
	if !strings.Contains(bagScreen(t, f, id), "Empunhar na pata") {
		t.Error("com o estilo, a folha da arma leve oferece a pata")
	}
	if refusal := bagCommand(t, f, id, fmt.Sprintf("itens/%d/equipa/leg", sword)); refusal != "" {
		t.Fatalf("empunhar na pata foi recusado: %q", refusal)
	}
	if got := equipped(t, f, sword); got != "leg" {
		t.Errorf("a espada não foi para a pata: %q", got)
	}
	screen := bagScreen(t, f, id)
	if !strings.Contains(screen, "Patas") || !strings.Contains(screen, "Pata insetoide") || !strings.Contains(screen, "1 pata livre") {
		t.Error("a tira das patas não mostra a espada na pata e a pata livre")
	}
	if !strings.Contains(inUseOf(t, screen), "0 / 2") {
		t.Error("a pata não ocupa mão")
	}
}

func TestTheLegRefusesAHeavyWeaponAndAThirdOne(t *testing.T) {
	f, id := fighterFixture(t)
	seedPowerChoices(t, f.s, id, `["membros-extras","estilo-de-duas-armas"]`, "{}")
	long := itemSemeia(t, f, id, "espada-longa", "Espada longa", "")
	if refusal := bagCommand(t, f, id, fmt.Sprintf("itens/%d/equipa/leg", long)); !strings.Contains(refusal, "Espada longa") {
		t.Errorf("a pata empunha só arma leve, e a recusa nomeia a arma: %q", refusal)
	}
	itemSemeia(t, f, id, "adaga", "Adaga", "leg")
	itemSemeia(t, f, id, "adaga", "Adaga", "leg")
	third := itemSemeia(t, f, id, "espada-curta", "Espada curta", "")
	if refusal := bagCommand(t, f, id, fmt.Sprintf("itens/%d/equipa/leg", third)); !strings.Contains(refusal, "2 patas") {
		t.Errorf("a terceira arma passa do teto de duas patas: %q", refusal)
	}
}
