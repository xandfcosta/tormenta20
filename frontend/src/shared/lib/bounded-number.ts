import { z } from 'zod'

/** A numeric control's valid domain: an inclusive range and a step increment
 *  (1 for integers, e.g. 0.25 for T20 ND). */
export type NumberRange = { min: number; max: number; step?: number }

/** zod schema for a value inside {@link NumberRange} — the source of truth the
 *  clamp below is guaranteed to satisfy (see bounded-number.test.ts). Use it to
 *  validate where a real submit exists; use {@link clampToRange} for the live,
 *  no-submit generator/filter knobs that must always yield a usable number. */
export function rangeSchema({ min, max }: NumberRange): z.ZodType<number> {
  return z.number().min(min).max(max)
}

/**
 * Validate a live control's value against its range and, when out of domain,
 * snap-to-step + clamp instead of rejecting. These knobs (encounter party
 * level/size, monster ND range, room count) recompute on every change with no
 * submit step, so a NaN or out-of-range typed value must be corrected in place
 * rather than blocking — otherwise it poisons the downstream generator math.
 *
 * @example clampToRange(99, { min: 1, max: 20 }) // 20
 * @example clampToRange(0.3, { min: 0, max: 20, step: 0.25 }) // 0.25
 */
export function clampToRange(value: number, range: NumberRange): number {
  const { min, max, step = 1 } = range
  if (rangeSchema(range).safeParse(value).success && isOnStep(value, min, step)) {
    return value
  }
  // NaN has no meaningful position; ±Infinity clamps naturally at the bounds.
  const base = Number.isNaN(value) ? min : value
  const snapped = min + Math.round((base - min) / step) * step
  const clamped = Math.min(max, Math.max(min, snapped))
  // Kill float drift from the step division (e.g. 0.1 + 0.2) before it shows.
  return Number(clamped.toFixed(4))
}

/** Whether `value` sits on the `min + k*step` grid (within float tolerance). */
function isOnStep(value: number, min: number, step: number): boolean {
  const steps = (value - min) / step
  return Math.abs(steps - Math.round(steps)) < 1e-9
}
