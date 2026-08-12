import type { CatalogItem, CatalogSpell, Condition } from '@tormenta20/t20-data'
import {
  classPowerCatalog,
  generalPowerCatalog,
  grantedPowers as grantedPowerCatalog,
} from '@/shared/lib/abilities-cache'
import { normalizeText } from '@/shared/lib/normalize-text'

/**
 * One power for the GM's lookup. The book scatters powers across several
 * catalogs (class abilities, general/combat feats, granted divine powers); the
 * GM just wants ONE searchable "Poderes" list, so they are flattened to a
 * common shape tagged by where they came from. Divine powers are left out —
 * their data carries a book page and no rules text to check.
 */
export type CatalogPower = {
  id: string
  name: string
  source: string
  description: string
}

/**
 * Every searchable power, sorted by name. A function and NOT a module-level
 * const: the abilities catalog is fetched and primed by the loader gate, so
 * reading it at import time — before priming — would freeze an empty list
 * (gotcha #13 of the port).
 */
export function catalogPowers(): CatalogPower[] {
  const fromClasses = classPowerCatalog().map((power) => ({
    id: power.id,
    name: power.name,
    source: power.className,
    description: power.description,
  }))
  const fromGeneral = generalPowerCatalog().map((power) => ({
    id: `general.${power.id}`,
    name: power.name,
    source: `Geral · ${power.kind}`,
    description: power.description,
  }))
  const fromGods = grantedPowerCatalog().map((power) => ({
    id: `granted.${power.id}`,
    name: power.name,
    source: `Concedido · ${power.deuses.join(', ')}`,
    description: power.effect,
  }))
  return [...fromClasses, ...fromGeneral, ...fromGods].sort((a, b) =>
    a.name.localeCompare(b.name, 'pt-BR'),
  )
}

/**
 * True when EVERY whitespace-separated term appears in one of the fields.
 * Terms are ANDed, so "luz cur" narrows to entries carrying both. An empty
 * query matches everything.
 *
 * Deliberately not `shared/lib/fuzzy-filter`'s `matchesQuery`: that one is
 * typo-tolerant ranking for picking ONE thing out of a list. Here the GM is
 * narrowing a reference by words they know, and fuzzy ranking would drag in
 * near-misses that make a rules lookup feel wrong.
 *
 * @example matchesAllTerms(['Bola de Fogo', 'dano de fogo'], 'bola fogo') // true
 */
export function matchesAllTerms(fields: readonly string[], query: string): boolean {
  const needle = normalizeText(query)
  if (!needle) return true
  const haystack = normalizeText(fields.join(' '))
  return needle.split(/\s+/).every((term) => haystack.includes(term))
}

// What each catalog matches on. Module-level so the browse tabs and the unified
// search agree by construction instead of by copy.
export const conditionSearch = (c: Condition): string[] => [c.name, c.description, ...c.tags]
export const spellSearch = (s: CatalogSpell): string[] => [s.name, s.baseEffect]
export const powerSearch = (p: CatalogPower): string[] => [p.name, p.source, p.description]
export const itemSearch = (i: CatalogItem): string[] => [i.name, i.category]

/** A flat row of the unified search: a per-catalog header followed by its hits,
 *  so ONE virtualized list can render results of four different shapes. */
export type CatalogResultRow =
  | { kind: 'header'; key: string; label: string; count: number }
  | { kind: 'condition'; key: string; value: Condition }
  | { kind: 'spell'; key: string; value: CatalogSpell }
  | { kind: 'power'; key: string; value: CatalogPower }
  | { kind: 'item'; key: string; value: CatalogItem }

export type SearchableCatalogs = {
  conditions: readonly Condition[]
  spells: readonly CatalogSpell[]
  powers: readonly CatalogPower[]
  items: readonly CatalogItem[]
}

/**
 * Filters every catalog by one query and flattens the hits into a single
 * grouped row list. Catalogs with no hits are omitted entirely — a header over
 * nothing is noise in a mid-combat lookup.
 *
 * @example catalogSearchRows('bola de fogo', catalogs) // [header Magias, spell…]
 */
export function catalogSearchRows(
  query: string,
  catalogs: SearchableCatalogs,
): CatalogResultRow[] {
  const rows: CatalogResultRow[] = []
  const hits = <T>(list: readonly T[], fields: (entry: T) => string[]) =>
    list.filter((entry) => matchesAllTerms(fields(entry), query))

  pushGroup(rows, 'Condições', hits(catalogs.conditions, conditionSearch), (c) => ({
    kind: 'condition',
    key: `condition.${c.id}`,
    value: c,
  }))
  pushGroup(rows, 'Magias', hits(catalogs.spells, spellSearch), (s) => ({
    kind: 'spell',
    key: `spell.${s.id}`,
    value: s,
  }))
  pushGroup(rows, 'Poderes', hits(catalogs.powers, powerSearch), (p) => ({
    kind: 'power',
    key: `power.${p.id}`,
    value: p,
  }))
  pushGroup(rows, 'Itens', hits(catalogs.items, itemSearch), (i) => ({
    kind: 'item',
    key: `item.${i.id}`,
    value: i,
  }))
  return rows
}

function pushGroup<T>(
  rows: CatalogResultRow[],
  label: string,
  matches: readonly T[],
  toRow: (entry: T) => CatalogResultRow,
): void {
  if (matches.length === 0) return
  rows.push({ kind: 'header', key: `header.${label}`, label, count: matches.length })
  for (const match of matches) rows.push(toRow(match))
}
