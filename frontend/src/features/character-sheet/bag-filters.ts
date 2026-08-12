import type { CharacterItem } from '@/shared/api/api'
import { getCatalogItem } from '@/shared/lib/catalog-cache'
import { normalize } from './normalize'

export type BagFilterKey = 'all' | 'weapons' | 'defense' | 'consumables' | 'other'

/** The bag's category chips, in the order they are shown. */
export const BAG_FILTERS: { key: BagFilterKey; label: string }[] = [
  { key: 'all', label: 'tudo' },
  { key: 'weapons', label: 'armas' },
  { key: 'defense', label: 'defesa' },
  { key: 'consumables', label: 'consumo' },
  { key: 'other', label: 'outros' },
]

const MATCHERS: Record<BagFilterKey, (category: string) => boolean> = {
  all: () => true,
  weapons: (c) => c.startsWith('weapon-'),
  defense: (c) => c.startsWith('armor-') || c === 'shield',
  consumables: (c) => c === 'consumable' || c === 'meal',
  other: (c) =>
    !c.startsWith('weapon-') &&
    !c.startsWith('armor-') &&
    !['shield', 'consumable', 'meal'].includes(c),
}

/**
 * Is this row shown under the given chip? A custom item has no catalog
 * category, so it counts as plain 'gear' — which lands it under "tudo" and
 * "outros" and never makes it vanish from the bag entirely.
 *
 * @example matchesBagFilter(escudoLeveRow, 'defense') // true
 */
export function matchesBagFilter(item: CharacterItem, key: BagFilterKey): boolean {
  if (key === 'all') return true
  const category = item.catalogId ? getCatalogItem(item.catalogId)?.category : undefined
  return MATCHERS[key](category ?? 'gear')
}

/**
 * The stowed rows a player is currently looking at: name search crossed with
 * the category chip. The search is accent-insensitive (like the Perícias one) —
 * "balsamo" has to find "Bálsamo restaurador".
 *
 * @example filterStowed(partition.stowed, 'espada', 'weapons')
 */
export function filterStowed(
  items: readonly CharacterItem[],
  search: string,
  key: BagFilterKey,
): CharacterItem[] {
  const query = normalize(search.trim())
  return items.filter(
    (item) =>
      (query === '' || normalize(item.name).includes(query)) && matchesBagFilter(item, key),
  )
}
