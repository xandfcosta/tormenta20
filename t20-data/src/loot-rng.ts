/**
 * Seedable RNG + dice helpers for deterministic loot generation.
 *
 * Uses mulberry32 — fast 32-bit PRNG with full uniform distribution
 * in [0, 1). Good enough for tabletop loot rolls (not cryptographic).
 *
 * Determinism: same seed → identical output sequence. Snapshot-test
 * friendly.
 */

export interface Rng {
  next(): number // float in [0, 1)
}

export function mulberry32(seed: number): Rng {
  let state = seed >>> 0
  return {
    next(): number {
      state = (state + 0x6d2b79f5) >>> 0
      let t = state
      t = Math.imul(t ^ (t >>> 15), t | 1)
      t ^= t + Math.imul(t ^ (t >>> 7), t | 61)
      return ((t ^ (t >>> 14)) >>> 0) / 4294967296
    },
  }
}

/** Inclusive integer in [min, max]. */
export function randInt(rng: Rng, min: number, max: number): number {
  if (min > max) throw new Error(`randInt: min ${min} > max ${max}`)
  return Math.floor(rng.next() * (max - min + 1)) + min
}

/** Single die roll [1, sides]. */
export function rollDie(rng: Rng, sides: number): number {
  if (sides < 1) throw new Error(`rollDie: sides must be ≥ 1, got ${sides}`)
  return randInt(rng, 1, sides)
}

/** Roll `count` dice of `sides`, sum them. */
export function rollDice(rng: Rng, count: number, sides: number): number {
  if (count < 0) throw new Error(`rollDice: count must be ≥ 0, got ${count}`)
  let total = 0
  for (let i = 0; i < count; i++) total += rollDie(rng, sides)
  return total
}

/** Standard d100 (percentile). Returns 1-100 inclusive. */
export function rollPercentile(rng: Rng): number {
  return rollDie(rng, 100)
}

/**
 * Parse a T20-style dice formula like:
 *   "4d6", "1d4×10", "4d6×10", "2d6+1×100", "1d10×1.000",
 *   "2d10×1.000", "6d12×10.000".
 *
 * Comma-thousands separator is supported (PDF uses "1.000"). Multiplier
 * uses "×" or "*". Flat constant prefix like "+1" within the dice term
 * is handled (e.g. "2d6+1×100" = (2d6 + 1) × 100).
 */
export function rollFormula(rng: Rng, formula: string): number {
  const normalized = formula
    .replace(/[×x*]/g, '*')
    .replace(/\./g, '') // 1.000 → 1000
    .replace(/\s+/g, '')

  // Match: <count>d<sides>(+<flat>)? optional (*<mult>)?
  const m = normalized.match(/^(\d+)d(\d+)(?:\+(\d+))?(?:\*(\d+))?$/)
  if (!m) {
    throw new Error(`rollFormula: cannot parse "${formula}"`)
  }
  const count = Number(m[1])
  const sides = Number(m[2])
  const flat = m[3] ? Number(m[3]) : 0
  const mult = m[4] ? Number(m[4]) : 1

  const diceSum = rollDice(rng, count, sides)
  return (diceSum + flat) * mult
}

/** Pick a random element from a non-empty readonly array. */
export function pickOne<T>(rng: Rng, items: readonly T[]): T {
  if (items.length === 0) throw new Error('pickOne: empty array')
  return items[randInt(rng, 0, items.length - 1)]!
}
