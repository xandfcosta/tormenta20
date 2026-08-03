import { describe, expect, it } from 'vitest'
import type { CharacterItem } from '@/shared/api/api'
import { partitionBag } from './bag-slots'

function item(id: number, equipped: CharacterItem['equipped']): CharacterItem {
  return {
    id,
    catalogId: null,
    name: `item-${id}`,
    quantity: 1,
    slots: 1,
    equipped,
    improvements: '[]',
    material: null,
  } as CharacterItem
}

describe('partitionBag', () => {
  it('separa mãos, vestidos e guardados', () => {
    const p = partitionBag([
      item(1, 'wielded'),
      item(2, 'vested'),
      item(3, null),
      item(4, 'wielded'),
    ])
    expect(p.wielded.map((i) => i.id)).toEqual([1, 4])
    expect(p.vested.map((i) => i.id)).toEqual([2])
    expect(p.stowed.map((i) => i.id)).toEqual([3])
    expect(p.twoHand).toBeUndefined()
    expect(p.handsUsed).toBe(2)
  })

  it('wielded2 ocupa as duas mãos', () => {
    const p = partitionBag([item(1, 'wielded2'), item(2, null)])
    expect(p.twoHand?.id).toBe(1)
    expect(p.handsUsed).toBe(2)
    expect(p.stowed.map((i) => i.id)).toEqual([2])
  })

  it('mochila vazia: tudo zerado', () => {
    const p = partitionBag([])
    expect(p.handsUsed).toBe(0)
    expect(p.stowed).toEqual([])
  })
})
