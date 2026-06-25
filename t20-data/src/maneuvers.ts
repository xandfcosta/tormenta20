/**
 * Manobras de combate — PDF book p233-234.
 *
 * "Uma manobra é um ataque corpo a corpo para fazer algo diferente de
 *  causar dano — como arrancar a arma do oponente ou empurrá-lo para um
 *  abismo. Não é possível fazer manobras de combate com ataques à
 *  distância." (p234)
 *
 * Core mechanic: ação padrão, opposed Luta test. Even if the defender
 * is wielding a ranged weapon, they oppose with their **Luta** value.
 * Ties go to the higher Luta bonus; same bonus → reroll.
 *
 * Ranged attacks targeting a creature involved in Agarrar have 50%
 * chance of hitting the wrong target.
 */
export const MANEUVER_IDS = [
  'agarrar',
  'derrubar',
  'desarmar',
  'empurrar',
  'quebrar',
] as const

export type ManeuverId = (typeof MANEUVER_IDS)[number]

export type ManeuverAction = 'padrao' | 'livre'

/**
 * The defender's roll: for most manobras it's their own Luta. Fintar
 * (not coded — sits between manobra and ação padrão in the book) uses
 * Reflexos. Quebrar against a held item: opposes Luta. Against a loose
 * item: a fixed attack vs the item's Defesa (not modeled here).
 */
export type ManeuverDefenderRoll = 'luta'

export type Maneuver = {
  id: ManeuverId
  name: string
  action: ManeuverAction
  /** Opposed roll: attacker uses Luta, defender uses one of these. */
  defenderRoll: ManeuverDefenderRoll
  /** PDF-paraphrased success text. */
  successEffect: string
  /** Whether the manobra grants an extra effect when the attacker wins
   *  by 5+ over the defender's roll (book "vencer por 5 pontos ou
   *  mais"). */
  hasFiveOverBonus: boolean
}

export const MANEUVERS: Record<ManeuverId, Maneuver> = {
  agarrar: {
    id: 'agarrar',
    name: 'Agarrar',
    action: 'padrao',
    defenderRoll: 'luta',
    successEffect:
      'Alvo fica agarrado: desprevenido, imóvel, -2 em ataques, só armas leves. Atacante mantém uma das mãos ocupada e tem deslocamento pela metade enquanto arrasta.',
    hasFiveOverBonus: false,
  },
  derrubar: {
    id: 'derrubar',
    name: 'Derrubar',
    action: 'padrao',
    defenderRoll: 'luta',
    successEffect: 'Alvo fica caído.',
    hasFiveOverBonus: true,
  },
  desarmar: {
    id: 'desarmar',
    name: 'Desarmar',
    action: 'padrao',
    defenderRoll: 'luta',
    successEffect:
      'O item segurado pelo alvo cai no chão (mesmo quadrado, salvo regra contrária).',
    hasFiveOverBonus: true,
  },
  empurrar: {
    id: 'empurrar',
    name: 'Empurrar',
    action: 'padrao',
    defenderRoll: 'luta',
    successEffect:
      'Empurra o alvo 1,5m, +1,5m a cada 5 pontos de margem; atacante pode gastar uma ação de movimento para avançar junto.',
    hasFiveOverBonus: false,
  },
  quebrar: {
    id: 'quebrar',
    name: 'Quebrar',
    action: 'padrao',
    defenderRoll: 'luta',
    successEffect:
      'Acerta o item segurado pelo alvo e causa dano normal; item destruído a 0 PV (consulte estatísticas de objetos, p239).',
    hasFiveOverBonus: false,
  },
}

export type ManeuverOutcome = {
  success: boolean
  /** Margin = attackerTotal - defenderTotal; only meaningful on success. */
  margin: number
  /** True when the manobra grants its five-over bonus
   *  (`hasFiveOverBonus && success && margin >= 5`). */
  fiveOverBonus: boolean
}

/**
 * Resolve a manobra outcome from the rolled totals.
 *
 *   ties go to the higher Luta bonus per PDF; the caller resolves the
 *   tie before passing the *total* values here (i.e., pre-tiebroken
 *   numbers). If they are still equal after tiebreak, the PDF says
 *   "reroll"; in that case, treat the manobra as a *failure* until the
 *   caller reruns the resolver with the new rolls.
 */
export function maneuverOutcome(
  id: ManeuverId,
  attackerTotal: number,
  defenderTotal: number,
): ManeuverOutcome {
  const success = attackerTotal > defenderTotal
  const margin = success ? attackerTotal - defenderTotal : 0
  const entry = MANEUVERS[id]
  return {
    success,
    margin,
    fiveOverBonus: success && entry.hasFiveOverBonus && margin >= 5,
  }
}
