import { describe, expect, it } from 'vitest'
import {
  CLASS_VITALS,
  classMpBase,
  classPvBase,
  multiclassMpPool,
  multiclassPvPool,
  pvPoolWithCon,
} from '../class-vitals'

/**
 * PDF Cap 1 — class entries (p36-83). Pinned values:
 *
 *   Class         PV inicial   PV/nível   PM/nível
 *   Arcanista     8            2          6
 *   Bárbaro       24           6          3
 *   Bardo         12           3          4
 *   Bucaneiro     16           4          3
 *   Caçador       16           4          4
 *   Cavaleiro     20           5          3
 *   Clérigo       16           4          5
 *   Druida        16           4          4
 *   Guerreiro     20           5          3
 *   Inventor      12           3          4
 *   Ladino        12           3          4
 *   Lutador       20           5          3
 *   Nobre         16           4          4
 *   Paladino      20           5          3 + Carisma (L1 one-time)
 *
 * The class entry text reads "começa com X PV + Constituição" — these
 * tests pin the PV inicial **before** Con. Same convention for PM.
 *
 * Multiclass rule (PDF p33 sidebar): use the **first** class's PV
 * inicial as the seed; other classes only contribute their per-level
 * grants. The PM rule has no seed distinction — every level grants its
 * full per-level PM.
 */
const PDF_VITALS: Record<
  string,
  { pvInicial: number; pvPerLevel: number; mpPerLevel: number }
> = {
  Arcanista: { pvInicial: 8, pvPerLevel: 2, mpPerLevel: 6 },
  'Bárbaro': { pvInicial: 24, pvPerLevel: 6, mpPerLevel: 3 },
  Bardo: { pvInicial: 12, pvPerLevel: 3, mpPerLevel: 4 },
  Bucaneiro: { pvInicial: 16, pvPerLevel: 4, mpPerLevel: 3 },
  'Caçador': { pvInicial: 16, pvPerLevel: 4, mpPerLevel: 4 },
  Cavaleiro: { pvInicial: 20, pvPerLevel: 5, mpPerLevel: 3 },
  'Clérigo': { pvInicial: 16, pvPerLevel: 4, mpPerLevel: 5 },
  Druida: { pvInicial: 16, pvPerLevel: 4, mpPerLevel: 4 },
  Guerreiro: { pvInicial: 20, pvPerLevel: 5, mpPerLevel: 3 },
  Inventor: { pvInicial: 12, pvPerLevel: 3, mpPerLevel: 4 },
  Ladino: { pvInicial: 12, pvPerLevel: 3, mpPerLevel: 4 },
  Lutador: { pvInicial: 20, pvPerLevel: 5, mpPerLevel: 3 },
  Nobre: { pvInicial: 16, pvPerLevel: 4, mpPerLevel: 4 },
  Paladino: { pvInicial: 20, pvPerLevel: 5, mpPerLevel: 3 },
}

describe('CLASS_VITALS — catalog completeness vs PDF', () => {
  it('covers all 14 base classes', () => {
    expect(Object.keys(CLASS_VITALS).sort()).toEqual(
      Object.keys(PDF_VITALS).sort(),
    )
  })

  for (const [className, expected] of Object.entries(PDF_VITALS)) {
    it(`${className}: PV ${expected.pvInicial}/${expected.pvPerLevel}, PM ${expected.mpPerLevel}/nível`, () => {
      const entry = CLASS_VITALS[className]
      expect(entry.pvInicial).toBe(expected.pvInicial)
      expect(entry.pvPerLevel).toBe(expected.pvPerLevel)
      expect(entry.mpPerLevel).toBe(expected.mpPerLevel)
    })
  }

  it('every numeric field is a positive integer', () => {
    for (const [className, entry] of Object.entries(CLASS_VITALS)) {
      for (const v of [entry.pvInicial, entry.pvPerLevel, entry.mpPerLevel]) {
        expect(Number.isInteger(v), `${className}: ${v}`).toBe(true)
        expect(v).toBeGreaterThan(0)
      }
    }
  })
})

