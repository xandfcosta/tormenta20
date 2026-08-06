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
