import { describe, expect, it } from 'vitest'
import { mulberry32 } from '../loot-rng'
import {
  POTION_ROWS,
  rollMagicPotion,
  tierForPotionRoll,
} from '../magic-potion-tables'

/**
 * PDF Cap 8 p341 (Tabela 8-12 Poções). Pinned:
 *  - 38 rows cobrindo d% [1, 100] contíguo.
 *  - Tier bands: menor 1-36 (T$ 30), medio 37-80 (T$ 120-270),
 *    maior 81-100 (T$ 750-3.000).
 *  - Determinismo total.
 */

describe('POTION_ROWS — shape & coverage', () => {
  it('cobre [1, 100] sem gaps', () => {
    let cursor = 1
    for (const row of POTION_ROWS) {
      expect(row[0]).toBe(cursor)
      cursor = row[1] + 1
    }
    expect(cursor).toBe(101)
  })

  it('38 rows total', () => {
    expect(POTION_ROWS.length).toBe(38)
  })

  it('primeira row 1-1 = Abençoar Alimentos T$30', () => {
    expect(POTION_ROWS[0]).toEqual([1, 1, 'Abençoar Alimentos (óleo)', 30])
  })

  it('última row 97-100 = Curar Ferimentos 11d8+11 T$3000', () => {
    expect(POTION_ROWS.at(-1)).toEqual([
      97,
      100,
      'Curar Ferimentos (11d8+11 PV)',
      3000,
    ])
  })

  it('frozen', () => {
    expect(Object.isFrozen(POTION_ROWS)).toBe(true)
  })

  it('preços em bandas menor/medio/maior', () => {
    for (const row of POTION_ROWS) {
      const price = row[3]
      const menor = row[1] <= 36
      const medio = row[0] >= 37 && row[1] <= 80
      const maior = row[0] >= 81
      if (menor) expect(price).toBe(30)
      if (medio) expect([120, 270]).toContain(price)
      if (maior) expect([750, 1080, 1470, 3000]).toContain(price)
    }
  })
})

describe('tierForPotionRoll — bandas p341', () => {
  it('1-36 = menor', () => {
    for (let r = 1; r <= 36; r++) {
      expect(tierForPotionRoll(r)).toBe('menor')
    }
  })

  it('37-80 = medio', () => {
    for (let r = 37; r <= 80; r++) {
      expect(tierForPotionRoll(r)).toBe('medio')
    }
  })

  it('81-100 = maior', () => {
    for (let r = 81; r <= 100; r++) {
      expect(tierForPotionRoll(r)).toBe('maior')
    }
  })
})

describe('rollMagicPotion', () => {
  it('sempre retorna { name, priceTs, tier, roll } válido', () => {
    const catalog = new Set(POTION_ROWS.map((r) => r[2]))
    const rng = mulberry32(7)
    for (let i = 0; i < 500; i++) {
      const r = rollMagicPotion(rng)
      expect(catalog.has(r.name)).toBe(true)
      expect(r.priceTs).toBeGreaterThan(0)
      expect(['menor', 'medio', 'maior']).toContain(r.tier)
      expect(r.roll).toBeGreaterThanOrEqual(1)
      expect(r.roll).toBeLessThanOrEqual(100)
    }
  })

  it('determinismo', () => {
    const a = mulberry32(2026)
    const b = mulberry32(2026)
    for (let i = 0; i < 20; i++) {
      expect(rollMagicPotion(a)).toEqual(rollMagicPotion(b))
    }
  })

  it('distribuição sanidade: rolls tier=menor mais comum (36% nominal)', () => {
    const rng = mulberry32(42)
    let menor = 0
    for (let i = 0; i < 1000; i++) {
      if (rollMagicPotion(rng).tier === 'menor') menor++
    }
    expect(menor / 1000).toBeGreaterThan(0.25)
    expect(menor / 1000).toBeLessThan(0.5)
  })
})
