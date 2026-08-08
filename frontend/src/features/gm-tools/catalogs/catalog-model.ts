import type {
  CatalogItem,
  CatalogSpell,
  Condition,
} from '@tormenta20/t20-data'
import {
  classPowerCatalog,
  generalPowerCatalog,
  grantedPowers as grantedPowerCatalog,
} from '@/shared/lib/abilities-cache'
import { normalizeText } from '@/shared/lib/normalize-text'

/**
 * Unified power entry for the GM catalog. The book scatters powers across
 * several catalogs (class abilities, general/combat feats, granted divine
 * powers); the GM just wants one searchable "Poderes" list, so we flatten
 * them to a common shape tagged by `source`. Divine powers are omitted here —
 * their data carries only a book-page reference, no rules text to check.
 */
export type CatalogPower = {
  id: string
  name: string
  source: string
  description: string
}

function classPowers(): CatalogPower[] {
  return classPowerCatalog().map((p) => ({
    id: p.id,
    name: p.name,
    source: p.className,
    description: p.description,
  }))
}

function generalPowers(): CatalogPower[] {
  return generalPowerCatalog().map((p) => ({
    id: `general.${p.id}`,
    name: p.name,
    source: `Geral · ${p.kind}`,
    description: p.description,
  }))
}

function grantedPowers(): CatalogPower[] {
  return grantedPowerCatalog().map((p) => ({
    id: `granted.${p.id}`,
    name: p.name,
    source: `Concedido · ${p.deuses.join(', ')}`,
    description: p.effect,
  }))
}

/**
 * All searchable powers, sorted by name (accent-insensitive). A function, not a
 * module-level const, because the abilities catalog is fetched + primed by the
 * loader gate — reading it at import time (before priming) would yield an empty
 * list. Call from a component/effect that runs after the gate (B.3).
 */
export function catalogPowers(): readonly CatalogPower[] {
  return [...classPowers(), ...generalPowers(), ...grantedPowers()].sort(
    (a, b) => a.name.localeCompare(b.name, 'pt-BR'),
  )
}

/**
 * True when every whitespace-separated term in `query` appears in one of the
 * entry's searchable fields. Empty query matches everything. Terms are ANDed
 * so "luz cur" narrows to spells with both.
 */
export function matchesQuery(fields: readonly string[], query: string): boolean {
  const q = normalizeText(query)
  if (!q) return true
  const hay = normalizeText(fields.join(' '))
  return q.split(/\s+/).every((term) => hay.includes(term))
}

// Searchable-field extractors per catalog. Stable module-level fns so the tab
// filter memoizes on their identity, and the single source of truth for what
// each catalog matches on (browse tabs + the unified search share them).
export const conditionSearch = (c: Condition): string[] => [
  c.name,
  c.description,
  ...c.tags,
]
export const spellSearch = (s: CatalogSpell): string[] => [s.name, s.baseEffect]
export const powerSearch = (p: CatalogPower): string[] => [
  p.name,
  p.source,
  p.description,
]
export const itemSearch = (i: CatalogItem): string[] => [i.name, i.category]

// A flat row in the unified (all-catalogs) search view: a per-catalog section
// header followed by its matching entries, so a single virtualized list can
// render heterogeneous results grouped by catalog (ALE-22).
export type CatalogResultRow =
  | { kind: 'header'; key: string; label: string; count: number }
  | { kind: 'condition'; key: string; value: Condition }
  | { kind: 'spell'; key: string; value: CatalogSpell }
  | { kind: 'power'; key: string; value: CatalogPower }
  | { kind: 'item'; key: string; value: CatalogItem }

function pushGroup<T>(
  rows: CatalogResultRow[],
  label: string,
  matches: readonly T[],
  toRow: (item: T) => CatalogResultRow,
): void {
  if (matches.length === 0) return
  rows.push({ kind: 'header', key: `header.${label}`, label, count: matches.length })
  for (const m of matches) rows.push(toRow(m))
}

/**
 * Filters every catalog by `query` and flattens the matches into one grouped
 * row list (header + entries per non-empty catalog). Empty catalogs are
 * omitted. Powers the unified GM search so "bola de fogo" finds the spell no
 * matter which tab was active (ALE-22).
 */
export function catalogSearchRows(
  query: string,
  catalogs: {
    conditions: readonly Condition[]
    spells: readonly CatalogSpell[]
    powers: readonly CatalogPower[]
    items: readonly CatalogItem[]
  },
): CatalogResultRow[] {
  const rows: CatalogResultRow[] = []
  const keep = <T>(list: readonly T[], search: (i: T) => string[]) =>
    list.filter((i) => matchesQuery(search(i), query))
  pushGroup(rows, 'Condições', keep(catalogs.conditions, conditionSearch), (c) => ({
    kind: 'condition',
    key: `condition.${c.id}`,
    value: c,
  }))
  pushGroup(rows, 'Magias', keep(catalogs.spells, spellSearch), (s) => ({
    kind: 'spell',
    key: `spell.${s.id}`,
    value: s,
  }))
  pushGroup(rows, 'Poderes', keep(catalogs.powers, powerSearch), (p) => ({
    kind: 'power',
    key: `power.${p.id}`,
    value: p,
  }))
  pushGroup(rows, 'Itens', keep(catalogs.items, itemSearch), (i) => ({
    kind: 'item',
    key: `item.${i.id}`,
    value: i,
  }))
  return rows
}
