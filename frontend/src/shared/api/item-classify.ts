import type { CatalogItem, ItemFamily } from './item-types'
import type { ProficiencyCategory } from './catalog-types'

/**
 * Classificadores puros de item, movidos do `t20-data` (ALE-109). Operam só
 * sobre `item.category`, sem tocar no catálogo — que hoje vem por HTTP.
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
