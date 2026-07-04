import { describe, expect, it } from 'vitest'
import {
  DISFARCE_COMPLEX_PENALTY,
  DISFARCE_DURATION_MINUTES,
  DISFARCE_KNOWS_SPECIFIC_PERSON_BONUS_PERCEPCAO,
  DISFARCE_WITHOUT_KIT_PENALTY,
  ENGANACAO_ARMOR_PENALTY,
  ENGANACAO_TRAINED_ONLY,
  ENGANACAO_USAGES,
  FALSIFICACAO_COMPLEX_PENALTY,
  INSINUACAO_CD,
  INSINUACAO_GARBLE_MARGIN,
  INTRIGA_CD,
  INTRIGA_EXPOSE_MARGIN,
  INTRIGA_IMPROVAVEL_CD,
  MENTIR_IMPLAUSIBLE_PENALTY,
  disfarcePenalty,
  disfarcePercepcaoBonus,
  enganacaoUsageByKind,
  falsificacaoPenalty,
  insinuacaoOutcome,
  intrigaCd,
  intrigaInvestigationCd,
  intrigaOutcome,
  mentirPenalty,
} from '../enganacao-skill-usages'

/**
 * PDF livro p118-119 — Perícia Enganação (CAR, aberta). 6 usos:
 *  1. Disfarce (p118) — vs Percepção; -5 complexo; -5 sem estojo; +10 na
 *     Percepção de quem conhece a pessoa; 10 min.
 *  2. Falsificação (p119) — vs Percepção; -10 documento complexo.
 *  3. Fintar (p119) — ação padrão vs Reflexos alcance curto; alvo desprevenido.
 *  4. Insinuação (p119) — CD 20; falha 5+ deturpa; obs Intuição oposta.
 *  5. Intriga (p119) — CD 20/30; falha 5+ expõe; Invest CD = teste.
 *  6. Mentir (p119) — vs Intuição; -10 implausível.
 */

