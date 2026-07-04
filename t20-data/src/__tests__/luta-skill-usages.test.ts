import { describe, expect, it } from 'vitest'
import {
  LUTA_ARMOR_PENALTY,
  LUTA_TRAINED_ONLY,
  LUTA_USAGES,
  ataqueCorpoACorpoCd,
  ataqueCorpoACorpoOutcome,
  lutaUsageByKind,
} from '../luta-skill-usages'

/**
 * PDF livro p121 — Perícia Luta (FOR, aberta, sem armor penalty).
 *  1. Ataque Corpo a Corpo — CD = Defesa do alvo; dano por arma
 */

describe('LUTA_USAGES — shape', () => {
  it('exatamente 1 uso', () => {
    expect(LUTA_USAGES.length).toBe(1)
  })

  it('frozen', () => {
    expect(Object.isFrozen(LUTA_USAGES)).toBe(true)
  })

  it('id canônico', () => {
    expect(LUTA_USAGES.map((u) => u.id)).toEqual(['ataque-corpo-a-corpo'])
  })

  it('bookPage 121', () => {
    for (const u of LUTA_USAGES) {
      expect(u.bookPage).toBe(121)
    }
  })
})

describe('Perícia flags — Tabela 2-1', () => {
  it('NÃO é somente treinada', () => {
    expect(LUTA_TRAINED_ONLY).toBe(false)
  })

  it('sem penalidade de armadura', () => {
    expect(LUTA_ARMOR_PENALTY).toBe(false)
  })
})

describe('Ataque Corpo a Corpo — p121', () => {
  const usage = lutaUsageByKind('ataque-corpo-a-corpo')

  it('CD = Defesa do alvo', () => {
    if (usage.kind !== 'ataque-corpo-a-corpo') throw new Error('narrow failed')
    expect(usage.cdEqualsTargetDefesa).toBe(true)
  })

  it('dano pela arma utilizada', () => {
    if (usage.kind !== 'ataque-corpo-a-corpo') throw new Error('narrow failed')
    expect(usage.damageBy).toBe('arma-utilizada')
  })

  it('cross-ref Cap 5: Jogando', () => {
    if (usage.kind !== 'ataque-corpo-a-corpo') throw new Error('narrow failed')
    expect(usage.crossRef).toBe('capitulo-5-jogando')
  })
})

describe('ataqueCorpoACorpoCd', () => {
  it('CD = Defesa do alvo', () => {
    expect(ataqueCorpoACorpoCd(15)).toBe(15)
    expect(ataqueCorpoACorpoCd(22)).toBe(22)
  })
})

describe('ataqueCorpoACorpoOutcome', () => {
  it('ataque > Defesa → hit', () => {
    expect(ataqueCorpoACorpoOutcome(20, 15)).toBe('hit')
  })

  it('ataque = Defesa → hit (empate favorece atacante)', () => {
    expect(ataqueCorpoACorpoOutcome(15, 15)).toBe('hit')
  })

  it('ataque < Defesa → miss', () => {
    expect(ataqueCorpoACorpoOutcome(14, 15)).toBe('miss')
  })
})

describe('lutaUsageByKind', () => {
  it('throws se kind inválido', () => {
    expect(() =>
      // @ts-expect-error — invalid kind on purpose
      lutaUsageByKind('agarrar'),
    ).toThrow(/unknown kind/)
  })

  it('resolve o kind único', () => {
    expect(lutaUsageByKind('ataque-corpo-a-corpo').kind).toBe(
      'ataque-corpo-a-corpo',
    )
  })
})
