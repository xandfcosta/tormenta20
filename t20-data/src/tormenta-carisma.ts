/**
 * Carisma loss from accumulated poderes da Tormenta — PURE math, NO powers
 * data. Split out of `./tormenta` (which holds the TORMENTA_POWERS catalog) so
 * the frontend can import it without anchoring that catalog into the bundle
 * (project_front_decouple_catalog). `./tormenta` re-exports it.
 *
 * Cada poder da Tormenta custa Carisma acumulativo (p136): o 1º custa 1, e a
 * cada dois poderes o custo por poder sobe 1. Sequência do total acumulado:
 * 1 → 2 → 4 → 6 → 9 → 12 → 16 → 20 → 25 → …
 */
export function carismaLossFromPowers(powerCount: number): number {
  if (powerCount < 0) {
    throw new Error(
      `carismaLossFromPowers: powerCount must be ≥ 0, got ${powerCount}`,
    )
  }
  let total = 0
  for (let k = 1; k <= powerCount; k++) {
    total += 1 + Math.floor((k - 1) / 2)
  }
  return total
}
