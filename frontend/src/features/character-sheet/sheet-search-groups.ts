import { sheetSearchFilter } from './sheet-search-filter'
import type { SheetSearchEntry } from './sheet-search-index'

export type SheetSearchGroup = { source: string; entries: SheetSearchEntry[] }

/** Where each source sits in the palette — the sheet's own reading order. */
const GROUP_ORDER = ['Perícia', 'Poderes & habilidades', 'Item', 'Magia', 'Condição']

/** Sources that keep their own group; anything else is a power-ish source. */
const OWN_GROUP = new Set(['Perícia', 'Item', 'Magia', 'Condição'])

const POWERS_GROUP = 'Poderes & habilidades'

/**
 * Scores the index against the query and drops what doesn't match, best first.
 * The score comes from `sheetSearchFilter`, so a prefix outranks a mid-word hit
 * and "furia" still finds "Fúria" — otherwise the obvious answer sinks below
 * the incidental one.
 *
 * An empty query returns the index untouched, in its own order.
 *
 * @example rankSheetEntries(index, 'furia')
 */
export function rankSheetEntries(
  index: readonly SheetSearchEntry[],
  query: string,
): SheetSearchEntry[] {
  if (!query.trim()) return [...index]
  return index
    .map((entry) => ({ entry, score: sheetSearchFilter(`${entry.name} ${entry.source}`, query) }))
    .filter((scored) => scored.score > 0)
    .sort((a, b) => b.score - a.score)
    .map((scored) => scored.entry)
}

/**
 * Buckets entries for display. Raça / Origem / Classe / Poder geral collapse
 * into one group: at the table they are the same question ("que poder é
 * esse?"), and four one-line groups read as noise.
 *
 * Group ORDER depends on whether the list was ranked. Browsing (no query)
 * follows the sheet's reading order. Searching keeps `rankSheetEntries`' order,
 * so the group holding the best match comes first — otherwise the palette's
 * cursor lands on the first row of a fixed group and Enter picks a weak
 * subsequence hit over the exact one the player typed.
 *
 * @example groupSheetEntries(rankSheetEntries(index, query), { ranked: true })
 */
export function groupSheetEntries(
  entries: readonly SheetSearchEntry[],
  options: { ranked?: boolean } = {},
): SheetSearchGroup[] {
  const bySource = new Map<string, SheetSearchEntry[]>()
  for (const entry of entries) {
    const key = OWN_GROUP.has(entry.source) ? entry.source : POWERS_GROUP
    bySource.set(key, [...(bySource.get(key) ?? []), entry])
  }
  const groups = [...bySource].map(([source, list]) => ({ source, entries: list }))
  // Map insertion order already follows the best-scoring entry when ranked.
  if (options.ranked) return groups
  return groups.sort((a, b) => GROUP_ORDER.indexOf(a.source) - GROUP_ORDER.indexOf(b.source))
}
