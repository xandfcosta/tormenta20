import {
  type DisplayFact,
  getCatalogItem,
  SPELL_CATALOG,
} from '@tormenta20/t20-data'

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

/**
 * Display-only facts (RD, immunities, senses, …) for a spell-sourced effect —
 * so an applied buff can surface its non-computed sub-effects as reference
 * chips. Empty for item sources / unknown ids.
 */
export function effectSourceFacts(catalogId: string): DisplayFact[] {
  return SPELL_CATALOG[catalogId]?.buff?.facts ?? []
}
