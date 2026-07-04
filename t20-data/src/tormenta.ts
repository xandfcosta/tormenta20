/**
 * Tormenta — contaminação, poderes, e insanidade dos lefeu.
 *
 * Sources:
 *  - Book p319 (Cap 7 Ameaças, entry "Tormenta") — daily save + area
 *    side effects + escalating condition ladder.
 *  - Book p127-128 + p135-137 — Poderes da Tormenta catalog and the
 *    Carisma-loss rule.
 *  - Book p313-316 — Lefeu monster statlines including the
 *    *Insanidade da Tormenta* ability (PM drain on sight).
 *  - Book p23 — Lefou racial `Cria da Tormenta` resistance.
 *
 * Key T20-isms encoded here:
 *  - There is NO "Pontos de Tormenta" score. Contamination is
 *    represented by escalating *condições* (frustrado → esmorecido →
 *    confuso → NPC insano) from failed daily Vontade saves.
 *  - There is NO mutation table tied to a points threshold. Mutations
 *    are voluntary `Poderes da Tormenta`; each one costs Carisma.
 *  - Lefeu / lefou are flatly immune to area effects (book p319).
 *  - The condition ladder is cleared by leaving the área (conditions
 *    last "pelo dia"); permanent power-based Carisma loss is NOT
 *    cleared by rest.
 */

/** Auto-applied condition the moment a character enters a área de Tormenta. */
export const TORMENTA_AREA_ENTRY_CONDITION = 'frustrado' as const

/** Save type & base CD for the daily area Vontade test (book p319). */
export const TORMENTA_AREA_SAVE_TYPE = 'vontade' as const
export const TORMENTA_AREA_BASE_CD = 25
export const TORMENTA_AREA_CD_PER_PRIOR_DAY = 2

/** While inside an área de Tormenta, every habilidade PM-fueled costs +2 PM. */
export const TORMENTA_AREA_PM_SURCHARGE = 2

/**
 * PV/PM recovery from rest is halved inside the área (book p319).
 * Multiplier applied AFTER any other rest modifier.
 */
export const TORMENTA_AREA_REST_FACTOR = 0.5

/**
 * Itens mágicos perdem **um encantamento** ao entrar na área. The bearer
 * picks which (book p319).
 */
export const TORMENTA_AREA_ENCHANT_LOSS = 1

/**
 * NPC conversion threshold from Poderes da Tormenta (book p136):
 * "Um personagem reduzido a menos que Car –5 torna-se um NPC sob
 * controle do mestre."
 */
export const TORMENTA_NPC_CARISMA_THRESHOLD = -5

/**
 * Estágios da escalada de insanidade dentro de uma área de Tormenta.
 *
 * Cadência (book p319):
 *  - Ao entrar, sem teste, o personagem fica `frustrado` para o dia.
 *  - No início de cada dia, faz teste de Vontade contra `tormentaAreaDailyCd`.
 *  - Falha: avança para o próximo estágio.
 *  - `insano-npc` é terminal — o personagem deixa o controle do jogador.
 */
export type TormentaInsanityStage =
  | 'none'
  | 'frustrado'
  | 'esmorecido'
  | 'confuso'
  | 'insano-npc'

const STAGE_NEXT: Readonly<Record<TormentaInsanityStage, TormentaInsanityStage>> = {
  none: 'frustrado',
  frustrado: 'esmorecido',
  esmorecido: 'confuso',
  confuso: 'insano-npc',
  'insano-npc': 'insano-npc',
}

/**
 * CD for the daily Vontade save (book p319):
 *   CD = 25 + 2 × (dias consecutivos anteriores no zone)
 *
 * `consecutivePriorDays` counts days already spent (so day 1 → CD 25,
 * day 2 → CD 27, …).
 */
export function tormentaAreaDailyCd(consecutivePriorDays: number): number {
  if (consecutivePriorDays < 0) {
    throw new Error(
      `tormentaAreaDailyCd: consecutivePriorDays must be ≥ 0, got ${consecutivePriorDays}`,
    )
  }
  return (
    TORMENTA_AREA_BASE_CD +
    TORMENTA_AREA_CD_PER_PRIOR_DAY * consecutivePriorDays
  )
}

