import type { ValidationError } from '@/shared/rules/rules-types'
// This module is DATA-FREE — the two catalog-dependent checks take the lookup
// RESULT as a param (spellExists / hasBuff) instead of importing SPELL_CATALOG,
// so the frontend can import these validators without bundling the ~175KB spell
// catalog (project_front_decouple_catalog A). Callers resolve the lookup: the
// backend from SPELL_CATALOG, the front from its primed spell-cache.

/**
 * Learning a spell: it must exist in the catalog and not already be in the
 * character's spellbook. Mirrors backend `learnSpell` (assertSpellExists +
 * unique(characterId, catalogSpellId)). `spellExists` is the catalog lookup
 * result (`!!SPELL_CATALOG[id]` / cache equivalent).
 */
export function validateLearnSpell(
  knownSpellIds: readonly string[],
  catalogSpellId: string,
  spellExists: boolean,
): ValidationError[] {
  if (!spellExists) {
    return [{ field: 'catalogSpellId', message: `Magia "${catalogSpellId}" não existe` }]
  }
  if (knownSpellIds.includes(catalogSpellId)) {
    return [{ field: 'catalogSpellId', message: 'Magia já conhecida' }]
  }
  return []
}

/** Preparing/toggling a spell requires it to already be learned. */
export function validateSpellLearned(
  knownSpellIds: readonly string[],
  catalogSpellId: string,
): ValidationError[] {
  if (!knownSpellIds.includes(catalogSpellId)) {
    return [{ field: 'catalogSpellId', message: 'Magia não aprendida' }]
  }
  return []
}

/**
 * Applying a spell as a scoped ActiveEffect requires the catalog entry to carry
 * a `buff` block. Mirrors backend `applyEffect`. `hasBuff` is the lookup result
 * (`!!SPELL_CATALOG[id]?.buff` / cache equivalent).
 */
export function validateApplyBuff(hasBuff: boolean): ValidationError[] {
  if (!hasBuff) {
    return [{ field: 'spellId', message: 'Magia sem efeito aplicável' }]
  }
  return []
}

/**
 * Cast preconditions. `totalPm` is the already-composed cost (base + augments −
 * catalisador); this only checks the aggregate invariants so the caller keeps
 * ownership of the PM math. Mirrors the Go `handleCastSpell`:
 *   - prepared casters must have the spell prepared;
 *   - circle > 0 spells: cost ≤ per-spell PM limit (p224), EXCEPT at the
 *     spell's own base cost — "mas você sempre pode usar a habilidade em seu
 *     custo mínimo" (p224). Without that clause a spell from outside the class
 *     became permanently uncastable;
 *   - cost ≤ current PM.
 */
export function validateCast(input: {
  circle: number
  basePm: number
  totalPm: number
  pmLimit: number
  mpCurrent: number
  needsPrep: boolean
  prepared: boolean
}): ValidationError[] {
  const errors: ValidationError[] = []
  if (input.needsPrep && !input.prepared) {
    errors.push({ field: 'prepared', message: 'Magia precisa estar preparada' })
  }
  if (input.circle > 0 && input.totalPm > input.pmLimit && input.totalPm > input.basePm) {
    errors.push({
      field: 'augments',
      message: `Limite PM excedido (${input.pmLimit})`,
    })
  }
  if (input.totalPm > input.mpCurrent) {
    errors.push({ field: 'mpCurrent', message: 'Sem PM suficiente' })
  }
  return errors
}

/**
 * Custo final de uma magia com os modificadores da p226 aplicados:
 *
 *   "Reduções de Custo. Reduções no custo de PM não são cumulativas. Uma
 *    habilidade nunca pode ter seu custo reduzido para menos de 1 PM."
 *
 * REDUÇÕES competem entre FONTES — a melhor vence — mas somam dentro de uma
 * fonte só, porque "os bônus dobram" da Força da Natureza (p63) é UMA habilidade
 * se duplicando. AUMENTOS (Alquebrado, p394) somam normalmente: o livro só
 * proíbe o acúmulo das reduções.
 *
 * Espelha `Catalogs.SpellPmCostFor` no Go, que é quem o servidor cobra — o
 * diálogo precisa mostrar o mesmo número, senão promete um preço que a API
 * recusa (ALE-110).
 *
 * @example spellPmCostWithMods(3, 0, [{ source: 'Força da Natureza', amount: -2 }]) // 1
 */
export function spellPmCostWithMods(
  basePm: number,
  augmentPm: number,
  contributions: readonly { source: string; amount: number }[],
): number {
  const raw = basePm + augmentPm
  if (raw <= 0) return 0

  const bySource = new Map<string, number>()
  for (const c of contributions) {
    bySource.set(c.source, (bySource.get(c.source) ?? 0) + c.amount)
  }
  let bestReduction = 0
  let increases = 0
  for (const amount of bySource.values()) {
    if (amount < bestReduction) bestReduction = amount
    if (amount > 0) increases += amount
  }
  return Math.max(1, raw + bestReduction + increases)
}
