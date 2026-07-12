/**
 * Accent- and case-insensitive normalize for free-text search across catalogs
 * (spells, powers, items, conditions). Strips diacritics so "aparicao"
 * matches "Aparição".
 */
export function normalizeText(s: string): string {
  return s
    .normalize('NFD')
    .replace(/[̀-ͯ]/g, '')
    .toLowerCase()
    .trim()
}
