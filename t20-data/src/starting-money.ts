/**
 * Tabela 3-1: Dinheiro Inicial (book p140). Level 1 rolls T$ 4d6 (see
 * `STARTING_TIBARES_DICE`); levels 2-20 start with a fixed amount. The app
 * treats these as editable defaults — GM homebrew may grant more/less.
 */
export const STARTING_MONEY_BY_LEVEL: Readonly<Record<number, number>> = {
  2: 300,
  3: 600,
  4: 1_000,
  5: 2_000,
  6: 3_000,
  7: 5_000,
  8: 7_000,
  9: 10_000,
  10: 13_000,
  11: 19_000,
  12: 27_000,
  13: 36_000,
  14: 49_000,
  15: 66_000,
  16: 88_000,
  17: 110_000,
  18: 150_000,
  19: 200_000,
  20: 260_000,
}

/**
 * Default starting tibares for a level. Level 1 has no fixed value (roll
 * 4d6) — returns null so the caller offers the roll instead. Out-of-range
 * levels throw with the offending value.
 */
export function startingMoneyForLevel(level: number): number | null {
  if (level === 1) return null
  const value = STARTING_MONEY_BY_LEVEL[level]
  if (value === undefined) {
    throw new Error(
      `startingMoneyForLevel: level must be 1-20, got ${level}`,
    )
  }
  return value
}
