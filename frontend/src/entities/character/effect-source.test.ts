import { describe, expect, it } from 'vitest'
import type { CharacterItem } from '@/shared/api/api'
import {
  effectSourceName,
  equippedItemFlagEffects,
  ITEM_FLAG_LABEL,
} from './effect-source'

describe('effectSourceName', () => {
  it('resolves the manual temp-PV pool id to its pt-BR label (F3)', () => {
    expect(effectSourceName('manual-temp-hp')).toBe('PV temporários (manual)')
  })

  it('still resolves power ids through the activation registry', () => {
    expect(effectSourceName('class.barbaro.alma-de-bronze')).toBe(
      'Alma de Bronze',
    )
  })

  it('falls back to the raw id for unknown sources', () => {
    expect(effectSourceName('totally-unknown')).toBe('totally-unknown')
  })
})

// Named fake — only the fields equippedItemFlagEffects reads matter.
function fakeItem(overrides: Partial<CharacterItem>): CharacterItem {
  return {
    id: 1,
    catalogId: null,
    name: 'Item',
    quantity: 1,
    slots: 1,
    equipped: null,
    improvements: '[]',
    material: null,
    ...overrides,
  }
}

describe('equippedItemFlagEffects', () => {
  it('lists the three always-on flags of a vested armadura completa', () => {
    const effects = equippedItemFlagEffects([
      fakeItem({
        catalogId: 'armadura-completa',
        name: 'Armadura completa',
        equipped: 'vested',
      }),
    ])
    expect(effects.map((e) => e.flag).sort()).toEqual([
      'armadura-pesada',
      'cannot-apply-dex-to-defense',
      'fatigue-on-sleep',
    ])
    for (const e of effects) {
      expect(e.source).toBe('Armadura completa')
      expect(e.label).toBe(ITEM_FLAG_LABEL[e.flag])
    }
  })

  it('labels fatigue-on-sleep as Fadiga ao dormir (matches the header warning)', () => {
    const effects = equippedItemFlagEffects([
      fakeItem({ catalogId: 'brunea', name: 'Brunea', equipped: 'vested' }),
    ])
    expect(effects.map((e) => e.label)).toContain('Fadiga ao dormir')
  })

  it('ignores unequipped items — flags only apply while worn', () => {
    const effects = equippedItemFlagEffects([
      fakeItem({ catalogId: 'armadura-completa', equipped: null }),
    ])
    expect(effects).toEqual([])
  })

  it('ignores custom items without a catalog entry', () => {
    const effects = equippedItemFlagEffects([
      fakeItem({ catalogId: null, equipped: 'vested' }),
    ])
    expect(effects).toEqual([])
  })

  it('returns nothing for equipped items without flag modifiers', () => {
    const effects = equippedItemFlagEffects([
      fakeItem({ catalogId: 'espada-curta', equipped: 'wielded' }),
    ])
    expect(effects).toEqual([])
  })
})