describe('classPvBase — single-class progression', () => {
  it('Guerreiro L1: 20 (seed only)', () => {
    expect(classPvBase([{ className: 'Guerreiro', level: 1 }])).toBe(20)
  })

  it('Guerreiro L5: 20 + 5*4 = 40', () => {
    expect(classPvBase([{ className: 'Guerreiro', level: 5 }])).toBe(40)
  })

  it('Guerreiro L20: 20 + 5*19 = 115', () => {
    expect(classPvBase([{ className: 'Guerreiro', level: 20 }])).toBe(115)
  })

  it('Bárbaro L20: 24 + 6*19 = 138 (tankiest class)', () => {
    expect(classPvBase([{ className: 'Bárbaro', level: 20 }])).toBe(138)
  })

  it('Arcanista L20: 8 + 2*19 = 46 (frailest)', () => {
    expect(classPvBase([{ className: 'Arcanista', level: 20 }])).toBe(46)
  })
})

describe('classPvBase — multiclass: seed = first class', () => {
  it('Guerreiro 5 / Arcanista 5 = 20 + 5*4 + 2*5 = 50', () => {
    // PV inicial 20 (Guerreiro seed) + per-level: 5*4 (Guerreiro) + 2*5 (Arcanista)
    expect(
      classPvBase([
        { className: 'Guerreiro', level: 5 },
        { className: 'Arcanista', level: 5 },
      ]),
    ).toBe(50)
  })

  it('Arcanista 5 / Guerreiro 5 = 8 + 2*4 + 5*5 = 41 (seed differs)', () => {
    // Note: swapping the order of classes changes the seed, so total PV differs.
    expect(
      classPvBase([
        { className: 'Arcanista', level: 5 },
        { className: 'Guerreiro', level: 5 },
      ]),
    ).toBe(41)
  })

  it('returns 0 when classes list is empty', () => {
    expect(classPvBase([])).toBe(0)
  })

  it('returns 0 when seed class is unknown', () => {
    expect(classPvBase([{ className: 'Hexer', level: 5 }])).toBe(0)
  })

  it('ignores unknown classes after the seed (no crash)', () => {
    expect(
      classPvBase([
        { className: 'Guerreiro', level: 3 },
        { className: 'Hexer', level: 4 },
      ]),
    ).toBe(20 + 5 * 2)
  })
})

describe('pvPoolWithCon — "sempre ganha pelo menos 1 PV ao subir de nível" (p34)', () => {
  it('CON positiva: idêntico à fórmula linear (Guerreiro L5 CON 2 = 50)', () => {
    // 20+2 no L1 + 4 níveis × (5+2) = 50 — mesmo resultado da soma antiga.
    expect(pvPoolWithCon(CLASS_VITALS.Guerreiro, 5, 2)).toBe(50)
  })

  it('CON negativa anula o PV/nível → aplica o piso de 1 (Arcanista CON −2 L5 = 10)', () => {
    // L1: 8−2 = 6; níveis 2-5: max(1, 2−2) = 1 cada → 6 + 4 = 10.
    // A fórmula linear antiga dava 8 + 4×2 − 2×5 = 6 (errado).
    expect(pvPoolWithCon(CLASS_VITALS.Arcanista, 5, -2)).toBe(10)
  })

  it('piso vale por nível mesmo com CON muito negativa (Guerreiro CON −5 L3 = 17)', () => {
    // L1: 20−5 = 15; níveis 2-3: max(1, 5−5) = 1 cada → 17.
    expect(pvPoolWithCon(CLASS_VITALS.Guerreiro, 3, -5)).toBe(17)
  })

  it('L1 não tem piso (não é "subir de nível"): Arcanista CON −2 L1 = 6', () => {
    expect(pvPoolWithCon(CLASS_VITALS.Arcanista, 1, -2)).toBe(6)
  })
})

