/**
 * Doenças + Venenos — PDF Cap 7 Aventura (doenças p318) e Cap 3
 * Equipamento (venenos alquímicos p161).
 *
 * **Vector taxonomy** (shared between doenças and venenos):
 *  - 'contato', 'inalacao', 'ingestao'. T20 has NO 'ferimento' vector —
 *    poisoned weapons fall under contato per rule (aplicar veneno em
 *    arma exige ação de mov + rolagem 1d6; 1 = autocontaminação).
 *
 * **Doenças system** (p318):
 *  - Save type: Fortitude vs CD específico da doença.
 *  - Exposição não cumulativa — re-exposição em vítima contaminada
 *    não tem efeito extra.
 *  - Progressão: novo Fortitude no início de cada dia. Falha = sofre
 *    o efeito do dia; sucesso = nada.
 *  - Cura: DOIS sucessos consecutivos no início de dias seguidos curam.
 *  - Tratamento (Cura, Apenas Treinado, ação completa) vs CD da doença
 *    concede +5 no próximo Fortitude. Maleta de medicamentos sob risco
 *    de -5 sem ela.
 *  - Marcador "*" no efeito = perda permanente; limite de 1 atributo
 *    por doença.
 *
 * **Venenos system** (p161):
 *  - Save type: Fortitude. CD é definida pelo aplicador via Int (CD
 *    base de fabricação = 20). Beladona e Pó de Lich têm modificador
 *    +5 na CD ("aumenta em +5").
 *  - Vítima ganha condição envenenada para efeitos prolongados; curar
 *    a condição encerra a sequência, mas não restaura PV já perdidos.
 *  - Single save model: falha = efeito completo; sucesso = efeito
 *    parcial menor (não nullified).
 *  - Tratamento (Cura): mesma mecânica de doenças.
 */

export type DiseasePoisonVector = 'contato' | 'inalacao' | 'ingestao'

// ─── DOENÇAS ────────────────────────────────────────────────────────
export type Disease = {
  id: string
  name: string
  vector: DiseasePoisonVector
  initialSaveCd: number
  incubationTime: string
  effect: string
  /** Progressão ladder (do mais brando ao mais severo). Vazio se efeito único. */
  progression: readonly string[]
  cureCd: number | null
  /** Indica se a doença pode causar perda permanente de atributo (marcador `*`). */
  hasPermanentEffect: boolean
  bookPage: number
}

export const DISEASES: readonly Disease[] = Object.freeze([
  {
    id: 'calafrio-diabolico',
    name: 'Calafrio Diabólico',
    vector: 'contato',
    initialSaveCd: 25,
    incubationTime: '1 dia',
    effect:
      'Causa fraqueza e, em casos graves, coma e morte. Progressivo conforme falhas consecutivas.',
    progression: ['fraca', 'debilitada', 'inconsciente', 'morre'],
    cureCd: 25,
    hasPermanentEffect: false,
    bookPage: 318,
  },
  {
    id: 'febre-do-riso',
    name: 'Febre do Riso',
    vector: 'inalacao',
    initialSaveCd: 20,
    incubationTime: '1 dia',
    effect:
      'Causa surtos de agitação e, em casos graves, loucura. Condição se ativa no início de cada cena.',
    progression: ['frustrada', 'esmorecida', 'confusa'],
    cureCd: 20,
    hasPermanentEffect: false,
    bookPage: 318,
  },
  {
    id: 'febre-mental',
    name: 'Febre Mental',
    vector: 'inalacao',
    initialSaveCd: 20,
    incubationTime: '1 dia',
    effect: 'Causa enxaqueca e torpor.',
    progression: ['frustrada', 'esmorecida'],
    cureCd: 20,
    hasPermanentEffect: false,
    bookPage: 318,
  },
  {
    id: 'infeccao-do-esgoto',
    name: 'Infecção do Esgoto',
    vector: 'contato',
    initialSaveCd: 15,
    incubationTime: '1 dia',
    effect:
      'Transmitida por ratos gigantes, otyughs e exposição a lugares imundos.',
    progression: ['fraca', 'debilitada'],
    cureCd: 15,
    hasPermanentEffect: false,
    bookPage: 318,
  },
  {
    id: 'maldicao-pegajosa',
    name: 'Maldição Pegajosa',
    vector: 'contato',
    initialSaveCd: 20,
    incubationTime: '1 dia',
    effect:
      'Transforma os órgãos internos em massa disforme. Exige três Fortitudes para se curar.',
    progression: ['perde 1d12 PV', 'perde 2d12 PV', 'perde 4d12 PV'],
    cureCd: 20,
    hasPermanentEffect: false,
    bookPage: 318,
  },
  {
    id: 'molestia-demoniaca',
    name: 'Moléstia Demoníaca',
    vector: 'contato',
    initialSaveCd: 20,
    incubationTime: '1 dia',
    effect:
      'Causa danos internos e, em casos graves, sequelas permanentes e morte.',
    progression: [
      'perde 1d12 PV',
      'perde 2d12 PV',
      'perde 1 de Constituição*',
      'morre',
    ],
    cureCd: 20,
    hasPermanentEffect: true,
    bookPage: 318,
  },
  {
    id: 'tremores',
    name: 'Tremores',
    vector: 'contato',
    initialSaveCd: 15,
    incubationTime: '1 dia',
    effect: 'Causa tremedeira e convulsões.',
    progression: ['vulnerável'],
    cureCd: 15,
    hasPermanentEffect: false,
    bookPage: 318,
  },
  {
    id: 'variola',
    name: 'Varíola',
    vector: 'inalacao',
    initialSaveCd: 20,
    incubationTime: '1 dia',
    effect:
      'Causa febre e vômitos. Em casos graves, úlceras e erupções deixam cicatrizes e podem levar à morte.',
    progression: ['enjoada', 'debilitada', 'perde 1 de Carisma*', 'morre'],
    cureCd: 20,
    hasPermanentEffect: true,
    bookPage: 318,
  },
])

