import { describe, expect, it } from 'vitest'
import {
  APOSTAR_TIER_MULTIPLIER,
  APOSTA_BASE_ROLL_EXPRESSION,
  APOSTA_CASSINO_LUXO_ROLL_EXPRESSION,
  APOSTA_PORTO_TAVERNA_ROLL_EXPRESSION,
  JOGATINA_ARMOR_PENALTY,
  JOGATINA_TRAINED_ONLY,
  JOGATINA_USAGES,
  apostaBaseRollExpression,
  apostarNetResult,
  apostarPayoutGross,
  apostarPayoutMultiplier,
  apostarTierByRoll,
  jogatinaUsageByKind,
} from '../jogatina-skill-usages'

/**
 * PDF livro p120 — Perícia Jogatina (CAR, treinada).
 *  1. Apostar — pague T$ 1d10; tabela: ≤9/10-14/15-19/20-29/30-39/≥40 = 0/½/1/2/3/5×
 */

describe('JOGATINA_USAGES — shape', () => {
  it('exatamente 1 uso', () => {
    expect(JOGATINA_USAGES.length).toBe(1)
  })

  it('frozen', () => {
    expect(Object.isFrozen(JOGATINA_USAGES)).toBe(true)
  })

  it('id canônico', () => {
    expect(JOGATINA_USAGES.map((u) => u.id)).toEqual(['apostar'])
  })

  it('bookPage 120', () => {
    for (const u of JOGATINA_USAGES) {
      expect(u.bookPage).toBe(120)
    }
  })
})

describe('Perícia flags — Tabela 2-1', () => {
  it('apenas treinada', () => {
    expect(JOGATINA_TRAINED_ONLY).toBe(true)
  })

  it('sem penalidade de armadura', () => {
    expect(JOGATINA_ARMOR_PENALTY).toBe(false)
  })
})

describe('Apostar — p120', () => {
  const usage = jogatinaUsageByKind('apostar')

  it('aposta padrão 1d10', () => {
    expect(APOSTA_BASE_ROLL_EXPRESSION).toBe('1d10')
    if (usage.kind !== 'apostar') throw new Error('narrow failed')
    expect(usage.defaultBetRoll).toBe('1d10')
  })

  it('duração uma noite', () => {
    if (usage.kind !== 'apostar') throw new Error('narrow failed')
    expect(usage.duration).toBe('uma-noite')
  })

  it('variação da aposta pelo mestre', () => {
    expect(APOSTA_PORTO_TAVERNA_ROLL_EXPRESSION).toBe('1d3')
    expect(APOSTA_CASSINO_LUXO_ROLL_EXPRESSION).toBe('1d10*1000')
  })
})

describe('APOSTAR_TIER_MULTIPLIER', () => {
  it('frozen', () => {
    expect(Object.isFrozen(APOSTAR_TIER_MULTIPLIER)).toBe(true)
  })

  it('multiplicadores verbatim', () => {
    expect(APOSTAR_TIER_MULTIPLIER.nenhum).toBe(0)
    expect(APOSTAR_TIER_MULTIPLIER.metade).toBe(0.5)
    expect(APOSTAR_TIER_MULTIPLIER.empate).toBe(1)
    expect(APOSTAR_TIER_MULTIPLIER.dobro).toBe(2)
    expect(APOSTAR_TIER_MULTIPLIER.triplo).toBe(3)
    expect(APOSTAR_TIER_MULTIPLIER.quintuplo).toBe(5)
  })
})

describe('apostarTierByRoll — faixas verbatim', () => {
  it.each([
    [0, 'nenhum'],
    [9, 'nenhum'],
    [10, 'metade'],
    [14, 'metade'],
    [15, 'empate'],
    [19, 'empate'],
    [20, 'dobro'],
    [29, 'dobro'],
    [30, 'triplo'],
    [39, 'triplo'],
    [40, 'quintuplo'],
    [100, 'quintuplo'],
  ] as const)('roll %s → %s', (roll, tier) => {
    expect(apostarTierByRoll(roll)).toBe(tier)
  })
})

describe('apostarPayoutMultiplier', () => {
  it('roll 5 → 0', () => {
    expect(apostarPayoutMultiplier(5)).toBe(0)
  })

  it('roll 12 → 0.5', () => {
    expect(apostarPayoutMultiplier(12)).toBe(0.5)
  })

  it('roll 17 → 1', () => {
    expect(apostarPayoutMultiplier(17)).toBe(1)
  })

  it('roll 25 → 2', () => {
    expect(apostarPayoutMultiplier(25)).toBe(2)
  })

  it('roll 35 → 3', () => {
    expect(apostarPayoutMultiplier(35)).toBe(3)
  })

  it('roll 40 → 5', () => {
    expect(apostarPayoutMultiplier(40)).toBe(5)
  })
})

describe('apostarPayoutGross', () => {
  it('aposta T$ 8, roll 25 → T$ 16 (dobro)', () => {
    expect(apostarPayoutGross(25, 8)).toBe(16)
  })

  it('aposta T$ 10, roll 12 → T$ 5 (metade)', () => {
    expect(apostarPayoutGross(12, 10)).toBe(5)
  })

  it('aposta T$ 4, roll 5 → T$ 0 (nenhum)', () => {
    expect(apostarPayoutGross(5, 4)).toBe(0)
  })

  it('aposta negativa lança', () => {
    expect(() => apostarPayoutGross(15, -1)).toThrow(
      /apostaTibar must be ≥ 0/,
    )
  })
})

describe('apostarNetResult', () => {
  it('aposta T$ 10, roll 5 → -10 (perde tudo)', () => {
    expect(apostarNetResult(5, 10)).toBe(-10)
  })

  it('aposta T$ 10, roll 12 → -5 (recebe metade)', () => {
    expect(apostarNetResult(12, 10)).toBe(-5)
  })

  it('aposta T$ 10, roll 17 → 0 (empate)', () => {
    expect(apostarNetResult(17, 10)).toBe(0)
  })

  it('aposta T$ 10, roll 25 → +10 (dobro)', () => {
    expect(apostarNetResult(25, 10)).toBe(10)
  })

  it('aposta T$ 10, roll 42 → +40 (quíntuplo)', () => {
    expect(apostarNetResult(42, 10)).toBe(40)
  })
})

describe('apostaBaseRollExpression', () => {
  it.each([
    ['porto-taverna', '1d3'],
    ['padrao', '1d10'],
    ['cassino-luxo', '1d10*1000'],
  ] as const)('%s → %s', (setting, exp) => {
    expect(apostaBaseRollExpression(setting)).toBe(exp)
  })
})

describe('jogatinaUsageByKind', () => {
  it('throws se kind inválido', () => {
    expect(() =>
      // @ts-expect-error — invalid kind on purpose
      jogatinaUsageByKind('trapacear'),
    ).toThrow(/unknown kind/)
  })

  it('resolve o kind único', () => {
    expect(jogatinaUsageByKind('apostar').kind).toBe('apostar')
  })
})
