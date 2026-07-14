import type { ValidationError } from './types'

/**
 * A consumable can be used only while at least one remains. Mirrors the
 * backend `consumeItem` guard (quantity ≥ 1 before decrement).
 *
 * @example validateConsumeQuantity(0) // [{ field: 'quantity', ... }]
 */
export function validateConsumeQuantity(quantity: number): ValidationError[] {
  if (!Number.isInteger(quantity) || quantity < 1) {
    return [{ field: 'quantity', message: 'Item esgotado' }]
  }
  return []
}
