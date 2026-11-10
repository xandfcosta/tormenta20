package api

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
)

// O MESTRE ENCANTA PELA FOLHA DO ITEM (decisão do dono). Encanto não se compra, se acha ou se fabrica
// (p333), e quem entrega o saque é o mestre: só ele vê o "Encantos" na folha, e só o pedido dele grava. O
// jogador lê os encantos na mesma folha e não tem o gesto. As regras do conjunto (família, teto, pré-requisito,
// a escolha da Anticriatura) são do `EnchantItem` e têm caso lá; aqui se prende QUEM e a LIGAÇÃO.

// enchantsAs pede o "Aplicar" do diálogo de encantos como `who`, e devolve a recusa.
func enchantsAs(t *testing.T, f sceneFixture, who, item int64, body string) string {
	t.Helper()
	path := fmt.Sprintf("/personagens/%d/itens/%d/encantos?tab=bag", f.charID, item)
	return bagRefusal(t, f.requests(t, who, http.MethodPost, path, body).Body.String())
}

func TestOnlyTheGMEnchantsAWeaponFromTheItemSheet(t *testing.T) {
	f := newSceneFixture(t)
	sword := itemSemeia(t, f, f.charID, "espada-longa", "Espada longa", "wielded")
	asked := `{"item_enchants":["encanto-anticriatura","encanto-venenosa"],"item_enchant_against":"morto-vivo"}`

	// O JOGADOR não tem o gesto na folha, e o pedido montado à mão é recusado sem gravar nada.
	playerSheet := renderNode(t, itemDialogOf(t, f.requests(t, f.player, http.MethodGet,
		fmt.Sprintf("/personagens/%d?tab=bag", f.charID), "").Body.String(), "Espada longa"))
	if strings.Contains(playerSheet, "Encantos de Espada longa") {
		t.Errorf("a folha do jogador oferece o diálogo de encantos:\n%s", playerSheet)
	}
	if refusal := enchantsAs(t, f, f.player, sword, asked); !strings.Contains(refusal, "mestre") {
		t.Errorf("o jogador encantou a própria arma, ou a recusa não diz quem encanta: %q", refusal)
	}
	if saved := itemEnchants(t, f, sword); len(saved) != 0 {
		t.Fatalf("a recusa ao jogador gravou %v", saved)
	}

	// O MESTRE tem o gesto e o diálogo, com a tabela da família da arma.
	gmScreen := f.requests(t, f.gm, http.MethodGet, fmt.Sprintf("/personagens/%d?tab=bag", f.charID), "").Body.String()
	if !strings.Contains(renderNode(t, itemDialogOf(t, gmScreen, "Espada longa")), "Encantos de Espada longa") {
		t.Error("a folha da espada não leva o mestre ao diálogo de encantos")
	}
	dialog := itemDialogOf(t, gmScreen, "Encantos de Espada longa")
	if dialog == nil {
		t.Fatal("o mestre não recebe o diálogo de encantos da espada")
	}
	written := renderNode(t, dialog)
	for _, want := range []string{"Anticriatura", "Flamejante", "Magnífica", "conta 2", "Contra", "morto-vivo"} {
		if !strings.Contains(written, want) {
			t.Errorf("o diálogo de encantos não escreve %q", want)
		}
	}
	if strings.Contains(written, "Sombrio") {
		t.Error("o diálogo da espada oferece um encanto de armadura (Tabela 8-10)")
	}

	if refusal := enchantsAs(t, f, f.gm, sword, asked); refusal != "" {
		t.Fatalf("o mestre foi recusado: %q", refusal)
	}
	if saved := itemEnchants(t, f, sword); !slices.Equal(saved, []string{"encanto-anticriatura", "encanto-venenosa"}) {
		t.Errorf("a espada ficou com %v; quer a Anticriatura e a Venenosa", saved)
	}
	if got := chosenKindOnTheSheet(t, f, f.charID, "Espada longa"); got != "morto-vivo" {
		t.Errorf("a escolha da Anticriatura ficou %q; quer morto-vivo", got)
	}
}

// A ARMADURA abre o mesmo diálogo, com a tabela dela (8-10, p338-339), e não a das armas.
func TestTheGMEnchantsAnArmorWithItsOwnTable(t *testing.T) {
	f := newSceneFixture(t)
	itemSemeia(t, f, f.charID, "armadura-couro", "Armadura de couro", "vested")
	screen := f.requests(t, f.gm, http.MethodGet, fmt.Sprintf("/personagens/%d?tab=bag", f.charID), "").Body.String()
	dialog := itemDialogOf(t, screen, "Encantos de Armadura de couro")
	if dialog == nil {
		t.Fatal("o mestre não recebe o diálogo de encantos da armadura")
	}
	written := renderNode(t, dialog)
	if !strings.Contains(written, "Sombrio") || strings.Contains(written, "Flamejante") {
		t.Errorf("o diálogo da armadura não é o da Tabela 8-10:\n%s", written)
	}
}
