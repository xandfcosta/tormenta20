/**
 * Structured view of `Origem.itensIniciais` (book p85-95 "Itens" lines).
 * The catalog stores the book's verbatim strings; several are CHOICES, not
 * fixed grants ("Arma marcial", "Um item estrangeiro (até T$ 100)",
 * "Estojo de disfarces OU gazua"). This parser classifies each entry so the
 * creation UI can render a picker instead of a dead text row.
 *
 * Parsing conventions (encoded in the data strings):
 *  - literal 'Arma simples' / 'Arma marcial' / 'Arma marcial ou exótica'
 *    → weapon pick by category.
 *  - '(até T$ N)' → any-item pick with a price cap.
 *  - ' OU ' (uppercase) → one-of between fixed alternatives; lowercase 'ou'
 *    inside parentheses is descriptive text, never a split point.
 *  - '(escolha)' suffix → one-of over the comma/'ou'-separated list.
 *  - 'T$ <dice>' → starting-money bonus, not an item.
 *  - anything else → fixed grant.
 */
import { ORIGENS } from './origens'

export type OrigemItemGrant =
  | { kind: 'fixed'; name: string }
  | {
      kind: 'weapon'
      categories: readonly ('weapon-simple' | 'weapon-martial' | 'weapon-exotic')[]
      label: string
    }
  | { kind: 'anyItem'; maxPrice: number; label: string }
  | { kind: 'oneOf'; options: readonly string[]; label: string }
  | { kind: 'money'; dice: string; label: string }

const WEAPON_ENTRIES: Readonly<
  Record<string, readonly ('weapon-simple' | 'weapon-martial' | 'weapon-exotic')[]>
> = {
  'Arma simples': ['weapon-simple'],
  'Arma marcial': ['weapon-martial'],
  'Arma marcial ou exótica': ['weapon-martial', 'weapon-exotic'],
}

export function parseOrigemItem(entry: string): OrigemItemGrant {
  const weapon = WEAPON_ENTRIES[entry]
  if (weapon) return { kind: 'weapon', categories: weapon, label: entry }
  const dice = entry.match(/^T\$ (\d+d\d+)/)
  if (dice) return { kind: 'money', dice: dice[1], label: entry }
  const cap = entry.match(/até T\$ (\d+)/)
  if (cap) return { kind: 'anyItem', maxPrice: Number(cap[1]), label: entry }
  if (entry.includes(' OU ')) {
    return { kind: 'oneOf', options: entry.split(' OU '), label: entry }
  }
  if (entry.endsWith('(escolha)')) {
    const list = entry.replace(/\s*\(escolha\)$/, '')
    const options = list
      .split(/,\s*|\s+ou\s+/)
      .map((s) => s.trim())
      .filter(Boolean)
      .map((s) => s.charAt(0).toUpperCase() + s.slice(1))
    return { kind: 'oneOf', options, label: entry }
  }
  return { kind: 'fixed', name: entry }
}

/** Structured item grants for an origem, by NAME (as the app stores it). */
export function origemItemGrantsByName(
  originName: string,
): OrigemItemGrant[] {
  const origem = Object.values(ORIGENS).find((o) => o.name === originName)
  return (origem?.itensIniciais ?? []).map(parseOrigemItem)
}
