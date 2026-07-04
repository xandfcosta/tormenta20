import { describe, expect, it } from 'vitest'
import {
  CONDUZIR_VEICULO_CD_EXTREMA,
  CONDUZIR_VEICULO_CD_RUIM,
  CONDUZIR_VEICULO_CRASH_MARGIN,
  PILOTAGEM_ARMOR_PENALTY,
  PILOTAGEM_TRAINED_ONLY,
  PILOTAGEM_USAGES,
  pilotagemCd,
  pilotagemOutcome,
  pilotagemUsageByKind,
} from '../pilotagem-skill-usages'

/**
 * PDF livro p122 — Perícia Pilotagem (DES, treinada).
 *  1. Conduzir Veículo — simples auto, ruim CD 15, extrema CD 25; falha = metade; falha 5+ = acidente
 */

describe('PILOTAGEM_USAGES — shape', () => {
  it('exatamente 1 uso', () => {
    expect(PILOTAGEM_USAGES.length).toBe(1)
  })

  it('frozen', () => {
    expect(Object.isFrozen(PILOTAGEM_USAGES)).toBe(true)
  })

  it('id canônico', () => {
    expect(PILOTAGEM_USAGES.map((u) => u.id)).toEqual(['conduzir-veiculo'])
  })

  it('bookPage 122', () => {
    for (const u of PILOTAGEM_USAGES) {
      expect(u.bookPage).toBe(122)
    }
  })
})

describe('Perícia flags — Tabela 2-1', () => {
  it('apenas treinada', () => {
    expect(PILOTAGEM_TRAINED_ONLY).toBe(true)
  })

  it('sem penalidade de armadura', () => {
    expect(PILOTAGEM_ARMOR_PENALTY).toBe(false)
  })
})

describe('Conduzir Veículo — p122', () => {
  const usage = pilotagemUsageByKind('conduzir-veiculo')

  it('CDs verbatim', () => {
    expect(CONDUZIR_VEICULO_CD_RUIM).toBe(15)
    expect(CONDUZIR_VEICULO_CD_EXTREMA).toBe(25)
    expect(CONDUZIR_VEICULO_CRASH_MARGIN).toBe(5)
    if (usage.kind !== 'conduzir-veiculo') throw new Error('narrow failed')
    expect(usage.cdRuim).toBe(15)
    expect(usage.cdExtrema).toBe(25)
  })

  it('ação de movimento', () => {
    if (usage.kind !== 'conduzir-veiculo') throw new Error('narrow failed')
    expect(usage.action).toBe('movimento')
  })

  it('falha simples avança metade + crash margin 5', () => {
    if (usage.kind !== 'conduzir-veiculo') throw new Error('narrow failed')
    expect(usage.halfSpeedOnFailure).toBe(true)
    expect(usage.crashMargin).toBe(5)
  })
})

describe('pilotagemCd', () => {
  it('simples → null (automático)', () => {
    expect(pilotagemCd('simples')).toBeNull()
  })

  it('ruim → 15', () => {
    expect(pilotagemCd('ruim')).toBe(15)
  })

  it('extrema → 25', () => {
    expect(pilotagemCd('extrema')).toBe(25)
  })
})

describe('pilotagemOutcome', () => {
  it('sucesso → success', () => {
    expect(pilotagemOutcome(15, 15)).toBe('success')
    expect(pilotagemOutcome(20, 15)).toBe('success')
  })

  it('falha < 5 → half-speed', () => {
    expect(pilotagemOutcome(14, 15)).toBe('half-speed')
    expect(pilotagemOutcome(11, 15)).toBe('half-speed')
  })

  it('falha ≥ 5 → crashes', () => {
    expect(pilotagemOutcome(10, 15)).toBe('crashes')
    expect(pilotagemOutcome(15, 25)).toBe('crashes')
  })
})

describe('pilotagemUsageByKind', () => {
  it('throws se kind inválido', () => {
    expect(() =>
      // @ts-expect-error — invalid kind on purpose
      pilotagemUsageByKind('reparar-veiculo'),
    ).toThrow(/unknown kind/)
  })

  it('resolve o kind único', () => {
    expect(pilotagemUsageByKind('conduzir-veiculo').kind).toBe(
      'conduzir-veiculo',
    )
  })
})