// ─── VENENOS ────────────────────────────────────────────────────────
export type Poison = {
  id: string
  name: string
  vector: DiseasePoisonVector
  /**
   * Modificador na CD do veneno em relação à CD-base do aplicador
   * (CD-fabricação = 20 + Int). Default 0. Beladona e Pó de Lich = +5.
   */
  saveCdModifier: number
  /** Efeito em falha no Fortitude. */
  initialEffect: string
  /** Efeito em sucesso (T20 não anula veneno; efeito parcial). null = sucesso anula. */
  partialEffect: string | null
  progression: string
  priceTibar: number
  bookPage: number
}

export const POISONS: readonly Poison[] = Object.freeze([
  {
    id: 'beladona',
    name: 'Beladona',
    vector: 'ingestao',
    saveCdModifier: 5,
    initialEffect: 'Vítima fica lenta por 3 rodadas.',
    partialEffect: null,
    progression: 'Condição lenta dura 3 rodadas.',
    priceTibar: 1500,
    bookPage: 161,
  },
  {
    id: 'bruma-sonolenta',
    name: 'Bruma Sonolenta',
    vector: 'inalacao',
    saveCdModifier: 0,
    initialEffect: 'Vítima fica inconsciente.',
    partialEffect: 'Enjoada por 1 rodada.',
    progression:
      'Efeito instantâneo (inconsciência); efeito menor em sucesso (enjoada 1 rodada).',
    priceTibar: 150,
    bookPage: 161,
  },
  {
    id: 'cicuta',
    name: 'Cicuta',
    vector: 'ingestao',
    saveCdModifier: 0,
    initialEffect: 'Perde 1d12 PV por rodada durante 3 rodadas.',
    partialEffect: 'Perde 1d12 PV (apenas uma vez).',
    progression: 'Dano recorrente 1d12 PV/rodada por 3 rodadas.',
    priceTibar: 60,
    bookPage: 161,
  },
  {
    id: 'essencia-de-sombra',
    name: 'Essência de Sombra',
    vector: 'contato',
    saveCdModifier: 0,
    initialEffect: 'Vítima fica debilitada.',
    partialEffect: 'Fraca.',
    progression: 'Aplica condição debilitada (em sucesso: fraca).',
    priceTibar: 100,
    bookPage: 161,
  },
  {
    id: 'nevoa-toxica',
    name: 'Névoa Tóxica',
    vector: 'inalacao',
    saveCdModifier: 0,
    initialEffect: 'Perde 1d12 PV por rodada durante 3 rodadas.',
    partialEffect: 'Perde 1d12 PV (uma vez).',
    progression:
      'Gás verde queima e corrói pele e pulmões; dano recorrente 1d12 PV/rodada por 3 rodadas.',
    priceTibar: 30,
    bookPage: 161,
  },
  {
    id: 'peconha-comum',
    name: 'Peçonha Comum',
    vector: 'contato',
    saveCdModifier: 0,
    initialEffect: 'Perde 1d12 PV.',
    partialEffect: null,
    progression: 'Perda instantânea de 1d12 PV em falha.',
    priceTibar: 15,
    bookPage: 161,
  },
  {
    id: 'peconha-concentrada',
    name: 'Peçonha Concentrada',
    vector: 'contato',
    saveCdModifier: 0,
    initialEffect: 'Perde 1d12 PV por rodada durante 3 rodadas.',
    partialEffect: 'Perde 1d12 PV (uma vez).',
    progression: 'Dano recorrente 1d12 PV/rodada por 3 rodadas.',
    priceTibar: 90,
    bookPage: 161,
  },
  {
    id: 'peconha-potente',
    name: 'Peçonha Potente',
    vector: 'contato',
    saveCdModifier: 0,
    initialEffect: 'Perde 2d12 PV por rodada durante 3 rodadas.',
    partialEffect: 'Perde 2d12 PV (uma vez).',
    progression: 'Dano recorrente 2d12 PV/rodada por 3 rodadas.',
    priceTibar: 600,
    bookPage: 161,
  },
  {
    id: 'po-de-lich',
    name: 'Pó de Lich',
    vector: 'ingestao',
    saveCdModifier: 5,
    initialEffect: 'Perde 4d12 PV por rodada durante 3 rodadas.',
    partialEffect: 'Perde 4d12 PV (uma vez).',
    progression:
      'Veneno letal para assassinar alvos poderosos; dano recorrente 4d12 PV/rodada por 3 rodadas.',
    priceTibar: 3000,
    bookPage: 161,
  },
  {
    id: 'riso-de-nimb',
    name: 'Riso de Nimb',
    vector: 'inalacao',
    saveCdModifier: 0,
    initialEffect: 'Vítima fica confusa.',
    partialEffect: 'Lenta por 1 rodada.',
    progression:
      'Gás púrpura faz a vítima rir descontroladamente e agir de forma caótica.',
    priceTibar: 150,
    bookPage: 161,
  },
])

