import { describe, expect, it } from 'vitest'
import {
  INICIATIVA_ARMOR_PENALTY,
  INICIATIVA_TRAINED_ONLY,
  INICIATIVA_USAGES,
  compareIniciativaDesc,
  iniciativaUsageByKind,
  sortByIniciativaDesc,
} from '../iniciativa-skill-usages'

/**
 * PDF livro p119 — Perícia Iniciativa (DES, aberta).
 *  1. Teste de Iniciativa — rolado no início da cena; ordem decrescente
 */

describe('INICIATIVA_USAGES — shape', () => {
  it('exatamente 1 uso', () => {
    expect(INICIATIVA_USAGES.length).toBe(1)
  })

  it('frozen', () => {
    expect(Object.isFrozen(INICIATIVA_USAGES)).toBe(true)
  })

  it('id canônico', () => {
    expect(INICIATIVA_USAGES.map((u) => u.id)).toEqual(['teste-iniciativa'])
  })

  it('bookPage 119', () => {
    for (const u of INICIATIVA_USAGES) {
      expect(u.bookPage).toBe(119)
    }
  })
})

describe('Perícia flags — Tabela 2-1', () => {
  it('NÃO é somente treinada', () => {
    expect(INICIATIVA_TRAINED_ONLY).toBe(false)
  })

  it('sem penalidade de armadura', () => {
    expect(INICIATIVA_ARMOR_PENALTY).toBe(false)
  })
})

describe('Teste de Iniciativa — p119', () => {
  const usage = iniciativaUsageByKind('teste-iniciativa')

  it('rolado ao início da cena; ordem decrescente', () => {
    if (usage.kind !== 'teste-iniciativa') throw new Error('narrow failed')
    expect(usage.rolledAtSceneStart).toBe(true)
    expect(usage.orderDescending).toBe(true)
  })
})

describe('compareIniciativaDesc', () => {
  it('maior primeiro', () => {
    expect(compareIniciativaDesc(20, 15)).toBeLessThan(0)
    expect(compareIniciativaDesc(15, 20)).toBeGreaterThan(0)
    expect(compareIniciativaDesc(15, 15)).toBe(0)
  })
})

describe('sortByIniciativaDesc', () => {
  it('ordena participantes descendente', () => {
    const combatants = [
      { name: 'A', iniciativa: 12 },
      { name: 'B', iniciativa: 20 },
      { name: 'C', iniciativa: 8 },
      { name: 'D', iniciativa: 18 },
    ]
    const sorted = sortByIniciativaDesc(combatants)
    expect(sorted.map((c) => c.name)).toEqual(['B', 'D', 'A', 'C'])
  })

  it('preserva ordem original em empates (stable)', () => {
    const combatants = [
      { name: 'A', iniciativa: 15 },
      { name: 'B', iniciativa: 20 },
      { name: 'C', iniciativa: 15 },
      { name: 'D', iniciativa: 15 },
    ]
    const sorted = sortByIniciativaDesc(combatants)
    expect(sorted.map((c) => c.name)).toEqual(['B', 'A', 'C', 'D'])
  })

  it('não modifica input', () => {
    const combatants = [
      { name: 'A', iniciativa: 10 },
      { name: 'B', iniciativa: 20 },
    ]
    sortByIniciativaDesc(combatants)
    expect(combatants.map((c) => c.name)).toEqual(['A', 'B'])
  })

  it('lista vazia → vazia', () => {
    expect(sortByIniciativaDesc([])).toEqual([])
  })
})

describe('iniciativaUsageByKind', () => {
  it('throws se kind inválido', () => {
    expect(() =>
      // @ts-expect-error — invalid kind on purpose
      iniciativaUsageByKind('surpresa'),
    ).toThrow(/unknown kind/)
  })

  it('resolve o kind único', () => {
    expect(iniciativaUsageByKind('teste-iniciativa').kind).toBe(
      'teste-iniciativa',
    )
  })
})
