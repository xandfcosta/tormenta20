import type { GrantedPowerOption } from '@/shared/api/catalog-types'

/**
 * Front-owned cache for the divine powers (poderes concedidos + rule text) the
 * devoto picker offers. Mirrors t20-data `grantedPowerOptionsFor` by filtering
 * the fetched list per deus — so the front reads it WITHOUT bundling
 * DIVINE_POWERS + DIVINE_POWER_DESCRIPTIONS (~24KB); fetched from
 * `GET /catalog/divine-powers` (the precomputed `allGrantedPowerOptions`) and
 * cached instead (project_front_decouple_catalog). Same prime-before-render
 * contract as the other *-cache modules.
 */
let options: readonly GrantedPowerOption[] = []
let primed = false

export function primeDivinePowers(opts: readonly GrantedPowerOption[]): void {
  options = opts
  primed = true
}

/** True once the divine-powers cache has been primed — for a render-time gate. */
export function isDivinePowersPrimed(): boolean {
  return primed
}

/** The devoto power options for a deus (or empty). Mirrors t20-data
 *  `grantedPowerOptionsFor` over the fetched list. */
export function grantedPowerOptionsFor(deusId: string): GrantedPowerOption[] {
  return options.filter((p) => p.deusId === deusId)
}
