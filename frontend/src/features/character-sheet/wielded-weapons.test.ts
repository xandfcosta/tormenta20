import { describe, expect, it } from 'vitest'
import type { CharacterItem, EquippedSlot } from '@/shared/api/api'
import { hasWieldedWeapon, wieldedWeaponEntries } from './wielded-weapons'

// Minimal CharacterItem stub — only the fields the wielded predicate reads.
function item(
  catalogId: string | null,
  equipped: EquippedSlot | null,
): CharacterItem {
  return {
    id: 1,
    catalogId,
    name: catalogId ?? 'x',
    quantity: 1,
    slots: 1,
    equipped,
    improvements: '[]',
    material: null,
  } as CharacterItem
}

describe('hasWieldedWeapon', () => {
  it('is true for a wielded catalog weapon', () => {
    expect(hasWieldedWeapon({ items: [item('espada-curta', 'wielded')] })).toBe(
      true,
    )
  })

  it('is true for an off-hand (wielded2) weapon', () => {
    expect(
      hasWieldedWeapon({ items: [item('espada-curta', 'wielded2')] }),
    ).toBe(true)
  })

  it('is false when the weapon is only carried, not wielded', () => {
    expect(hasWieldedWeapon({ items: [item('espada-curta', null)] })).toBe(
      false,
    )
  })

  it('is false for a wielded non-weapon (shield)', () => {
    expect(hasWieldedWeapon({ items: [item('escudo-leve', 'wielded')] })).toBe(
      false,
    )
  })

  it('is false for a custom item without a catalog entry', () => {
    expect(hasWieldedWeapon({ items: [item(null, 'wielded')] })).toBe(false)
  })
})

describe('wieldedWeaponEntries', () => {
  it('resolves catalog weapon stats and keeps the item name', () => {
    const [entry] = wieldedWeaponEntries({
      items: [item('espada-curta', 'wielded')],
    })
    expect(entry?.name).toBe('espada-curta')
    expect(entry?.weapon.damage).toBeTruthy()
  })

  it('caps at two entries', () => {
    const items = [
      item('espada-curta', 'wielded'),
      item('espada-curta', 'wielded2'),
      item('espada-curta', 'wielded'),
    ]
    expect(wieldedWeaponEntries({ items })).toHaveLength(2)
  })
})
