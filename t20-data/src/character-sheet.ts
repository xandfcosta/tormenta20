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
import {
  CONDITIONS,
  type Condition,
  type ConditionId,
} from './conditions'
import type { DamageType, WeaponHand, WeaponPurpose } from './items/types'
import {
  NON_SELF_STACKING_SOURCES,
  bestBonusOnly,
  sumStackingBonuses,
  type ModifierSource,
} from './modifier-stacking'
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
   * com flag `armorPenalty` no `SKILL_INDEX`. Se `equipment` estiver
   * presente, é ignorada — a penalidade é derivada de armor + shield.
   */
  armorPenalty?: number
  /**
   * Equipamento atual do personagem. Se presente, o orchestrator deriva
   * bônus de Defesa (armor + shield), penalidade de armadura para
   * skills e ataques por arma equipada.
   */
  equipment?: CharacterEquipment
  /**
   * Efeitos ativos (buffs de magia, poderes toggled, itens ativados,
   * bônus de parceiro, ambiente). Aplicados por modifier-stacking:
   * fontes selfStacking somam entre effectIds diferentes; fontes
   * nonSelfStacking mantêm apenas o melhor por effectId único.
   */
  activeEffects?: readonly ActiveEffect[]
  /**
   * Condições ativas (Fraco, Cego, Atordoado…). Expostas no output
   * como resumo textual; modificações mecânicas por condição não são
   * aplicadas automaticamente em v4 (heterogêneo demais). Caller pode
   * traduzir em ActiveEffect quando quiser aplicar (ex: Fraco →
   * effect com source 'ambiente' + attribute FOR -2).
   */
  activeConditions?: readonly ConditionId[]
}

export type ActiveEffect = {
  /** Identificador único do efeito ativo (ex: 'bencao-clerigo-1'). */
  id: string
  /** Nome legível. */
  name: string
  /** Fonte para regras de stacking. */
  source: ModifierSource
  modifiers: readonly EffectModifier[]
}

export type EffectModifier = {
  target: EffectTarget
  amount: number
}

export type EffectTarget =
  | { k: 'attribute'; attribute: AttributeKey }
  | { k: 'defense' }
  | { k: 'attack' }
  | { k: 'damage' }
  | { k: 'save'; save: 'fortitude' | 'reflexos' | 'vontade' }
  | { k: 'skill'; skill: SkillId }

export type CharacterEquipment = {
  armor?: EquippedArmor
  shield?: EquippedShield
  mainHand?: EquippedWeapon
  offHand?: EquippedWeapon
}

export type EquippedArmor = {
  name?: string
  /** Bônus de armadura na Defesa (positivo). */
  defense: number
  /** Penalidade de armadura em skills sensíveis (positivo; subtraído). */
  penalty: number
  heavy?: boolean
}

export type EquippedShield = {
  name?: string
  defense: number
  penalty: number
  heavy?: boolean
}

export type EquippedWeapon = {
  name?: string
  hand: WeaponHand
  purpose: WeaponPurpose
  /** Dado de dano base (ex: "1d8"). */
  damage: string
  critRange: number
  critMult: number
  damageType: DamageType
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
    /** Bônus de armadura equipada (0 se sem). */
    armor: number
    /** Bônus de escudo equipado (0 se sem). */
    shield: number
    total: number
  }
  saves: {
    fortitude: number
    reflexos: number
    vontade: number
  }
  skills: Record<SkillId, SkillComputed>
  attacks: {
    mainHand: ComputedAttack | null
    offHand: ComputedAttack | null
  }
  /** Condições ativas — apenas exposição (sem aplicação mecânica em v4). */
  conditions: readonly ConditionSummary[]
  /** Bônus totais aplicados por alvo, para debug/UI. */
  buffs: BuffsSummary
  deslocamento: number
  tamanho: Tamanho
  warnings: readonly string[]
}

export type ConditionSummary = Pick<
  Condition,
  'id' | 'name' | 'description' | 'tags'
>

export type BuffsSummary = {
  /** Total por alvo (attribute:FOR, defense, save:fortitude, skill:ladinagem, attack, damage). */
  totals: Record<string, number>
  /** Contribuições — cada efeito que participou (mesmo os anulados por non-stacking). */
  contributions: readonly BuffContribution[]
}

export type BuffContribution = {
  effectId: string
  effectName: string
  source: ModifierSource
  targetKey: string
  amount: number
  applied: boolean
}

