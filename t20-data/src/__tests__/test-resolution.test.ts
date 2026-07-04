import { describe, expect, it } from 'vitest'
import {
  AID_ANOTHER_BASE_BONUS,
  AID_ANOTHER_CD,
  AID_ANOTHER_INCREMENT_PER_10_ABOVE,
  DIFFICULTY_CD,
  EXTENDED_CUMULATIVE_CD_INCREMENT,
  EXTENDED_CUMULATIVE_FAILURE_PENALTY,
  EXTENDED_MAX_FAILURES,
  EXTENDED_SUCCESSES_REQUIRED,
  NATURAL_1_ALWAYS_FAILS,
  NATURAL_20_ALWAYS_SUCCEEDS,
  RESISTANCE_CD_BASE,
  TAKE_TEN_VALUE,
  TAKE_TWENTY_TIME_MULTIPLIER,
  TAKE_TWENTY_VALUE,
  TOOLS_MISSING_PENALTY,
  aidAnotherBonus,
  automaticOutcome,
  difficultyCd,
  extendedCumulativeCd,
  extendedCumulativePenalty,
  extendedSuccessesRequired,
  extendedTestState,
  opposedTestOutcome,
  resistanceCd,
  takeTenResult,
  takeTwentyResult,
  takeTwentyTime,
  takeZeroSucceeds,
  toolsMissingPenalty,
} from '../test-resolution'

/**
 * PDF livro Cap 5 p220-223, p227 — Test Resolution rules.
 */

describe('Constantes', () => {
  it('sucesso/falha automáticos', () => {
    expect(NATURAL_20_ALWAYS_SUCCEEDS).toBe(true)
    expect(NATURAL_1_ALWAYS_FAILS).toBe(true)
  })

  it('Take 10/20 e time multiplier', () => {
    expect(TAKE_TEN_VALUE).toBe(10)
    expect(TAKE_TWENTY_VALUE).toBe(20)
    expect(TAKE_TWENTY_TIME_MULTIPLIER).toBe(20)
  })

  it('Ajudar CD 10, +1 base, +1 por 10 acima', () => {
    expect(AID_ANOTHER_CD).toBe(10)
    expect(AID_ANOTHER_BASE_BONUS).toBe(1)
    expect(AID_ANOTHER_INCREMENT_PER_10_ABOVE).toBe(1)
  })

  it('Estendido: 3 falhas máx, +2 CD/teste, -2/falha', () => {
    expect(EXTENDED_MAX_FAILURES).toBe(3)
    expect(EXTENDED_CUMULATIVE_CD_INCREMENT).toBe(2)
    expect(EXTENDED_CUMULATIVE_FAILURE_PENALTY).toBe(-2)
  })

  it('Ferramentas ausentes -5', () => {
    expect(TOOLS_MISSING_PENALTY).toBe(-5)
  })

  it('Base 10 CD de resistência', () => {
    expect(RESISTANCE_CD_BASE).toBe(10)
  })
})

describe('DIFFICULTY_CD — Tabela 5-1 verbatim', () => {
  it('frozen', () => {
    expect(Object.isFrozen(DIFFICULTY_CD)).toBe(true)
  })

  it.each([
    ['facil', 5],
    ['media', 10],
    ['dificil', 15],
    ['desafiadora', 20],
    ['formidavel', 25],
    ['heroica', 30],
    ['quase-impossivel', 40],
  ] as const)('%s → CD %s', (tier, cd) => {
    expect(DIFFICULTY_CD[tier]).toBe(cd)
    expect(difficultyCd(tier)).toBe(cd)
  })

  it('NÃO existe tier CD 35', () => {
    expect(Object.values(DIFFICULTY_CD).includes(35)).toBe(false)
  })
})

describe('EXTENDED_SUCCESSES_REQUIRED — Tabela 5-2 verbatim', () => {
  it('frozen', () => {
    expect(Object.isFrozen(EXTENDED_SUCCESSES_REQUIRED)).toBe(true)
  })

  it.each([
    ['baixa', 3],
    ['media', 5],
    ['alta', 7],
  ] as const)('%s → %s sucessos', (c, s) => {
    expect(extendedSuccessesRequired(c)).toBe(s)
  })
})

describe('resistanceCd — fórmula p227', () => {
  it('exemplo livro: nível 10, CAR 4 → 19', () => {
    expect(resistanceCd(10, 4)).toBe(19)
  })

  it('nível 1, mod 0 → 10', () => {
    expect(resistanceCd(1, 0)).toBe(10)
  })

  it('nível 5, mod 3 → 15 (10 + 2 + 3)', () => {
    expect(resistanceCd(5, 3)).toBe(15)
  })

  it('nível 20, mod 5 → 25 (10 + 10 + 5)', () => {
    expect(resistanceCd(20, 5)).toBe(25)
  })

  it('mod negativo aceito', () => {
    expect(resistanceCd(10, -2)).toBe(13)
  })

  it('nível 0 lança', () => {
    expect(() => resistanceCd(0, 3)).toThrow(/level must be ≥ 1/)
  })
})

