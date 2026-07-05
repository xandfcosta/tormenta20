/**
 * Clima — PDF Cap 6 p267. Três eixos independentes:
 *
 *   1. Calor/Frio: thresholds de temperatura + Fort CD 15 (cumulativo)
 *      por dia (ou por minuto em extremos), dano por falha.
 *
 *   2. Precipitações: chuva / granizo / neve / tempestade. Cada uma tem
 *      penalidade de Percepção, efeito de vento associado + eventual
 *      dano de rodada.
 *
 *   3. Vento: forte / vendaval / furacão / tornado. Penalidade de ataque
 *      à distância, chamas/névoas, e para furacão/tornado uma Fort CD
 *      que causa queda + arrasto + dano.
 *
 * Neblina (p267): fornece camuflagem, mas não tem CD/dano por si só —
 * codificada em `environmental-hazards.ts` como concealment; fica de
 * fora aqui.
 */

// ─── Calor e Frio (p267) ─────────────────────────────────────────────

/** Temperatura acima da qual o Fort CD começa (°C). */
export const HEAT_THRESHOLD_C = 50
/** Temperatura abaixo da qual o Fort CD começa (°C). */
export const COLD_THRESHOLD_C = -10
/** Temperatura acima da qual o teste passa a ser por minuto (°C). */
export const HEAT_EXTREME_C = 60
/** Temperatura abaixo da qual o teste passa a ser por minuto (°C). */
export const COLD_EXTREME_C = -20

export type TemperatureExposure =
  | { kind: 'safe' }
  | {
      kind: 'hot' | 'cold'
      baseCd: 15
      /** true = teste por minuto; false = teste por dia. */
      extreme: boolean
      damagePerFailure: '1d6'
      damageType: 'fogo' | 'frio'
    }

export function temperatureExposure(temperatureC: number): TemperatureExposure {
  if (temperatureC >= HEAT_EXTREME_C) {
    return {
      kind: 'hot',
      baseCd: 15,
      extreme: true,
      damagePerFailure: '1d6',
      damageType: 'fogo',
    }
  }
  if (temperatureC > HEAT_THRESHOLD_C) {
    return {
      kind: 'hot',
      baseCd: 15,
      extreme: false,
      damagePerFailure: '1d6',
      damageType: 'fogo',
    }
  }
  if (temperatureC <= COLD_EXTREME_C) {
    return {
      kind: 'cold',
      baseCd: 15,
      extreme: true,
      damagePerFailure: '1d6',
      damageType: 'frio',
    }
  }
  if (temperatureC < COLD_THRESHOLD_C) {
    return {
      kind: 'cold',
      baseCd: 15,
      extreme: false,
      damagePerFailure: '1d6',
      damageType: 'frio',
    }
  }
  return { kind: 'safe' }
}

/**
 * CD do teste seguinte na sequência acumulativa "CD 15 + 1 por teste
 * anterior" (p267 tanto Calor/Frio quanto Marcha Forçada seguem esse
 * padrão). `priorFailures` = quantos testes já falharam antes.
 */
export function heatColdCdAfter(priorTests: number): number {
  if (priorTests < 0 || !Number.isInteger(priorTests)) {
    throw new Error(`priorTests must be a non-negative integer, got ${priorTests}`)
  }
  return 15 + priorTests
}

// ─── Precipitações (p267) ────────────────────────────────────────────

export const PRECIPITATIONS = [
  'chuva',
  'granizo',
  'neve',
  'tempestade',
] as const

export type Precipitation = (typeof PRECIPITATIONS)[number]

export type PrecipitationRow = {
  kind: Precipitation
  label: string
  perceptionPenalty: number
  /** Efeitos de vento equivalente (p267 encadeia precipitação em vento). */
  windEquivalent: 'vento-forte' | 'vendaval'
  /** Efeito extra por rodada (dano/terreno). */
  perRoundEffect:
    | { kind: 'none' }
    | { kind: 'impact', damage: '1' }
    | { kind: 'difficult-terrain' }
    | { kind: 'lightning-strike', chancePct: 10, damage: '8d10', damageType: 'eletricidade' }
}

