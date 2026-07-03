/**
 * Viagens + Marcha Forçada (PDF Cap 6 p270 + Tabela 6-4).
 *
 * Regras verbatim:
 *  - "Deslocamento x 0,5 km" por hora
 *  - "Deslocamento por hora x 8 km" por dia (8h de marcha)
 *  - Marcha Forçada: dobra distância/hora; Fort CD 15+n/hora ou 1d6 PV
 *  - Terreno difícil OU clima ruim: distância × ½ (cumulativo)
 *
 * Velocidade do grupo = do membro mais lento (p270).
 * "Correr" no ritmo de viagem NÃO é modelado no PDF por hora — Tabela
 * 6-4 é deslocamento-agnostic (plugue base m, obtenha km/h).
 *
 * Cross-ref:
 *  - `hazards-cansaco.ts` — Fort CD 15+n padrão T20
 *  - `environmental-hazards.ts` — clima extremo (>50°C/<-10°C) usa
 *    fórmula própria (1d6/dia fogo/frio)
 */

// ─── Formulas base p270 ──────────────────────────────────────────────
/** 8h padrão de marcha por dia (Tabela 6-4 nota ²). */
export const TRAVEL_HOURS_PER_DAY = 8

/** Fator km/h por metro de deslocamento (Tabela 6-4 nota ¹). */
export const KM_PER_HOUR_PER_METER = 0.5

// ─── Marcha Forçada p270 ─────────────────────────────────────────────
/** Multiplicador de distância por hora sob marcha forçada. */
export const MARCHA_FORCADA_MULTIPLIER = 2

/** CD base do Fort de marcha forçada. */
export const MARCHA_FORCADA_FORT_CD_BASE = 15

/** Progressão do CD por teste anterior (padrão T20). */
export const MARCHA_FORCADA_CD_PROGRESSAO = 1

/** Dano em PV ao falhar o Fort (verbatim "1d6"). */
export const MARCHA_FORCADA_DANO_FALHA = '1d6' as const

// ─── Modificadores de terreno/clima p270 ─────────────────────────────
/** Fator de redução por terreno difícil (florestas, pântanos, etc). */
export const DIFFICULT_TERRAIN_MULTIPLIER = 0.5

/** Fator de redução por clima ruim (chuva, neblina). */
export const BAD_WEATHER_MULTIPLIER = 0.5

// ─── Tabela 6-4 canônica (p270) ──────────────────────────────────────
export type TravelRow = {
  deslocamentoM: number
  kmPerHour: number
  kmPerDay: number
}

export const TABELA_6_4: readonly TravelRow[] = Object.freeze([
  { deslocamentoM: 4.5, kmPerHour: 2.25, kmPerDay: 18 },
  { deslocamentoM: 6, kmPerHour: 3, kmPerDay: 24 },
  { deslocamentoM: 7.5, kmPerHour: 3.75, kmPerDay: 30 },
  { deslocamentoM: 9, kmPerHour: 4.5, kmPerDay: 36 },
  { deslocamentoM: 12, kmPerHour: 6, kmPerDay: 48 },
])

// ─── Helpers ─────────────────────────────────────────────────────────
/**
 * Distância viajada por hora dado deslocamento em metros.
 * Fórmula p270 (Tabela 6-4 nota ¹): `deslocamentoM × 0,5`.
 */
export function kmPerHour(deslocamentoM: number): number {
  if (deslocamentoM < 0) {
    throw new Error(
      `kmPerHour: deslocamentoM must be ≥ 0, got ${deslocamentoM}`,
    )
  }
  return deslocamentoM * KM_PER_HOUR_PER_METER
}

/**
 * Distância viajada por dia (padrão 8h de marcha).
 * Fórmula p270 (Tabela 6-4 nota ²): `kmPerHour × 8`.
 */
export function kmPerDay(
  deslocamentoM: number,
  hoursPerDay: number = TRAVEL_HOURS_PER_DAY,
): number {
  if (hoursPerDay < 0) {
    throw new Error(
      `kmPerDay: hoursPerDay must be ≥ 0, got ${hoursPerDay}`,
    )
  }
  return kmPerHour(deslocamentoM) * hoursPerDay
}

/** Distância por hora sob marcha forçada (dobra normal). */
export function kmPerHourForced(deslocamentoM: number): number {
  return kmPerHour(deslocamentoM) * MARCHA_FORCADA_MULTIPLIER
}

/**
 * CD do teste de Fortitude por hora de marcha forçada.
 * `previousTests = 0` → 15 (primeira hora).
 */
export function marchaForcadaFortCd(previousTests: number): number {
  if (previousTests < 0) {
    throw new Error(
      `marchaForcadaFortCd: previousTests must be ≥ 0, got ${previousTests}`,
    )
  }
  return (
    MARCHA_FORCADA_FORT_CD_BASE + previousTests * MARCHA_FORCADA_CD_PROGRESSAO
  )
}

/**
 * Aplica modificadores de terreno + clima (p270). Multiplicadores são
 * cumulativos: terreno difícil AND clima ruim → × ¼.
 */
export function applyTerrainWeatherModifier(
  distanceKm: number,
  opts: { difficultTerrain?: boolean; badWeather?: boolean } = {},
): number {
  let result = distanceKm
  if (opts.difficultTerrain) result *= DIFFICULT_TERRAIN_MULTIPLIER
  if (opts.badWeather) result *= BAD_WEATHER_MULTIPLIER
  return result
}

/**
 * Distância por hora da marcha do grupo — usa o deslocamento do
 * membro mais lento (p270 "Velocidade de Viagem").
 */
export function partyKmPerHour(deslocamentosM: readonly number[]): number {
  if (deslocamentosM.length === 0) {
    throw new Error('partyKmPerHour: at least one member required')
  }
  const slowest = Math.min(...deslocamentosM)
  return kmPerHour(slowest)
}