describe('automaticOutcome — 20 nat / 1 nat', () => {
  it('20 → auto-success', () => {
    expect(automaticOutcome(20)).toBe('auto-success')
  })

  it('1 → auto-failure', () => {
    expect(automaticOutcome(1)).toBe('auto-failure')
  })

  it('outros → undefined', () => {
    expect(automaticOutcome(10)).toBeUndefined()
    expect(automaticOutcome(15)).toBeUndefined()
  })
})

describe('takeTenResult / takeTwentyResult / takeTwentyTime', () => {
  it('take 10 com +3 → 13', () => {
    expect(takeTenResult(3)).toBe(13)
  })

  it('take 20 com +5 → 25', () => {
    expect(takeTwentyResult(5)).toBe(25)
  })

  it('take 20 dobra tempo × 20', () => {
    expect(takeTwentyTime(1)).toBe(20)
    expect(takeTwentyTime(3)).toBe(60)
  })

  it('take 20 tempo ≤ 0 lança', () => {
    expect(() => takeTwentyTime(0)).toThrow(/baseTimeUnits must be > 0/)
  })
})

describe('takeZeroSucceeds', () => {
  it('bônus ≥ CD → sim', () => {
    expect(takeZeroSucceeds(10, 10)).toBe(true)
    expect(takeZeroSucceeds(15, 10)).toBe(true)
  })

  it('bônus < CD → não', () => {
    expect(takeZeroSucceeds(9, 10)).toBe(false)
  })
})

describe('aidAnotherBonus — CD 10, +1 base, +1 por 10 acima', () => {
  it('roll < 10 → 0 (falha)', () => {
    expect(aidAnotherBonus(9)).toBe(0)
    expect(aidAnotherBonus(0)).toBe(0)
  })

  it('roll 10 → +1', () => {
    expect(aidAnotherBonus(10)).toBe(1)
  })

  it('roll 19 → +1 (só 9 acima)', () => {
    expect(aidAnotherBonus(19)).toBe(1)
  })

  it('roll 20 (natural 20 exemplo livro) → +2', () => {
    expect(aidAnotherBonus(20)).toBe(2)
  })

  it('roll 30 → +3', () => {
    expect(aidAnotherBonus(30)).toBe(3)
  })

  it('roll 40 → +4', () => {
    expect(aidAnotherBonus(40)).toBe(4)
  })
})

describe('extendedTestState — 3 falhas ou X sucessos', () => {
  it('baixa complexidade — 3 sucessos → success', () => {
    expect(extendedTestState(3, 0, 'baixa')).toBe('success')
    expect(extendedTestState(3, 2, 'baixa')).toBe('success')
  })

  it('baixa — 2 sucessos, 3 falhas → failure', () => {
    expect(extendedTestState(2, 3, 'baixa')).toBe('failure')
  })

  it('baixa — 2 sucessos, 2 falhas → in-progress', () => {
    expect(extendedTestState(2, 2, 'baixa')).toBe('in-progress')
  })

  it('média — 5 sucessos → success', () => {
    expect(extendedTestState(5, 2, 'media')).toBe('success')
  })

  it('alta — 7 sucessos → success', () => {
    expect(extendedTestState(7, 2, 'alta')).toBe('success')
  })

  it('valores negativos lançam', () => {
    expect(() => extendedTestState(-1, 0, 'baixa')).toThrow(/must be ≥ 0/)
    expect(() => extendedTestState(0, -1, 'baixa')).toThrow(/must be ≥ 0/)
  })
})

describe('extendedCumulativeCd', () => {
  it('sem testes anteriores → base', () => {
    expect(extendedCumulativeCd(15, 0)).toBe(15)
  })

  it('após 3 testes → +6', () => {
    expect(extendedCumulativeCd(15, 3)).toBe(21)
  })

  it('negativo lança', () => {
    expect(() => extendedCumulativeCd(15, -1)).toThrow(/testsAlreadyMade must be ≥ 0/)
  })
})

describe('extendedCumulativePenalty', () => {
  it('0 falhas → 0', () => {
    expect(extendedCumulativePenalty(0)).toBe(0)
  })

  it('2 falhas → -4', () => {
    expect(extendedCumulativePenalty(2)).toBe(-4)
  })

  it('negativo lança', () => {
    expect(() => extendedCumulativePenalty(-1)).toThrow(/failureCount must be ≥ 0/)
  })
})

describe('toolsMissingPenalty', () => {
  it('com ferramenta → 0', () => {
    expect(toolsMissingPenalty(true)).toBe(0)
  })

  it('sem ferramenta → -5', () => {
    expect(toolsMissingPenalty(false)).toBe(-5)
  })
})

describe('opposedTestOutcome — p220', () => {
  it('maior valor total → vence', () => {
    expect(opposedTestOutcome(20, 5, 15, 3)).toBe('attacker-wins')
    expect(opposedTestOutcome(15, 3, 20, 5)).toBe('defender-wins')
  })

  it('empate no total → maior modificador vence', () => {
    expect(opposedTestOutcome(20, 8, 20, 5)).toBe('attacker-wins')
    expect(opposedTestOutcome(20, 5, 20, 8)).toBe('defender-wins')
  })

  it('empate total e modificador → reroll', () => {
    expect(opposedTestOutcome(20, 5, 20, 5)).toBe('reroll')
  })
})
