import { describe, expect, it } from 'vitest'
import type { CharacterItem } from '@/shared/api/api'
import { equipOptionsFor } from './equip-options'

function item(catalogId: string | null): CharacterItem {
  return {
    id: 1,
    catalogId,
    name: 'x',
    quantity: 1,
    slots: 1,
    equipped: null,
    improvements: '[]',
    material: null,
  } as CharacterItem
}

function values(catalogId: string | null): string[] {
  return equipOptionsFor(item(catalogId)).map((o) => o.value)
}

// Regression (bug E): every item used to offer —/Vestido/1 mão/2 mãos, letting
// a shield be persisted as "Vestido". Options now follow the catalog entry.
describe('equipOptionsFor', () => {
  it('armor (vested) → only — and Vestido', () => {
    expect(values('armadura-completa')).toEqual(['', 'vested'])
  })

  it('shield (wielded, 1 hand) → only — and 1 mão', () => {
    expect(values('escudo-leve')).toEqual(['', 'wielded'])
  })

  it('two-handed weapon → only — and 2 mãos', () => {
    expect(values('montante')).toEqual(['', 'wielded2'])
  })

  it('versátil weapon → —, 1 mão and 2 mãos', () => {
    expect(values('mangual')).toEqual(['', 'wielded', 'wielded2'])
  })

  it('consumable → only —', () => {
    expect(values('balsamo-restaurador')).toEqual([''])
  })

  it('either-slot gear → —, Vestido and 1 mão', () => {
    expect(values('algemas')).toEqual(['', 'vested', 'wielded'])
  })

  it('custom item (no catalog) keeps the full list', () => {
    expect(values(null)).toEqual(['', 'vested', 'wielded', 'wielded2'])
  })

  it('homebrew registry: Medalhão de prata also offers Vestido', () => {
    // HOMEBREW_VESTED_OK — wearable esotérico; bonus still gated by the
    // Efeitos toggle, but the slot itself must be selectable.
    expect(values('medalhao-de-prata')).toEqual(['', 'vested', 'wielded'])
  })

  it('non-registry esotérico stays wielded-only (orbe)', () => {
    expect(values('orbe-cristalino')).toEqual(['', 'wielded'])
  })
})
