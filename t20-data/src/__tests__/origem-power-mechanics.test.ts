import { describe, expect, it } from 'vitest'
import { ORIGENS } from '../origens'
import {
  ORIGEM_POWERS,
  activeOrigemPowers,
  origemPowerOf,
  pmConsumingOrigemPowers,
} from '../origem-power-mechanics'

/**
 * PDF Cap 1 Origens p85-95. Pinned:
 *  - 35 origens × 1 poder único = 35 entries.
 *  - Nomes de OrigemPower.name batem com Origem.poderUnico.
 *  - Só 5 poderes ativos: Busca Interior, Cultura Exótica, Detetive,
 *    Palpite Fundamentado, Truque de Mágica.
 */

describe('ORIGEM_POWERS — shape', () => {
  it('35 entries (uma por origem)', () => {
    expect(ORIGEM_POWERS.length).toBe(35)
  })

  it('frozen', () => {
    expect(Object.isFrozen(ORIGEM_POWERS)).toBe(true)
  })

  it('cada origemId existe em ORIGENS', () => {
    for (const p of ORIGEM_POWERS) {
      expect(ORIGENS[p.origemId]).toBeDefined()
    }
  })

  it('bookPage entre 85 e 95', () => {
    for (const p of ORIGEM_POWERS) {
      expect(p.bookPage).toBeGreaterThanOrEqual(85)
      expect(p.bookPage).toBeLessThanOrEqual(95)
    }
  })

  it('cada bookPage bate com Origem.bookPage', () => {
    for (const p of ORIGEM_POWERS) {
      const origem = ORIGENS[p.origemId]!
      expect(p.bookPage).toBe(origem.bookPage)
    }
  })

  it('nomes batem com Origem.poderUnico', () => {
    for (const p of ORIGEM_POWERS) {
      const origem = ORIGENS[p.origemId]!
      expect(origem.poderUnico).toBe(p.name)
    }
  })
})

describe('ORIGEM_POWERS — pinned entries ativos', () => {
  it('Eremita: Busca Interior completa/1PM', () => {
    const p = origemPowerOf('eremita')!
    expect(p.action).toBe('completa')
    expect(p.pmCost).toBe(1)
    expect(p.uses).toBeNull()
  })

  it('Forasteiro: Cultura Exótica livre/1PM', () => {
    const p = origemPowerOf('forasteiro')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(1)
  })

  it('Guarda: Detetive livre/1PM/cena', () => {
    const p = origemPowerOf('guarda')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(1)
    expect(p.uses).toBe('cena')
  })

  it('Estudioso: Palpite Fundamentado livre/2PM', () => {
    const p = origemPowerOf('estudioso')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(2)
  })

  it('Circense: Truque de Mágica varia/variavel', () => {
    const p = origemPowerOf('circense')!
    expect(p.action).toBe('varia')
    expect(p.pmCost).toBe('variavel')
  })
})

describe('ORIGEM_POWERS — passivos pinned', () => {
  it('Acólito: Membro da Igreja passivo', () => {
    const p = origemPowerOf('acolito')!
    expect(p.action).toBe('passivo')
    expect(p.pmCost).toBe(0)
  })

  it('Aristocrata: Sangue Azul passivo', () => {
    const p = origemPowerOf('aristocrata')!
    expect(p.action).toBe('passivo')
  })

  it('Criminoso: Punguista passivo/dia', () => {
    const p = origemPowerOf('criminoso')!
    expect(p.action).toBe('passivo')
    expect(p.uses).toBe('dia')
  })

  it('Seguidor: Antigo Mestre passivo/cena', () => {
    const p = origemPowerOf('seguidor')!
    expect(p.action).toBe('passivo')
    expect(p.uses).toBe('cena')
  })
})

describe('origemPowerOf', () => {
  it('miss retorna undefined', () => {
    expect(origemPowerOf('inexistente')).toBeUndefined()
  })
})

describe('activeOrigemPowers', () => {
  it('exatamente 5 poderes ativos', () => {
    expect(activeOrigemPowers().length).toBe(5)
  })

  it('inclui os 5 conhecidos', () => {
    const names = activeOrigemPowers().map((p) => p.name).sort()
    expect(names).toEqual([
      'Busca Interior',
      'Cultura Exótica',
      'Detetive',
      'Palpite Fundamentado',
      'Truque de Mágica',
    ])
  })
})

describe('pmConsumingOrigemPowers', () => {
  it('cobre exatamente os 5 ativos', () => {
    expect(pmConsumingOrigemPowers().length).toBe(5)
  })
})