export type ComputedAttack = {
  weaponName?: string
  /** Perícia usada (Luta corpo a corpo, Pontaria à distância/arremesso). */
  skill: 'luta' | 'pontaria'
  /** Bônus total de ataque (mesmo valor da perícia associada). */
  attackTotal: number
  /** Dado de dano base. */
  damageDice: string
  /** Bônus adicional de atributo no dano (FOR para corpo a corpo). */
  damageAttributeBonus: number
  damageType: DamageType
  critRange: number
  critMult: number
  hand: WeaponHand
  purpose: WeaponPurpose
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

  const buffs = resolveBuffs(input.activeEffects ?? [])

  // Aplicar buffs de atributo primeiro (afetam derivações posteriores).
  for (const key of ATTRIBUTE_KEYS) {
    const buffKey = `attribute:${key}`
    const delta = buffs.totals[buffKey] ?? 0
    attributes[key] = {
      ...attributes[key],
      total: attributes[key].total + delta,
    }
  }

  const vitals = computeVitals(input, attributes, warnings)
  const defenseAttr = attributes.dexterity.total
  const armorBonus = input.equipment?.armor?.defense ?? 0
  const shieldBonus = input.equipment?.shield?.defense ?? 0
  const defenseBuff = buffs.totals.defense ?? 0
  const defense: ComputedSheet['defense'] = {
    base: DEFENSE_BASE,
    attribute: defenseAttr,
    armor: armorBonus,
    shield: shieldBonus,
    total: DEFENSE_BASE + defenseAttr + armorBonus + shieldBonus + defenseBuff,
  }

  const half = halfLevel(input.level)
  const saves = {
    fortitude:
      half + attributes.constitution.total + (buffs.totals['save:fortitude'] ?? 0),
    reflexos:
      half + attributes.dexterity.total + (buffs.totals['save:reflexos'] ?? 0),
    vontade:
      half + attributes.wisdom.total + (buffs.totals['save:vontade'] ?? 0),
  }

  const { deslocamento, tamanho } = resolveRaceMovement(input, warnings)
  const skills = computeSkills(input, attributes, warnings, buffs.totals)
  const attacks = computeAttacks(input, skills, attributes, buffs.totals)
  const conditions = resolveConditions(input, warnings)

  return {
    level: input.level,
    className: input.className,
    attributes,
    vitals,
    defense,
    saves,
    skills,
    attacks,
    conditions,
    buffs: { totals: buffs.totals, contributions: buffs.contributions },
    deslocamento,
    tamanho,
    warnings,
  }
}

// ─── Conditions summary ──────────────────────────────────────────────
function resolveConditions(
  input: CharacterInput,
  warnings: string[],
): readonly ConditionSummary[] {
  const ids = input.activeConditions ?? []
  const summaries: ConditionSummary[] = []
  for (const id of ids) {
    const cond = CONDITIONS[id]
    if (!cond) {
      warnings.push(`condição desconhecida: ${id}`)
      continue
    }
    summaries.push({
      id: cond.id,
      name: cond.name,
      description: cond.description,
      tags: cond.tags,
    })
  }
  return summaries
}

// ─── Buffs / active effects ──────────────────────────────────────────
/**
 * Chave textual canônica de um EffectTarget — usada como key em maps
 * agregadores.
 */
export function effectTargetKey(t: EffectTarget): string {
  switch (t.k) {
    case 'attribute':
      return `attribute:${t.attribute}`
    case 'defense':
      return 'defense'
    case 'attack':
      return 'attack'
    case 'damage':
      return 'damage'
    case 'save':
      return `save:${t.save}`
    case 'skill':
      return `skill:${t.skill}`
  }
}

type FlatModifier = {
  effectId: string
  effectName: string
  source: ModifierSource
  amount: number
}

/**
 * Agrega bônus para um alvo aplicando regras de stacking do livro:
 *   - Diferentes fontes acumulam.
 *   - Habilidade/perícia acumulam entre effectIds diferentes.
 *   - Item/magia/parceiro/ambiente: melhor por source (mesmo com
 *     effectIds diferentes) — só o maior aplica dentro da fonte.
 *   - effectId duplicado (mesma habilidade lançada 2x): só o maior.
 *
 * Devolve o total + lista de contribuições marcando quais entraram
 * no total (`applied: true`) e quais foram anuladas por non-stacking.
 */
export function stackModifiersForTarget(
  mods: readonly FlatModifier[],
): { total: number; contributions: BuffContribution[]; targetKey: string } {
  // Primeiro colapsa duplicatas do MESMO effectId (best-only).
  const perEffectId = new Map<string, FlatModifier>()
  for (const m of mods) {
    const key = `${m.source}:${m.effectId}`
    const existing = perEffectId.get(key)
    if (!existing || m.amount > existing.amount) {
      perEffectId.set(key, m)
    }
  }

  // Agrupa por source.
  const bySource = new Map<ModifierSource, FlatModifier[]>()
  for (const m of perEffectId.values()) {
    const arr = bySource.get(m.source) ?? []
    arr.push(m)
    bySource.set(m.source, arr)
  }

  // Aplica stacking por source. Marca applied em cada contribuição.
  const contributions: BuffContribution[] = []
  let total = 0
  for (const [source, arr] of bySource) {
    const amounts = arr.map((m) => m.amount)
    let sourceTotal: number
    let appliedAmount: number
    if (NON_SELF_STACKING_SOURCES.has(source)) {
      sourceTotal = bestBonusOnly(amounts)
      appliedAmount = sourceTotal
    } else {
      sourceTotal = sumStackingBonuses(amounts)
      appliedAmount = sourceTotal
    }
    total += sourceTotal
    for (const m of arr) {
      const applied = NON_SELF_STACKING_SOURCES.has(source)
        ? m.amount === appliedAmount
        : true
      contributions.push({
        effectId: m.effectId,
        effectName: m.effectName,
        source: m.source,
        targetKey: '', // preenchido pelo caller
        amount: m.amount,
        applied,
      })
    }
  }
  return { total, contributions, targetKey: '' }
}

