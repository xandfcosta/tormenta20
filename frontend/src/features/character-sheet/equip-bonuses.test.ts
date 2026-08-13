import { describe, expect, it } from 'vitest'
import type { CatalogItem } from '@/shared/api/item-types'
import type { CharacterItem } from '@/shared/api/api'
import { catalogEquipChips, equipBonuses } from './equip-bonuses'

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

  // Regression (bug A): armors carry a defense modifier mirroring the base
  // stat, which used to render "Defesa +10 / Defesa +10" and break React keys.
  it('shows exactly one Defesa chip for armor whose modifier mirrors the base', () => {
    const defesa = equipBonuses(item('armadura-completa')).filter(
      (b) => b === 'Defesa +10',
    )
    expect(defesa).toHaveLength(1)
  })

  it('shows exactly one Defesa chip for a shield', () => {
    const defesa = equipBonuses(item('escudo-leve')).filter(
      (b) => b === 'Defesa +1',
    )
    expect(defesa).toHaveLength(1)
  })

  // Regression (bug B): flag targets used to leak as "Efeito: <slug> +1".
  it('labels flags in PT-BR without an amount suffix', () => {
    const chips = equipBonuses(item('armadura-completa'))
    expect(chips).toContain('Não soma Destreza na Defesa')
    expect(chips).toContain('Fadiga ao dormir')
    expect(chips).toContain('Conta como armadura pesada')
    expect(chips.some((c) => c.includes('Efeito:'))).toBe(false)
    expect(
      chips.filter((c) => c.startsWith('Não soma') || c.startsWith('Fadiga')),
    ).toEqual(['Não soma Destreza na Defesa', 'Fadiga ao dormir'])
  })
})

describe('catalogEquipChips', () => {
  // Named fake: an armor whose catalog entry has ONLY the base stat — the
  // chip must fall back to `armor.defense` instead of vanishing.
  const fakeArmorWithoutDefenseModifier: CatalogItem = {
    id: 'fake-armadura',
    name: 'Fake Armadura',
    category: 'armor-light',
    price: 0,
    slots: 1,
    equip: 'vested',
    armor: { defense: 3, penalty: 0, heavy: false },
    modifiers: [],
  }

  it('keeps the base Defesa chip when no defense modifier exists', () => {
    expect(catalogEquipChips(fakeArmorWithoutDefenseModifier)).toEqual([
      'Defesa +3',
    ])
  })
})
