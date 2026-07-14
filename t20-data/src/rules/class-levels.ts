import type { ValidationError } from './types'

/** Max total character level across all classes (PDF: nível 20). */
export const MAX_TOTAL_LEVEL = 20

/**
 * Class-level invariants: every class level ≥ 1 and the sum ≤ 20. Mirrors the
 * backend guard in `updateClassLevel` / character create.
 *
 * @example validateTotalLevel([{ className: 'Guerreiro', level: 21 }]) // sum > 20 → error
 */
export function validateTotalLevel(
  classes: readonly { className: string; level: number }[],
): ValidationError[] {
  const errors: ValidationError[] = []
  for (const c of classes) {
    if (!Number.isInteger(c.level) || c.level < 1) {
      errors.push({
        field: 'classes',
        message: `Nível de ${c.className} deve ser inteiro ≥ 1`,
      })
    }
  }
  const total = classes.reduce((sum, c) => sum + c.level, 0)
  if (total > MAX_TOTAL_LEVEL) {
    errors.push({
      field: 'classes',
      message: `Soma dos níveis (${total}) excede o máximo de ${MAX_TOTAL_LEVEL}`,
    })
  }
  return errors
}
