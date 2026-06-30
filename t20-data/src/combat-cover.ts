/**
 * Cobertura + Camuflagem — PDF Cap 5 (Combate), p238-239, Tabela 5-3.
 *
 * T20 simplifies the D&D 3.5 cover/concealment system:
 *
 *  - **Cobertura** has TWO tiers only (not ¼/½/¾): "Leve" → +5 Defesa,
 *    "Total" → alvo não pode ser atacado. NO Reflex bonus from
 *    cobertura (diverges from D&D 3.5).
 *
 *  - **Camuflagem** ("ocultamento") also has two tiers: "Leve" → 20%
 *    miss; "Total" → 50% miss. Rolled as a parallel **d10** alongside
 *    the d20 attack: result ≤ 2 (leve) or ≤ 5 (total) makes the attack
 *    miss regardless of the d20 total. NOT a reroll.
 *
 * The line-of-sight tracing rule (canto-a-canto entre quadrados) is
 * captured in the description text but not modeled here — the grid
 * layer owns geometry; this module owns the numbers.
 */

export type CoberturaTier = 'none' | 'leve' | 'total'
export type CamuflagemTier = 'none' | 'leve' | 'total'

export const COBERTURA_LEVE_DEFENSE_BONUS = 5
export const CAMUFLAGEM_LEVE_MISS_THRESHOLD = 2 // d10 ≤ 2 → erra
export const CAMUFLAGEM_TOTAL_MISS_THRESHOLD = 5 // d10 ≤ 5 → erra
export const CAMUFLAGEM_DIE_SIDES = 10

export type CoberturaDef = {
  id: CoberturaTier
  name: string
  defenseBonus: number
  /** true if attacks can land at all when the alvo has this cobertura. */
  attackable: boolean
  description: string
  bookPage: number
}

export type CamuflagemDef = {
  id: CamuflagemTier
  name: string
  missChancePct: number
  /** Threshold on a 1d10 parallel roll: d10 ≤ threshold → miss. */
  d10MissThreshold: number
  description: string
  bookPage: number
}

export const COBERTURA: Record<CoberturaTier, CoberturaDef> = Object.freeze({
  none: {
    id: 'none',
    name: 'Sem Cobertura',
    defenseBonus: 0,
    attackable: true,
    description: 'Sem cobertura. Sem bônus.',
    bookPage: 239,
  },
  leve: {
    id: 'leve',
    name: 'Cobertura Leve',
    defenseBonus: COBERTURA_LEVE_DEFENSE_BONUS,
    attackable: true,
    description:
      'Você recebe cobertura leve quando está atrás de algo que bloqueia o ataque dos inimigos — árvore, muralha de castelo, lateral de uma carroça, criatura maior. Cobertura leve fornece +5 na Defesa. No mapa, atacante e alvo escolhem cantos do quadrado em que estão; traça-se uma linha reta entre os cantos. Se a linha é interrompida por obstáculo ou criatura, o alvo tem cobertura leve. Não recebe cobertura se a linha seguir ao longo do obstáculo ou apenas tocar sua ponta.',
    bookPage: 239,
  },
  total: {
    id: 'total',
    name: 'Cobertura Total',
    defenseBonus: 0,
    attackable: false,
    description:
      'Você recebe cobertura total quando seus inimigos não podem alcançá-lo — por exemplo, atrás de uma parede. Cobertura total impede que você seja atacado.',
    bookPage: 239,
  },
})

export const CAMUFLAGEM: Record<CamuflagemTier, CamuflagemDef> = Object.freeze({
  none: {
    id: 'none',
    name: 'Sem Camuflagem',
    missChancePct: 0,
    d10MissThreshold: 0,
    description: 'Sem camuflagem. Sem chance de falha.',
    bookPage: 238,
  },
  leve: {
    id: 'leve',
    name: 'Camuflagem',
    missChancePct: 20,
    d10MissThreshold: CAMUFLAGEM_LEVE_MISS_THRESHOLD,
    description:
      'Você recebe camuflagem leve quando um efeito dificulta a visão dos inimigos — escuridão leve, neblina, folhagens ou efeito similar no local onde você está ou no espaço entre você e o oponente. Ao atacar, o atacante rola 1d10 junto com o d20 do ataque; d10 ≤ 2 = ataque erra, independentemente do resultado do d20.',
    bookPage: 238,
  },
  total: {
    id: 'total',
    name: 'Camuflagem Total',
    missChancePct: 50,
    d10MissThreshold: CAMUFLAGEM_TOTAL_MISS_THRESHOLD,
    description:
      'Você recebe camuflagem total quando um efeito impede a visão dos inimigos — por exemplo, câmara em escuridão total. Rola-se 1d10 junto com o d20 do ataque; d10 ≤ 5 = ataque erra, independentemente do d20.',
    bookPage: 239,
  },
})

/**
 * Effective Defesa do alvo após cobertura. Returns the base Defesa
 * unchanged for `none`/`total` (total = no attack possible; caller
 * should short-circuit via {@link canBeAttacked} first).
 */
export function defesaComCobertura(
  baseDefesa: number,
  cobertura: CoberturaTier,
): number {
  return baseDefesa + COBERTURA[cobertura].defenseBonus
}

/**
 * Whether an attack can even be attempted against a target. Returns
 * `false` only for cobertura total.
 */
export function canBeAttacked(cobertura: CoberturaTier): boolean {
  return COBERTURA[cobertura].attackable
}

/**
 * Resolve the camuflagem miss roll. The atacante rolled a parallel d10
 * alongside the d20 attack; returns true if the camuflagem makes the
 * attack miss (regardless of the d20 outcome).
 *
 * Throws if `d10Result` is outside [1, 10] — caller must roll a real
 * d10.
 */
export function camuflagemCausesMiss(
  d10Result: number,
  camuflagem: CamuflagemTier,
): boolean {
  if (d10Result < 1 || d10Result > CAMUFLAGEM_DIE_SIDES) {
    throw new Error(
      `d10Result must be 1..${CAMUFLAGEM_DIE_SIDES}, got ${d10Result}`,
    )
  }
  const threshold = CAMUFLAGEM[camuflagem].d10MissThreshold
  return d10Result <= threshold
}