export const PRECIPITATION_TABLE: readonly PrecipitationRow[] = Object.freeze([
  {
    kind: 'chuva',
    label: 'Chuva',
    perceptionPenalty: -5,
    windEquivalent: 'vento-forte',
    perRoundEffect: { kind: 'none' },
  },
  {
    kind: 'granizo',
    label: 'Granizo',
    perceptionPenalty: -5,
    windEquivalent: 'vento-forte',
    perRoundEffect: { kind: 'impact', damage: '1' },
  },
  {
    kind: 'neve',
    label: 'Neve',
    perceptionPenalty: -5,
    windEquivalent: 'vento-forte',
    perRoundEffect: { kind: 'difficult-terrain' },
  },
  {
    kind: 'tempestade',
    label: 'Tempestade',
    perceptionPenalty: -10,
    windEquivalent: 'vendaval',
    perRoundEffect: {
      kind: 'lightning-strike',
      chancePct: 10,
      damage: '8d10',
      damageType: 'eletricidade',
    },
  },
])

export function precipitationRow(kind: Precipitation): PrecipitationRow {
  const row = PRECIPITATION_TABLE.find((r) => r.kind === kind)
  if (!row) throw new Error(`Unknown Precipitation: ${kind}`)
  return row
}

// ─── Vento (p267) ────────────────────────────────────────────────────

export const WINDS = ['vento-forte', 'vendaval', 'furacao', 'tornado'] as const

export type Wind = (typeof WINDS)[number]

export type WindRow = {
  kind: Wind
  label: string
  /** Penalidade em teste de ataque à distância. */
  rangedAttackPenalty: number | 'impossivel'
  /** Comportamento com chamas ao ar livre. */
  chamas: 'nenhum' | '50pct-por-rodada-apaga' | 'apaga'
  /** Comportamento com névoas ao ar livre. */
  nevoas: 'nenhum' | '50pct-por-rodada-dissipa' | 'dissipa'
  /**
   * Efeito por rodada para criaturas até certo tamanho. `null` = sem
   * efeito (vento-forte, vendaval). Furacão / tornado exigem Fort CD e
   * causam queda + arrasto + dano de impacto ao falhar.
   */
  knockdown: null | {
    fortCd: number
    /** Maior tamanho ainda afetado. Furacão até Médio; tornado até Grande. */
    maxAffectedSize: 'medio' | 'grande'
    /** Distância arrastada (multiplicado por 1,5m). */
    dragDice: '1d4' | '1d12'
    dragDirection: 'direcao-do-vento' | 'aleatoria'
    damagePer1_5m: '1d6'
    damageType: 'impacto'
  }
}

export const WIND_TABLE: readonly WindRow[] = Object.freeze([
  {
    kind: 'vento-forte',
    label: 'Vento Forte',
    rangedAttackPenalty: -2,
    chamas: '50pct-por-rodada-apaga',
    nevoas: '50pct-por-rodada-dissipa',
    knockdown: null,
  },
  {
    kind: 'vendaval',
    label: 'Vendaval',
    rangedAttackPenalty: -5,
    chamas: 'apaga',
    nevoas: 'dissipa',
    knockdown: null,
  },
  {
    kind: 'furacao',
    label: 'Furacão',
    rangedAttackPenalty: 'impossivel',
    chamas: 'apaga',
    nevoas: 'dissipa',
    knockdown: {
      fortCd: 15,
      maxAffectedSize: 'medio',
      dragDice: '1d4',
      dragDirection: 'direcao-do-vento',
      damagePer1_5m: '1d6',
      damageType: 'impacto',
    },
  },
  {
    kind: 'tornado',
    label: 'Tornado',
    rangedAttackPenalty: 'impossivel',
    chamas: 'apaga',
    nevoas: 'dissipa',
    knockdown: {
      fortCd: 25,
      maxAffectedSize: 'grande',
      dragDice: '1d12',
      dragDirection: 'aleatoria',
      damagePer1_5m: '1d6',
      damageType: 'impacto',
    },
  },
])

export function windRow(kind: Wind): WindRow {
  const row = WIND_TABLE.find((r) => r.kind === kind)
  if (!row) throw new Error(`Unknown Wind: ${kind}`)
  return row
}
