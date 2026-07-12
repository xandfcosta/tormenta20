import { getCatalogItem } from '@tormenta20/t20-data'
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
  const out: string[] = []
  if (catalog.armor) out.push(`Defesa ${signed(catalog.armor.defense)}`)
  if (catalog.shield) out.push(`Defesa ${signed(catalog.shield.defense)}`)
  if (catalog.weapon) out.push(`Dano ${catalog.weapon.damage}`)
  for (const m of catalog.modifiers ?? []) {
    const label = describeModifierTarget(m.target)
    out.push(m.amount ? `${label} ${signed(m.amount)}` : label)
  }
  return out
}
