import { describe, expect, it } from 'vitest'
import { WEAPONS } from '../catalog/weapons'

/**
 * PDF Cap 3 (Equipamento, p142-145, Tabela 3-3: Armas). Pinned spot checks
 * from the book table:
 *
 * Simples Leves:    Adaga 1d4/19/Perfuração, Espada curta 1d6/19/Perf,
 *                   Foice 1d6/x3/Corte
 * Simples 1-mão:    Clava 1d6/x2/Imp, Lança 1d6/x2/Curto/Perf,
 *                   Maça 1d8/x2/Imp
 * Simples 2-mão:    Bordão 1d6/1d6/x2/Imp/dupla, Pique 1d8/x2/Perf/alongada,
 *                   Tacape 1d10/x2/Imp
 * Marciais 1-mão:   Cimitarra 1d6/18/Corte/agil, Espada longa 1d8/19/Corte,
 *                   Florete 1d8/18/Perf/agil, Machado de batalha 1d8/x3/Corte,
 *                   Mangual 1d8/x2/Imp/versatil, Martelo guerra 1d8/x3/Imp,
 *                   Picareta 1d6/x4/Perf, Tridente 1d8/x2/Curto/Perf/versatil
 *
 * These are the cells of Tabela 3-3 that the engine surfaces directly to the
 * UI (damage dice, crit range, crit multiplier, damage type, traits).
 */

type BookRow = {
  damage: string
  critRange: number
  critMult: number
  type: string
  hand?: string
  purpose?: string
  range?: string
  traits?: string[]
}

const BOOK_TABELA_3_3: Record<string, BookRow> = {
  adaga: {
    damage: '1d4',
    critRange: 19,
    critMult: 2,
    type: 'perfuracao',
    hand: 'light',
    purpose: 'melee',
    range: 'curto',
  },
  'espada-curta': {
    damage: '1d6',
    critRange: 19,
    critMult: 2,
    type: 'perfuracao',
    hand: 'light',
  },
  foice: {
    damage: '1d6',
    critRange: 20,
    critMult: 3,
    type: 'corte',
    hand: 'light',
  },
  clava: {
    damage: '1d6',
    critRange: 20,
    critMult: 2,
    type: 'impacto',
    hand: 'one',
  },
  lanca: {
    damage: '1d6',
    critRange: 20,
    critMult: 2,
    type: 'perfuracao',
    purpose: 'thrown',
    range: 'curto',
  },
  maca: {
    damage: '1d8',
    critRange: 20,
    critMult: 2,
    type: 'impacto',
    hand: 'one',
  },
  bordao: {
    damage: '1d6/1d6',
    critRange: 20,
    critMult: 2,
    type: 'impacto',
    hand: 'two',
    traits: ['dupla'],
  },
  pique: {
    damage: '1d8',
    critRange: 20,
    critMult: 2,
    type: 'perfuracao',
    hand: 'two',
    traits: ['alongada'],
  },
  tacape: {
    damage: '1d10',
    critRange: 20,
    critMult: 2,
    type: 'impacto',
    hand: 'two',
  },
  cimitarra: {
    damage: '1d6',
    critRange: 18,
    critMult: 2,
    type: 'corte',
    traits: ['agil'],
  },
  'espada-longa': {
    damage: '1d8',
    critRange: 19,
    critMult: 2,
    type: 'corte',
  },
  florete: {
    damage: '1d8',
    critRange: 18,
    critMult: 2,
    type: 'perfuracao',
    traits: ['agil'],
  },
  'machado-batalha': {
    damage: '1d8',
    critRange: 20,
    critMult: 3,
    type: 'corte',
  },
  mangual: {
    damage: '1d8',
    critRange: 20,
    critMult: 2,
    type: 'impacto',
    traits: ['versatil'],
  },
  'martelo-guerra': {
    damage: '1d8',
    critRange: 20,
    critMult: 3,
    type: 'impacto',
  },
  picareta: {
    damage: '1d6',
    critRange: 20,
    critMult: 4,
    type: 'perfuracao',
  },
  tridente: {
    damage: '1d8',
    critRange: 20,
    critMult: 2,
    type: 'perfuracao',
    purpose: 'thrown',
    range: 'curto',
    traits: ['versatil'],
  },
}

describe('WEAPONS vs PDF Tabela 3-3', () => {
  for (const [id, row] of Object.entries(BOOK_TABELA_3_3)) {
    it(`${id}: damage/crit/type match the book`, () => {
      const weapon = WEAPONS.find((w) => w.id === id)
      expect(weapon, `${id} missing from catalog`).toBeDefined()
      const stats = weapon!.weapon!
      expect(stats.damage).toBe(row.damage)
      expect(stats.critRange).toBe(row.critRange)
      expect(stats.critMult).toBe(row.critMult)
      expect(stats.type).toBe(row.type)
      if (row.hand) expect(stats.hand).toBe(row.hand)
      if (row.purpose) expect(stats.purpose).toBe(row.purpose)
      if (row.range) expect(stats.range).toBe(row.range)
      if (row.traits) expect(stats.traits.sort()).toEqual(row.traits.sort())
    })
  }
})

describe('Weapon catalog invariants', () => {
  it('every weapon-* entry carries weapon stats', () => {
    for (const w of WEAPONS) {
      expect(
        w.category.startsWith('weapon-'),
        `${w.id} should be weapon-*`,
      ).toBe(true)
      expect(w.weapon, `${w.id} missing weapon stats`).toBeDefined()
    }
  })

  it('every weapon is wielded, not vested', () => {
    for (const w of WEAPONS) {
      expect(w.equip).toBe('wielded')
      expect(w.hands === 1 || w.hands === 2).toBe(true)
    }
  })

  it('two-handed weapons report hand=two and hands=2', () => {
    const twoHanded = WEAPONS.filter((w) => w.weapon?.hand === 'two')
    for (const w of twoHanded) expect(w.hands).toBe(2)
  })

  it('light-hand weapons report hands=1', () => {
    const light = WEAPONS.filter((w) => w.weapon?.hand === 'light')
    for (const w of light) expect(w.hands).toBe(1)
  })

  it('crit range is between 18 and 20', () => {
    for (const w of WEAPONS) {
      const r = w.weapon!.critRange
      expect(r).toBeGreaterThanOrEqual(18)
      expect(r).toBeLessThanOrEqual(20)
    }
  })

  it('crit multiplier is 2, 3 or 4', () => {
    for (const w of WEAPONS) {
      expect([2, 3, 4]).toContain(w.weapon!.critMult)
    }
  })

  it('ranged/thrown weapons carry a range tier', () => {
    for (const w of WEAPONS) {
      const p = w.weapon!.purpose
      if (p === 'ranged' || p === 'thrown') {
        expect(w.weapon!.range, `${w.id} ${p} missing range`).toBeDefined()
      }
    }
  })

  it('every weapon id is unique', () => {
    const ids = WEAPONS.map((w) => w.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('covers all 4 proficiency categories', () => {
    const cats = new Set(WEAPONS.map((w) => w.category))
    expect(cats).toEqual(
      new Set([
        'weapon-simple',
        'weapon-martial',
        'weapon-exotic',
        'weapon-firearm',
      ]),
    )
  })
})
