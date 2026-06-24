import { describe, expect, it } from 'vitest'
import {
  IRRELEVANT_ND_GAP,
  MAX_LEVEL,
  OUTCOME_MULTIPLIER,
  encounterXp,
  levelForXp,
  levelUpBenefits,
  xpToReachLevel,
} from '../xp'

/**
 * PDF Cap 8 (book p326-332). XP comes from challenges:
 *   ND × 1.000 × outcome multiplier, split by partySize
 *
 * Outcome multipliers (book p326):
 *   - Vitória: 1
 *   - Empate:  ½
 *   - Derrota: ¼
 *
 * "Desafios Irrelevantes" gate (p326): ND ≤ partyLevel - 5 → 0 XP.
 *
 * Sidebar p332: "um grupo de quatro personagens deve vencer quatro
 * ameaças de ND igual ao seu nível para subir para o próximo" — for a
 * 4-char party, ~level × 1.000 XP per character per level. Cumulative:
 * 1.000 + 2.000 + … (level - 1) × 1.000.
 *
 * Treinamento (p277) — level-up grants per the rulebook's closest
 * enumeration:
 *   - PV equivalentes ao próximo nível
 *   - PM equivalentes ao próximo nível
 *   - Uma habilidade de classe do próximo nível
 *   - +1 em todas as perícias treinadas (apenas se nível for par)
 */
describe('OUTCOME_MULTIPLIER', () => {
  it('matches PDF p326 (win=1, draw=½, loss=¼)', () => {
    expect(OUTCOME_MULTIPLIER).toEqual({ win: 1, draw: 0.5, loss: 0.25 })
  })
})

describe('encounterXp — PDF p326 formula', () => {
  it('ND × 1000 / partySize for a full win', () => {
    // ND 4 challenge, 4 players, full win → 4 × 1000 / 4 = 1000 each.
    expect(
      encounterXp({ nd: 4, partyLevel: 4, partySize: 4, outcome: 'win' }),
    ).toBe(1000)
  })

  it('draws award half the XP', () => {
    expect(
      encounterXp({ nd: 4, partyLevel: 4, partySize: 4, outcome: 'draw' }),
    ).toBe(500)
  })

  it('losses award one-quarter of the XP', () => {
    expect(
      encounterXp({ nd: 4, partyLevel: 4, partySize: 4, outcome: 'loss' }),
    ).toBe(250)
  })

  it('Desafios Irrelevantes: ND ≤ partyLevel - 5 awards 0 XP', () => {
    // L8 party fighting ND 3 (gap = 5) → 0.
    expect(
      encounterXp({ nd: 3, partyLevel: 8, partySize: 4, outcome: 'win' }),
    ).toBe(0)
    // ND 2 (gap = 6) → still 0.
    expect(
      encounterXp({ nd: 2, partyLevel: 8, partySize: 4, outcome: 'win' }),
    ).toBe(0)
  })

  it('the irrelevant gap is exactly 5 (not 4)', () => {
    // L8 party fighting ND 4 (gap = 4) → still awarded.
    expect(
      encounterXp({ nd: 4, partyLevel: 8, partySize: 4, outcome: 'win' }),
    ).toBe(1000)
    expect(IRRELEVANT_ND_GAP).toBe(5)
  })

  it('splits XP across the actual party size (3-char party gets more each)', () => {
    // ND 4 / 3 chars = 1333.33 → floored to 1333.
    expect(
      encounterXp({ nd: 4, partyLevel: 4, partySize: 3, outcome: 'win' }),
    ).toBe(1333)
  })

  it('returns 0 when partySize ≤ 0 (defensive)', () => {
    expect(
      encounterXp({ nd: 4, partyLevel: 4, partySize: 0, outcome: 'win' }),
    ).toBe(0)
    expect(
      encounterXp({ nd: 4, partyLevel: 4, partySize: -1, outcome: 'win' }),
    ).toBe(0)
  })

  it('returns 0 when ND ≤ 0 (defensive)', () => {
    expect(
      encounterXp({ nd: 0, partyLevel: 4, partySize: 4, outcome: 'win' }),
    ).toBe(0)
  })
})

describe('xpToReachLevel — cumulative party-of-four approximation', () => {
  it('L1 requires 0 XP (starting level)', () => {
    expect(xpToReachLevel(1)).toBe(0)
    expect(xpToReachLevel(0)).toBe(0)
    expect(xpToReachLevel(-5)).toBe(0)
  })

  it('L2 = 1000 XP (one full L1 challenge per char)', () => {
    expect(xpToReachLevel(2)).toBe(1000)
  })

  it('L3 = 1000 + 2000 = 3000', () => {
    expect(xpToReachLevel(3)).toBe(3000)
  })

  it('L10 = sum 1..9 × 1000 = 45000', () => {
    expect(xpToReachLevel(10)).toBe(45_000)
  })

  it(`L${MAX_LEVEL} = sum 1..19 × 1000 = 190000`, () => {
    // 19*20/2 = 190
    expect(xpToReachLevel(20)).toBe(190_000)
  })

  it('clamps to MAX_LEVEL for level > 20', () => {
    expect(xpToReachLevel(21)).toBe(xpToReachLevel(MAX_LEVEL))
    expect(xpToReachLevel(999)).toBe(xpToReachLevel(MAX_LEVEL))
  })
})

describe('levelForXp — inverse of xpToReachLevel', () => {
  it('returns 1 below the L2 threshold', () => {
    expect(levelForXp(0)).toBe(1)
    expect(levelForXp(999)).toBe(1)
  })

  it('returns 2 at exactly 1000', () => {
    expect(levelForXp(1000)).toBe(2)
  })

  it('returns 3 at exactly 3000', () => {
    expect(levelForXp(3000)).toBe(3)
  })

  it('does not pre-advance: 2999 XP is still L2', () => {
    expect(levelForXp(2999)).toBe(2)
  })

  it('clamps at MAX_LEVEL for very large totals', () => {
    expect(levelForXp(1_000_000)).toBe(MAX_LEVEL)
  })

  it('round-trips with xpToReachLevel for every level 1..20', () => {
    for (let level = 1; level <= MAX_LEVEL; level++) {
      expect(levelForXp(xpToReachLevel(level))).toBe(level)
    }
  })
})

describe('levelUpBenefits — Treinamento p277 enumeration', () => {
  it('always grants PV/PM next-level + 1 class power slot', () => {
    expect(levelUpBenefits(2)).toEqual(
      expect.arrayContaining([
        'pv-next-level',
        'pm-next-level',
        'class-power-slot',
      ]),
    )
  })

  it('grants the +1 in trained perícias ONLY on even levels', () => {
    expect(levelUpBenefits(2)).toContain('expertise-plus-one')
    expect(levelUpBenefits(3)).not.toContain('expertise-plus-one')
    expect(levelUpBenefits(4)).toContain('expertise-plus-one')
    expect(levelUpBenefits(5)).not.toContain('expertise-plus-one')
    expect(levelUpBenefits(20)).toContain('expertise-plus-one')
  })

  it('returns [] for an invalid next level (<2 or >20)', () => {
    expect(levelUpBenefits(1)).toEqual([])
    expect(levelUpBenefits(0)).toEqual([])
    expect(levelUpBenefits(21)).toEqual([])
  })
})
