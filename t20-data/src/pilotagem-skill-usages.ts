/**
 * Perícia Pilotagem (DES, treinada, sem penalidade de armadura) — 1 uso.
 *
 * PDF Cap 2 Perícias — Pilotagem (livro p122). Header verbatim:
 * "PILOTAGEM — DES · TREINADA".
 * Corpo verbatim: "Você sabe operar veículos como carroças, barcos e
 * balões. Ações simples não exigem testes — você pode atrelhar seus
 * trobos a uma carroça e conduzi-la pela estrada, ou levantar âncora
 * e velejar seu navio em águas tranquilas, automaticamente. Porém,
 * conduzir um veículo em situações ruins (terreno acidentado para
 * veículos terrestres, chuva ou ventania para veículos aquáticos ou
 * aéreos), exige uma ação de movimento e um teste de Pilotagem contra
 * CD 15 por turno ou cena, de acordo com o mestre. Se falhar, você
 * avança metade do deslocamento. Se falhar por 5 ou mais, se acidenta
 * de alguma forma. Situações extremas (terreno com obstáculos,
 * tempestade...) aumentam a CD para 25."
 *
 * Único uso:
 *  - Conduzir Veículo (p122) — ação de movimento; CD por condição:
 *    simples auto, ruim CD 15, extrema CD 25 (por turno ou cena,
 *    a critério do mestre). Falha = metade do deslocamento; falha
 *    por 5+ = acidenta-se.
 *
 * Nota Tabela 2-1 p115: APENAS TREINADA; NÃO sofre penalidade de
 * armadura. Cross-ref externa: Adestramento/Manejar Animal (p115)
 * pode substituir Pilotagem para veículos de tração animal.
 */

import { makeUsageByKind } from './skill-usage-resolver'

// ─── Types ────────────────────────────────────────────────────────────
export type PilotagemUsageKind = 'conduzir-veiculo'

/** Categoria do veículo pilotado (p122 verbatim). */
export type VehicleKind = 'terrestre' | 'aquatico' | 'aereo'

/** Condição/situação da condução (p122). */
export type PilotagemCondition = 'simples' | 'ruim' | 'extrema'

/** Cadência dos testes em situação ruim/extrema (a critério do mestre). */
export type TestCadence = 'por-turno' | 'por-cena'

type UsageCommon = {
  id: PilotagemUsageKind
  name: string
  effect: string
  bookPage: 122
}

export type PilotagemConduzirVeiculo = UsageCommon & {
  kind: 'conduzir-veiculo'
  action: 'movimento'
  cdRuim: 15
  cdExtrema: 25
  /** Falha simples avança metade do deslocamento. */
  halfSpeedOnFailure: true
  /** Falha por 5+ resulta em acidente. */
  crashMargin: 5
}

export type PilotagemUsage = PilotagemConduzirVeiculo

// ─── Constantes ──────────────────────────────────────────────────────
// Conduzir Veículo (p122 verbatim)
export const CONDUZIR_VEICULO_CD_RUIM = 15
export const CONDUZIR_VEICULO_CD_EXTREMA = 25
export const CONDUZIR_VEICULO_CRASH_MARGIN = 5

// Flags Tabela 2-1 p115
export const PILOTAGEM_TRAINED_ONLY = true
export const PILOTAGEM_ARMOR_PENALTY = false

// ─── Catálogo ─────────────────────────────────────────────────────────
export const PILOTAGEM_USAGES: readonly PilotagemUsage[] = Object.freeze([
  {
    id: 'conduzir-veiculo',
    kind: 'conduzir-veiculo',
    name: 'Conduzir Veículo',
    action: 'movimento',
    cdRuim: 15,
    cdExtrema: 25,
    halfSpeedOnFailure: true,
    crashMargin: 5,
    effect:
      'Ações simples automáticas; situação ruim CD 15 (movimento, por turno/cena); situação extrema CD 25; falha = metade do deslocamento; falha 5+ = acidenta-se.',
    bookPage: 122,
  },
])

export const pilotagemUsageByKind = makeUsageByKind<PilotagemUsageKind, PilotagemUsage>(
  PILOTAGEM_USAGES,
  'pilotagemUsageByKind',
)

// ─── Helpers — Conduzir Veículo ─────────────────────────────────────
/**
 * CD por condição:
 *  - simples → null (automático, não exige teste)
 *  - ruim → 15
 *  - extrema → 25
 */
export function pilotagemCd(condition: PilotagemCondition): number | null {
  switch (condition) {
    case 'simples':
      return null
    case 'ruim':
      return CONDUZIR_VEICULO_CD_RUIM
    case 'extrema':
      return CONDUZIR_VEICULO_CD_EXTREMA
  }
}

export type PilotagemOutcome = 'success' | 'half-speed' | 'crashes'

/**
 * Resolve Pilotagem (Conduzir Veículo em situação ruim/extrema):
 *  - roll ≥ CD → success.
 *  - falha por < 5 → half-speed (avança metade do deslocamento).
 *  - falha por ≥ 5 → crashes (acidente).
 */
export function pilotagemOutcome(
  rollResult: number,
  cd: number,
): PilotagemOutcome {
  const delta = rollResult - cd
  if (delta >= 0) return 'success'
  if (Math.abs(delta) >= CONDUZIR_VEICULO_CRASH_MARGIN) return 'crashes'
  return 'half-speed'
}
