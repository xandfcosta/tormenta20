/**
 * Skill Index — Tabela 2-1 (PDF p115) canônica das 29 perícias T20.
 *
 * Fonte de verdade única para: id → { atributo-chave, treinada-apenas,
 * penalidade-de-armadura }. Consumido pelo character-sheet orchestrator
 * para calcular valor total das perícias.
 *
 * Formula T20 Cap 2:
 *   Sem treinamento: valor = modificador do atributo-chave
 *   Com treinamento: valor = ½ nível (mín +1) + atributo + 2 (treino)
 *                          - penalidade de armadura (se aplicável)
 *
 * Bônus de treinamento base é +2. Poderes/habilidades (ex: Perícia
 * expandida, p228) podem elevá-lo — orchestrator aceita override via
 * `trainedBonusOverride` no futuro (v3+).
 */

import type { AttributeKey } from './attributes'

// ─── IDs ─────────────────────────────────────────────────────────────
export const SKILL_IDS = [
  'acrobacia',
  'adestramento',
  'atletismo',
  'atuacao',
  'cavalgar',
  'conhecimento',
  'cura',
  'diplomacia',
  'enganacao',
  'fortitude',
  'furtividade',
  'guerra',
  'iniciativa',
  'intimidacao',
  'intuicao',
  'investigacao',
  'jogatina',
  'ladinagem',
  'luta',
  'misticismo',
  'nobreza',
  'oficio',
  'percepcao',
  'pilotagem',
  'pontaria',
  'reflexos',
  'religiao',
  'sobrevivencia',
  'vontade',
] as const

export type SkillId = (typeof SKILL_IDS)[number]

export type SkillMeta = {
  id: SkillId
  /** Nome canônico do livro (Tabela 2-1). */
  name: string
  /** Atributo-chave (PDF Tabela 2-1). */
  keyAttribute: AttributeKey
  /** Perícia só pode ser usada por personagens treinados nela. */
  trainedOnly: boolean
  /** Perícia sofre penalidade de armadura quando o portador usa armadura. */
  armorPenalty: boolean
  /** Página do livro que descreve a perícia. */
  bookPage: number
}

