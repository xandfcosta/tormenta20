import type { ValidationError } from '@/shared/rules/rules-types'
/** The three ways an item can be worn/held. `null` = stowed. */
export type EquippedState = 'vested' | 'wielded' | 'wielded2'

/** Hand-slots an equip state occupies. Mirrors backend `assertEquipLimits`. */
export const HANDS_LIMIT = 2
export const VESTED_LIMIT = 4

function handsFor(slot: EquippedState | null): number {
  return slot === 'wielded' ? 1 : slot === 'wielded2' ? 2 : 0
}

/**
 * Equip caps (PDF: 4 vestidos, 2 mãos). Pass the equip state of every OTHER
 * item on the character (exclude the one being changed) plus the `incoming`
 * slot; returns the same field errors the backend would throw.
 *
 * @example
 * validateEquipChange(['vested', 'vested'], 'vested') // [] (3 ≤ 4)
 * validateEquipChange(['wielded2'], 'wielded') // hands 2+1 > 2 → error
 */
export function validateEquipChange(
  otherEquipped: readonly (EquippedState | null)[],
  incoming: EquippedState,
): ValidationError[] {
  const vested = otherEquipped.filter((s) => s === 'vested').length
  const hands = otherEquipped.reduce((sum, s) => sum + handsFor(s), 0)
  const errors: ValidationError[] = []
  if (vested + (incoming === 'vested' ? 1 : 0) > VESTED_LIMIT) {
    errors.push({
      field: 'equipped',
      message: `Limite de ${VESTED_LIMIT} itens vestidos atingido`,
    })
  }
  if (hands + handsFor(incoming) > HANDS_LIMIT) {
    errors.push({
      field: 'equipped',
      message: `Limite de ${HANDS_LIMIT} mãos atingido`,
    })
  }
  return errors
}
