import { describe, expect, it } from 'vitest'
import { mulberry32 } from '../loot-rng'
import {
  SPECIFIC_WEAPON_ROWS,
  WEAPON_ENCHANT_ROWS,
  rollMagicWeapon,
} from '../magic-weapon-tables'

/**
 * PDF Cap 8 p336-337 (Tabelas 8-8 + 8-9). Pinned:
 *  - 8-8 cobertura d% [1, 100] contígua com 29 rows.
 *  - 8-9 cobertura d% [1, 100] contígua com 18 rows.
 *  - Footnote *: Energética/Lancinante/Magnífica countsAsTwo → menor reroll.
 *  - Row 91-100 na 8-8 rota pra 8-9.
 *  - Determinismo total.
 */

describe('WEAPON_ENCHANT_ROWS — Tabela 8-8', () => {
  it('cobre [1, 100] sem gaps', () => {
    let cursor = 1
    for (const row of WEAPON_ENCHANT_ROWS) {
      expect(row[0]).toBe(cursor)
      cursor = row[1] + 1
    }
    expect(cursor).toBe(101)
  })

  it('29 rows', () => {
    expect(WEAPON_ENCHANT_ROWS.length).toBe(29)
  })

  it('Energética/Lancinante/Magnífica marcadas countsAsTwo', () => {
    const marks = WEAPON_ENCHANT_ROWS.filter((r) => r[3] === true).map(
      (r) => r[2],
    )
    expect(marks).toEqual(['Energética', 'Lancinante', 'Magnífica'])
  })

  it('sentinel 91-100 = "Arma específica"', () => {
    const last = WEAPON_ENCHANT_ROWS.at(-1)!
    expect(last[0]).toBe(91)
    expect(last[1]).toBe(100)
    expect(last[2]).toBe('Arma específica')
  })

  it('frozen', () => {
    expect(Object.isFrozen(WEAPON_ENCHANT_ROWS)).toBe(true)
  })
})

describe('SPECIFIC_WEAPON_ROWS — Tabela 8-9', () => {
  it('cobre [1, 100] sem gaps', () => {
    let cursor = 1
    for (const row of SPECIFIC_WEAPON_ROWS) {
      expect(row[0]).toBe(cursor)
      cursor = row[1] + 1
    }
    expect(cursor).toBe(101)
  })

  it('18 rows', () => {
    expect(SPECIFIC_WEAPON_ROWS.length).toBe(18)
  })

  it('primeira row 1-5 = Azagaia dos relâmpagos', () => {
    expect(SPECIFIC_WEAPON_ROWS[0]).toEqual([1, 5, 'Azagaia dos relâmpagos'])
  })

  it('última row 96-100 = Vingadora sagrada', () => {
    expect(SPECIFIC_WEAPON_ROWS.at(-1)).toEqual([96, 100, 'Vingadora sagrada'])
  })
})

describe('rollMagicWeapon', () => {
  const ENCHANT_NAMES = new Set(
    WEAPON_ENCHANT_ROWS.filter((r) => r[2] !== 'Arma específica').map(
      (r) => r[2],
    ),
  )
  const SPECIFIC_NAMES = new Set(SPECIFIC_WEAPON_ROWS.map((r) => r[2]))

  it('nomes encanto vêm de 8-8 (excluindo sentinel)', () => {
    const rng = mulberry32(1)
    let encantoHits = 0
    for (let i = 0; i < 500; i++) {
      const r = rollMagicWeapon(rng, 'medio')
      if (r.kind === 'encanto') {
        encantoHits++
        expect(ENCHANT_NAMES.has(r.name)).toBe(true)
      }
    }
    expect(encantoHits).toBeGreaterThan(0)
  })

  it('nomes specific vêm de 8-9', () => {
    const rng = mulberry32(2)
    let specificHits = 0
    for (let i = 0; i < 500; i++) {
      const r = rollMagicWeapon(rng, 'medio')
      if (r.kind === 'specific') {
        specificHits++
        expect(SPECIFIC_NAMES.has(r.name)).toBe(true)
      }
    }
    expect(specificHits).toBeGreaterThan(0)
  })

  it('menor: countsAsTwo encantos nunca aparecem (footnote reroll)', () => {
    const rng = mulberry32(3)
    for (let i = 0; i < 2000; i++) {
      const r = rollMagicWeapon(rng, 'menor')
      if (r.kind === 'encanto') {
        expect(r.countsAsTwo).toBe(false)
      }
    }
  })

  it('medio/maior: countsAsTwo encantos aparecem', () => {
    const rng = mulberry32(4)
    let seenCountsAsTwo = 0
    for (let i = 0; i < 2000; i++) {
      const r = rollMagicWeapon(rng, 'medio')
      if (r.kind === 'encanto' && r.countsAsTwo) seenCountsAsTwo++
    }
    expect(seenCountsAsTwo).toBeGreaterThan(0)
  })

  it('tier propagado no result', () => {
    const rng = mulberry32(5)
    for (const tier of ['menor', 'medio', 'maior'] as const) {
      for (let i = 0; i < 50; i++) {
        const r = rollMagicWeapon(rng, tier)
        expect(r.tier).toBe(tier)
      }
    }
  })

  it('determinismo', () => {
    const a = mulberry32(2026)
    const b = mulberry32(2026)
    for (let i = 0; i < 30; i++) {
      expect(rollMagicWeapon(a, 'medio')).toEqual(rollMagicWeapon(b, 'medio'))
    }
  })

  it('specific ~10% rolls (row 91-100 é 10% da distribuição)', () => {
    const rng = mulberry32(42)
    let specific = 0
    for (let i = 0; i < 3000; i++) {
      if (rollMagicWeapon(rng, 'medio').kind === 'specific') specific++
    }
    expect(specific / 3000).toBeGreaterThan(0.05)
    expect(specific / 3000).toBeLessThan(0.2)
  })
})
