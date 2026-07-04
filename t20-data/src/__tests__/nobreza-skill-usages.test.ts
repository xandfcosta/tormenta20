import { describe, expect, it } from 'vitest'
import {
  ETIQUETA_CD,
  NOBREZA_ARMOR_PENALTY,
  NOBREZA_INFORMACAO_CD_COMPLEXA,
  NOBREZA_INFORMACAO_CD_MISTERIO,
  NOBREZA_TRAINED_ONLY,
  NOBREZA_USAGES,
  etiquetaCd,
  nobrezaInformacaoCd,
  nobrezaUsageByKind,
} from '../nobreza-skill-usages'

/**
 * PDF livro p121 — Perícia Nobreza (INT, treinada).
 *  1. Etiqueta — CD 15
 *  2. Informação — sem teste / CD 20 / CD 30
 */

describe('NOBREZA_USAGES — shape', () => {
  it('exatamente 2 usos', () => {
    expect(NOBREZA_USAGES.length).toBe(2)
  })

  it('frozen', () => {
    expect(Object.isFrozen(NOBREZA_USAGES)).toBe(true)
  })

  it('ids canônicos', () => {
    expect(NOBREZA_USAGES.map((u) => u.id).sort()).toEqual([
      'etiqueta',
      'informacao',
    ])
  })

  it('todos em p121', () => {
    for (const u of NOBREZA_USAGES) {
      expect(u.bookPage).toBe(121)
    }
  })
})

describe('Perícia flags — Tabela 2-1', () => {
  it('apenas treinada', () => {
    expect(NOBREZA_TRAINED_ONLY).toBe(true)
  })

  it('sem penalidade de armadura', () => {
    expect(NOBREZA_ARMOR_PENALTY).toBe(false)
  })
})

describe('Etiqueta — p121', () => {
  it('CD 15', () => {
    expect(ETIQUETA_CD).toBe(15)
    const usage = nobrezaUsageByKind('etiqueta')
    if (usage.kind !== 'etiqueta') throw new Error('narrow failed')
    expect(usage.dc).toBe(15)
  })
})

describe('etiquetaCd', () => {
  it('sempre 15', () => {
    expect(etiquetaCd()).toBe(15)
  })
})

describe('Informação — p121', () => {
  it('CDs verbatim', () => {
    expect(NOBREZA_INFORMACAO_CD_COMPLEXA).toBe(20)
    expect(NOBREZA_INFORMACAO_CD_MISTERIO).toBe(30)
  })

  it('simples não exige teste', () => {
    const usage = nobrezaUsageByKind('informacao')
    if (usage.kind !== 'informacao') throw new Error('narrow failed')
    expect(usage.simplesRequiresNoTest).toBe(true)
  })
})

describe('nobrezaInformacaoCd', () => {
  it('simples → null', () => {
    expect(nobrezaInformacaoCd('simples')).toBeNull()
  })

  it('complexa → 20', () => {
    expect(nobrezaInformacaoCd('complexa')).toBe(20)
  })

  it('mistério/enigma → 30', () => {
    expect(nobrezaInformacaoCd('misterio-ou-enigma')).toBe(30)
  })
})

describe('nobrezaUsageByKind', () => {
  it('throws se kind inválido', () => {
    expect(() =>
      // @ts-expect-error — invalid kind on purpose
      nobrezaUsageByKind('heraldica'),
    ).toThrow(/unknown kind/)
  })

  it('resolve todos', () => {
    for (const k of ['etiqueta', 'informacao'] as const) {
      expect(nobrezaUsageByKind(k).kind).toBe(k)
    }
  })
})
