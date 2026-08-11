import { normalize } from './normalize'

/**
 * cmdk `filter` for the sheet palette: accent/case-insensitive with a fuzzy
 * subsequence fallback, so "furia", "conc comb" and "ccombate" all find
 * "Concentração de Combate". Returns 0..1 — 0 hides the row.
 *
 * Example: `sheetSearchFilter('Fúria Classe · Bárbaro', 'furia') === 1`.
 */
export function sheetSearchFilter(value: string, search: string): number {
  const haystack = normalize(value)
  const query = normalize(search).trim()
  if (!query) return 1
  // Every whitespace-separated term must hit; the row scores its worst term.
  let worst = 1
  for (const term of query.split(/\s+/)) {
    const score = termScore(haystack, term)
    if (score === 0) return 0
    worst = Math.min(worst, score)
  }
  return worst
}

/** Prefix (1) > word-start (0.9) > infix (0.8) > subsequence (<0.7). */
function termScore(haystack: string, term: string): number {
  const at = haystack.indexOf(term)
  if (at === 0) return 1
  if (at > 0) return haystack[at - 1] === ' ' ? 0.9 : 0.8
  return subsequenceScore(haystack, term)
}

/** In-order scattered chars ("ftvd" → "FurTiViDade"); tighter spread = higher. */
function subsequenceScore(haystack: string, term: string): number {
  let matched = 0
  let first = -1
  let last = -1
  for (let i = 0; i < haystack.length && matched < term.length; i++) {
    if (haystack[i] !== term[matched]) continue
    if (first < 0) first = i
    last = i
    matched++
  }
  if (matched < term.length) return 0
  return 0.3 + 0.4 * (term.length / (last - first + 1))
}