// ─── Helpers ────────────────────────────────────────────────────────
const diseaseById = new Map(DISEASES.map((d) => [d.id, d]))
const poisonById = new Map(POISONS.map((p) => [p.id, p]))

export function diseaseById_(id: string): Disease | undefined {
  return diseaseById.get(id)
}

export function poisonById_(id: string): Poison | undefined {
  return poisonById.get(id)
}

export function diseasesByVector(
  vector: DiseasePoisonVector,
): readonly Disease[] {
  return DISEASES.filter((d) => d.vector === vector)
}

export function poisonsByVector(
  vector: DiseasePoisonVector,
): readonly Poison[] {
  return POISONS.filter((p) => p.vector === vector)
}

/**
 * CD final para resistir a um veneno = CD-base do aplicador + modifier.
 * CD-base default = 20 (p161: CD-fabricação base).
 */
export const POISON_BASE_CRAFT_CD = 20

export function poisonResistCd(
  poison: Poison,
  applicatorIntelligence: number = 0,
): number {
  return POISON_BASE_CRAFT_CD + applicatorIntelligence + poison.saveCdModifier
}

/**
 * Required number of consecutive successful Fortitude saves to cure a
 * doença (p318: "dois testes de Fortitude bem-sucedidos consecutivos").
 */
export const DISEASE_CURE_SAVES_REQUIRED = 2

/**
 * Modelo simples de avanço diário de doença. Recebe estado atual e o
 * resultado do Fortitude do dia; retorna novo estado.
 *
 *  - `step` = índice atual na ladder de progressão (-1 = ainda não
 *    sofreu efeito; 0 = primeiro efeito; ...).
 *  - `consecutiveSuccesses` = sucessos seguidos acumulados (0..2).
 *
 * Falha: avança um step (clamp ao topo). Reseta sucessos.
 * Sucesso: incrementa contador. Quando atinge 2, retorna `cured: true`.
 */
export function advanceDisease(
  disease: Disease,
  state: { step: number; consecutiveSuccesses: number },
  fortitudeSuccess: boolean,
): { step: number; consecutiveSuccesses: number; cured: boolean } {
  if (fortitudeSuccess) {
    const consecutiveSuccesses = state.consecutiveSuccesses + 1
    const cured = consecutiveSuccesses >= DISEASE_CURE_SAVES_REQUIRED
    return { step: state.step, consecutiveSuccesses, cured }
  }
  const lastStep = disease.progression.length - 1
  const nextStep = Math.min(state.step + 1, lastStep)
  return { step: nextStep, consecutiveSuccesses: 0, cured: false }
}
