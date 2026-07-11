/**
 * Encounter difficulty math — Book Cap 7 p282 rules for computing the
 * effective ND of a group of monsters.
 *
 *   ND < 1 → groupNd = monsterNd × quantity
 *            (four ND 1/4 = ND 1; two ND 1/2 = ND 1)
 *   ND ≥ 1 → groupNd = monsterNd + 2 for each doubling of quantity
 *            (two ND 1 = ND 3; four ND 5 = ND 9)
 *
 * `Math.log2(quantity)` extends the rule to non-integer doublings so
 * a group of 3 lands between 1× and 2×. Callers that want a cleaner
 * integer ND can Math.round the result.
 */
export function computeGroupNd(
  monsterNd: number,
  quantity: number,
): number {
  if (quantity <= 0) return 0
  if (monsterNd < 1) return monsterNd * quantity
  if (quantity === 1) return monsterNd
  return monsterNd + 2 * Math.log2(quantity)
}
