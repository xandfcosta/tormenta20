import type { ActivationSpec } from '@/shared/api/catalog-types'

/**
 * Front-owned cache for the power-activation registry (every power source's
 * id → pmCost/action/uses spec), with SYNC accessors mirroring t20-data's
 * `getActivation`/`findActivationByName`/`ACTIVATION_SPECS`. Lets the core
 * derive (effect-source label resolution) + the power-use UI read activation
 * specs WITHOUT a build-time `import` of power-activation.ts — which eagerly
 * builds the registry from ALL 14 classes + racial + origem + divine power
 * mechanics (~40KB). Fetched from `GET /catalog/activations` (the prebuilt
 * `ACTIVATION_SPECS`) and cached instead (project_front_decouple_catalog).
 * Same prime-before-render contract as the other *-cache modules.
 */
let byId: ReadonlyMap<string, ActivationSpec> = new Map()
let byNameClass: ReadonlyMap<string, ActivationSpec> = new Map()
let byNameAny: ReadonlyMap<string, ActivationSpec> = new Map()
let specs: readonly ActivationSpec[] = []
let primed = false

// Mirror of t20-data's internal `slug` (abilities/classes/_helpers) so
// findActivationByName matches the source's name normalization exactly.
function slug(text: string): string {
  return text
    .toLowerCase()
    .normalize('NFD')
    .replace(/[̀-ͯ]/g, '')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
}

export function primeActivations(list: readonly ActivationSpec[]): void {
  specs = list
  byId = new Map(list.map((s) => [s.id, s]))
  const cls = new Map<string, ActivationSpec>()
  const any = new Map<string, ActivationSpec>()
  for (const spec of list) {
    const key = slug(spec.name)
    if (!any.has(key)) any.set(key, spec)
    if (spec.id.startsWith('class.') && !cls.has(key)) cls.set(key, spec)
  }
  byNameClass = cls
  byNameAny = any
  primed = true
}

/** True once the activation cache has been primed — for a render-time gate. */
export function isActivationCachePrimed(): boolean {
  return primed
}

/** Lookup by full id — mirrors t20-data `getActivation`. */
export function getActivation(id: string): ActivationSpec | undefined {
  return byId.get(id)
}

/** Fallback by name (accent/case-insensitive; prefers a class-power source) —
 *  mirrors t20-data `findActivationByName`. */
export function findActivationByName(name: string): ActivationSpec | undefined {
  const key = slug(name)
  return byNameClass.get(key) ?? byNameAny.get(key)
}

/** All specs — was the t20-data `ACTIVATION_SPECS` const. Read after the gate. */
export function activationSpecs(): readonly ActivationSpec[] {
  return specs
}
