import { useQuery } from '@tanstack/react-query'
import type { CatalogSpell } from '@tormenta20/t20-data'
import { spellCatalogQueryOptions } from './queries'

export type SpellCatalog = Record<string, CatalogSpell>

/**
 * The id-keyed spell catalog from the backend. `data` is undefined until the
 * (cached-forever) fetch resolves — consumers that used to read the synchronous
 * `SPELL_CATALOG` import must now tolerate a loading tick.
 *
 * @example const { data: spells } = useSpellCatalog(); spells?.[id]?.name
 */
export function useSpellCatalog() {
  return useQuery(spellCatalogQueryOptions)
}

/** Sorted spell list off a loaded catalog (empty until it resolves). */
export function spellList(catalog: SpellCatalog | undefined): CatalogSpell[] {
  return Object.values(catalog ?? {}).sort((a, b) => a.name.localeCompare(b.name))
}

/** Buff spells only (mid-combat effect pickers). Empty until loaded. */
export function buffSpells(catalog: SpellCatalog | undefined): CatalogSpell[] {
  return Object.values(catalog ?? {}).filter((s) => s.buff)
}
