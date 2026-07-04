import { describe, expect, it } from 'vitest'
import {
  ACALMAR_ANIMAL_CD,
  ADESTRAMENTO_ARMOR_PENALTY,
  ADESTRAMENTO_TRAINED_ONLY,
  ADESTRAMENTO_USAGES,
  MANEJAR_ANIMAL_CD,
  MANEJAR_ANIMAL_EXAMPLE_COMMANDS,
  acalmarAnimalCd,
  adestramentoUsageByKind,
  canSubstitutePilotagemForVehicle,
  manejarAnimalCd,
} from '../adestramento-skill-usages'

/**
 * PDF livro p115 — Perícia Adestramento (CAR, treinada).
 * T20 core simplifica: SÓ 2 usos.
 *  1. Acalmar Animal — CD 25, ação completa, alvo não precisa treinado
 *  2. Manejar Animal — CD 15, ação de movimento, alvo precisa treinado
 */

describe('ADESTRAMENTO_USAGES — shape', () => {
  it('exatamente 2 usos (p115)', () => {
    expect(ADESTRAMENTO_USAGES.length).toBe(2)
  })

  it('frozen', () => {
    expect(Object.isFrozen(ADESTRAMENTO_USAGES)).toBe(true)
  })

  it('ids únicos', () => {
    const ids = ADESTRAMENTO_USAGES.map((u) => u.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('todos bookPage 115', () => {
    for (const u of ADESTRAMENTO_USAGES) expect(u.bookPage).toBe(115)
  })

  it('ids canônicos', () => {
    expect(ADESTRAMENTO_USAGES.map((u) => u.id).sort()).toEqual([
      'acalmar-animal',
      'manejar-animal',
    ])
  })
})

describe('Perícia flags — Tabela 2-1', () => {
  it('apenas treinada', () => {
    expect(ADESTRAMENTO_TRAINED_ONLY).toBe(true)
  })

  it('sem penalidade de armadura', () => {
    expect(ADESTRAMENTO_ARMOR_PENALTY).toBe(false)
  })
})

describe('Acalmar Animal — p115', () => {
  const usage = adestramentoUsageByKind('acalmar-animal')

  it('CD 25', () => {
    expect(ACALMAR_ANIMAL_CD).toBe(25)
    expect(acalmarAnimalCd()).toBe(25)
  })

  it('ação completa', () => {
    expect(usage.action).toBe('completa')
  })

  it('alvo NÃO precisa estar treinado', () => {
    if (usage.kind !== 'acalmar-animal') throw new Error('narrow failed')
    expect(usage.requiresTrainedTarget).toBe(false)
  })

  it('CD no catálogo bate com constante', () => {
    expect(usage.dc).toBe(ACALMAR_ANIMAL_CD)
  })
})

describe('Manejar Animal — p115', () => {
  const usage = adestramentoUsageByKind('manejar-animal')

  it('CD 15', () => {
    expect(MANEJAR_ANIMAL_CD).toBe(15)
    expect(manejarAnimalCd()).toBe(15)
  })

  it('ação de movimento', () => {
    expect(usage.action).toBe('movimento')
  })

  it('alvo PRECISA estar treinado para tarefa', () => {
    if (usage.kind !== 'manejar-animal') throw new Error('narrow failed')
    expect(usage.requiresTrainedTarget).toBe(true)
  })

  it('substitui Pilotagem para veículos de tração animal', () => {
    if (usage.kind !== 'manejar-animal') throw new Error('narrow failed')
    expect(usage.substitutesPilotagemForAnimalDrawnVehicles).toBe(true)
  })

  it('CD no catálogo bate com constante', () => {
    expect(usage.dc).toBe(MANEJAR_ANIMAL_CD)
  })
})

describe('MANEJAR_ANIMAL_EXAMPLE_COMMANDS — verbatim inline', () => {
  it('frozen', () => {
    expect(Object.isFrozen(MANEJAR_ANIMAL_EXAMPLE_COMMANDS)).toBe(true)
  })

  it('contém os 3 exemplos citados', () => {
    expect([...MANEJAR_ANIMAL_EXAMPLE_COMMANDS].sort()).toEqual([
      'atacar',
      'sentar',
      'vigiar',
    ])
  })
})

describe('canSubstitutePilotagemForVehicle', () => {
  it('veículo de tração animal → true', () => {
    expect(canSubstitutePilotagemForVehicle(true)).toBe(true)
  })

  it('veículo não-animal → false', () => {
    expect(canSubstitutePilotagemForVehicle(false)).toBe(false)
  })
})

describe('adestramentoUsageByKind', () => {
  it('throws se kind inválido', () => {
    expect(() =>
      // @ts-expect-error — invalid kind on purpose
      adestramentoUsageByKind('treinar-animal'),
    ).toThrow(/unknown kind/)
  })

  it('resolve acalmar', () => {
    expect(adestramentoUsageByKind('acalmar-animal').name).toBe(
      'Acalmar Animal',
    )
  })

  it('resolve manejar', () => {
    expect(adestramentoUsageByKind('manejar-animal').name).toBe(
      'Manejar Animal',
    )
  })
})
