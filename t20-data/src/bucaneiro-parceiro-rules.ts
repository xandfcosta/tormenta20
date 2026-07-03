/**
 * Bucaneiro — poderes que geram parceiros (PDF p47).
 *
 * "Amigos no Porto" é o único poder do Bucaneiro que gera parceiro.
 * Concede um veterano de tipo escolhido por 1 dia apenas.
 *
 * Deltas vs Autoridade Feudal (Cavaleiro/Nobre p54/p79):
 *  - Mesmo minLevel 6, MAS tier veterano (não iniciante).
 *  - Duração 1 dia (não "até fim da aventura").
 *  - Requer teste de Carisma CD 10 (não gasto de PM).
 *  - Localização: comunidade portuária (não influência genérica).
 *
 * Cross-ref: `parceiro-rules.ts:parceiroLimit` — conta contra o limite.
 */

import type { ParceiroTier } from './parceiro-rules'
import type { ParceiroDuration } from './nobre-parceiro-rules'

export type BucaneiroParceiroPowerId = 'amigos-no-porto'

export type BucaneiroParceiroPower = {
  id: BucaneiroParceiroPowerId
  name: string
  bookPage: 47
  minLevel: number
  countsAgainstLimit: boolean
  grantedTier: ParceiroTier
  duration: ParceiroDuration
  /** CD do teste de perícia para acionar. null = automático. */
  activationCd: number | null
  /** Perícia usada no teste de ativação. */
  activationSkill: string | null
  additionalPrereqs: readonly string[]
}

export const AMIGOS_NO_PORTO_MIN_LEVEL = 6
export const AMIGOS_NO_PORTO_ACTIVATION_CD = 10

const RAW: readonly BucaneiroParceiroPower[] = [
  {
    id: 'amigos-no-porto',
    name: 'Amigos no Porto',
    bookPage: 47,
    minLevel: AMIGOS_NO_PORTO_MIN_LEVEL,
    countsAgainstLimit: true,
    grantedTier: 'veterano',
    duration: 'dia',
    activationCd: AMIGOS_NO_PORTO_ACTIVATION_CD,
    activationSkill: 'Carisma',
    additionalPrereqs: ['Car 1', 'comunidade portuária'],
  },
]

export const BUCANEIRO_PARCEIRO_POWERS: readonly BucaneiroParceiroPower[] =
  Object.freeze(RAW)

export function bucaneiroParceiroPowerById(
  id: BucaneiroParceiroPowerId,
): BucaneiroParceiroPower | undefined {
  return BUCANEIRO_PARCEIRO_POWERS.find((p) => p.id === id)
}

export function unlockedBucaneiroParceiroPowers(
  bucaneiroLevel: number,
): readonly BucaneiroParceiroPower[] {
  if (bucaneiroLevel < 1) {
    throw new Error(
      `unlockedBucaneiroParceiroPowers: bucaneiroLevel must be ≥ 1, got ${bucaneiroLevel}`,
    )
  }
  return BUCANEIRO_PARCEIRO_POWERS.filter((p) => p.minLevel <= bucaneiroLevel)
}
