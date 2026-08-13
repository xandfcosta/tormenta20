import { getCatalogItem } from '@/shared/lib/catalog-cache'
import type { CatalogItem } from '@/shared/api/item-types'
import type { Modifier } from '@/shared/api/item-types'
import type { CharacterItem } from '@/shared/api/api'
import { describeModifierTarget } from './item-describe'
import { signed } from './signed'

/**
 * Short labels of what an equipped item grants — its base combat stat plus
 * each granted modifier — for display on the item's Equipado slot. Custom
 * items (no `catalogId`) grant nothing lookup-able, so they return `[]`.
 *
 * @example equipBonuses(bandana) // ["Perícia Intimidação +1"]
 */
export function equipBonuses(item: CharacterItem): string[] {
  if (!item.catalogId) return []
  const catalog = getCatalogItem(item.catalogId)
  if (!catalog) return []
  return catalogEquipChips(catalog)
}

/**
 * Chip list for a catalog entry. Deduplicated so callers can key React
 * elements on the chip text itself without collisions.
 *
 * @example catalogEquipChips(getCatalogItem('armadura-couro')!) // ["Defesa +2"]
 */
export function catalogEquipChips(catalog: CatalogItem): string[] {
  const chips = [
    ...baseDefenseChip(catalog),
    ...(catalog.weapon ? [`Dano ${catalog.weapon.damage}`] : []),
    ...catalog.modifiers.map(modifierChip),
  ]
  return [...new Set(chips)]
}

/**
 * Base armor/shield Defense chip — skipped when the catalog already carries
 * a `defense` modifier of the same amount (armors and shields do), which
 * would render the chip twice ("Defesa +2 / Defesa +2").
 */
function baseDefenseChip(catalog: CatalogItem): string[] {
  const defense = catalog.armor?.defense ?? catalog.shield?.defense
  if (!defense) return []
  const duplicated = catalog.modifiers.some(
    (m) => m.target.k === 'defense' && m.amount === defense,
  )
  return duplicated ? [] : [`Defesa ${signed(defense)}`]
}

/** Flag targets are boolean — the chip is the label alone, never an amount. */
function modifierChip(m: Modifier): string {
  const label = describeModifierTarget(m.target)
  if (m.target.k === 'flag' || !m.amount) return label
  return `${label} ${signed(m.amount)}`
}
