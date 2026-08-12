/**
 * Sum of `count` dice with `sides` faces. The Forja rolls in two places — the
 * starting money (4d6, p140) and an origin's T$ grant (2d6) — and a second
 * inline copy is how the two would drift apart.
 *
 * @example rollDice(4, 6) // 14
 */
export function rollDice(count: number, sides: number): number {
  let total = 0
  for (let i = 0; i < count; i++) total += 1 + Math.floor(Math.random() * sides)
  return total
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
