import { describe, expect, it } from 'vitest'
import type { CharacterItem } from '@/shared/api/api'
import { equipBonuses } from './equip-bonuses'

// Minimal CharacterItem stub — only the fields equipBonuses reads matter.
function item(catalogId: string | null): CharacterItem {
  return {
    id: 1,
    catalogId,
    name: 'x',
    quantity: 1,
    slots: 1,
    equipped: 'vested',
    improvements: '[]',
    material: null,
  } as CharacterItem
}

describe('equipBonuses', () => {
  it('lists an apparel expertise bonus', () => {
    expect(equipBonuses(item('bandana'))).toContain('Perícia Intimidação +1')
  })

  it('lists armor defense', () => {
    expect(equipBonuses(item('armadura-couro'))).toContain('Defesa +2')
  })

  it('lists weapon damage', () => {
    expect(equipBonuses(item('espada-curta'))).toContain('Dano 1d6')
  })

  it('returns nothing for a custom item (no catalog)', () => {
    expect(equipBonuses(item(null))).toEqual([])
  })
})