// ─── Catálogo ────────────────────────────────────────────────────────
export const SKILL_INDEX: Readonly<Record<SkillId, SkillMeta>> = Object.freeze({
  acrobacia: {
    id: 'acrobacia',
    name: 'Acrobacia',
    keyAttribute: 'dexterity',
    trainedOnly: false,
    armorPenalty: true,
    bookPage: 115,
  },
  adestramento: {
    id: 'adestramento',
    name: 'Adestramento',
    keyAttribute: 'charisma',
    trainedOnly: false,
    armorPenalty: false,
    bookPage: 115,
  },
  atletismo: {
    id: 'atletismo',
    name: 'Atletismo',
    keyAttribute: 'strength',
    trainedOnly: false,
    armorPenalty: false,
    bookPage: 116,
  },
  atuacao: {
    id: 'atuacao',
    name: 'Atuação',
    keyAttribute: 'charisma',
    trainedOnly: false,
    armorPenalty: false,
    bookPage: 116,
  },
  cavalgar: {
    id: 'cavalgar',
    name: 'Cavalgar',
    keyAttribute: 'dexterity',
    trainedOnly: false,
    armorPenalty: false,
    bookPage: 116,
  },
  conhecimento: {
    id: 'conhecimento',
    name: 'Conhecimento',
    keyAttribute: 'intelligence',
    trainedOnly: true,
    armorPenalty: false,
    bookPage: 117,
  },
  cura: {
    id: 'cura',
    name: 'Cura',
    keyAttribute: 'wisdom',
    trainedOnly: false,
    armorPenalty: false,
    bookPage: 117,
  },
  diplomacia: {
    id: 'diplomacia',
    name: 'Diplomacia',
    keyAttribute: 'charisma',
    trainedOnly: false,
    armorPenalty: false,
    bookPage: 118,
  },
  enganacao: {
    id: 'enganacao',
    name: 'Enganação',
    keyAttribute: 'charisma',
    trainedOnly: false,
    armorPenalty: false,
    bookPage: 118,
  },
  fortitude: {
    id: 'fortitude',
    name: 'Fortitude',
    keyAttribute: 'constitution',
    trainedOnly: false,
    armorPenalty: false,
    bookPage: 119,
  },
  furtividade: {
    id: 'furtividade',
    name: 'Furtividade',
    keyAttribute: 'dexterity',
    trainedOnly: false,
    armorPenalty: true,
    bookPage: 119,
  },
  guerra: {
    id: 'guerra',
    name: 'Guerra',
    keyAttribute: 'intelligence',
    trainedOnly: true,
    armorPenalty: false,
    bookPage: 119,
  },
  iniciativa: {
    id: 'iniciativa',
    name: 'Iniciativa',
    keyAttribute: 'dexterity',
    trainedOnly: false,
    armorPenalty: false,
    bookPage: 119,
  },
  intimidacao: {
    id: 'intimidacao',
    name: 'Intimidação',
    keyAttribute: 'charisma',
    trainedOnly: false,
    armorPenalty: false,
    bookPage: 120,
  },
  intuicao: {
    id: 'intuicao',
    name: 'Intuição',
    keyAttribute: 'wisdom',
    trainedOnly: false,
    armorPenalty: false,
    bookPage: 120,
  },
  investigacao: {
    id: 'investigacao',
    name: 'Investigação',
    keyAttribute: 'intelligence',
    trainedOnly: false,
    armorPenalty: false,
    bookPage: 120,
  },
  jogatina: {
    id: 'jogatina',
    name: 'Jogatina',
    keyAttribute: 'charisma',
    trainedOnly: true,
    armorPenalty: false,
    bookPage: 120,
  },
  ladinagem: {
    id: 'ladinagem',
    name: 'Ladinagem',
    keyAttribute: 'dexterity',
    trainedOnly: true,
    armorPenalty: true,
    bookPage: 120,
  },
  luta: {
    id: 'luta',
    name: 'Luta',
    keyAttribute: 'strength',
    trainedOnly: false,
    armorPenalty: false,
    bookPage: 121,
  },
  misticismo: {
    id: 'misticismo',
    name: 'Misticismo',
    keyAttribute: 'intelligence',
    trainedOnly: true,
    armorPenalty: false,
    bookPage: 121,
  },
  nobreza: {
    id: 'nobreza',
    name: 'Nobreza',
    keyAttribute: 'intelligence',
    trainedOnly: true,
    armorPenalty: false,
    bookPage: 121,
  },
  oficio: {
    id: 'oficio',
    name: 'Ofício',
    keyAttribute: 'intelligence',
    trainedOnly: true,
    armorPenalty: false,
    bookPage: 121,
  },
  percepcao: {
    id: 'percepcao',
    name: 'Percepção',
    keyAttribute: 'wisdom',
    trainedOnly: false,
    armorPenalty: false,
    bookPage: 122,
  },
  pilotagem: {
    id: 'pilotagem',
    name: 'Pilotagem',
    keyAttribute: 'dexterity',
    trainedOnly: true,
    armorPenalty: false,
    bookPage: 122,
  },
  pontaria: {
    id: 'pontaria',
    name: 'Pontaria',
    keyAttribute: 'dexterity',
    trainedOnly: false,
    armorPenalty: false,
    bookPage: 122,
  },
  reflexos: {
    id: 'reflexos',
    name: 'Reflexos',
    keyAttribute: 'dexterity',
    trainedOnly: false,
    armorPenalty: false,
    bookPage: 122,
  },
  religiao: {
    id: 'religiao',
    name: 'Religião',
    keyAttribute: 'wisdom',
    trainedOnly: true,
    armorPenalty: false,
    bookPage: 122,
  },
  sobrevivencia: {
    id: 'sobrevivencia',
    name: 'Sobrevivência',
    keyAttribute: 'wisdom',
    trainedOnly: false,
    armorPenalty: false,
    bookPage: 123,
  },
  vontade: {
    id: 'vontade',
    name: 'Vontade',
    keyAttribute: 'wisdom',
    trainedOnly: false,
    armorPenalty: false,
    bookPage: 123,
  },
})

// ─── Constantes de fórmula ───────────────────────────────────────────
/** Bônus base de treinamento (PDF p115). Modificáveis por poderes. */
export const TRAINED_BONUS_BASE = 2

/** ½ nível ao ser treinado nunca é menor que +1 (PDF p115). */
export const TRAINED_HALF_LEVEL_MIN = 1

// ─── Helpers ─────────────────────────────────────────────────────────
export function skillMetaById(id: SkillId): SkillMeta {
  const meta = SKILL_INDEX[id]
  if (!meta) {
    throw new Error(`skillMetaById: unknown skill id ${id}`)
  }
  return meta
}

/**
 * Componente treinado da perícia: ½ nível (mínimo +1) + TRAINED_BONUS_BASE.
 * Se não treinado, retorna 0.
 */
export function trainedComponent(level: number, trained: boolean): number {
  if (!trained) return 0
  return Math.max(TRAINED_HALF_LEVEL_MIN, Math.floor(level / 2)) + TRAINED_BONUS_BASE
}

/**
 * Valor total de uma perícia.
 *
 *   Sem treinamento: valor = modificador do atributo-chave
 *   Com treinamento: valor = ½ nível (mín +1) + atributo + 2 - penalidade
 *
 * Penalidade de armadura só se aplica quando a perícia tem flag
 * `armorPenalty` e o personagem tem armadura equipada com penalidade > 0.
 */
export function skillValue(params: {
  level: number
  attributeValue: number
  trained: boolean
  armorPenaltyApplies: boolean
  armorPenalty: number
}): number {
  const { level, attributeValue, trained, armorPenaltyApplies, armorPenalty } = params
  const trainedBonus = trainedComponent(level, trained)
  const penalty = armorPenaltyApplies ? armorPenalty : 0
  return attributeValue + trainedBonus - penalty
}
