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
 * Vitals CHOKE POINT: PV/PM máximos pelo motor Go/WASM, em produção E nos
 * testes. O ramo TS que existia aqui atrás de `import.meta.env.MODE === 'test'`
 * saiu: ele fazia a suíte medir uma implementação que a produção não roda
 * (ALE-109). A cópia TS continua viva, mas só como IMPLEMENTAÇÃO DE REFERÊNCIA
 * do oráculo de paridade — ver `tsVitalPools` abaixo.
 */
export function computeVitalPools(ctx: VitalContext): VitalPools {
  if (!areEngineCatalogsPrimed()) {
    throw new Error('vitals: WASM engine not primed — ensureEngineCatalogs() must resolve first')
  }
  return engineComputeVitals(ctx)
}

/**
 * A mesma conta em TypeScript — a IMPLEMENTAÇÃO DE REFERÊNCIA que gera o oráculo
 * de paridade.
 *
 * Ela precisa ser chamada EXPLICITAMENTE, e não por um `if` de ambiente: assim
 * que o harness passou a atravessar o choke point, o oráculo virou "o Go dizendo
 * o que o Go acha" — que é justamente a ilusão que a fatia 5 quer evitar enquanto
 * ainda há duas implementações. Morre com o `t20-data`.
 */
export function tsVitalPools(ctx: VitalContext): VitalPools {
  const con = ctx.attrTotals.constitution
  const grants = collectVitalGrants(toGrantContext(ctx), frontVitalResolver)
  return {
    pvMax: Math.max(0, multiclassPvPool(ctx.classes, con) + grants.pv),
    pmMax: Math.max(0, multiclassMpPool(ctx.classes) + grants.pm),
  }
}

/** Adapt VitalContext → t20-data VitalGrantContext (adds the single-class
 *  fallback `className`; the shapes otherwise match). */
function toGrantContext(ctx: VitalContext): VitalGrantContext {
  return { ...ctx, className: ctx.classes[0]?.className ?? '' }
}
