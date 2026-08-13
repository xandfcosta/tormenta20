import {
} from '@tormenta20/t20-data'
import {
  areEngineCatalogsPrimed,
  computeVitals as engineComputeVitals,
  type VitalContext,
  type VitalPools,
} from '@/shared/lib/engine-wasm'

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


