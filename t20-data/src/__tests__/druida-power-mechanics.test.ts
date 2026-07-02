import { describe, expect, it } from 'vitest'
import {
  DRUIDA_ELECTIVES,
  activeDruidaPowers,
  aspectoPowers,
  druidaElectives,
  druidaPowerById,
  formaSelvagemPowers,
} from '../druida-power-mechanics'

/**
 * PDF Cap 1 Druida p60-63. Pinned:
 *  - 22 eletivos
 *  - 4 Aspectos: Primavera + Inverno passivos; Verão + Outono livre 1 PM
 *  - 3 Forma Selvagem tiers: completa 3/6/10 PM
 *  - Força dos Penhascos: reacao pmCost variavel
 *  - Tranquilidade dos Lagos: reacao 1 PM rodada
 *  - Espírito dos Equinócios: livre 4 PM cena
 */

describe('DRUIDA_ELECTIVES — shape', () => {
  it('22 eletivos total', () => {
    expect(DRUIDA_ELECTIVES.length).toBe(22)
  })

  it('frozen', () => {
    expect(Object.isFrozen(DRUIDA_ELECTIVES)).toBe(true)
  })

  it('bookPage entre 61 e 63', () => {
    for (const p of DRUIDA_ELECTIVES) {
      expect(p.bookPage).toBeGreaterThanOrEqual(61)
      expect(p.bookPage).toBeLessThanOrEqual(63)
    }
  })

  it('IDs únicos', () => {
    const ids = DRUIDA_ELECTIVES.map((p) => p.id)
    expect(new Set(ids).size).toBe(ids.length)
  })
})

describe('aspectoPowers', () => {
  it('exatamente 4 Aspectos', () => {
    expect(aspectoPowers().length).toBe(4)
  })

  it('IDs corretos', () => {
    const ids = aspectoPowers().map((p) => p.id).sort()
    expect(ids).toEqual([
      'aspecto-da-primavera',
      'aspecto-do-inverno',
      'aspecto-do-outono',
      'aspecto-do-verao',
    ])
  })

  it('Primavera + Inverno passivos', () => {
    expect(druidaPowerById('aspecto-da-primavera')!.action).toBe('passivo')
    expect(druidaPowerById('aspecto-do-inverno')!.action).toBe('passivo')
  })

  it('Verão + Outono ativos com 1 PM', () => {
    const verao = druidaPowerById('aspecto-do-verao')!
    const outono = druidaPowerById('aspecto-do-outono')!
    expect(verao.action).toBe('livre')
    expect(verao.pmCost).toBe(1)
    expect(outono.action).toBe('livre')
    expect(outono.pmCost).toBe(1)
  })
})

describe('formaSelvagemPowers', () => {
  it('exatamente 3 tiers', () => {
    expect(formaSelvagemPowers().length).toBe(3)
  })

  it('progressão de PM 3 → 6 → 10 (completa)', () => {
    expect(druidaPowerById('forma-selvagem')!.pmCost).toBe(3)
    expect(druidaPowerById('forma-selvagem-aprimorada')!.pmCost).toBe(6)
    expect(druidaPowerById('forma-selvagem-superior')!.pmCost).toBe(10)
  })

  it('todas action completa', () => {
    for (const p of formaSelvagemPowers()) {
      expect(p.action).toBe('completa')
    }
  })
})

describe('DRUIDA_ELECTIVES — pinned entries', () => {
  it('Espírito dos Equinócios: livre 4 PM cena', () => {
    const p = druidaPowerById('espirito-dos-equinocios')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(4)
    expect(p.uses).toBe('cena')
  })

  it('Força dos Penhascos: reacao pmCost variavel', () => {
    const p = druidaPowerById('forca-dos-penhascos')!
    expect(p.action).toBe('reacao')
    expect(p.pmCost).toBe('variavel')
  })

  it('Tranquilidade dos Lagos: reacao 1 PM rodada', () => {
    const p = druidaPowerById('tranquilidade-dos-lagos')!
    expect(p.action).toBe('reacao')
    expect(p.pmCost).toBe(1)
    expect(p.uses).toBe('rodada')
  })
})

describe('druidaPowerById', () => {
  it('miss retorna undefined', () => {
    expect(druidaPowerById('inexistente')).toBeUndefined()
  })
})

describe('druidaElectives', () => {
  it('retorna todos', () => {
    expect(druidaElectives()).toBe(DRUIDA_ELECTIVES)
  })
})

describe('activeDruidaPowers', () => {
  it('exclui passivos', () => {
    for (const p of activeDruidaPowers()) {
      expect(p.action).not.toBe('passivo')
    }
  })

  it('inclui todas 3 Formas + Verão + Outono + Equinócios + Penhascos + Lagos', () => {
    const ids = activeDruidaPowers().map((p) => p.id).sort()
    expect(ids).toEqual([
      'aspecto-do-outono',
      'aspecto-do-verao',
      'espirito-dos-equinocios',
      'forca-dos-penhascos',
      'forma-selvagem',
      'forma-selvagem-aprimorada',
      'forma-selvagem-superior',
      'tranquilidade-dos-lagos',
    ])
  })
})
