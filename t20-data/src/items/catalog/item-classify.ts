import type { ProficiencyCategory } from '../../proficiencies'
import type { CatalogItem, ItemFamily } from '../types'

/**
 * Pure item classifiers — split out of `index.ts` so importing them does NOT
 * pull the `CATALOG_ITEMS` data (index.ts builds a `byId` Map over it at module
 * load). Consumers that only need `familyFor`/`requiredProficiency` (the sheet
 * derive, the overlay picker) can then tree-shake the ~44KB item catalog out of
 * the frontend bundle. Both operate purely on `item.category`.
 */

/** Coarse family gate for which improvements/materials attach to a base item. */
export function familyFor(item: CatalogItem): ItemFamily {
  if (item.category.startsWith('weapon-')) return 'weapon'
  if (item.category.startsWith('armor-')) return 'armor'
  if (item.category === 'shield') return 'shield'
  return 'apparel'
}

/** The proficiency an item requires to use without penalty, or null. */
export function requiredProficiency(
  item: CatalogItem,
): ProficiencyCategory | null {
  switch (item.category) {
    case 'weapon-simple':
      return 'armas-simples'
    case 'weapon-martial':
      return 'armas-marciais'
    case 'weapon-exotic':
      return 'armas-exoticas'
    case 'weapon-firearm':
      return 'armas-de-fogo'
    case 'armor-light':
      return 'armaduras-leves'
    case 'armor-heavy':
      return 'armaduras-pesadas'
    case 'shield':
      return 'escudos'
    default:
      return null
  }
}
