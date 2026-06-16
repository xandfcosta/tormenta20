import { describe, expect, it } from 'vitest'
import { MATERIALS } from '../catalog/materials'
import type { ItemFamily } from '../types'

/**
 * PDF Cap 3 (Equipamento) — Materiais Especiais (p162). Six materials are
 * implemented as catalog overlays attached via CharacterItem.material:
 *
 *  - Aço-rubi (T$1000)        — weapon only,           +2 dano vs vivos
 *  - Adamante (T$5000)        — weapon/armor/shield,   ignora RD não-mágica
 *  - Mitral (T$1000)          — armor/shield,          -2 penalidade
 *  - Gelo eterno (T$800)      — weapon only,           +1d6 frio
 *  - Madeira tollon (T$600)   — weapon/shield,         flavor-only
 *  - Matéria vermelha (T$1500) — weapon/armor/shield,  flavor-only
 */
type MaterialRow = {
  price: number
  appliesTo: ItemFamily[]
}

const BOOK_MATERIALS: Record<string, MaterialRow> = {
  'material-aco-rubi': { price: 1000, appliesTo: ['weapon'] },
  'material-adamante': {
    price: 5000,
    appliesTo: ['weapon', 'armor', 'shield'],
  },
  'material-mitral': { price: 1000, appliesTo: ['armor', 'shield'] },
  'material-gelo-eterno': { price: 800, appliesTo: ['weapon'] },
  'material-madeira-tollon': {
    price: 600,
    appliesTo: ['weapon', 'shield'],
  },
  'material-materia-vermelha': {
    price: 1500,
    appliesTo: ['weapon', 'armor', 'shield'],
  },
}

describe('MATERIALS vs PDF p162', () => {
  it('catalog contains all 6 materials', () => {
    expect(MATERIALS).toHaveLength(6)
  })

  for (const [id, row] of Object.entries(BOOK_MATERIALS)) {
    it(`${id}: price + appliesTo match the book`, () => {
      const material = MATERIALS.find((m) => m.id === id)
      expect(material, `${id} missing from catalog`).toBeDefined()
      expect(material!.price).toBe(row.price)
      expect([...material!.appliesTo!].sort()).toEqual(
        [...row.appliesTo].sort(),
      )
    })
  }
})

describe('Material catalog invariants', () => {
  it('every material has category=material and zero slots', () => {
    for (const m of MATERIALS) {
      expect(m.category).toBe('material')
      expect(m.slots).toBe(0)
    }
  })

  it('every material is overlay-shaped (appliesTo set, equip=either)', () => {
    for (const m of MATERIALS) {
      expect(m.appliesTo, `${m.id} missing appliesTo`).toBeDefined()
      expect(m.appliesTo!.length).toBeGreaterThan(0)
      expect(m.equip).toBe('either')
    }
  })

  it('appliesTo families are limited to weapon/armor/shield (no apparel/any)', () => {
    const valid: ItemFamily[] = ['weapon', 'armor', 'shield']
    for (const m of MATERIALS) {
      for (const fam of m.appliesTo!) {
        expect(valid).toContain(fam)
      }
    }
  })
})
