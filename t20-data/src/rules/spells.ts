import { SPELL_CATALOG } from '../spell-catalog'
import type { ValidationError } from './types'

/**
 * Learning a spell: it must exist in the catalog and not already be in the
 * character's spellbook. Mirrors backend `learnSpell` (assertSpellExists +
 * unique(characterId, catalogSpellId)).
 */
export function validateLearnSpell(
  knownSpellIds: readonly string[],
  catalogSpellId: string,
): ValidationError[] {
  if (!SPELL_CATALOG[catalogSpellId]) {
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
 * a `buff` block. Mirrors backend `applyEffect`.
 */
export function validateApplyBuff(spellId: string): ValidationError[] {
  const spell = SPELL_CATALOG[spellId]
  if (!spell?.buff) {
    return [{ field: 'spellId', message: 'Magia sem efeito aplicável' }]
  }
  return []
}

/**
 * Cast preconditions (PDF p171). `totalPm` is the already-composed cost (base +
 * augments − catalisador); this only checks the aggregate invariants so the
 * caller keeps ownership of the PM math. Mirrors backend `castSpell`:
 *   - prepared casters must have the spell prepared;
 *   - circle > 0 spells: cost ≤ per-spell PM limit (⌊nível/2⌋ + item bonuses);
 *   - cost ≤ current PM.
 */
export function validateCast(input: {
  circle: number
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
  if (input.circle > 0 && input.totalPm > input.pmLimit) {
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
