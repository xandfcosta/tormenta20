/**
 * Case + accent-insensitive normalizer for on-sheet search boxes.
 * Used by both the expertise search + the catalog picker so a player
 * typing "furt" finds "Furtividade".
 */
export function normalize(s: string): string {
  return s.normalize('NFD').replace(/\p{Diacritic}/gu, '').toLowerCase()
}
