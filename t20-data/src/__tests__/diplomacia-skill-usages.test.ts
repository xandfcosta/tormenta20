import { describe, expect, it } from 'vitest'
import {
  ATITUDE_ORDER,
  BARGANHA_BASE_DISCOUNT_PCT,
  BARGANHA_BIG_DISCOUNT_MARGIN,
  BARGANHA_BIG_DISCOUNT_PCT,
  BARGANHA_OFFENSE_MARGIN,
  BARGANHA_OFFENSE_REFUSAL_WEEKS,
  DIPLOMACIA_ARMOR_PENALTY,
  DIPLOMACIA_TRAINED_ONLY,
  DIPLOMACIA_USAGES,
  MUDAR_ATITUDE_BIG_SHIFT_MARGIN,
  MUDAR_ATITUDE_MAX_PER_TARGET_PER_DAY,
  MUDAR_ATITUDE_REVERSE_FAIL_MARGIN,
  MUDAR_ATITUDE_RUSHED_PENALTY,
  PERSUASAO_CD,
  PERSUASAO_COSTLY_PENALTY,
  PERSUASAO_DANGEROUS_PENALTY,
  applyAtitudeShift,
  atitudeIndex,
  barganhaDiscountPct,
  barganhaOutcome,
  diplomaciaUsageByKind,
  mudarAtitudeActionPenalty,
  mudarAtitudeShift,
  persuasaoModifier,
} from '../diplomacia-skill-usages'

/**
 * PDF livro p118 — Perícia Diplomacia (CAR, aberta). 3 usos:
 *  1. Barganha — vs Vontade; -10%, margem 10+ = -20%; falha 5+ = 1 semana
 *  2. Mudar Atitude — vs Vontade; 1 categoria (2 se margem 10+);
 *     falha 5+ = 1 categoria oposta; 1 min ou completa @ -10; 1x/alvo/dia
 *  3. Persuasão — CD 20; -5 custoso; -10 perigoso; oposto opcional
 * Todos os 3 são efeitos mentais.
 */

describe('DIPLOMACIA_USAGES — shape', () => {
  it('exatamente 3 usos', () => {
    expect(DIPLOMACIA_USAGES.length).toBe(3)
  })

  it('frozen', () => {
    expect(Object.isFrozen(DIPLOMACIA_USAGES)).toBe(true)
  })

  it('ids canônicos', () => {
    expect(DIPLOMACIA_USAGES.map((u) => u.id).sort()).toEqual([
      'barganha',
      'mudar-atitude',
      'persuasao',
    ])
  })

  it('todos bookPage 118', () => {
    for (const u of DIPLOMACIA_USAGES) expect(u.bookPage).toBe(118)
  })

  it('todos são efeitos mentais', () => {
    for (const u of DIPLOMACIA_USAGES) expect(u.isMentalEffect).toBe(true)
  })
})

describe('Perícia flags — Tabela 2-1', () => {
  it('perícia aberta', () => {
    expect(DIPLOMACIA_TRAINED_ONLY).toBe(false)
  })

  it('sem penalidade de armadura', () => {
    expect(DIPLOMACIA_ARMOR_PENALTY).toBe(false)
  })
})

describe('Barganha — p118', () => {
  const usage = diplomaciaUsageByKind('barganha')

  it('oposto por Vontade', () => {
    if (usage.kind !== 'barganha') throw new Error('narrow failed')
    expect(usage.opposedBy).toBe('vontade')
  })

  it('constantes verbatim', () => {
    expect(BARGANHA_BASE_DISCOUNT_PCT).toBe(10)
    expect(BARGANHA_BIG_DISCOUNT_MARGIN).toBe(10)
    expect(BARGANHA_BIG_DISCOUNT_PCT).toBe(20)
    expect(BARGANHA_OFFENSE_MARGIN).toBe(5)
    expect(BARGANHA_OFFENSE_REFUSAL_WEEKS).toBe(1)
  })
})

describe('barganhaOutcome', () => {
  it('empate → discount (-10%)', () => {
    expect(barganhaOutcome(20, 20)).toBe('discount')
  })

  it('vitória por 9 → discount', () => {
    expect(barganhaOutcome(29, 20)).toBe('discount')
  })

  it('vitória por 10 → big-discount', () => {
    expect(barganhaOutcome(30, 20)).toBe('big-discount')
  })

  it('derrota por 1 → no-change', () => {
    expect(barganhaOutcome(19, 20)).toBe('no-change')
  })

  it('derrota por 4 → no-change', () => {
    expect(barganhaOutcome(16, 20)).toBe('no-change')
  })

  it('derrota por 5 → refused (ofensa)', () => {
    expect(barganhaOutcome(15, 20)).toBe('refused')
  })
})

describe('barganhaDiscountPct', () => {
  it('big-discount → 20', () => {
    expect(barganhaDiscountPct('big-discount')).toBe(20)
  })

  it('discount → 10', () => {
    expect(barganhaDiscountPct('discount')).toBe(10)
  })

  it('no-change → 0', () => {
    expect(barganhaDiscountPct('no-change')).toBe(0)
  })

  it('refused → 0', () => {
    expect(barganhaDiscountPct('refused')).toBe(0)
  })
})

