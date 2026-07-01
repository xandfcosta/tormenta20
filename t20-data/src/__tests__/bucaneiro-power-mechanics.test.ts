import { describe, expect, it } from 'vitest'
import {
  BUCANEIRO_ELECTIVES,
  activeBucaneiroPowers,
  bucaneiroElectives,
  bucaneiroPowerById,
} from '../bucaneiro-power-mechanics'

/**
 * PDF Cap 1 Bucaneiro p45-48. Pinned:
 *  - 19 eletivos
 *  - 6 active: Aparar, Aventureiro Ávido, En Garde, Flagelo dos Mares,
 *    Ripostar, Touché
 *  - Aparar/Ripostar: reacao 1 PM
 *  - En Garde: movimento 1 PM cena
 *  - Aventureiro Ávido: livre 5 PM rodada
 *  - Touché: livre 2 PM
 *  - Flagelo dos Mares: varia pmCost variavel
 */

describe('BUCANEIRO_ELECTIVES — shape', () => {
  it('19 eletivos total', () => {
    expect(BUCANEIRO_ELECTIVES.length).toBe(19)
  })

  it('frozen', () => {
    expect(Object.isFrozen(BUCANEIRO_ELECTIVES)).toBe(true)
  })

  it('bookPage entre 47 e 48', () => {
    for (const p of BUCANEIRO_ELECTIVES) {
      expect(p.bookPage).toBeGreaterThanOrEqual(47)
      expect(p.bookPage).toBeLessThanOrEqual(48)
    }
  })

  it('IDs únicos', () => {
    const ids = BUCANEIRO_ELECTIVES.map((p) => p.id)
    expect(new Set(ids).size).toBe(ids.length)
  })
})

describe('BUCANEIRO_ELECTIVES — pinned entries', () => {
  it('Aparar: reacao, 1 PM', () => {
    const p = bucaneiroPowerById('aparar')!
    expect(p.action).toBe('reacao')
    expect(p.pmCost).toBe(1)
  })

  it('Ripostar: reacao, 1 PM', () => {
    const p = bucaneiroPowerById('ripostar')!
    expect(p.action).toBe('reacao')
    expect(p.pmCost).toBe(1)
  })

  it('En Garde: movimento, 1 PM, cena', () => {
    const p = bucaneiroPowerById('en-garde')!
    expect(p.action).toBe('movimento')
    expect(p.pmCost).toBe(1)
    expect(p.uses).toBe('cena')
  })

  it('Aventureiro Ávido: livre, 5 PM, rodada', () => {
    const p = bucaneiroPowerById('aventureiro-avido')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(5)
    expect(p.uses).toBe('rodada')
  })

  it('Touché: livre, 2 PM', () => {
    const p = bucaneiroPowerById('touche')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(2)
  })

  it('Flagelo dos Mares: varia, pmCost variavel (Amedrontar-like)', () => {
    const p = bucaneiroPowerById('flagelo-dos-mares')!
    expect(p.action).toBe('varia')
    expect(p.pmCost).toBe('variavel')
  })

  it('Amigos no Porto: passivo, uses dia', () => {
    const p = bucaneiroPowerById('amigos-no-porto')!
    expect(p.action).toBe('passivo')
    expect(p.uses).toBe('dia')
  })

  it('Apostador: passivo, uses dia', () => {
    const p = bucaneiroPowerById('apostador')!
    expect(p.uses).toBe('dia')
  })

  it('Esgrimista: passivo', () => {
    const p = bucaneiroPowerById('esgrimista')!
    expect(p.action).toBe('passivo')
  })
})

describe('bucaneiroPowerById', () => {
  it('miss retorna undefined', () => {
    expect(bucaneiroPowerById('inexistente')).toBeUndefined()
  })
})

describe('bucaneiroElectives', () => {
  it('retorna todos', () => {
    expect(bucaneiroElectives()).toBe(BUCANEIRO_ELECTIVES)
  })
})

describe('activeBucaneiroPowers', () => {
  it('exclui passivos', () => {
    for (const p of activeBucaneiroPowers()) {
      expect(p.action).not.toBe('passivo')
    }
  })

  it('exatamente 6 poderes ativos', () => {
    expect(activeBucaneiroPowers().length).toBe(6)
  })

  it('IDs corretos', () => {
    const ids = activeBucaneiroPowers().map((p) => p.id).sort()
    expect(ids).toEqual([
      'aparar',
      'aventureiro-avido',
      'en-garde',
      'flagelo-dos-mares',
      'ripostar',
      'touche',
    ])
  })
})
