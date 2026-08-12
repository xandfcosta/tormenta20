/**
 * Sum of `count` dice with `sides` faces. Rolled in four places already — the
 * Forja's starting money (4d6, p140), an origin's T$ grant (2d6), a player's
 * initiative and the GM dropping a monster into it — and every inline copy is
 * one more place for them to drift apart.
 *
 * @example rollDice(4, 6) // 14
 */
export function rollDice(count: number, sides: number): number {
  let total = 0
  for (let i = 0; i < count; i++) total += 1 + Math.floor(Math.random() * sides)
  return total
}

/** The initiative die. Monsters carry no DEX mod here — the GM adjusts. */
export function rollD20(): number {
  return rollDice(1, 20)
}

/** Parses the "2d6" notation the origin grants carry. Returns null for junk,
 *  so a bad catalog string cannot roll a NaN into the player's wallet. */
export function parseDiceNotation(notation: string): { count: number; sides: number } | null {
  const match = /^(\d+)d(\d+)$/.exec(notation.trim())
  if (!match) return null
  const count = Number(match[1])
  const sides = Number(match[2])
  return count > 0 && sides > 0 ? { count, sides } : null
}
