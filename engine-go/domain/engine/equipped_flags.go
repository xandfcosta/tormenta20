package engine

import "sort"

// EquippedFlag is one always-on flag carried by an equipped item, with the
// item's name as provenance. Mirrors entities/character/effect-source.ts
// equippedItemFlagEffects; the pt-BR label is added on the front (ITEM_FLAG_LABEL).
type EquippedFlag struct {
	Flag   string `json:"flag"`
	Source string `json:"source"`
}

// ComputeEquippedFlags ports effect-source.ts equippedItemFlagEffects: for each
// equipped catalog item, resolves ONLY its base catalog modifiers (no melhorias/
// material/penalties/mirrors — the display just wants the item's own always-on
// flags) through the item engine so wear conditions (vested/wielded) apply, then
// emits the resulting flags with the item name as source. Flags are sorted per
// item for a deterministic order (matching ItemEffects.MarshalJSON).
func (c *Catalogs) ComputeEquippedFlags(items []CharacterItem) []EquippedFlag {
	out := []EquippedFlag{}
	for _, it := range items {
		if it.Equipped == nil || it.CatalogID == nil {
			continue
		}
		catalog := c.getCatalogItem(*it.CatalogID)
		if catalog == nil {
			continue
		}
		effects := ComputeItemEffects([]ActiveItem{
			{Source: it.Name, Equipped: it.Equipped, Modifiers: catalog.Modifiers},
		})
		flags := make([]string, 0, len(effects.Flags))
		for f := range effects.Flags {
			flags = append(flags, f)
		}
		sort.Strings(flags)
		for _, f := range flags {
			out = append(out, EquippedFlag{Flag: f, Source: it.Name})
		}
	}
	return out
}