describe('Mudar Atitude — p118', () => {
  const usage = diplomaciaUsageByKind('mudar-atitude')

  it('oposto por Vontade', () => {
    if (usage.kind !== 'mudar-atitude') throw new Error('narrow failed')
    expect(usage.opposedBy).toBe('vontade')
  })

  it('constantes verbatim', () => {
    expect(MUDAR_ATITUDE_BIG_SHIFT_MARGIN).toBe(10)
    expect(MUDAR_ATITUDE_REVERSE_FAIL_MARGIN).toBe(5)
    expect(MUDAR_ATITUDE_RUSHED_PENALTY).toBe(-10)
    expect(MUDAR_ATITUDE_MAX_PER_TARGET_PER_DAY).toBe(1)
  })

  it('referência tabela p259', () => {
    if (usage.kind !== 'mudar-atitude') throw new Error('narrow failed')
    expect(usage.attitudeTablePage).toBe(259)
  })
})

describe('mudarAtitudeActionPenalty', () => {
  it('um-minuto → 0', () => {
    expect(mudarAtitudeActionPenalty('um-minuto')).toBe(0)
  })

  it('completa → -10', () => {
    expect(mudarAtitudeActionPenalty('completa')).toBe(-10)
  })
})

describe('mudarAtitudeShift', () => {
  it('empate → +1', () => {
    expect(mudarAtitudeShift(15, 15)).toBe(1)
  })

  it('vitória por 9 → +1', () => {
    expect(mudarAtitudeShift(24, 15)).toBe(1)
  })

  it('vitória por 10 → +2', () => {
    expect(mudarAtitudeShift(25, 15)).toBe(2)
  })

  it('derrota por 4 → 0', () => {
    expect(mudarAtitudeShift(11, 15)).toBe(0)
  })

  it('derrota por 5 → -1', () => {
    expect(mudarAtitudeShift(10, 15)).toBe(-1)
  })
})

describe('Atitudes — eixo numérico', () => {
  it('ATITUDE_ORDER frozen', () => {
    expect(Object.isFrozen(ATITUDE_ORDER)).toBe(true)
  })

  it('5 categorias na ordem canônica', () => {
    expect([...ATITUDE_ORDER]).toEqual([
      'hostil',
      'inamistoso',
      'indiferente',
      'amistoso',
      'prestativo',
    ])
  })

  it.each([
    ['hostil', 0],
    ['inamistoso', 1],
    ['indiferente', 2],
    ['amistoso', 3],
    ['prestativo', 4],
  ] as const)('atitudeIndex(%s) = %s', (a, i) => {
    expect(atitudeIndex(a)).toBe(i)
  })

  it('atitudeIndex throws se inválida', () => {
    // @ts-expect-error — invalid atitude on purpose
    expect(() => atitudeIndex('neutro')).toThrow(/unknown atitude/)
  })
})

describe('applyAtitudeShift', () => {
  it('indiferente + 1 up → amistoso', () => {
    expect(applyAtitudeShift('indiferente', 1, 'up')).toBe('amistoso')
  })

  it('indiferente + 2 up → prestativo', () => {
    expect(applyAtitudeShift('indiferente', 2, 'up')).toBe('prestativo')
  })

  it('indiferente + 1 down → inamistoso', () => {
    expect(applyAtitudeShift('indiferente', 1, 'down')).toBe('inamistoso')
  })

  it('clamp low: hostil + 1 down = hostil', () => {
    expect(applyAtitudeShift('hostil', 1, 'down')).toBe('hostil')
  })

  it('clamp high: prestativo + 1 up = prestativo', () => {
    expect(applyAtitudeShift('prestativo', 1, 'up')).toBe('prestativo')
  })

  it('grande shift saturado: hostil + 10 up = prestativo', () => {
    expect(applyAtitudeShift('hostil', 10, 'up')).toBe('prestativo')
  })
})

describe('Persuasão — p118', () => {
  const usage = diplomaciaUsageByKind('persuasao')

  it('CD 20', () => {
    expect(PERSUASAO_CD).toBe(20)
    if (usage.kind !== 'persuasao') throw new Error('narrow failed')
    expect(usage.dc).toBe(20)
  })

  it('constantes verbatim', () => {
    expect(PERSUASAO_COSTLY_PENALTY).toBe(-5)
    expect(PERSUASAO_DANGEROUS_PENALTY).toBe(-10)
  })

  it('oposto opcional Vontade', () => {
    if (usage.kind !== 'persuasao') throw new Error('narrow failed')
    expect(usage.optionalOpposedBy).toBe('vontade')
  })
})

describe('persuasaoModifier', () => {
  it('trivial → 0, sem autoFail', () => {
    expect(persuasaoModifier('trivial')).toEqual({ penalty: 0, autoFail: false })
  })

  it('custoso → -5, sem autoFail', () => {
    expect(persuasaoModifier('costly')).toEqual({
      penalty: -5,
      autoFail: false,
    })
  })

  it('perigoso default → -10, sem autoFail', () => {
    expect(persuasaoModifier('dangerous')).toEqual({
      penalty: -10,
      autoFail: false,
    })
  })

  it('perigoso com autoFail flag → -10 + autoFail', () => {
    expect(persuasaoModifier('dangerous', { autoFailIfDangerous: true })).toEqual(
      { penalty: -10, autoFail: true },
    )
  })
})

describe('diplomaciaUsageByKind', () => {
  it('throws se kind inválido', () => {
    expect(() =>
      // @ts-expect-error — invalid kind on purpose
      diplomaciaUsageByKind('intimidar'),
    ).toThrow(/unknown kind/)
  })

  it('resolve todos', () => {
    for (const k of ['barganha', 'mudar-atitude', 'persuasao'] as const) {
      expect(diplomaciaUsageByKind(k).kind).toBe(k)
    }
  })
})
