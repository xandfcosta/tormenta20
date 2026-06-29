import type { CatalogItem, ItemFamily } from '../types'
import type { ProficiencyCategory } from '../../proficiencies'
import { ANIMALS } from './animals'
import { APPAREL } from './apparel'
import { ARMORS } from './armors'
import { CONSUMABLES } from './consumables'
import { GEAR } from './gear'
import { IMPROVEMENTS } from './improvements'
import { MATERIALS } from './materials'
import { SHIELDS } from './shields'
import { VEHICLES } from './vehicles'
import { WEAPONS } from './weapons'

export {
  ANIMALS,
  APPAREL,
  ARMORS,
  CONSUMABLES,
  GEAR,
  IMPROVEMENTS,
  MATERIALS,
  SHIELDS,
  VEHICLES,
  WEAPONS,
}

export const CATALOG_ITEMS: CatalogItem[] = [
  ...WEAPONS,
  ...ARMORS,
  ...SHIELDS,
  ...APPAREL,
  ...GEAR,
  ...CONSUMABLES,
  ...IMPROVEMENTS,
  ...MATERIALS,
  ...ANIMALS,
  ...VEHICLES,
]

/**
 * Coarse family for a base item — used to gate which overlays may attach.
 */
export function familyFor(item: CatalogItem): ItemFamily {
  if (item.category.startsWith('weapon-')) return 'weapon'
  if (item.category.startsWith('armor-')) return 'armor'
  if (item.category === 'shield') return 'shield'
  return 'apparel'
}

const byId = new Map<string, CatalogItem>(
  CATALOG_ITEMS.map((it) => [it.id, it]),
)

export function getCatalogItem(id: string): CatalogItem | undefined {
  return byId.get(id)
}

export function isCatalogId(id: string): boolean {
  return byId.has(id)
}

/**
 * Proficiency category required to use this item without penalty. Returns
 * null for items that have no proficiency requirement (apparel, consumables,
 * tools, overlays). Used by the engine to inject -5 attack / lose-Dex
 * synthetic modifiers when an equipped item is non-proficient.
 */
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