describe('ENGANACAO_USAGES — shape', () => {
  it('exatamente 6 usos', () => {
    expect(ENGANACAO_USAGES.length).toBe(6)
  })

  it('frozen', () => {
    expect(Object.isFrozen(ENGANACAO_USAGES)).toBe(true)
  })

  it('ids únicos', () => {
    const ids = ENGANACAO_USAGES.map((u) => u.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('ids canônicos', () => {
    expect(ENGANACAO_USAGES.map((u) => u.id).sort()).toEqual([
      'disfarce',
      'falsificacao',
      'fintar',
      'insinuacao',
      'intriga',
      'mentir',
    ])
  })

  it('Disfarce em p118, resto em p119', () => {
    for (const u of ENGANACAO_USAGES) {
      const expected = u.id === 'disfarce' ? 118 : 119
      expect(u.bookPage).toBe(expected)
    }
  })
})

describe('Perícia flags — Tabela 2-1', () => {
  it('perícia aberta', () => {
    expect(ENGANACAO_TRAINED_ONLY).toBe(false)
  })

  it('sem penalidade de armadura', () => {
    expect(ENGANACAO_ARMOR_PENALTY).toBe(false)
  })
})

describe('Disfarce — p118', () => {
  const usage = enganacaoUsageByKind('disfarce')

  it('oposto por Percepção', () => {
    expect(usage.opposedBy).toBe('percepcao')
  })

  it('duração 10 min', () => {
    expect(usage.action).toBe('10-minutos')
    expect(DISFARCE_DURATION_MINUTES).toBe(10)
  })

  it('exige estojo', () => {
    if (usage.kind !== 'disfarce') throw new Error('narrow failed')
    expect(usage.requiresDisguiseKit).toBe(true)
  })

  it('penalidades verbatim', () => {
    expect(DISFARCE_COMPLEX_PENALTY).toBe(-5)
    expect(DISFARCE_WITHOUT_KIT_PENALTY).toBe(-5)
    expect(DISFARCE_KNOWS_SPECIFIC_PERSON_BONUS_PERCEPCAO).toBe(10)
  })
})

describe('disfarcePenalty', () => {
  it('sem penalidades → 0', () => {
    expect(disfarcePenalty({})).toBe(0)
  })

  it('só complexo → -5', () => {
    expect(disfarcePenalty({ complex: true })).toBe(-5)
  })

  it('só sem estojo → -5', () => {
    expect(disfarcePenalty({ withoutKit: true })).toBe(-5)
  })

  it('complexo + sem estojo → -10', () => {
    expect(disfarcePenalty({ complex: true, withoutKit: true })).toBe(-10)
  })
})

describe('disfarcePercepcaoBonus', () => {
  it('conhece pessoa → +10', () => {
    expect(disfarcePercepcaoBonus(true)).toBe(10)
  })

  it('não conhece → 0', () => {
    expect(disfarcePercepcaoBonus(false)).toBe(0)
  })
})

describe('Falsificação — p119', () => {
  const usage = enganacaoUsageByKind('falsificacao')

  it('oposto por Percepção', () => {
    expect(usage.opposedBy).toBe('percepcao')
  })

  it('combina com Ofício', () => {
    if (usage.kind !== 'falsificacao') throw new Error('narrow failed')
    expect(usage.combinesWithOficio).toBe(true)
  })

  it('penalidade -10 verbatim', () => {
    expect(FALSIFICACAO_COMPLEX_PENALTY).toBe(-10)
  })
})

describe('falsificacaoPenalty', () => {
  it('simples → 0', () => {
    expect(falsificacaoPenalty(false)).toBe(0)
  })

  it('complexo/assinatura → -10', () => {
    expect(falsificacaoPenalty(true)).toBe(-10)
  })
})

describe('Fintar — p119', () => {
  const usage = enganacaoUsageByKind('fintar')

  it('ação padrão', () => {
    expect(usage.action).toBe('padrao')
  })

  it('oposto por Reflexos', () => {
    expect(usage.opposedBy).toBe('reflexos')
  })

  it('alcance curto', () => {
    if (usage.kind !== 'fintar') throw new Error('narrow failed')
    expect(usage.range).toBe('curto')
  })

  it('deixa alvo desprevenido', () => {
    if (usage.kind !== 'fintar') throw new Error('narrow failed')
    expect(usage.makesTargetDesprevenido).toBe(true)
  })
})

describe('Insinuação — p119', () => {
  const usage = enganacaoUsageByKind('insinuacao')

  it('CD 20', () => {
    expect(INSINUACAO_CD).toBe(20)
    if (usage.kind !== 'insinuacao') throw new Error('narrow failed')
    expect(usage.dc).toBe(20)
  })

  it('garble margin 5', () => {
    expect(INSINUACAO_GARBLE_MARGIN).toBe(5)
  })

  it('observadores fazem Intuição oposta', () => {
    if (usage.kind !== 'insinuacao') throw new Error('narrow failed')
    expect(usage.observersRollIntuicaoOpposed).toBe(true)
  })
})

describe('insinuacaoOutcome', () => {
  it('sucesso (20) → received', () => {
    expect(insinuacaoOutcome(20)).toBe('received')
  })

  it('sucesso alto (25) → received', () => {
    expect(insinuacaoOutcome(25)).toBe('received')
  })

  it('falha por 1 (19) → failed', () => {
    expect(insinuacaoOutcome(19)).toBe('failed')
  })

  it('falha por 4 (16) → failed', () => {
    expect(insinuacaoOutcome(16)).toBe('failed')
  })

  it('falha por 5 (15) → garbled', () => {
    expect(insinuacaoOutcome(15)).toBe('garbled')
  })

  it('falha por 20 (0) → garbled', () => {
    expect(insinuacaoOutcome(0)).toBe('garbled')
  })
})

describe('Intriga — p119', () => {
  const usage = enganacaoUsageByKind('intriga')

  it('CD 20 normal', () => {
    expect(INTRIGA_CD).toBe(20)
  })

  it('CD 30 improvável', () => {
    expect(INTRIGA_IMPROVAVEL_CD).toBe(30)
  })

  it('expose margin 5', () => {
    expect(INTRIGA_EXPOSE_MARGIN).toBe(5)
  })

  it('ação dia ou mais', () => {
    expect(usage.action).toBe('dia-ou-mais')
  })
})

describe('intrigaCd', () => {
  it('normal → 20', () => {
    expect(intrigaCd(false)).toBe(20)
  })

  it('improvável → 30', () => {
    expect(intrigaCd(true)).toBe(30)
  })
})

describe('intrigaOutcome', () => {
  it('passa CD 20 → spread', () => {
    expect(intrigaOutcome(25, 20)).toBe('spread')
  })

  it('falha por 4 (CD 20 vs 16) → failed', () => {
    expect(intrigaOutcome(16, 20)).toBe('failed')
  })

  it('falha por 5 (CD 20 vs 15) → exposed', () => {
    expect(intrigaOutcome(15, 20)).toBe('exposed')
  })

  it('falha por 10 (CD 30 vs 20) → exposed', () => {
    expect(intrigaOutcome(20, 30)).toBe('exposed')
  })
})

describe('intrigaInvestigationCd', () => {
  it('CD Invest = teste de Enganação', () => {
    expect(intrigaInvestigationCd(27)).toBe(27)
  })
})

describe('Mentir — p119', () => {
  const usage = enganacaoUsageByKind('mentir')

  it('oposto por Intuição', () => {
    expect(usage.opposedBy).toBe('intuicao')
  })

  it('penalidade -10 verbatim', () => {
    expect(MENTIR_IMPLAUSIBLE_PENALTY).toBe(-10)
    if (usage.kind !== 'mentir') throw new Error('narrow failed')
    expect(usage.implausiblePenalty).toBe(-10)
  })
})

describe('mentirPenalty', () => {
  it('plausível → 0', () => {
    expect(mentirPenalty(false)).toBe(0)
  })

  it('implausível → -10', () => {
    expect(mentirPenalty(true)).toBe(-10)
  })
})

describe('enganacaoUsageByKind', () => {
  it('throws se kind inválido', () => {
    expect(() =>
      // @ts-expect-error — invalid kind on purpose
      enganacaoUsageByKind('blefar'),
    ).toThrow(/unknown kind/)
  })

  it('resolve todos', () => {
    const kinds = [
      'disfarce',
      'falsificacao',
      'fintar',
      'insinuacao',
      'intriga',
      'mentir',
    ] as const
    for (const k of kinds) {
      expect(enganacaoUsageByKind(k).kind).toBe(k)
    }
  })
})
