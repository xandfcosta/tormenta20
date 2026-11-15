package main

import (
	"context"
	"fmt"

	"t20engine/domain/sheet"
)

// O QUE A SEMENTE FAZ NOS ITENS DO ELENCO, pelos mesmos gestos da Mochila e da folha do item: as marcas, antes
// das sobreposições, e as escolhas, depois delas — a magia e a habilidade são de uma MELHORIA, que tem de
// existir antes.

// markItems marca as armas de categoria maior e as "feita por mim" (o Ferreiro e o Cano Raiado, p69).
func markItems(ctx context.Context, square casaDaSeed, id int64, items []sheet.ItemDTO, ch seedCharacter) error {
	for _, weapon := range ch.OversizedWeapons {
		line := lineOfCatalogItem(items, weapon)
		if line == 0 {
			return fmt.Errorf("o personagem %d não tem a arma %q para marcar como de categoria maior", id, weapon)
		}
		if err := square.ToggleOversizedWeapon(ctx, line); err != nil {
			return fmt.Errorf("marcar a %q do personagem %d: %w", weapon, id, err)
		}
	}
	for _, weapon := range ch.MadeByOwner {
		line := lineOfCatalogItem(items, weapon)
		if line == 0 {
			return fmt.Errorf("o personagem %d não tem a arma %q para marcar como feita por ele: "+
				"o `madeByOwner` aponta para um `create.items[].catalogId` que não está lá", id, weapon)
		}
		if err := square.ToggleItemMadeByOwner(ctx, line); err != nil {
			return fmt.Errorf("marcar a %q do personagem %d como feita por ele: %w", weapon, id, err)
		}
	}
	return nil
}

// chooseOnItems faz as escolhas que os itens e as melhorias pedem: o deus do símbolo, a magia do Harmonizado e a
// habilidade da arma Harmonizada.
func chooseOnItems(ctx context.Context, square casaDaSeed, id int64, items []sheet.ItemDTO, ch seedCharacter) error {
	for _, want := range ch.ItemGods {
		line := lineOfCatalogItem(items, want.Item)
		if line == 0 {
			return fmt.Errorf("o personagem %d não tem o item %q para dar o deus %q: "+
				"o `itemGods` aponta para um `create.items[].catalogId` que não está lá", id, want.Item, want.God)
		}
		if err := square.ChooseItemGod(ctx, line, want.God); err != nil {
			return fmt.Errorf("dar o deus %q ao %q do personagem %d: %w", want.God, want.Item, id, err)
		}
	}
	for _, want := range ch.ImprovementSpells {
		line := lineOfCatalogItem(items, want.Item)
		if line == 0 {
			return fmt.Errorf("o personagem %d não tem o item %q para dar a magia %q: "+
				"o `improvementSpells` aponta para um `create.items[].catalogId` que não está lá", id, want.Item, want.Spell)
		}
		if err := square.ChooseImprovementSpell(ctx, line, want.Improvement, want.Spell); err != nil {
			return fmt.Errorf("dar a magia %q ao %q do personagem %d: %w", want.Spell, want.Item, id, err)
		}
	}
	for _, want := range ch.ImprovementAbilities {
		line := lineOfCatalogItem(items, want.Item)
		if line == 0 {
			return fmt.Errorf("o personagem %d não tem o item %q para dar a habilidade %q: "+
				"o `improvementAbilities` aponta para um `create.items[].catalogId` que não está lá", id, want.Item, want.Ability)
		}
		if err := square.ChooseImprovementAbility(ctx, line, want.Improvement, want.Ability); err != nil {
			return fmt.Errorf("dar a habilidade %q ao %q do personagem %d: %w", want.Ability, want.Item, id, err)
		}
	}
	return nil
}
