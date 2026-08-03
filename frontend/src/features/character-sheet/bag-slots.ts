import type { CharacterItem } from '@/shared/api/api'

export type BagPartition = {
  /** A `wielded2` item occupies both hands (at most one can exist). */
  twoHand: CharacterItem | undefined
  wielded: CharacterItem[]
  vested: CharacterItem[]
  stowed: CharacterItem[]
  handsUsed: number
}

/**
 * Split the character's items into the bag's display groups — the two equip
 * capacity pools the rules track (Mãos ≤2, Vestidos ≤4, PDF p141) plus the
 * stowed remainder shown as the Mochila tile grid.
 *
 * @example partitionBag(items).handsUsed // 2 while a montante is wielded2
 */
export function partitionBag(items: readonly CharacterItem[]): BagPartition {
  const twoHand = items.find((i) => i.equipped === 'wielded2')
  const wielded = items.filter((i) => i.equipped === 'wielded')
  const vested = items.filter((i) => i.equipped === 'vested')
  const stowed = items.filter((i) => i.equipped === null)
  return {
    twoHand,
    wielded,
    vested,
    stowed,
    handsUsed: twoHand ? 2 : wielded.length,
  }
}
