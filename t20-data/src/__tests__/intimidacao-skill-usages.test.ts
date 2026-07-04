import { describe, expect, it } from 'vitest'
import {
  ASSUSTAR_APAVORADO_ROUNDS,
  ASSUSTAR_BIG_SUCCESS_MARGIN,
  COAGIR_DANGEROUS_ORDER_VONTADE_BONUS,
  COAGIR_TARGET_MIN_INT,
  INTIMIDACAO_ARMOR_PENALTY,
  INTIMIDACAO_TRAINED_ONLY,
  INTIMIDACAO_USAGES,
  assustarOutcome,
  coagirOutcome,
  coagirTargetIsEligible,
  coagirTargetVontadeModifier,
  intimidacaoUsageByKind,
} from '../intimidacao-skill-usages'

/**
 * PDF livro p120 — Perícia Intimidação (CAR, aberta). 2 usos:
 *  1. Assustar — ação padrão vs Vontade, alcance curto; sucesso abalado;
 *     margem 10+ = apavorado 1 rodada + abalado cena
 *  2. Coagir — 1 min+ vs Vontade adjacente; Int ≥ -3;
 *     ordem perigosa +5 no Vontade (ou passa auto); alvo hostil
 * Ambos são "efeitos de medo" (não mental).
 */

describe('INTIMIDACAO_USAGES — shape', () => {
  it('exatamente 2 usos', () => {
    expect(INTIMIDACAO_USAGES.length).toBe(2)
  })

  it('frozen', () => {
    expect(Object.isFrozen(INTIMIDACAO_USAGES)).toBe(true)
  })

  it('ids canônicos', () => {
    expect(INTIMIDACAO_USAGES.map((u) => u.id).sort()).toEqual([
      'assustar',
      'coagir',
    ])
  })

  it('todos bookPage 120', () => {
    for (const u of INTIMIDACAO_USAGES) expect(u.bookPage).toBe(120)
  })

  it('todos efeitos de medo', () => {
    for (const u of INTIMIDACAO_USAGES) expect(u.isFearEffect).toBe(true)
  })

  it('todos opostos por Vontade', () => {
    for (const u of INTIMIDACAO_USAGES) expect(u.opposedBy).toBe('vontade')
  })
})

describe('Perícia flags — Tabela 2-1', () => {
  it('perícia aberta', () => {
    expect(INTIMIDACAO_TRAINED_ONLY).toBe(false)
  })

  it('sem penalidade de armadura', () => {
    expect(INTIMIDACAO_ARMOR_PENALTY).toBe(false)
  })
})

describe('Assustar — p120', () => {
  const usage = intimidacaoUsageByKind('assustar')

  it('ação padrão', () => {
    if (usage.kind !== 'assustar') throw new Error('narrow failed')
    expect(usage.action).toBe('padrao')
  })

  it('alcance curto', () => {
    if (usage.kind !== 'assustar') throw new Error('narrow failed')
    expect(usage.range).toBe('curto')
  })

  it('sucesso base = abalado cena', () => {
    if (usage.kind !== 'assustar') throw new Error('narrow failed')
    expect(usage.successCondition).toBe('abalado')
    expect(usage.successConditionDuration).toBe('resto-da-cena')
  })

  it('margem 10+ = apavorado 1 rodada + abalado', () => {
    if (usage.kind !== 'assustar') throw new Error('narrow failed')
    expect(usage.bigSuccessMargin).toBe(10)
    expect(usage.bigSuccessBriefCondition).toBe('apavorado')
    expect(usage.bigSuccessBriefDurationRounds).toBe(1)
    expect(usage.bigSuccessThenCondition).toBe('abalado')
  })

  it('constantes verbatim', () => {
    expect(ASSUSTAR_BIG_SUCCESS_MARGIN).toBe(10)
    expect(ASSUSTAR_APAVORADO_ROUNDS).toBe(1)
  })
})

