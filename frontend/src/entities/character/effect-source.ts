import { SPELL_CATALOG, getCatalogItem } from '@tormenta20/t20-data'

/**
 * Display name for an ActiveEffect's source id. Item-sourced effects resolve
 * through the item catalog; spell buffs (Phase-1 `SpellBuff`) through the spell
 * catalog; anything else falls back to the raw id. `getCatalogItem` returns
 * undefined for a spell id, which is exactly the case this bridges.
 */
export function effectSourceName(catalogId: string): string {
  return (
    getCatalogItem(catalogId)?.name ??
    SPELL_CATALOG[catalogId]?.name ??
    catalogId
  )
}
