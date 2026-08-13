/**
 * Up to two initials — the monogram standing in for art everywhere in this app:
 * character stage, side peeks, filmstrip chips, campaign emblems.
 *
 * It lives in `shared/lib` and not next to its first caller because
 * `shared/ui/character-portrait` needs it too, and shared may not import from
 * features — which is precisely how the third copy of this eight-line function
 * came to exist.
 *
 * @example initials('Thal, o Errante') // 'TO'
 */
export function initials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean)
  if (parts.length === 0) return '?'
  return parts
    .slice(0, 2)
    .map((word) => word[0])
    .join('')
    .toUpperCase()
}

