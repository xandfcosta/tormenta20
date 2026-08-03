import { getCatalogItem, HOMEBREW_VESTED_OK } from '@tormenta20/t20-data'
import type { CatalogItem } from '@tormenta20/t20-data'
import type { CharacterItem, EquippedSlot } from '@/shared/api/api'

export type EquipOption = { value: '' | EquippedSlot; label: string }

const NONE: EquipOption = { value: '', label: '—' }
const VESTED: EquipOption = { value: 'vested', label: 'Vestido' }
const ONE_HAND: EquipOption = { value: 'wielded', label: '1 mão' }
const TWO_HANDS: EquipOption = { value: 'wielded2', label: '2 mãos' }

export const ALL_EQUIP_OPTIONS: EquipOption[] = [
  NONE,
  VESTED,
  ONE_HAND,
  TWO_HANDS,
]

/**
 * Equip slots that make sense for this item, pre-validating what the server
 * accepts (a shield can't be "Vestido"). Custom items (no catalogId) keep the
 * full list — nothing is known about them.
 *
 * @example equipOptionsFor(escudoLeveRow) // [—, 1 mão]
 */
export function equipOptionsFor(item: CharacterItem): EquipOption[] {
  if (!item.catalogId) return ALL_EQUIP_OPTIONS
  const catalog = getCatalogItem(item.catalogId)
  if (!catalog) return ALL_EQUIP_OPTIONS
  return equipOptionsForCatalog(catalog)
}

/**
 * Same slot rules resolved straight from a catalog entry — the add-item
 * picker uses this before a CharacterItem row exists (and hides the equip
 * control entirely when only "—" remains, e.g. consumables).
 *
 * @example equipOptionsForCatalog(balsamo) // [—]
 */
export function equipOptionsForCatalog(catalog: CatalogItem): EquipOption[] {
  if (catalog.category === 'consumable' || catalog.category === 'meal')
    return [NONE]
  if (catalog.equip === 'vested') return [NONE, VESTED]
  if (catalog.equip === 'wielded') {
    // Homebrew registry: wearable esotéricos (Medalhão de prata) also offer
    // Vestido; the bonus still needs the Efeitos homebrew toggle.
    return HOMEBREW_VESTED_OK.has(catalog.id)
      ? [NONE, VESTED, ...handOptions(catalog)]
      : [NONE, ...handOptions(catalog)]
  }
  return [NONE, VESTED, ...handOptions(catalog)]
}

/**
 * "2 mãos" only when it's mandatory (`hands: 2`) or mechanically meaningful
 * (versátil weapons deal bigger damage two-handed, PDF p150). A plain
 * one-hand item gains nothing from occupying both hands.
 */
function handOptions(catalog: CatalogItem): EquipOption[] {
  if (catalog.hands === 2) return [TWO_HANDS]
  if (catalog.weapon?.traits.includes('versatil')) return [ONE_HAND, TWO_HANDS]
  return [ONE_HAND]
}
