package api

import (
	"context"
	"strings"
	"testing"

	"t20engine/domain/sheet"
)

// O TIPO DA ANTICRIATURA É ESCOLHIDO AO ENCANTAR (p335, decisão do dono), e a ficha o lê de volta: é ele que
// diz contra quem o gasto de 2 PM vale. Encantar sem a escolha é recusado com as opções do livro, e a recusa
// não grava nada.
func TestTheAnticreatureKindIsChosenWhenTheWeaponIsEnchanted(t *testing.T) {
	f := newSceneFixture(t)
	sword := itemSemeia(t, f, f.charID, "espada-longa", "Espada longa", "wielded")
	plays := f.s.characterPlays()
	ctx := context.Background()

	err := plays.EnchantItem(ctx, sword, []string{"encanto-anticriatura"}, nil)
	if err == nil || !strings.Contains(err.Error(), "escolha o tipo de criatura") {
		t.Fatalf("a Anticriatura sem tipo foi gravada, ou a recusa não pede a escolha: %v", err)
	}
	if saved := itemEnchants(t, f, sword); len(saved) != 0 {
		t.Fatalf("a recusa gravou o encanto: %v", saved)
	}

	if err := plays.EnchantItem(ctx, sword, []string{"encanto-anticriatura"},
		map[string]string{"encanto-anticriatura": "morto-vivo"}); err != nil {
		t.Fatalf("encantar a espada contra morto-vivo: %v", err)
	}
	if got := chosenKindOnTheSheet(t, f, f.charID, "Espada longa"); got != "morto-vivo" {
		t.Errorf("a ficha leu a escolha da Anticriatura como %q; quer morto-vivo", got)
	}

	// A CÓPIA DA FICHA PARA A MESA leva a escolha na arma copiada, como leva o encanto.
	copied := cloneForCampaignTest(t, f)
	if got := chosenKindOnTheSheet(t, f, copied, "Espada longa"); got != "morto-vivo" {
		t.Errorf("a cópia da ficha leu a escolha da Anticriatura como %q; quer morto-vivo", got)
	}
}

// chosenKindOnTheSheet é a escolha da Anticriatura na arma de nome `weapon`, como a ficha a carrega.
func chosenKindOnTheSheet(t *testing.T, f sceneFixture, characterID int64, weapon string) string {
	t.Helper()
	ctx := context.Background()
	row, err := f.s.sceneCore().Queries().GetCharacter(ctx, characterID)
	if err != nil {
		t.Fatal(err)
	}
	dto, err := sheet.Load(ctx, f.s.sceneCore().Queries(), f.s.catalogs, row)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range dto.Items {
		if item.Name == weapon {
			return item.EnchantChoices["encanto-anticriatura"]
		}
	}
	t.Fatalf("a ficha %d não tem a arma %q", characterID, weapon)
	return ""
}
