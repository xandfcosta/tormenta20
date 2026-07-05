/**
 * Character Sheet Orchestrator (v1 — MVP).
 *
 * Consolida os módulos de mecânica em uma função `computeCharacterSheet`
 * que aceita o estado bruto do personagem e devolve valores derivados
 * (atributos totais, PV/PM máximos, Defesa, salvações, deslocamento).
 *
 * Escopo v1 (MVP): atributos + vitals + defesa base + saves + deslocamento.
 * Fora do escopo v1: perícias com valores totais, equipamentos, buffs de
 * magia, condições ativas, ataques por arma. Esses ficam em v2/v3/v4 e
 * podem ser adicionados sem quebrar a API atual.
 *
 * Warnings coletados em array; caller decide como surfacer (UI toast /
 * validação de criação de personagem / log).
 */

import { ATTRIBUTE_KEYS, type AttributeKey } from './attributes'
import { CLASS_VITALS } from './class-vitals'
import { racaById, resolveAtributoMod, type Raca, type Tamanho } from './racas'
import { SKILL_IDS, SKILL_INDEX, skillValue, type SkillId } from './skill-index'

// ─── Input ───────────────────────────────────────────────────────────
export type CharacterInput = {
  /** Nível do personagem (1-20). */
  level: number
  /** Nome canônico da classe (ex: 'Bárbaro', 'Arcanista'). */
  className: string
  /** ID canônico da raça (ex: 'humano'). Se omitido, race mod = 0. */
  raceId?: string
  /**
   * Atributos escolhidos para raças com bônus flutuante (Humano, Lefou,
   * Osteon, Sereia). Ignorado para raças fixed/subraca-gated.
   */
  raceFloatingPicks?: readonly AttributeKey[]
  /** Ascendência escolhida para raças subraca-gated (Suraggel). */
  raceAscendencia?: string
  /**
   * Valores base dos atributos antes de mod racial. Em T20 o valor do
   * atributo já é o modificador (não há d20 conversion).
   */
  baseAttributes: Record<AttributeKey, number>
  /** PV atual (opcional; default = pvMax). */
  currentPv?: number
  /** PM atual (opcional; default = pmMax). */
  currentPm?: number
  /**
   * Perícias em que o personagem é treinado. IDs canônicos do
   * `SKILL_INDEX`. Perícias treinada-apenas exigem estar aqui para
   * poder ser usadas.
   */
  trainedSkills?: readonly SkillId[]
  /**
   * Penalidade de armadura atual (não-negativa). Aplicada às perícias
   * com flag `armorPenalty` no `SKILL_INDEX`. Origem: armadura +
   * escudo equipados (calculada por camada de equipamento em v3).
   */
  armorPenalty?: number
}

// ─── Output ──────────────────────────────────────────────────────────
export type AttributeComputed = {
  base: number
  raceMod: number
  total: number
}

export type ComputedSheet = {
  level: number
  className: string
  attributes: Record<AttributeKey, AttributeComputed>
  vitals: {
    pvMax: number
    pmMax: number
    pvCurrent: number
    pmCurrent: number
  }
  defense: {
    /** 10 fixo por regra base (p106). */
    base: 10
    /** Componente de Destreza. */
    attribute: number
    /** Total sem armadura/escudo (v1 não modela equipamento). */
    total: number
  }
  saves: {
    fortitude: number
    reflexos: number
    vontade: number
  }
  skills: Record<SkillId, SkillComputed>
  deslocamento: number
  tamanho: Tamanho
  warnings: readonly string[]
}

export type SkillComputed = {
  total: number
  trained: boolean
  keyAttribute: AttributeKey
  /** True se a perícia é treinada-apenas E o personagem não é treinado. */
  cannotUse: boolean
  /** Penalidade de armadura aplicada (0 se não relevante). */
  armorPenaltyApplied: number
}

// ─── Constantes ──────────────────────────────────────────────────────
/** Defesa base fixa por regra do livro (p106). */
export const DEFENSE_BASE = 10

// ─── Helpers ─────────────────────────────────────────────────────────
/** ½ nível arredondado para baixo — base de perícias e saves. */
export function halfLevel(level: number): number {
  return Math.floor(level / 2)
}

/**
 * Resolve mod racial em cada atributo, respeitando o tipo do mod
 * (fixed / floating / subraca-gated). Warnings para inputs inválidos.
 */
function resolveRaceMods(
  input: CharacterInput,
  warnings: string[],
): Record<AttributeKey, number> {
  const zero: Record<AttributeKey, number> = {
    strength: 0,
    dexterity: 0,
    constitution: 0,
    intelligence: 0,
    wisdom: 0,
    charisma: 0,
  }
  if (!input.raceId) return zero

  let raca: Raca
  try {
    raca = racaById(input.raceId)
  } catch {
    warnings.push(`raça desconhecida: ${input.raceId}`)
    return zero
  }

  try {
    const mods = resolveAtributoMod(raca, {
      floatingPicks: input.raceFloatingPicks,
      ascendencia: input.raceAscendencia,
    })
    return { ...zero, ...mods }
  } catch (err) {
    warnings.push(
      `mod racial inválido para ${input.raceId}: ${(err as Error).message}`,
    )
    return zero
  }
}