describe('multiclassPvPool / multiclassMpPool — p34-35 (com CON + piso)', () => {
  it('Zaled (p35): 1º nível de Paladino novo dá 5 PV, não 20', () => {
    // Guerreiro 1 → Paladino 1: seed Guerreiro 20 + con 0, Paladino soma 5.
    expect(
      multiclassPvPool(
        [
          { className: 'Guerreiro', level: 1 },
          { className: 'Paladino', level: 1 },
        ],
        0,
      ),
    ).toBe(25)
  })

  it('CON conta em todos os níveis, com o piso do p34 por nível', () => {
    // Arcanista 2 / Guerreiro 2, CON −2: seed 8−2 + 1×max(1,0) + 2×max(1,3) = 13.
    expect(
      multiclassPvPool(
        [
          { className: 'Arcanista', level: 2 },
          { className: 'Guerreiro', level: 2 },
        ],
        -2,
      ),
    ).toBe(13)
  })

  it('classe única degenera para pvPoolWithCon', () => {
    expect(multiclassPvPool([{ className: 'Guerreiro', level: 5 }], 2)).toBe(
      pvPoolWithCon(CLASS_VITALS.Guerreiro, 5, 2),
    )
  })

  it('PM: "some os PM fornecidos por cada classe" (p35), sem rider de Carisma', () => {
    // Guerreiro 4 (3×4) + Arcanista 4 (6×4) = 36. O +Car do Paladino vem do
    // modifier de Abençoado, nunca daqui.
    expect(
      multiclassMpPool([
        { className: 'Guerreiro', level: 4 },
        { className: 'Arcanista', level: 4 },
      ]),
    ).toBe(36)
    expect(multiclassMpPool([{ className: 'Paladino', level: 2 }])).toBe(6)
  })

  it('lista vazia / seed desconhecida → 0', () => {
    expect(multiclassPvPool([], 3)).toBe(0)
    expect(multiclassPvPool([{ className: 'Hexer', level: 5 }], 3)).toBe(0)
  })
})

describe('classMpBase — uniform per-level + Paladino L1 Carisma bonus', () => {
  it('Arcanista L5: 6*5 = 30', () => {
    expect(classMpBase([{ className: 'Arcanista', level: 5 }], 0)).toBe(30)
  })

  it('Guerreiro L1: 3 (no Cha bonus — not a Paladino)', () => {
    expect(classMpBase([{ className: 'Guerreiro', level: 1 }], 4)).toBe(3)
  })

  it('Paladino L1 with Carisma 3 → 3 + 3 = 6 (Abençoado one-time bonus)', () => {
    expect(classMpBase([{ className: 'Paladino', level: 1 }], 3)).toBe(6)
  })

  it('Paladino L20 with Carisma 5 → 3*20 + 5 = 65', () => {
    expect(classMpBase([{ className: 'Paladino', level: 20 }], 5)).toBe(65)
  })

  it('multiclass: sums each class\'s mpPerLevel * level', () => {
    // Arcanista 5 = 30; Guerreiro 5 = 15; total 45.
    expect(
      classMpBase(
        [
          { className: 'Arcanista', level: 5 },
          { className: 'Guerreiro', level: 5 },
        ],
        0,
      ),
    ).toBe(45)
  })

  it('Paladino multiclass applies Carisma bonus once, not per Paladino level', () => {
    // 3*5 (Paladino) + 6*5 (Arcanista) + 4 (Cha) = 49
    expect(
      classMpBase(
        [
          { className: 'Paladino', level: 5 },
          { className: 'Arcanista', level: 5 },
        ],
        4,
      ),
    ).toBe(49)
  })

  it('negative Carisma reduces a Paladino\'s PM (penalty applies)', () => {
    // PDF: PM bonus is + Carisma, so a Cha of -1 reduces by 1.
    expect(classMpBase([{ className: 'Paladino', level: 1 }], -1)).toBe(2)
  })

  it('returns 0 for an empty classes list', () => {
    expect(classMpBase([], 5)).toBe(0)
  })

  it('ignores unknown class names (no crash)', () => {
    expect(classMpBase([{ className: 'Hexer', level: 5 }], 0)).toBe(0)
  })
})
