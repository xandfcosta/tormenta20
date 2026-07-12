import {
  CLASS_POWERS_CATALOG,
  GENERAL_POWERS_CATALOG,
  GRANTED_POWERS,
} from '@tormenta20/t20-data'
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
  return CLASS_POWERS_CATALOG.map((p) => ({
    id: p.id,
    name: p.name,
    source: p.className,
    description: p.description,
  }))
}

function generalPowers(): CatalogPower[] {
  return GENERAL_POWERS_CATALOG.map((p) => ({
    id: `general.${p.id}`,
    name: p.name,
    source: `Geral · ${p.kind}`,
    description: p.description,
  }))
}

function grantedPowers(): CatalogPower[] {
  return GRANTED_POWERS.map((p) => ({
    id: `granted.${p.id}`,
    name: p.name,
    source: `Concedido · ${p.deuses.join(', ')}`,
    description: p.effect,
  }))
}

/** All searchable powers, sorted by name (accent-insensitive). */
export const CATALOG_POWERS: readonly CatalogPower[] = [
  ...classPowers(),
  ...generalPowers(),
  ...grantedPowers(),
].sort((a, b) => a.name.localeCompare(b.name, 'pt-BR'))

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
