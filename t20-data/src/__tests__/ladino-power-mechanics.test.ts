import { describe, expect, it } from 'vitest'
import {
  LADINO_ELECTIVES,
  activeLadinoPowers,
  ladinoElectives,
  ladinoPowerById,
} from '../ladino-power-mechanics'

/**
 * PDF Cap 1 Ladino p72-75. Pinned:
 *  - 20 eletivos
 *  - Assassinar: movimento 3 PM
 *  - Emboscar: livre 2 PM rodada
 *  - Fuga Formidável: completa 1 PM cena
 *  - Oportunismo: reacao 2 PM rodada
 *  - Rolamento Defensivo: reacao 2 PM
 *  - Mãos Rápidas: livre 1 PM rodada
 *  - Velocidade Ladina: livre 2 PM rodada
 *  - Truque Mágico: varia variavel
 */

describe('LADINO_ELECTIVES — shape', () => {
  it('20 eletivos total', () => {
    expect(LADINO_ELECTIVES.length).toBe(20)
  })

  it('frozen', () => {
    expect(Object.isFrozen(LADINO_ELECTIVES)).toBe(true)
  })

  it('bookPage entre 73 e 74', () => {
    for (const p of LADINO_ELECTIVES) {
      expect(p.bookPage).toBeGreaterThanOrEqual(73)
      expect(p.bookPage).toBeLessThanOrEqual(74)
    }
  })

  it('IDs únicos', () => {
    const ids = LADINO_ELECTIVES.map((p) => p.id)
    expect(new Set(ids).size).toBe(ids.length)
  })
})

describe('LADINO_ELECTIVES — pinned entries', () => {
  it('Assassinar: movimento 3 PM', () => {
    const p = ladinoPowerById('assassinar')!
    expect(p.action).toBe('movimento')
    expect(p.pmCost).toBe(3)
  })

  it('Emboscar: livre 2 PM rodada', () => {
    const p = ladinoPowerById('emboscar')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(2)
    expect(p.uses).toBe('rodada')
  })

  it('Fuga Formidável: completa 1 PM cena', () => {
    const p = ladinoPowerById('fuga-formidavel')!
    expect(p.action).toBe('completa')
    expect(p.pmCost).toBe(1)
    expect(p.uses).toBe('cena')
  })

  it('Oportunismo: reacao 2 PM rodada', () => {
    const p = ladinoPowerById('oportunismo')!
    expect(p.action).toBe('reacao')
    expect(p.pmCost).toBe(2)
    expect(p.uses).toBe('rodada')
  })

  it('Rolamento Defensivo: reacao 2 PM', () => {
    const p = ladinoPowerById('rolamento-defensivo')!
    expect(p.action).toBe('reacao')
    expect(p.pmCost).toBe(2)
  })

  it('Mãos Rápidas: livre 1 PM rodada', () => {
    const p = ladinoPowerById('maos-rapidas')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(1)
    expect(p.uses).toBe('rodada')
  })

  it('Velocidade Ladina: livre 2 PM rodada', () => {
    const p = ladinoPowerById('velocidade-ladina')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(2)
    expect(p.uses).toBe('rodada')
  })

  it('Truque Mágico: varia pmCost variavel', () => {
    const p = ladinoPowerById('truque-magico')!
    expect(p.action).toBe('varia')
    expect(p.pmCost).toBe('variavel')
  })
})

describe('ladinoPowerById', () => {
  it('miss retorna undefined', () => {
    expect(ladinoPowerById('inexistente')).toBeUndefined()
  })
})

describe('ladinoElectives', () => {
  it('retorna todos', () => {
    expect(ladinoElectives()).toBe(LADINO_ELECTIVES)
  })
})

describe('activeLadinoPowers', () => {
  it('exclui passivos', () => {
    for (const p of activeLadinoPowers()) {
      expect(p.action).not.toBe('passivo')
    }
  })

  it('exatamente 8 poderes ativos', () => {
    expect(activeLadinoPowers().length).toBe(8)
  })

  it('IDs corretos', () => {
    const ids = activeLadinoPowers().map((p) => p.id).sort()
    expect(ids).toEqual([
      'assassinar',
      'emboscar',
      'fuga-formidavel',
      'maos-rapidas',
      'oportunismo',
      'rolamento-defensivo',
      'truque-magico',
      'velocidade-ladina',
    ])
  })
})
