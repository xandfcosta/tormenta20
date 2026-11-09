package api

import (
	"fmt"
	"strings"
	"testing"
)

// A LINHA DO ATAQUE ADICIONAL: o que ela diz e o que ela oferece, como a linha do primeiro golpe.

// A linha diz de onde o ataque vem.
func TestTheAdditionalLineNamesItsSource(t *testing.T) {
	f, _, _ := styleOnTurn(t, "estilo-de-duas-armas", true, "bordao")
	surface := superficieDeAcoes(t, acoesDaMesa(t, f))
	if strings.Contains(surface, "Disparo Rápido") || !strings.Contains(surface, "Estilo de Duas Armas: depois de agredir") {
		t.Errorf("a linha do bordão não diz Estilo de Duas Armas")
	}
}

// O adicional com a adaga também pode ser arremesso, como o primeiro golpe.
func TestTheAdditionalAttackOffersTheOtherUse(t *testing.T) {
	f, id, goblin := styleOnTurn(t, "estilo-de-duas-armas", true, "espada-longa", "adaga")
	if surface := superficieDeAcoes(t, acoesDaMesa(t, f)); !strings.Contains(surface, "/acoes/ataque-extra/1/outro-uso") {
		t.Error("o adicional não oferece o arremesso")
	}
	rollsFromTheActions(t, f, id, "atacar/0", goblin)
	gmRules(t, f, "errou")
	if refusal := rollsFromTheActions(t, f, id, "ataque-extra/1/outro-uso", goblin); refusal != "" {
		t.Fatalf("o arremesso adicional foi recusado: %q", refusal)
	}
	if pa := pendingOf(t, f); pa == nil || pa.Melee {
		t.Errorf("o adicional devia ser à distância: %+v", pa)
	}
}

// A dose escolhida vale no adicional como vale no primeiro golpe.
func TestTheAdditionalAttackTakesTheChosenDose(t *testing.T) {
	f, goblin := doseAttackOnTurn(t)
	seedPowerChoices(t, f.s, f.charID, `["estilo-de-duas-armas"]`, "{}")
	weapon := weaponWithInjection(t, f)
	loaded(t, f, weapon, "acido", "Ácido")
	loaded(t, f, weapon, "fogo-alquimico", "Fogo alquímico")
	itemSemeia(t, f, f.charID, "adaga", "Adaga", "wielded")
	turnsOnStyle(t, f, f.charID, "estilo-de-duas-armas")
	if surface := superficieDeAcoes(t, acoesDaMesa(t, f)); !strings.Contains(surface, "Ataque adicional · ") {
		t.Fatal("as Ações não oferecem o adicional")
	}
	if refusal := rollsFromTheActions(t, f, f.charID, "atacar/1", goblin); refusal != "" {
		t.Fatalf("o primeiro golpe: %q", refusal)
	}
	gmRules(t, f, "errou")
	signals := fmt.Sprintf(`{"turn_target":%q,"dose_weapon":0,"dose_choice":"fogo-alquimico"}`, goblin)
	if refusal := sceneRefusal(f.posts(t, f.player, fmt.Sprintf("/personagens/%d/acoes/ataque-extra/0?embutida=1", f.charID), signals)); refusal != "" {
		t.Fatalf("o adicional: %q", refusal)
	}
	gmRules(t, f, "acertou")
	expectDoses(t, f, weapon, "acido")
}
