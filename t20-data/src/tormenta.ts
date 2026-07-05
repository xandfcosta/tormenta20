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
  /**
   * Verbatim rule text lifted from PDF Cap 2 p135-137. Every power in
   * this catalog now carries a description so consumers can render
   * without a second lookup. See `feat(t20-data): 22 tormenta descs`.
   */
  description: string
  /** Number of OTHER Tormenta powers required to unlock this one. */
  requiresOtherPowers: number
  /** Specific prerequisite power, when stated. */
  requiresPower?: TormentaPowerId
  /** Book page anchor (135, 136 or 137). */
  bookPage: 135 | 136 | 137
}

const POWERS: readonly TormentaPower[] = [
  {
    id: 'anatomia-insana',
    name: 'Anatomia Insana',
    description:
      '25% de chance (resultado "1" em 1d4) de ignorar o dano adicional de um acerto crítico ou ataque furtivo. A chance aumenta em +25% para cada dois outros poderes da Tormenta que você possui.',
    requiresOtherPowers: 0,
    bookPage: 136,
  },
  {
    id: 'antenas',
    name: 'Antenas',
    description:
      '+1 em Iniciativa, Percepção e Vontade. Este bônus aumenta em +1 para cada dois outros poderes da Tormenta que você possui.',
    requiresOtherPowers: 0,
    bookPage: 136,
  },
  {
    id: 'armamento-aberrante',
    name: 'Armamento Aberrante',
    description:
      'Ação de movimento + 1 PM para produzir versão orgânica de qualquer arma corpo a corpo ou de arremesso com a qual seja proficiente — brota do braço/ombro/costas como planta grotesca e então se desprende. Dano da arma aumenta em um passo para cada dois outros poderes da Tormenta. Dura pela cena, então desfaz em gosma.',
    requiresOtherPowers: 1,
    bookPage: 136,
  },
  {
    id: 'articulacoes-flexiveis',
    name: 'Articulações Flexíveis',
    description:
      '+1 em Acrobacia, Furtividade e Reflexos. Bônus aumenta em +1 para cada dois outros poderes da Tormenta.',
    requiresOtherPowers: 0,
    bookPage: 136,
  },
  {
    id: 'asas-insetoides',
    name: 'Asas Insetoides',
    description:
      '1 PM para deslocamento de voo 9m até o fim do turno. Deslocamento aumenta +1,5m para cada outro poder da Tormenta que você possui.',
    requiresOtherPowers: 4,
    bookPage: 136,
  },
  {
    id: 'carapaca',
    name: 'Carapaça',
    description:
      'Pele recoberta por placas quitinosas: +1 na Defesa. Bônus aumenta em +1 para cada dois outros poderes da Tormenta.',
    requiresOtherPowers: 0,
    bookPage: 136,
  },
  {
    id: 'corpo-aberrante',
    name: 'Corpo Aberrante',
    description:
      'Crostas vermelhas tornam ataques desarmados mais perigosos. Dano desarmado aumenta em um passo, mais um passo para cada quatro outros poderes da Tormenta.',
    requiresOtherPowers: 1,
    bookPage: 136,
  },
  {
    id: 'cuspir-enxame',
    name: 'Cuspir Enxame',
    description:
      'Ação completa + 2 PM para criar enxame de insetos rubros em ponto à escolha em alcance curto (sustentada). Enxame tamanho Médio, atravessa espaço de outras criaturas. Ação de movimento move enxame 9m. Fim do turno causa 2d6 dano ácido a criaturas no espaço. +1 PM por dois outros poderes da Tormenta aumenta dano em +1d6.',
    requiresOtherPowers: 0,
    bookPage: 136,
  },
  {
    id: 'dentes-afiados',
    name: 'Dentes Afiados',
    description:
      'Arma natural de mordida (dano 1d4, crítico x2, corte). Uma vez por rodada quando agride com outra arma, gasta 1 PM para ataque corpo a corpo extra com a mordida.',
    requiresOtherPowers: 0,
    bookPage: 137,
  },
  {
    id: 'desprezar-a-realidade',
    name: 'Desprezar a Realidade',
    description:
      '2 PM para ficar no limiar da realidade até o início do próximo turno: ignora terreno difícil e causa 20% de chance de falha em efeitos usados contra você (não apenas ataques). +5% por dois outros poderes da Tormenta (máximo 50%).',
    requiresOtherPowers: 4,
    bookPage: 137,
  },
  {
    id: 'empunhadura-rubra',
    name: 'Empunhadura Rubra',
    description:
      '1 PM para cobrir mãos com carapaça rubra. Até fim da cena: +1 em Luta. Bônus aumenta em +1 para cada dois outros poderes da Tormenta.',
    requiresOtherPowers: 0,
    bookPage: 137,
  },
  {
    id: 'fome-de-mana',
    name: 'Fome de Mana',
    description:
      'Ao passar em teste de resistência contra habilidade mágica de inimigo, recebe 1 PM temporário cumulativo. Máximo de PM temporários por cena = número de poderes da Tormenta que possui.',
    requiresOtherPowers: 0,
    bookPage: 137,
  },
  {
    id: 'larva-explosiva',
    name: 'Larva Explosiva',
    description:
      'Se criatura que sofreu dano de sua mordida na cena é reduzida a 0 PV ou menos, explode em chuva cáustica (morre) causando 4d4 ácido em criaturas adjacentes. +2d4 por dois outros poderes da Tormenta. Imune a esse dano.',
    requiresOtherPowers: 0,
    requiresPower: 'dentes-afiados',
    bookPage: 137,
  },
  {
    id: 'legiao-aberrante',
    name: 'Legião Aberrante',
    description:
      'Corpo se transforma em massa de insetos rubros. Atravessa espaços por onde passa moeda (esses espaços contam como terreno difícil). +1 em testes contra manobras e resistência a efeitos direcionados a você (não área). Bônus aumenta +1 para cada dois outros poderes da Tormenta.',
    requiresOtherPowers: 3,
    requiresPower: 'anatomia-insana',
    bookPage: 137,
  },
  {
    id: 'maos-membranosas',
    name: 'Mãos Membranosas',
    description:
      '+1 em Atletismo, Fortitude e testes de agarrar. Bônus aumenta em +1 para cada dois outros poderes da Tormenta.',
    requiresOtherPowers: 0,
    bookPage: 137,
  },
  {
    id: 'membros-estendidos',
    name: 'Membros Estendidos',
    description:
      'Braços e armas naturais grotescamente mais longos: alcance natural corpo a corpo +1,5m. Alcance aumenta +1,5m para cada quatro outros poderes da Tormenta.',
    requiresOtherPowers: 0,
    bookPage: 137,
  },
  {
    id: 'membros-extras',
    name: 'Membros Extras',
    description:
      'Duas armas naturais de patas insetoides das costas/ombros/flancos. Uma vez por rodada ao agredir com outra arma, gasta 2 PM para ataque corpo a corpo extra com cada uma (dano 1d4, crítico x2, corte). Com Ambidestria ou Estilo de Duas Armas: empunha armas leves nas patas (ainda paga 2 PM + sofre -2 em todos os ataques).',
    requiresOtherPowers: 4,
    bookPage: 137,
  },
  {
    id: 'mente-aberrante',
    name: 'Mente Aberrante',
    description:
      'Resistência a efeitos mentais +1. Ao passar em teste de Vontade contra habilidade, a criatura usuária sofre 1d6 psíquico. Bônus e dano aumentam +1/+1d6 para cada dois outros poderes da Tormenta.',
    requiresOtherPowers: 0,
    bookPage: 137,
  },
  {
    id: 'olhos-vermelhos',
    name: 'Olhos Vermelhos',
    description:
      'Visão no escuro e +1 em Intimidação. Bônus em Intimidação aumenta em +1 para cada dois outros poderes da Tormenta.',
    requiresOtherPowers: 0,
    bookPage: 137,
  },
  {
    id: 'pele-corrompida',
    name: 'Pele Corrompida',
    description:
      'Carne mesclada à matéria vermelha: redução de ácido, eletricidade, fogo, frio, luz e trevas 2. RD aumenta +2 para cada dois outros poderes da Tormenta.',
    requiresOtherPowers: 0,
    bookPage: 137,
  },
  {
    id: 'sangue-acido',
    name: 'Sangue Ácido',
    description:
      'Quando sofre dano por ataque corpo a corpo, atacante sofre 1 dano ácido por poder da Tormenta que você possui.',
    requiresOtherPowers: 0,
    bookPage: 137,
  },
  {
    id: 'visco-rubro',
    name: 'Visco Rubro',
    description:
      '1 PM para expelir líquido grosso e corrosivo. Até fim da cena: +1 nas rolagens de dano corpo a corpo. Bônus aumenta em +1 para cada dois outros poderes da Tormenta.',
    requiresOtherPowers: 0,
    bookPage: 137,
  },
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
