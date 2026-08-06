import {
  collectVitalGrants,
  multiclassMpPool,
  multiclassPvPool,
  type VitalGrantContext,
} from '@tormenta20/t20-data'
import {
  areEngineCatalogsPrimed,
  computeVitals as engineComputeVitals,
  type VitalContext,
  type VitalPools,
} from '@/shared/lib/engine-wasm'
import { frontVitalResolver } from './vital-resolver'

export type { VitalContext, VitalPools }

/**
 * Vitals CHOKE POINT (Inc.3): PV/PM máximos via the Go/WASM engine — the single
 * source of truth. Same MODE-gate as the item-effects derive (derived.ts
 * `resolveEffects`): in production/dev the engine computes everything; the TS
 * branch (t20-data `collectVitalGrants` + `frontVitalResolver` + the multiclass
 * pools) is TEST-ONLY, so it stays the parity oracle without wasm. Because
 * `import.meta.env.MODE` is statically `'production'` in the app build, that
 * branch — and the TS vitals pipeline it reaches — is dead-code-eliminated, so
 * the bundle ships only the Go engine. Parity is proven by the `vitals` oracle.
 */
export function computeVitalPools(ctx: VitalContext): VitalPools {
  if (import.meta.env.MODE === 'test') {
    const con = ctx.attrTotals.constitution
    const grants = collectVitalGrants(toGrantContext(ctx), frontVitalResolver)
    return {
      pvMax: Math.max(0, multiclassPvPool(ctx.classes, con) + grants.pv),
      pmMax: Math.max(0, multiclassMpPool(ctx.classes) + grants.pm),
    }
  }
  if (!areEngineCatalogsPrimed()) {
    throw new Error('vitals: WASM engine not primed — ensureEngineCatalogs() must resolve first')
  }
  return engineComputeVitals(ctx)
}

/** Adapt VitalContext → t20-data VitalGrantContext (adds the single-class
 *  fallback `className`; the shapes otherwise match). */
function toGrantContext(ctx: VitalContext): VitalGrantContext {
  return { ...ctx, className: ctx.classes[0]?.className ?? '' }
}
