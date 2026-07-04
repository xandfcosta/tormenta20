import { describe, expect, it } from 'vitest'
import {
  CONHECIMENTO_ARMOR_PENALTY,
  CONHECIMENTO_INFORMACAO_CD_COMPLEXA,
  CONHECIMENTO_INFORMACAO_CD_MISTERIO,
  CONHECIMENTO_TRAINED_ONLY,
  CONHECIMENTO_USAGES,
  IDIOMAS_CD_EXOTICO_OU_ANTIGO,
  IDIOMAS_CD_PADRAO,
  IDIOMAS_WRONG_CONCLUSION_MARGIN,
  conhecimentoInformacaoCd,
  conhecimentoUsageByKind,
  idiomasCd,
  idiomasOutcome,
} from '../conhecimento-skill-usages'

/**
 * PDF livro p117 — Perícia Conhecimento (INT, treinada).
 *  1. Idiomas — CD 20 padrão / 30 exótico; falha 5+ = conclusão falsa
 *  2. Informação — sem teste / CD 20 / CD 30
 */

describe('CONHECIMENTO_USAGES — shape', () => {
  it('exatamente 2 usos', () => {
    expect(CONHECIMENTO_USAGES.length).toBe(2)
  })

  it('frozen', () => {
    expect(Object.isFrozen(CONHECIMENTO_USAGES)).toBe(true)
  })

  it('ids canônicos', () => {
    expect(CONHECIMENTO_USAGES.map((u) => u.id).sort()).toEqual([
      'idiomas',
      'informacao',
    ])
  })

  it('todos em p117', () => {
    for (const u of CONHECIMENTO_USAGES) {
      expect(u.bookPage).toBe(117)
    }
  })
})

describe('Perícia flags — Tabela 2-1', () => {
  it('apenas treinada', () => {
    expect(CONHECIMENTO_TRAINED_ONLY).toBe(true)
  })

  it('sem penalidade de armadura', () => {
    expect(CONHECIMENTO_ARMOR_PENALTY).toBe(false)
  })
})

describe('Idiomas — p117', () => {
  const usage = conhecimentoUsageByKind('idiomas')

  it('CDs verbatim', () => {
    expect(IDIOMAS_CD_PADRAO).toBe(20)
    expect(IDIOMAS_CD_EXOTICO_OU_ANTIGO).toBe(30)
    expect(IDIOMAS_WRONG_CONCLUSION_MARGIN).toBe(5)
    if (usage.kind !== 'idiomas') throw new Error('narrow failed')
    expect(usage.cdPadrao).toBe(20)
    expect(usage.cdExoticoOuAntigo).toBe(30)
    expect(usage.wrongConclusionMargin).toBe(5)
  })
})

describe('idiomasCd', () => {
  it('padrão → 20', () => {
    expect(idiomasCd('padrao')).toBe(20)
  })

  it('exótico/antigo → 30', () => {
    expect(idiomasCd('exotico-ou-antigo')).toBe(30)
  })
})

describe('idiomasOutcome', () => {
  it('sucesso → success', () => {
    expect(idiomasOutcome(20, 20)).toBe('success')
    expect(idiomasOutcome(25, 20)).toBe('success')
  })

  it('falha < 5 → failed', () => {
    expect(idiomasOutcome(16, 20)).toBe('failed')
  })

  it('falha ≥ 5 → wrong-conclusion', () => {
    expect(idiomasOutcome(15, 20)).toBe('wrong-conclusion')
    expect(idiomasOutcome(10, 20)).toBe('wrong-conclusion')
  })
})

describe('Informação — p117', () => {
  it('CDs verbatim', () => {
    expect(CONHECIMENTO_INFORMACAO_CD_COMPLEXA).toBe(20)
    expect(CONHECIMENTO_INFORMACAO_CD_MISTERIO).toBe(30)
  })

  it('simples não exige teste', () => {
    const usage = conhecimentoUsageByKind('informacao')
    if (usage.kind !== 'informacao') throw new Error('narrow failed')
    expect(usage.simplesRequiresNoTest).toBe(true)
  })
})

describe('conhecimentoInformacaoCd', () => {
  it('simples → null', () => {
    expect(conhecimentoInformacaoCd('simples')).toBeNull()
  })

  it('complexa → 20', () => {
    expect(conhecimentoInformacaoCd('complexa')).toBe(20)
  })

  it('mistério/enigma → 30', () => {
    expect(conhecimentoInformacaoCd('misterio-ou-enigma')).toBe(30)
  })
})

describe('conhecimentoUsageByKind', () => {
  it('throws se kind inválido', () => {
    expect(() =>
      // @ts-expect-error — invalid kind on purpose
      conhecimentoUsageByKind('astronomia'),
    ).toThrow(/unknown kind/)
  })

  it('resolve todos', () => {
    for (const k of ['idiomas', 'informacao'] as const) {
      expect(conhecimentoUsageByKind(k).kind).toBe(k)
    }
  })
})
