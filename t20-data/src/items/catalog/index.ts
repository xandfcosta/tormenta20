import type { CatalogItem } from '../types'
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

// Pure classifiers live in item-classify.ts so importing them doesn't pull the
// CATALOG_ITEMS data below (lets the frontend tree-shake the item catalog).
export { familyFor, requiredProficiency } from './item-classify'

const byId = new Map<string, CatalogItem>(
  CATALOG_ITEMS.map((it) => [it.id, it]),
)

export function getCatalogItem(id: string): CatalogItem | undefined {
  return byId.get(id)
}

export function isCatalogId(id: string): boolean {
  return byId.has(id)
}