// ─── Orchestrator ────────────────────────────────────────────────────
/**
 * Calcula a folha derivada a partir do estado bruto do personagem.
 *
 * Nunca lança — inputs inválidos viram entradas no array `warnings` e
 * a folha retorna com defaults conservadores. Isso permite renderizar
 * uma folha parcial na UI enquanto o jogador ainda está preenchendo.
 */
export function computeCharacterSheet(input: CharacterInput): ComputedSheet {
  const warnings: string[] = []

  if (input.level < 1 || input.level > 20) {
    warnings.push(`nível fora do range T20 (1-20): ${input.level}`)
  }

  const raceMods = resolveRaceMods(input, warnings)
  const attributes = {} as Record<AttributeKey, AttributeComputed>
  for (const key of ATTRIBUTE_KEYS) {
    const base = input.baseAttributes[key] ?? 0
    const raceMod = raceMods[key]
    attributes[key] = { base, raceMod, total: base + raceMod }
  }

  const vitals = computeVitals(input, attributes, warnings)
  const defenseAttr = attributes.dexterity.total
  const defense: ComputedSheet['defense'] = {
    base: DEFENSE_BASE,
    attribute: defenseAttr,
    total: DEFENSE_BASE + defenseAttr,
  }

  const half = halfLevel(input.level)
  const saves = {
    fortitude: half + attributes.constitution.total,
    reflexos: half + attributes.dexterity.total,
    vontade: half + attributes.wisdom.total,
  }

  const { deslocamento, tamanho } = resolveRaceMovement(input, warnings)
  const skills = computeSkills(input, attributes, warnings)

  return {
    level: input.level,
    className: input.className,
    attributes,
    vitals,
    defense,
    saves,
    skills,
    deslocamento,
    tamanho,
    warnings,
  }
}

// ─── Skills ──────────────────────────────────────────────────────────
function computeSkills(
  input: CharacterInput,
  attributes: Record<AttributeKey, AttributeComputed>,
  warnings: string[],
): Record<SkillId, SkillComputed> {
  const trainedSet = new Set<SkillId>(input.trainedSkills ?? [])
  const armorPenalty = input.armorPenalty ?? 0
  if (armorPenalty < 0) {
    warnings.push(`armorPenalty deve ser não-negativa, got ${armorPenalty}`)
  }
  const penalty = Math.max(0, armorPenalty)

  // Warn on unknown trained skill ids
  for (const id of trainedSet) {
    if (!(id in SKILL_INDEX)) {
      warnings.push(`perícia treinada desconhecida: ${id}`)
    }
  }

  const skills = {} as Record<SkillId, SkillComputed>
  for (const id of SKILL_IDS) {
    const meta = SKILL_INDEX[id]
    const trained = trainedSet.has(id)
    const attributeValue = attributes[meta.keyAttribute].total
    const total = skillValue({
      level: input.level,
      attributeValue,
      trained,
      armorPenaltyApplies: meta.armorPenalty && penalty > 0,
      armorPenalty: penalty,
    })
    skills[id] = {
      total,
      trained,
      keyAttribute: meta.keyAttribute,
      cannotUse: meta.trainedOnly && !trained,
      armorPenaltyApplied: meta.armorPenalty ? penalty : 0,
    }
  }
  return skills
}

// ─── Vitals ──────────────────────────────────────────────────────────
function computeVitals(
  input: CharacterInput,
  attributes: Record<AttributeKey, AttributeComputed>,
  warnings: string[],
): ComputedSheet['vitals'] {
  const vitals = CLASS_VITALS[input.className]
  if (!vitals) {
    warnings.push(`classe desconhecida: ${input.className}`)
    return { pvMax: 0, pmMax: 0, pvCurrent: 0, pmCurrent: 0 }
  }
  const con = attributes.constitution.total
  const pvMax = vitals.pvInicial + (input.level - 1) * vitals.pvPerLevel + con * input.level
  const paladinoBonus =
    vitals.paladinoMpAtL1Bonus === 'charisma'
      ? attributes.charisma.total
      : 0
  const pmMax = vitals.mpPerLevel * input.level + paladinoBonus
  const pvCurrent = input.currentPv ?? pvMax
  const pmCurrent = input.currentPm ?? pmMax
  return {
    pvMax: Math.max(0, pvMax),
    pmMax: Math.max(0, pmMax),
    pvCurrent: Math.min(pvCurrent, pvMax),
    pmCurrent: Math.min(pmCurrent, pmMax),
  }
}

// ─── Race movement ───────────────────────────────────────────────────
function resolveRaceMovement(
  input: CharacterInput,
  warnings: string[],
): { deslocamento: number; tamanho: Tamanho } {
  if (!input.raceId) {
    return { deslocamento: 9, tamanho: 'Médio' }
  }
  try {
    const raca = racaById(input.raceId)
    return { deslocamento: raca.deslocamento, tamanho: raca.tamanho }
  } catch {
    warnings.push(`raça desconhecida ao resolver deslocamento: ${input.raceId}`)
    return { deslocamento: 9, tamanho: 'Médio' }
  }
}