describe('assustarOutcome', () => {
  it('empate → abalado', () => {
    expect(assustarOutcome(15, 15)).toBe('abalado')
  })

  it('vitória por 9 → abalado', () => {
    expect(assustarOutcome(24, 15)).toBe('abalado')
  })

  it('vitória por 10 → apavorado-then-abalado', () => {
    expect(assustarOutcome(25, 15)).toBe('apavorado-then-abalado')
  })

  it('vitória por 20 → apavorado-then-abalado', () => {
    expect(assustarOutcome(35, 15)).toBe('apavorado-then-abalado')
  })

  it('derrota por 1 → no-effect', () => {
    expect(assustarOutcome(14, 15)).toBe('no-effect')
  })

  it('derrota por 10 → no-effect', () => {
    expect(assustarOutcome(5, 15)).toBe('no-effect')
  })
})

describe('Coagir — p120', () => {
  const usage = intimidacaoUsageByKind('coagir')

  it('ação 1 min ou mais', () => {
    if (usage.kind !== 'coagir') throw new Error('narrow failed')
    expect(usage.action).toBe('um-minuto-ou-mais')
  })

  it('alvo adjacente', () => {
    if (usage.kind !== 'coagir') throw new Error('narrow failed')
    expect(usage.positioning).toBe('adjacente')
  })

  it('alvo Int ≥ -3', () => {
    if (usage.kind !== 'coagir') throw new Error('narrow failed')
    expect(usage.targetMinIntelligence).toBe(-3)
    expect(COAGIR_TARGET_MIN_INT).toBe(-3)
  })

  it('bônus verbatim +5', () => {
    expect(COAGIR_DANGEROUS_ORDER_VONTADE_BONUS).toBe(5)
  })

  it('alvo pode passar auto se ordem perigosa', () => {
    if (usage.kind !== 'coagir') throw new Error('narrow failed')
    expect(usage.dangerousOrderCanAutoPass).toBe(true)
  })

  it('sucesso deixa alvo hostil', () => {
    if (usage.kind !== 'coagir') throw new Error('narrow failed')
    expect(usage.leavesTargetHostile).toBe(true)
  })
})

describe('coagirTargetIsEligible', () => {
  it('Int -3 → elegível', () => {
    expect(coagirTargetIsEligible(-3)).toBe(true)
  })

  it('Int -2 → elegível', () => {
    expect(coagirTargetIsEligible(-2)).toBe(true)
  })

  it('Int 0 → elegível', () => {
    expect(coagirTargetIsEligible(0)).toBe(true)
  })

  it('Int -4 → não elegível (animal)', () => {
    expect(coagirTargetIsEligible(-4)).toBe(false)
  })

  it('Int -5 → não elegível', () => {
    expect(coagirTargetIsEligible(-5)).toBe(false)
  })
})

describe('coagirTargetVontadeModifier', () => {
  it('ordem normal → bonus 0, sem autoPass', () => {
    expect(coagirTargetVontadeModifier(false)).toEqual({
      bonus: 0,
      canAutoPass: false,
    })
  })

  it('ordem perigosa → bonus +5, autoPass', () => {
    expect(coagirTargetVontadeModifier(true)).toEqual({
      bonus: 5,
      canAutoPass: true,
    })
  })
})

describe('coagirOutcome', () => {
  it('empate → obedece', () => {
    expect(coagirOutcome(20, 20)).toBe('obeys-and-hostile')
  })

  it('vitória → obedece', () => {
    expect(coagirOutcome(25, 20)).toBe('obeys-and-hostile')
  })

  it('derrota → refused', () => {
    expect(coagirOutcome(15, 20)).toBe('refused')
  })

  it('ordem perigosa vira empate em derrota (20 vs 15+5=20 → obedece)', () => {
    expect(
      coagirOutcome(20, 15, { dangerousOrContraryOrder: true }),
    ).toBe('obeys-and-hostile')
  })

  it('ordem perigosa: 22 vs 20+5=25 → refused', () => {
    expect(
      coagirOutcome(22, 20, { dangerousOrContraryOrder: true }),
    ).toBe('refused')
  })
})

describe('intimidacaoUsageByKind', () => {
  it('throws se kind inválido', () => {
    expect(() =>
      // @ts-expect-error — invalid kind on purpose
      intimidacaoUsageByKind('interrogatorio'),
    ).toThrow(/unknown kind/)
  })

  it('resolve assustar', () => {
    expect(intimidacaoUsageByKind('assustar').name).toBe('Assustar')
  })

  it('resolve coagir', () => {
    expect(intimidacaoUsageByKind('coagir').name).toBe('Coagir')
  })
})
