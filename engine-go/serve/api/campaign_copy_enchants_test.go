package api

import (
	"context"
	"testing"
)

// O encanto é da arma, e a arma entra na mesa com a cópia da ficha. A cópia tem ids
// novos, e o encanto tem de cair na mesma arma: há duas adagas na mochila de propósito,
// porque pelo catálogo elas seriam a mesma.
func TestTheCampaignCopyKeepsTheEnchantsOnTheirWeapon(t *testing.T) {
	f := newSceneFixture(t)
	itemSemeia(t, f, f.charID, "adaga", "Adaga comum", "")
	enchanted := itemSemeia(t, f, f.charID, "adaga", "Adaga encantada", "wielded")
	if err := f.s.characterPlays().EnchantItem(context.Background(), enchanted, []string{"encanto-formidavel"}); err != nil {
		t.Fatalf("encantar a adaga: %v", err)
	}

	copied := cloneForCampaignTest(t, f)

	var name, enchant string
	if err := f.s.db.QueryRowContext(context.Background(), `
		SELECT i.name, e.enchantId FROM character_item_enchants e
		JOIN character_items i ON i.id = e.itemId WHERE i.characterId = ?`, copied).Scan(&name, &enchant); err != nil {
		t.Fatalf("a cópia não trouxe o encanto: %v", err)
	}
	if name != "Adaga encantada" || enchant != "encanto-formidavel" {
		t.Errorf("a cópia pôs %q em %q, want encanto-formidavel em Adaga encantada", enchant, name)
	}
}