/**
 * Processa todos os ActiveEffect do personagem, agrupando por
 * `effectTargetKey` e resolvendo cada alvo via `stackModifiersForTarget`.
 */
function resolveBuffs(
  effects: readonly ActiveEffect[],
): {
  totals: Record<string, number>
  contributions: BuffContribution[]
} {
  const byTarget = new Map<string, FlatModifier[]>()
  const keysByTarget = new Map<string, string>()
  for (const eff of effects) {
    for (const mod of eff.modifiers) {
      const key = effectTargetKey(mod.target)
      keysByTarget.set(key, key)
      const flat: FlatModifier = {
        effectId: eff.id,
        effectName: eff.name,
        source: eff.source,
        amount: mod.amount,
      }
      const arr = byTarget.get(key) ?? []
      arr.push(flat)
      byTarget.set(key, arr)
    }
  }

  const totals: Record<string, number> = {}
  const contributions: BuffContribution[] = []
  for (const [key, mods] of byTarget) {
    const res = stackModifiersForTarget(mods)
    totals[key] = res.total
    for (const c of res.contributions) {
      contributions.push({ ...c, targetKey: key })
    }
  }
  return { totals, contributions }
}

// ─── Armor penalty derivation ────────────────────────────────────────
/**
 * Deriva penalidade de armadura efetiva:
 *  - Se `equipment.armor` ou `equipment.shield` presentes, soma penalidades.
 *  - Caso contrário, usa `armorPenalty` override (v2 API).
 *  - Valores negativos viram 0 + warning.
 */
function deriveArmorPenalty(
  input: CharacterInput,
  warnings: string[],
): number {
  const eq = input.equipment
  if (eq && (eq.armor || eq.shield)) {
    return (eq.armor?.penalty ?? 0) + (eq.shield?.penalty ?? 0)
  }
  const armorPenalty = input.armorPenalty ?? 0
  if (armorPenalty < 0) {
    warnings.push(`armorPenalty deve ser não-negativa, got ${armorPenalty}`)
  }
  return Math.max(0, armorPenalty)
}

// ─── Skills ──────────────────────────────────────────────────────────
function computeSkills(
  input: CharacterInput,
  attributes: Record<AttributeKey, AttributeComputed>,
  warnings: string[],
  buffTotals: Record<string, number>,
): Record<SkillId, SkillComputed> {
  const trainedSet = new Set<SkillId>(input.trainedSkills ?? [])
  const penalty = deriveArmorPenalty(input, warnings)

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
    const baseTotal = skillValue({
      level: input.level,
      attributeValue,
      trained,
      armorPenaltyApplies: meta.armorPenalty && penalty > 0,
      armorPenalty: penalty,
    })
    const skillBuff = buffTotals[`skill:${id}`] ?? 0
    skills[id] = {
      total: baseTotal + skillBuff,
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

// ─── Attacks ─────────────────────────────────────────────────────────
/**
 * Arma corpo a corpo usa Luta (FOR); arma à distância/arremesso usa
 * Pontaria (DES). Bônus de dano de atributo:
 *  - Melee: FOR total
 *  - Thrown: FOR total (ainda somam FOR no dano em T20)
 *  - Ranged: 0 (bows/firearms não somam atributo no dano por padrão)
 */
function computeAttackFor(
  weapon: EquippedWeapon,
  skills: Record<SkillId, SkillComputed>,
  attributes: Record<AttributeKey, AttributeComputed>,
  attackBuff: number,
  damageBuff: number,
): ComputedAttack {
  const skillId: 'luta' | 'pontaria' =
    weapon.purpose === 'ranged' ? 'pontaria' : 'luta'
  const attackTotal = skills[skillId].total + attackBuff
  const damageAttributeBonus =
    weapon.purpose === 'ranged' ? 0 : attributes.strength.total
  return {
    weaponName: weapon.name,
    skill: skillId,
    attackTotal,
    damageDice: weapon.damage,
    damageAttributeBonus: damageAttributeBonus + damageBuff,
    damageType: weapon.damageType,
    critRange: weapon.critRange,
    critMult: weapon.critMult,
    hand: weapon.hand,
    purpose: weapon.purpose,
  }
}

function computeAttacks(
  input: CharacterInput,
  skills: Record<SkillId, SkillComputed>,
  attributes: Record<AttributeKey, AttributeComputed>,
  buffTotals: Record<string, number>,
): ComputedSheet['attacks'] {
  const eq = input.equipment
  const attackBuff = buffTotals.attack ?? 0
  const damageBuff = buffTotals.damage ?? 0
  return {
    mainHand: eq?.mainHand
      ? computeAttackFor(eq.mainHand, skills, attributes, attackBuff, damageBuff)
      : null,
    offHand: eq?.offHand
      ? computeAttackFor(eq.offHand, skills, attributes, attackBuff, damageBuff)
      : null,
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
