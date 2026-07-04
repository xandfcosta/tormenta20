import { describe, expect, it } from 'vitest'
import {
  PONTARIA_ARMOR_PENALTY,
  PONTARIA_TRAINED_ONLY,
  PONTARIA_USAGES,
  ataqueADistanciaCd,
  ataqueADistanciaOutcome,
  pontariaUsageByKind,
} from '../pontaria-skill-usages'

/**
 * PDF livro p122 — Perícia Pontaria (DES, aberta, sem armor penalty).
 *  1. Ataque à Distância — CD = Defesa do alvo; dano por arma
 */

describe('PONTARIA_USAGES — shape', () => {
  it('exatamente 1 uso', () => {
    expect(PONTARIA_USAGES.length).toBe(1)
  })

  it('frozen', () => {
    expect(Object.isFrozen(PONTARIA_USAGES)).toBe(true)
  })

  it('id canônico', () => {
    expect(PONTARIA_USAGES.map((u) => u.id)).toEqual(['ataque-a-distancia'])
  })

  it('bookPage 122', () => {
    for (const u of PONTARIA_USAGES) {
      expect(u.bookPage).toBe(122)
    }
  })
})

describe('Perícia flags — Tabela 2-1', () => {
  it('NÃO é somente treinada', () => {
    expect(PONTARIA_TRAINED_ONLY).toBe(false)
  })

  it('sem penalidade de armadura', () => {
    expect(PONTARIA_ARMOR_PENALTY).toBe(false)
  })
})

describe('Ataque à Distância — p122', () => {
  const usage = pontariaUsageByKind('ataque-a-distancia')

  it('CD = Defesa do alvo', () => {
    if (usage.kind !== 'ataque-a-distancia') throw new Error('narrow failed')
    expect(usage.cdEqualsTargetDefesa).toBe(true)
  })

  it('dano pela arma utilizada', () => {
    if (usage.kind !== 'ataque-a-distancia') throw new Error('narrow failed')
    expect(usage.damageBy).toBe('arma-utilizada')
  })

  it('cross-ref Cap 5: Jogando', () => {
    if (usage.kind !== 'ataque-a-distancia') throw new Error('narrow failed')
    expect(usage.crossRef).toBe('capitulo-5-jogando')
  })
})

describe('ataqueADistanciaCd', () => {
  it('CD = Defesa do alvo', () => {
    expect(ataqueADistanciaCd(15)).toBe(15)
    expect(ataqueADistanciaCd(22)).toBe(22)
  })
})

describe('ataqueADistanciaOutcome', () => {
  it('ataque > Defesa → hit', () => {
    expect(ataqueADistanciaOutcome(20, 15)).toBe('hit')
  })

  it('ataque = Defesa → hit (empate favorece atacante)', () => {
    expect(ataqueADistanciaOutcome(15, 15)).toBe('hit')
  })

  it('ataque < Defesa → miss', () => {
    expect(ataqueADistanciaOutcome(14, 15)).toBe('miss')
  })
})

describe('pontariaUsageByKind', () => {
  it('throws se kind inválido', () => {
    expect(() =>
      // @ts-expect-error — invalid kind on purpose
      pontariaUsageByKind('mirar'),
    ).toThrow(/unknown kind/)
  })

  it('resolve o kind único', () => {
    expect(pontariaUsageByKind('ataque-a-distancia').kind).toBe(
      'ataque-a-distancia',
    )
  })
})
