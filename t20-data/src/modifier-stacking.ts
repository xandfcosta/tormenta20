/**
 * Regras de acumulação de efeitos (Modifier Stacking).
 *
 * PDF Cap 5 Jogando p226 — seções "Acumulando Efeitos", "Efeitos que
 * Afetam Testes", sidebar "Clarificações de Regras".
 *
 * IMPORTANTE: Tormenta 20 **NÃO usa** taxonomia D&D/Pathfinder de bônus
 * tipados (armor/enhancement/morale/dodge/etc.). Acumulação é por
 * **fonte** (habilidade, perícia, item, magia, parceiro, ambiente), não
 * por tipo de bônus. O type `BonusType` em `items/types.ts` cobre
 * agrupamento de origem para o inventário; para regras de stack, use
 * `ModifierSource` desta módulo.
 *
 * Regras principais:
 *  - Diferentes fontes: acumulam (ex.: +1 item + +1 magia = +2).
 *  - Mesma fonte de itens/magias/parceiros/ambiente: NÃO acumula (só
 *    o maior aplica).
 *  - Habilidades e perícias: acumulam entre si, exceto quando vierem
 *    da MESMA habilidade/perícia (caller filtra pelo effectId).
 *  - Armadura + escudo + 1 outro item: bônus de defesa e penalidade
 *    de armadura empilham.
 *  - Atributo em perícia: soma o atributo-chave uma única vez, mesmo
 *    que outra habilidade também some o mesmo atributo.
 *  - Chance de falha: cap 75% (min 1 em 4 de acertar).
 *  - Redução de custo em PM: piso 1 PM.
 *  - Timing: modificadores usados antes do rolo; re-rolagens antes
 *    do mestre declarar sucesso/falha.
 *  - Arredondar: sempre para baixo salvo indicação.
 *  - Múltiplos multiplicadores: primeiro aplica cheio, subsequentes
 *    contribuem `m - 1`. Ex.: 2× + 2× = 3×; 2× + 3× = 4×.
 */

// ─── Types ────────────────────────────────────────────────────────────
export type ModifierSource =
  | 'habilidade'
  | 'pericia'
  | 'item'
  | 'magia'
  | 'parceiro'
  | 'ambiente'

// ─── Constantes ──────────────────────────────────────────────────────
/** Chance de falha máxima acumulável (p226). */
export const MISS_CHANCE_CAP = 0.75

/** Chance mínima de acerto (1 em 4). */
export const MIN_HIT_CHANCE = 0.25

/** Custo mínimo em PM após reduções (p226). */
export const PM_COST_FLOOR = 1

/** Timing padrão: modificador declarado antes do rolo (p226). */
export const MODIFIER_TIMING = 'before-roll' as const

/** Arredondamento padrão em T20 (p226). */
export const ROUNDING_RULE = 'floor' as const

/** Bônus de defesa e penalidade de armadura de escudo+armadura empilham com 1 item extra (p226). */
export const ARMOR_SHIELD_PLUS_ONE_ITEM_STACKS = true

/**
 * Set de fontes que NÃO acumulam consigo mesmas (só o maior aplica).
 * Frozen.
 */
export const NON_SELF_STACKING_SOURCES: ReadonlySet<ModifierSource> = new Set([
  'item',
  'magia',
  'parceiro',
  'ambiente',
])

/**
 * Set de fontes que acumulam entre si mesmas (efeitos diferentes de
 * uma habilidade + outra habilidade empilham). Frozen.
 */
export const SELF_STACKING_SOURCES: ReadonlySet<ModifierSource> = new Set([
  'habilidade',
  'pericia',
])

// ─── Helpers — stacksWith ───────────────────────────────────────────
/**
 * True se um modificador de fonte `a` empilha com outro de fonte `b`.
 *
 * Regras (p226):
 *  - Fontes diferentes SEMPRE empilham (ex.: item + magia = 2).
 *  - Mesma fonte em {item, magia, parceiro, ambiente} NÃO empilha.
 *  - Mesma fonte em {habilidade, pericia} empilha — desde que sejam
 *    habilidades/perícias distintas (caller checa pelo effectId).
 *
 * Não avalia `effectId`. Para bloquear duplicação da MESMA habilidade
 * ou MESMA perícia, use `sameEffectStacks(effectIdA, effectIdB)`.
 */
export function stacksWith(a: ModifierSource, b: ModifierSource): boolean {
  if (a !== b) return true
  return SELF_STACKING_SOURCES.has(a)
}

/**
 * True se dois modificadores com o mesmo `effectId` empilham. Regra
 * simples: não empilham (mesma habilidade/perícia/item não conta duas
 * vezes, só o maior aplica).
 */
export function sameEffectStacks(
  effectIdA: string,
  effectIdB: string,
): boolean {
  return effectIdA !== effectIdB
}

// ─── Helpers — cap / floor ──────────────────────────────────────────
/**
 * Aplica cap de 75% à chance de falha acumulada (p226).
 * Aceita 0..1 (ou percentual normalizado).
 */
export function capMissChance(chance: number): number {
  if (chance < 0) {
    throw new Error(`capMissChance: chance must be ≥ 0, got ${chance}`)
  }
  return Math.min(chance, MISS_CHANCE_CAP)
}

/**
 * Chance de acerto residual após aplicar o cap na chance de falha.
 * Nunca fica abaixo de 25% (1 em 4).
 */
export function hitChanceAfterMissCap(missChance: number): number {
  return 1 - capMissChance(missChance)
}

/**
 * Aplica piso de 1 PM no custo reduzido de uma magia (p226).
 * Reduções abaixo de 1 PM não são permitidas.
 */
export function floorPmCost(reducedCost: number): number {
  return Math.max(reducedCost, PM_COST_FLOOR)
}

// ─── Helpers — múltiplos multiplicadores (p226) ─────────────────────
/**
 * Combina múltiplos multiplicadores conforme p226:
 * primeiro aplica cheio, subsequentes contribuem `m - 1`.
 *
 * Ex.: [2, 2] → 3; [2, 3] → 4; [2, 2, 2] → 4; [3, 3] → 5.
 * Lista vazia → 1 (identidade).
 */
export function combineMultipliers(
  multipliers: readonly number[],
): number {
  if (multipliers.length === 0) return 1
  for (const m of multipliers) {
    if (m < 1) {
      throw new Error(
        `combineMultipliers: each multiplier must be ≥ 1, got ${m}`,
      )
    }
  }
  const [first, ...rest] = multipliers
  return first + rest.reduce((sum, m) => sum + (m - 1), 0)
}

// ─── Helpers — arredondamento (p226) ────────────────────────────────
/** Sempre arredonda para baixo (regra padrão T20). */
export function roundDown(value: number): number {
  return Math.floor(value)
}

// ─── Helpers — resolução de múltiplos bônus da mesma fonte ──────────
/**
 * Dado um conjunto de bônus da mesma fonte (mesma categoria), retorna
 * apenas o maior — comportamento para item/magia/parceiro/ambiente.
 */
export function bestBonusOnly(bonuses: readonly number[]): number {
  if (bonuses.length === 0) return 0
  return Math.max(...bonuses)
}

/**
 * Dado um conjunto de bônus da mesma fonte "empilhável" (habilidade ou
 * perícia), retorna a soma — comportamento quando efeitos vêm de
 * habilidades/perícias distintas.
 */
export function sumStackingBonuses(bonuses: readonly number[]): number {
  return bonuses.reduce((sum, b) => sum + b, 0)
}
