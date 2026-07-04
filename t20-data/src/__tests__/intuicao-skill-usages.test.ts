import { describe, expect, it } from 'vitest'
import {
  IMPLAUSIBLE_LIE_ENGANACAO_PENALTY,
  INTUICAO_ARMOR_PENALTY,
  INTUICAO_TRAINED_ONLY,
  INTUICAO_USAGES,
  PRESSENTIMENTO_CD,
  enganacaoAdjustmentForLie,
  intuicaoUsageByKind,
  pressentimentoCd,
  resolvePerceberMentira,
} from '../intuicao-skill-usages'

/**
 * PDF livro p120 — Perícia Intuição (SAB). 2 usos:
 *  1. Perceber Mentira — teste oposto vs Enganação-Mentir (aberto)
 *  2. Pressentimento — CD 20, Apenas Treinado
 * Cross-ref Enganação p119: implausible lie → -10 no Enganação do mentiroso.
 */

describe('INTUICAO_USAGES — shape', () => {
  it('exatamente 2 usos (p120)', () => {
    expect(INTUICAO_USAGES.length).toBe(2)
  })

  it('frozen', () => {
    expect(Object.isFrozen(INTUICAO_USAGES)).toBe(true)
  })

  it('ids únicos', () => {
    const ids = INTUICAO_USAGES.map((u) => u.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('todos bookPage 120', () => {
    for (const u of INTUICAO_USAGES) expect(u.bookPage).toBe(120)
  })

  it('só Pressentimento é treinado', () => {
    const treinados = INTUICAO_USAGES.filter((u) => u.trainedOnly).map(
      (u) => u.id,
    )
    expect(treinados).toEqual(['pressentimento'])
  })
})

describe('Perícia flags — Tabela 2-1', () => {
  it('perícia aberta (não é apenas treinada)', () => {
    expect(INTUICAO_TRAINED_ONLY).toBe(false)
  })

  it('sem penalidade de armadura', () => {
    expect(INTUICAO_ARMOR_PENALTY).toBe(false)
  })
})

describe('Perceber Mentira — p120', () => {
  const usage = intuicaoUsageByKind('perceber-mentira')

  it('teste oposto vs Enganação-Mentir', () => {
    if (usage.kind !== 'perceber-mentira') throw new Error('narrow failed')
    expect(usage.contest).toBe('opposed-vs-enganacao-mentir')
  })

  it('não treinado', () => {
    expect(usage.trainedOnly).toBe(false)
  })

  it('penalidade cross-ref -10', () => {
    if (usage.kind !== 'perceber-mentira') throw new Error('narrow failed')
    expect(usage.liarImplausiblePenalty).toBe(-10)
  })
})

describe('Pressentimento — p120', () => {
  const usage = intuicaoUsageByKind('pressentimento')

  it('CD 20', () => {
    expect(PRESSENTIMENTO_CD).toBe(20)
    expect(pressentimentoCd()).toBe(20)
  })

  it('apenas treinado', () => {
    expect(usage.trainedOnly).toBe(true)
  })

  it('não revela causa (só anomalia)', () => {
    if (usage.kind !== 'pressentimento') throw new Error('narrow failed')
    expect(usage.revealsCause).toBe(false)
  })

  it('CD no catálogo bate com constante', () => {
    if (usage.kind !== 'pressentimento') throw new Error('narrow failed')
    expect(usage.dc).toBe(PRESSENTIMENTO_CD)
  })
})

describe('IMPLAUSIBLE_LIE_ENGANACAO_PENALTY', () => {
  it('-10 verbatim Enganação p119', () => {
    expect(IMPLAUSIBLE_LIE_ENGANACAO_PENALTY).toBe(-10)
  })
})

describe('enganacaoAdjustmentForLie', () => {
  it('mentira plausível → 0', () => {
    expect(enganacaoAdjustmentForLie(false)).toBe(0)
  })

  it('mentira implausível → -10', () => {
    expect(enganacaoAdjustmentForLie(true)).toBe(-10)
  })
})

describe('resolvePerceberMentira — teste oposto', () => {
  it('Intuição > Enganação → detecta', () => {
    expect(resolvePerceberMentira(25, 20)).toBe('detected-lie')
  })

  it('Intuição < Enganação → acredita', () => {
    expect(resolvePerceberMentira(15, 22)).toBe('believes-lie')
  })

  it('empate → tied', () => {
    expect(resolvePerceberMentira(18, 18)).toBe('tied')
  })

  it('cenário implausível: Intuição 18 vs (Enganação 25 - 10) = detecta', () => {
    const enganacaoAdjusted = 25 + enganacaoAdjustmentForLie(true)
    expect(resolvePerceberMentira(18, enganacaoAdjusted)).toBe('detected-lie')
  })
})

describe('intuicaoUsageByKind', () => {
  it('throws se kind inválido', () => {
    expect(() =>
      // @ts-expect-error — invalid kind on purpose
      intuicaoUsageByKind('julgar-personalidade'),
    ).toThrow(/unknown kind/)
  })

  it('resolve perceber-mentira', () => {
    expect(intuicaoUsageByKind('perceber-mentira').name).toBe(
      'Perceber Mentira',
    )
  })

  it('resolve pressentimento', () => {
    expect(intuicaoUsageByKind('pressentimento').name).toBe('Pressentimento')
  })
})
