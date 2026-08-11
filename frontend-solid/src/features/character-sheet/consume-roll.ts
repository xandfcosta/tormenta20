import { z } from 'zod'

/**
 * Dice-roll validation for the consumable "Usar" dialog. The player rolls a
 * physical die (e.g. Bálsamo restaurador = 2d4 PV) and types the result; we
 * validate it client-side (zod) before applying so a typo (empty, negative, or
 * a value the die can't produce) is caught before the round-trip.
 */

export type InstantRoll = { dice: string; bonus?: number }

/**
 * Min/max total for a dice string like "2d4" — each of the N dice rolls in
 * [1, sides], so the total is in [N, N*sides]. Returns null for a malformed
 * string (or the fixed "0") so callers skip range validation but still require
 * a value.
 *
 * @example rollBounds('2d4') // { min: 2, max: 8 }
 */
export function rollBounds(dice: string): { min: number; max: number } | null {
  const match = /^(\d+)d(\d+)$/.exec(dice.trim())
  if (!match) return null
  const count = Number(match[1])
  const sides = Number(match[2])
  if (count < 1 || sides < 1) return null
  return { min: count, max: count * sides }
}

/**
 * Zod schema for a rolled die result (kept as the raw string the field holds).
 * Empty is rejected, non-digits are rejected, and the total is range-checked
 * against {@link rollBounds} when the die string is well-formed.
 *
 * @example rollValueSchema({ dice: '2d4' }).safeParse('9').success // false
 */
export function rollValueSchema(roll: InstantRoll) {
  const bounds = rollBounds(roll.dice)
  return z
    .string()
    .trim()
    .min(1, 'Informe o resultado do dado')
    .refine((v) => /^\d+$/.test(v), 'Valor inválido')
    .refine(
      (v) => !bounds || (Number(v) >= bounds.min && Number(v) <= bounds.max),
      bounds
        ? `Fora do intervalo (${bounds.min}–${bounds.max})`
        : 'Valor inválido',
    )
}