/** Advance one rung on the condition ladder; `insano-npc` is sticky. */
export function escalateTormentaInsanity(
  current: TormentaInsanityStage,
): TormentaInsanityStage {
  return STAGE_NEXT[current]
}

/**
 * Whether the character is now an NPC controlled by the GM, EITHER from
 * the daily-save escalation reaching `insano-npc` OR from Carisma being
 * reduced below the threshold by power acquisition.
 */
export function isTormentaNpc(args: {
  insanityStage: TormentaInsanityStage
  carismaAfterLoss: number
}): boolean {
  return (
    args.insanityStage === 'insano-npc' ||
    args.carismaAfterLoss < TORMENTA_NPC_CARISMA_THRESHOLD
  )
}

/**
 * Poder da Tormenta catalog (book p127-128 + p135-137). Each is a
 * permanent voluntary mutation. Some require N other powers already
 * taken before they unlock.
 */
export type TormentaPowerId =
  | 'anatomia-insana'
  | 'antenas'
  | 'armamento-aberrante'
  | 'articulacoes-flexiveis'
  | 'asas-insetoides'
  | 'carapaca'
  | 'corpo-aberrante'
  | 'cuspir-enxame'
  | 'dentes-afiados'
  | 'desprezar-a-realidade'
  | 'empunhadura-rubra'
  | 'fome-de-mana'
  | 'larva-explosiva'
  | 'legiao-aberrante'
  | 'maos-membranosas'
  | 'membros-estendidos'
  | 'membros-extras'
  | 'mente-aberrante'
  | 'olhos-vermelhos'
  | 'pele-corrompida'
  | 'sangue-acido'
  | 'visco-rubro'

export type TormentaPower = {
  id: TormentaPowerId
  name: string
  /** Number of OTHER Tormenta powers required to unlock this one. */
  requiresOtherPowers: number
  /** Specific prerequisite power, when stated. */
  requiresPower?: TormentaPowerId
}

const POWERS: readonly TormentaPower[] = [
  { id: 'anatomia-insana', name: 'Anatomia Insana', requiresOtherPowers: 0 },
  { id: 'antenas', name: 'Antenas', requiresOtherPowers: 0 },
  { id: 'armamento-aberrante', name: 'Armamento Aberrante', requiresOtherPowers: 1 },
  { id: 'articulacoes-flexiveis', name: 'Articulações Flexíveis', requiresOtherPowers: 0 },
  { id: 'asas-insetoides', name: 'Asas Insetoides', requiresOtherPowers: 4 },
  { id: 'carapaca', name: 'Carapaça', requiresOtherPowers: 0 },
  { id: 'corpo-aberrante', name: 'Corpo Aberrante', requiresOtherPowers: 1 },
  { id: 'cuspir-enxame', name: 'Cuspir Enxame', requiresOtherPowers: 0 },
  { id: 'dentes-afiados', name: 'Dentes Afiados', requiresOtherPowers: 0 },
  { id: 'desprezar-a-realidade', name: 'Desprezar a Realidade', requiresOtherPowers: 4 },
  { id: 'empunhadura-rubra', name: 'Empunhadura Rubra', requiresOtherPowers: 0 },
  { id: 'fome-de-mana', name: 'Fome de Mana', requiresOtherPowers: 0 },
  {
    id: 'larva-explosiva',
    name: 'Larva Explosiva',
    requiresOtherPowers: 0,
    requiresPower: 'dentes-afiados',
  },
  {
    id: 'legiao-aberrante',
    name: 'Legião Aberrante',
    requiresOtherPowers: 3,
    requiresPower: 'anatomia-insana',
  },
  { id: 'maos-membranosas', name: 'Mãos Membranosas', requiresOtherPowers: 0 },
  { id: 'membros-estendidos', name: 'Membros Estendidos', requiresOtherPowers: 0 },
  { id: 'membros-extras', name: 'Membros Extras', requiresOtherPowers: 4 },
  { id: 'mente-aberrante', name: 'Mente Aberrante', requiresOtherPowers: 0 },
  { id: 'olhos-vermelhos', name: 'Olhos Vermelhos', requiresOtherPowers: 0 },
  { id: 'pele-corrompida', name: 'Pele Corrompida', requiresOtherPowers: 0 },
  { id: 'sangue-acido', name: 'Sangue Ácido', requiresOtherPowers: 0 },
  { id: 'visco-rubro', name: 'Visco Rubro', requiresOtherPowers: 0 },
]

