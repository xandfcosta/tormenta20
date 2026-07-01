import { describe, expect, it } from 'vitest'
import { mulberry32 } from '../loot-rng'
import {
  ACCESSORIES_MAIOR_ROWS,
  ACCESSORIES_MEDIO_ROWS,
  ACCESSORIES_MENOR_ROWS,
  rollMagicAccessory,
} from '../magic-accessory-tables'

/**
 * PDF Cap 8, p342-343 (Tabelas 8-13/8-14/8-15). Pinned:
 *  - 3 tabelas cobertura d% [1, 100] no gaps.
 *  - Cada tier retorna nome do próprio catálogo.
 *  - Determinismo total.
 */

const TABLES = [
  { name: 'menor', rows: ACCESSORIES_MENOR_ROWS },
  { name: 'medio', rows: ACCESSORIES_MEDIO_ROWS },
  { name: 'maior', rows: ACCESSORIES_MAIOR_ROWS },
] as const

describe('accessory tables — shape & coverage', () => {
  for (const t of TABLES) {
    it(`${t.name}: cobertura [1, 100] sem gaps`, () => {
      let cursor = 1
      for (const row of t.rows) {
        expect(row[0]).toBe(cursor)
        cursor = row[1] + 1
      }
      expect(cursor).toBe(101)
    })

    it(`${t.name}: frozen`, () => {
      expect(Object.isFrozen(t.rows)).toBe(true)
    })

    it(`${t.name}: nomes únicos`, () => {
      const names = t.rows.map((r) => r[2])
      expect(new Set(names).size).toBe(names.length)
    })
  }
})

describe('rollMagicAccessory — pinned entries', () => {
  it('menor 1-2 = Anel do sustento', () => {
    expect(ACCESSORIES_MENOR_ROWS[0]).toEqual([1, 2, 'Anel do sustento'])
  })

  it('menor 99-100 = Pingente da saúde', () => {
    expect(ACCESSORIES_MENOR_ROWS.at(-1)).toEqual([99, 100, 'Pingente da saúde'])
  })

  it('medio 95-100 = Tiara da sapiência', () => {
    expect(ACCESSORIES_MEDIO_ROWS.at(-1)).toEqual([
      95,
      100,
      'Tiara da sapiência',
    ])
  })

  it('maior 99-100 = Espelho do aprisionamento', () => {
    expect(ACCESSORIES_MAIOR_ROWS.at(-1)).toEqual([
      99,
      100,
      'Espelho do aprisionamento',
    ])
  })
})

describe('rollMagicAccessory — roll behavior', () => {
  it('tier=menor retorna nome do catálogo menor', () => {
    const catalog = new Set(ACCESSORIES_MENOR_ROWS.map((r) => r[2]))
    const rng = mulberry32(1)
    for (let i = 0; i < 200; i++) {
      const r = rollMagicAccessory(rng, 'menor')
      expect(r.tier).toBe('menor')
      expect(catalog.has(r.name)).toBe(true)
    }
  })

  it('tier=medio retorna nome do catálogo medio', () => {
    const catalog = new Set(ACCESSORIES_MEDIO_ROWS.map((r) => r[2]))
    const rng = mulberry32(2)
    for (let i = 0; i < 200; i++) {
      const r = rollMagicAccessory(rng, 'medio')
      expect(catalog.has(r.name)).toBe(true)
    }
  })

  it('tier=maior retorna nome do catálogo maior', () => {
    const catalog = new Set(ACCESSORIES_MAIOR_ROWS.map((r) => r[2]))
    const rng = mulberry32(3)
    for (let i = 0; i < 200; i++) {
      const r = rollMagicAccessory(rng, 'maior')
      expect(catalog.has(r.name)).toBe(true)
    }
  })

  it('determinismo — mesma seed → mesmo output em todos tiers', () => {
    for (const tier of ['menor', 'medio', 'maior'] as const) {
      const a = mulberry32(2026)
      const b = mulberry32(2026)
      for (let i = 0; i < 20; i++) {
        expect(rollMagicAccessory(a, tier)).toEqual(rollMagicAccessory(b, tier))
      }
    }
  })
})
