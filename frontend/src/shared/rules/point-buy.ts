import { ATTRIBUTE_KEYS } from '@/shared/api/attribute-keys'
import type { AttributeKey } from '@/shared/api/attribute-keys'
/**
 * Compra de pontos (book p17, Tabela 1-1): all six attributes start at 0 and
 * the player gets 10 points. Costs: 1→1, 2→2, 3→4, 4→7. Exactly ONE attribute
 * may be reduced to −1 to refund 1 point ("reduzir UM atributo para −1").
 * Point-buy range is −1..4 (racial mods apply after, outside the budget).
 */

export const POINT_BUY_BUDGET = 10
export const POINT_BUY_MIN = -1
export const POINT_BUY_MAX = 4

const COSTS = new Map<number, number>([
  [-1, -1],
  [0, 0],
  [1, 1],
  [2, 2],
  [3, 4],
  [4, 7],
])

/** Cost of one attribute value. Throws for values outside −1..4. */
export function pointBuyCost(value: number): number {
  const cost = COSTS.get(value)
  if (cost === undefined) {
    throw new Error(
      `pointBuyCost: value must be in [${POINT_BUY_MIN}, ${POINT_BUY_MAX}], got ${value}`,
    )
  }
  return cost
}

/** Total points spent by a full BASE attribute spread (pre-race). */
export function pointBuySpent(
  attrs: Readonly<Record<AttributeKey, number>>,
): number {
  return ATTRIBUTE_KEYS.reduce((sum, k) => sum + pointBuyCost(attrs[k] ?? 0), 0)
}

/**
 * Advisory validation of a spread against the p17 rules. Empty = legal.
 * Out-of-range values are reported (not thrown) so a live-editing UI can
 * show the problem without crashing.
 */
export function pointBuyWarnings(
  attrs: Readonly<Record<AttributeKey, number>>,
): string[] {
  const warnings: string[] = []
  let spent = 0
  let reduced = 0
  for (const k of ATTRIBUTE_KEYS) {
    const v = attrs[k] ?? 0
    if (v < POINT_BUY_MIN || v > POINT_BUY_MAX) {
      warnings.push(
        `compra de pontos: ${k} = ${v} fora do intervalo [${POINT_BUY_MIN}, ${POINT_BUY_MAX}]`,
      )
      continue
    }
    if (v === -1) reduced++
    spent += pointBuyCost(v)
  }
  if (reduced > 1) {
    warnings.push(
      `compra de pontos: apenas UM atributo pode ser reduzido a −1 (p17), há ${reduced}`,
    )
  }
  if (spent > POINT_BUY_BUDGET) {
    warnings.push(
      `compra de pontos: ${spent} pontos gastos excedem o limite de ${POINT_BUY_BUDGET}`,
    )
  }
  return warnings
}