export const TORMENTA_POWERS: Readonly<Record<TormentaPowerId, TormentaPower>> =
  Object.freeze(
    POWERS.reduce<Record<TormentaPowerId, TormentaPower>>((acc, p) => {
      acc[p.id] = p
      return acc
    }, {} as Record<TormentaPowerId, TormentaPower>),
  )

export const TORMENTA_POWER_IDS: readonly TormentaPowerId[] = POWERS.map((p) => p.id)

/**
 * Whether the picker may add `candidate` to their pool, given the
 * currently-held powers. Checks both `requiresOtherPowers` and
 * `requiresPower`.
 */
export function canTakePower(
  candidate: TormentaPowerId,
  held: readonly TormentaPowerId[],
): boolean {
  const power = TORMENTA_POWERS[candidate]
  if (held.includes(candidate)) return false
  if (power.requiresPower && !held.includes(power.requiresPower)) return false
  const othersCount = power.requiresPower
    ? held.length - 1
    : held.length
  if (othersCount < power.requiresOtherPowers) return false
  return true
}

/**
 * Total Carisma loss from taking N Tormenta powers.
 *
 * Rule (book p136): "Quando escolhe um poder da Tormenta, você perde 1
 * de Carisma. Para cada dois outros poderes da Tormenta, você perde
 * mais 1 de Carisma." Interpreted incrementally: picking the N-th
 * power costs `1 + floor((N-1) / 2)` Carisma (so 0 prior others adds
 * +0, 2 prior others adds +1, 4 prior others adds +2, …).
 *
 * Sequence: 1 → 2 → 4 → 6 → 9 → 12 → 16 → 20 → 25 → …
 */
export function carismaLossFromPowers(powerCount: number): number {
  if (powerCount < 0) {
    throw new Error(
      `carismaLossFromPowers: powerCount must be ≥ 0, got ${powerCount}`,
    )
  }
  let total = 0
  for (let k = 1; k <= powerCount; k++) {
    total += 1 + Math.floor((k - 1) / 2)
  }
  return total
}

/**
 * Lefeu insanity (book p313-316 monster ability): sight of one or more
 * lefeu forces a Vontade test; on failure the viewer loses PM. Reaching
 * 0 PM imposes `confuso`. Once per day per viewer.
 *
 * Sample lefeu CDs / drains pinned from the book monster section:
 *   uktril       — ND 3,  Von CD 17, 1d6 PM
 *   geraktril    — ND 6,  Von CD 22, 2d4 PM
 *   reishid      — ND 8,  Von CD 26, 2d6 PM
 *   thuwarokk    — ND 16, Von CD 42, 2d12 PM
 */
export type LefeuInsanity = {
  saveCd: number
  pmDrainDice: string
  nd: number
}

export const LEFEU_INSANITY_SAMPLES: Readonly<Record<string, LefeuInsanity>> = {
  uktril: { saveCd: 17, pmDrainDice: '1d6', nd: 3 },
  geraktril: { saveCd: 22, pmDrainDice: '2d4', nd: 6 },
  reishid: { saveCd: 26, pmDrainDice: '2d6', nd: 8 },
  thuwarokk: { saveCd: 42, pmDrainDice: '2d12', nd: 16 },
}

/** Lefou racial (book p23) — +5 a testes de resistência vs Tormenta. */
export const LEFOU_TORMENTA_RESISTANCE_BONUS = 5
