/**
 * Cansaço ladder — helpers computacionais para Fome/Sede + Sono forçado.
 *
 * PDF Cap 7 p318 (Sono + Fome e Sede). Ambos usam efeito "Cansaço"
 * (p221): "Diminui as capacidades físicas do alvo. Construtos e
 * mortos-vivos são imunes."
 *
 * Complementa entradas em prosa de `environmental-hazards.ts`
 * (fome-sede + sono) com math resolver.
 *
 * Cross-ref:
 *  - `rest.ts:sonoForcadoFortCd(previousChecks)` — mesma fórmula,
 *    mantido lá para compatibilidade da API existente.
 *  - `environmental-hazards.ts:escalatingFortitudeCd(base, prev)` —
 *    fórmula genérica reutilizada aqui.
 */

/** Estágios do ladder de cansaço para Fome/Sede (4 níveis + morte). */
export type CansacoStage =
  | 'ok'
  | 'fatigado'
  | 'exausto'
  | 'inconsciente'
  | 'letal'

/** Estágios do ladder de sono (3 níveis, sem letal). */
export type SonoStage = 'ok' | 'fatigado' | 'exausto' | 'inconsciente'

/** PDF p318: 1 dia de tolerância sem comida/água antes dos testes. */
export const FOME_SEDE_TOLERANCIA_DIAS = 1

/** PDF p318: CD base do Fort para Fome/Sede + Sono forçado. */
export const CANSACO_FORT_CD_BASE = 15

/** Aumento de CD por teste anterior (padrão T20). */
export const CANSACO_CD_PROGRESSAO = 1

/**
 * PDF p318 verbatim: "Condições causadas por fome e sede só podem
 * ser curadas por comida e bebida. Metabolismo."
 */
export const FOME_SEDE_RECOVERY_METHOD = 'comida-bebida' as const

/** PDF p318: recuperação de sono forçado exige dormir 8h. */
export const SONO_RECOVERY_METHOD = 'dormir-8h' as const

/** Tipos imunes ao efeito "Cansaço" (PDF p221). */
export const CANSACO_IMUNE_TIPOS: readonly ['construto', 'morto-vivo'] =
  Object.freeze(['construto', 'morto-vivo'])

/**
 * CD do teste de Fortitude por dia sem comida/água (PDF p318).
 * `previousChecks = 0` (primeiro teste após tolerância) → 15.
 */
export function fomeSedeFortCd(previousChecks: number): number {
  if (previousChecks < 0) {
    throw new Error(
      `fomeSedeFortCd: previousChecks must be ≥ 0, got ${previousChecks}`,
    )
  }
  return CANSACO_FORT_CD_BASE + previousChecks * CANSACO_CD_PROGRESSAO
}

/**
 * Ladder de Fome/Sede (PDF p318). Escalada por falhas consecutivas:
 *  0 = ok
 *  1 = fatigado
 *  2 = exausto
 *  3 = inconsciente
 *  4+ = letal
 */
export function fomeSedeStage(consecutiveFailures: number): CansacoStage {
  if (consecutiveFailures < 0) {
    throw new Error(
      `fomeSedeStage: consecutiveFailures must be ≥ 0, got ${consecutiveFailures}`,
    )
  }
  if (consecutiveFailures === 0) return 'ok'
  if (consecutiveFailures === 1) return 'fatigado'
  if (consecutiveFailures === 2) return 'exausto'
  if (consecutiveFailures === 3) return 'inconsciente'
  return 'letal'
}

/**
 * Ladder de Sono forçado (PDF p318). Sem letal (o pior é inconsciente
 * até dormir 8h).
 *  0 = ok
 *  1 = fatigado
 *  2 = exausto
 *  3+ = inconsciente
 */
export function sonoStage(consecutiveFailures: number): SonoStage {
  if (consecutiveFailures < 0) {
    throw new Error(
      `sonoStage: consecutiveFailures must be ≥ 0, got ${consecutiveFailures}`,
    )
  }
  if (consecutiveFailures === 0) return 'ok'
  if (consecutiveFailures === 1) return 'fatigado'
  if (consecutiveFailures === 2) return 'exausto'
  return 'inconsciente'
}

/** True se o tipo de criatura é imune a Cansaço (p221). */
export function isImuneCansaco(tipo: string): boolean {
  return (CANSACO_IMUNE_TIPOS as readonly string[]).includes(tipo)
}
