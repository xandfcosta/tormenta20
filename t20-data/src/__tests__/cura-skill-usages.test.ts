import { describe, expect, it } from 'vitest'
import {
  CURA_USAGES,
  cuidadosProlongadosHpBonus,
  cuidadosProlongadosMaxPatients,
  curaUsageByKind,
  necropsiaCd,
} from '../cura-skill-usages'

/**
 * PDF p117 — Perícia Cura (SAB). 4 usos canônicos:
 *  1. Primeiros Socorros — CD 15, padrão, NÃO treinado
 *  2. Tratamento — CD doença/veneno, completa, treinado, +5 Fort
 *  3. Cuidados Prolongados — CD 15, 1h, treinado, +1 PV/nível
 *  4. Necropsia — CD 20 (30 raras), 10min, treinado
 * Regras globais: kit obrigatório (-5 sem), -5 auto-uso (em outro módulo).
 */

describe('CURA_USAGES — shape & invariants', () => {
  it('exatamente 4 usos (p117)', () => {
    expect(CURA_USAGES.length).toBe(4)
  })

  it('catálogo frozen', () => {
    expect(Object.isFrozen(CURA_USAGES)).toBe(true)
  })

  it('todos ids únicos', () => {
    const ids = CURA_USAGES.map((u) => u.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('todos requerem maleta', () => {
    for (const u of CURA_USAGES) expect(u.requiresMaleta).toBe(true)
  })

  it('todos bookPage 117', () => {
    for (const u of CURA_USAGES) expect(u.bookPage).toBe(117)
  })

  it('só Primeiros Socorros é non-treinado', () => {
    const nonTreinado = CURA_USAGES.filter((u) => !u.trainedOnly).map((u) => u.id)
    expect(nonTreinado).toEqual(['primeiros-socorros'])
  })
})

describe('Primeiros Socorros', () => {
  const usage = curaUsageByKind('primeiros-socorros')

  it('CD 15', () => {
    if (usage.kind !== 'primeiros-socorros') throw new Error('narrow failed')
    expect(usage.dc).toBe(15)
  })

  it('ação padrão', () => {
    expect(usage.action).toBe('padrao')
  })

  it('não treinado', () => {
    expect(usage.trainedOnly).toBe(false)
  })
})

describe('Tratamento', () => {
  const usage = curaUsageByKind('tratamento')

  it('CD variável = doença/veneno', () => {
    if (usage.kind !== 'tratamento') throw new Error('narrow failed')
    expect(usage.dc).toBe('doenca-ou-veneno-cd')
  })

  it('ação completa', () => {
    expect(usage.action).toBe('completa')
  })

  it('treinado', () => {
    expect(usage.trainedOnly).toBe(true)
  })

  it('+5 no próximo Fort', () => {
    if (usage.kind !== 'tratamento') throw new Error('narrow failed')
    expect(usage.bonusNextFortitude).toBe(5)
  })
})

describe('Cuidados Prolongados', () => {
  const usage = curaUsageByKind('cuidados-prolongados')

  it('CD 15', () => {
    if (usage.kind !== 'cuidados-prolongados') throw new Error('narrow failed')
    expect(usage.dc).toBe(15)
  })

  it('duração 1 hora', () => {
    expect(usage.action).toBe('1-hora')
  })

  it('treinado', () => {
    expect(usage.trainedOnly).toBe(true)
  })

  it('+1 PV por nível', () => {
    if (usage.kind !== 'cuidados-prolongados') throw new Error('narrow failed')
    expect(usage.hpBonusPerLevel).toBe(1)
  })
})

describe('cuidadosProlongadosHpBonus', () => {
  it('nível 1 → +1 PV', () => {
    expect(cuidadosProlongadosHpBonus(1)).toBe(1)
  })

  it('nível 10 → +10 PV', () => {
    expect(cuidadosProlongadosHpBonus(10)).toBe(10)
  })

  it('nível 20 → +20 PV', () => {
    expect(cuidadosProlongadosHpBonus(20)).toBe(20)
  })

  it('throws se nível < 1', () => {
    expect(() => cuidadosProlongadosHpBonus(0)).toThrow(/curatorLevel/)
  })
})

describe('cuidadosProlongadosMaxPatients', () => {
  it('nível 1 → 1 paciente', () => {
    expect(cuidadosProlongadosMaxPatients(1)).toBe(1)
  })

  it('nível 5 → 5 pacientes', () => {
    expect(cuidadosProlongadosMaxPatients(5)).toBe(5)
  })

  it('nível 20 → 20 pacientes', () => {
    expect(cuidadosProlongadosMaxPatients(20)).toBe(20)
  })

  it('throws se nível < 1', () => {
    expect(() => cuidadosProlongadosMaxPatients(0)).toThrow(/curatorLevel/)
  })
})

describe('Necropsia', () => {
  const usage = curaUsageByKind('necropsia')

  it('CD base 20', () => {
    if (usage.kind !== 'necropsia') throw new Error('narrow failed')
    expect(usage.dcBase).toBe(20)
  })

  it('CD rara 30', () => {
    if (usage.kind !== 'necropsia') throw new Error('narrow failed')
    expect(usage.dcRareCause).toBe(30)
  })

  it('duração 10 minutos', () => {
    expect(usage.action).toBe('10-minutos')
  })

  it('treinado', () => {
    expect(usage.trainedOnly).toBe(true)
  })
})

describe('necropsiaCd', () => {
  it('causa normal → 20', () => {
    expect(necropsiaCd(false)).toBe(20)
  })

  it('causa rara (veneno/maldição) → 30', () => {
    expect(necropsiaCd(true)).toBe(30)
  })
})

describe('curaUsageByKind', () => {
  it.each(['primeiros-socorros', 'tratamento', 'cuidados-prolongados', 'necropsia'] as const)(
    'resolve %s',
    (kind) => {
      expect(curaUsageByKind(kind).kind).toBe(kind)
    },
  )
})
