import { describe, expect, it } from 'vitest'
import { mulberry32 } from '../loot-rng'
import {
  ARMOR_ENCHANT_ROWS,
  SPECIFIC_ARMOR_ROWS,
  rollMagicArmor,
} from '../magic-armor-tables'

/**
 * PDF Cap 8 p339-340 (Tabelas 8-10 + 8-11). Pinned:
 *  - 8-10 cobertura d% [1, 100] contígua com 26 rows.
 *  - 8-11 cobertura d% [1, 100] contígua com 13 rows.
 *  - Footnote ¹: Animado, Esmagador shieldOnly.
 *  - Footnote ²: Guardião countsAsTwo (menor reroll).
 *  - Row 91-100 na 8-10 rota pra 8-11.
 *  - Determinismo total.
 */

describe('ARMOR_ENCHANT_ROWS — Tabela 8-10', () => {
  it('cobre [1, 100] sem gaps', () => {
    let cursor = 1
    for (const row of ARMOR_ENCHANT_ROWS) {
      expect(row[0]).toBe(cursor)
      cursor = row[1] + 1
    }
    expect(cursor).toBe(101)
  })

  it('26 rows', () => {
    expect(ARMOR_ENCHANT_ROWS.length).toBe(26)
  })

  it('Animado + Esmagador marcados shieldOnly', () => {
    const shieldOnly = ARMOR_ENCHANT_ROWS.filter(
      (r) => r[3] === 'shieldOnly',
    ).map((r) => r[2])
    expect(shieldOnly).toEqual(['Animado', 'Esmagador'])
  })

  it('Guardião marcado countsAsTwo', () => {
    const countsAsTwo = ARMOR_ENCHANT_ROWS.filter(
      (r) => r[3] === 'countsAsTwo',
    ).map((r) => r[2])
    expect(countsAsTwo).toEqual(['Guardião'])
  })

  it('sentinel 91-100 = "Item específico"', () => {
    const last = ARMOR_ENCHANT_ROWS.at(-1)!
    expect(last[0]).toBe(91)
    expect(last[1]).toBe(100)
    expect(last[2]).toBe('Item específico')
  })

  it('frozen', () => {
    expect(Object.isFrozen(ARMOR_ENCHANT_ROWS)).toBe(true)
  })
})

describe('SPECIFIC_ARMOR_ROWS — Tabela 8-11', () => {
  it('cobre [1, 100] sem gaps', () => {
    let cursor = 1
    for (const row of SPECIFIC_ARMOR_ROWS) {
      expect(row[0]).toBe(cursor)
      cursor = row[1] + 1
    }
    expect(cursor).toBe(101)
  })

  it('13 rows', () => {
    expect(SPECIFIC_ARMOR_ROWS.length).toBe(13)
  })

  it('primeira row 1-10 = Cota élfica', () => {
    expect(SPECIFIC_ARMOR_ROWS[0]).toEqual([1, 10, 'Cota élfica'])
  })

  it('última row 94-100 = Armadura da luz', () => {
    expect(SPECIFIC_ARMOR_ROWS.at(-1)).toEqual([94, 100, 'Armadura da luz'])
  })
})

describe('rollMagicArmor', () => {
  const ENCHANT_NAMES = new Set(
    ARMOR_ENCHANT_ROWS.filter((r) => r[2] !== 'Item específico').map(
      (r) => r[2],
    ),
  )
  const SPECIFIC_NAMES = new Set(SPECIFIC_ARMOR_ROWS.map((r) => r[2]))

  it('nomes encanto vêm de 8-10 (excluindo sentinel)', () => {
    const rng = mulberry32(1)
    let hits = 0
    for (let i = 0; i < 500; i++) {
      const r = rollMagicArmor(rng, 'medio', true)
      if (r.kind === 'encanto') {
        hits++
        expect(ENCHANT_NAMES.has(r.name)).toBe(true)
      }
    }
    expect(hits).toBeGreaterThan(0)
  })

  it('nomes specific vêm de 8-11', () => {
    const rng = mulberry32(2)
    let hits = 0
    for (let i = 0; i < 500; i++) {
      const r = rollMagicArmor(rng, 'medio', false)
      if (r.kind === 'specific') {
        hits++
        expect(SPECIFIC_NAMES.has(r.name)).toBe(true)
      }
    }
    expect(hits).toBeGreaterThan(0)
  })

  it('armor (isShield=false) nunca retorna Animado/Esmagador', () => {
    const rng = mulberry32(3)
    for (let i = 0; i < 2000; i++) {
      const r = rollMagicArmor(rng, 'medio', false)
      if (r.kind === 'encanto') {
        expect(r.name).not.toBe('Animado')
        expect(r.name).not.toBe('Esmagador')
      }
    }
  })

  it('shield (isShield=true) às vezes retorna Animado ou Esmagador', () => {
    const rng = mulberry32(4)
    let shieldOnlySeen = 0
    for (let i = 0; i < 2000; i++) {
      const r = rollMagicArmor(rng, 'medio', true)
      if (
        r.kind === 'encanto' &&
        (r.name === 'Animado' || r.name === 'Esmagador')
      ) {
        shieldOnlySeen++
      }
    }
    expect(shieldOnlySeen).toBeGreaterThan(0)
  })

  it('menor: countsAsTwo (Guardião) nunca aparece', () => {
    const rng = mulberry32(5)
    for (let i = 0; i < 2000; i++) {
      const r = rollMagicArmor(rng, 'menor', false)
      if (r.kind === 'encanto') {
        expect(r.countsAsTwo).toBe(false)
        expect(r.name).not.toBe('Guardião')
      }
    }
  })

  it('medio: Guardião aparece com countsAsTwo=true', () => {
    const rng = mulberry32(6)
    let guardiaoHits = 0
    for (let i = 0; i < 2000; i++) {
      const r = rollMagicArmor(rng, 'medio', false)
      if (r.kind === 'encanto' && r.name === 'Guardião') {
        guardiaoHits++
        expect(r.countsAsTwo).toBe(true)
      }
    }
    expect(guardiaoHits).toBeGreaterThan(0)
  })

  it('tier + isShield propagados no result', () => {
    const rng = mulberry32(7)
    for (const tier of ['menor', 'medio', 'maior'] as const) {
      for (const isShield of [true, false]) {
        for (let i = 0; i < 20; i++) {
          const r = rollMagicArmor(rng, tier, isShield)
          expect(r.tier).toBe(tier)
          expect(r.isShield).toBe(isShield)
        }
      }
    }
  })

  it('determinismo', () => {
    const a = mulberry32(2026)
    const b = mulberry32(2026)
    for (let i = 0; i < 30; i++) {
      expect(rollMagicArmor(a, 'medio', true)).toEqual(
        rollMagicArmor(b, 'medio', true),
      )
    }
  })

  it('specific ~10% rolls (row 91-100 é 10% da distribuição)', () => {
    const rng = mulberry32(42)
    let specific = 0
    for (let i = 0; i < 3000; i++) {
      if (rollMagicArmor(rng, 'medio', false).kind === 'specific') specific++
    }
    expect(specific / 3000).toBeGreaterThan(0.05)
    expect(specific / 3000).toBeLessThan(0.2)
  })
})
