import { describe, expect, it } from 'vitest'
import {
  GUERREIRO_ELECTIVES,
  activeGuerreiroPowers,
  guerreiroElectives,
  guerreiroPowerById,
} from '../guerreiro-power-mechanics'

/**
 * PDF Cap 1 Guerreiro p64-67. Pinned:
 *  - 19 eletivos
 *  - Ataque Reflexo: reacao 1 PM rodada
 *  - Golpe de Raspão: reacao 2 PM rodada
 *  - Golpe Demolidor: livre 2 PM
 *  - Tornado de Dor: padrao 2 PM
 *  - Planejamento Marcial: varia 3 PM dia
 *  - Golpe Pessoal: varia pmCost variavel (technique construct)
 */

describe('GUERREIRO_ELECTIVES — shape', () => {
  it('19 eletivos total', () => {
    expect(GUERREIRO_ELECTIVES.length).toBe(19)
  })

  it('frozen', () => {
    expect(Object.isFrozen(GUERREIRO_ELECTIVES)).toBe(true)
  })

  it('bookPage entre 65 e 67', () => {
    for (const p of GUERREIRO_ELECTIVES) {
      expect(p.bookPage).toBeGreaterThanOrEqual(65)
      expect(p.bookPage).toBeLessThanOrEqual(67)
    }
  })

  it('IDs únicos', () => {
    const ids = GUERREIRO_ELECTIVES.map((p) => p.id)
    expect(new Set(ids).size).toBe(ids.length)
  })
})

describe('GUERREIRO_ELECTIVES — pinned entries', () => {
  it('Ataque Reflexo: reacao 1 PM rodada', () => {
    const p = guerreiroPowerById('ataque-reflexo')!
    expect(p.action).toBe('reacao')
    expect(p.pmCost).toBe(1)
    expect(p.uses).toBe('rodada')
  })

  it('Golpe de Raspão: reacao 2 PM rodada', () => {
    const p = guerreiroPowerById('golpe-de-raspao')!
    expect(p.action).toBe('reacao')
    expect(p.pmCost).toBe(2)
    expect(p.uses).toBe('rodada')
  })

  it('Golpe Demolidor: livre 2 PM', () => {
    const p = guerreiroPowerById('golpe-demolidor')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(2)
  })

  it('Tornado de Dor: padrao 2 PM', () => {
    const p = guerreiroPowerById('tornado-de-dor')!
    expect(p.action).toBe('padrao')
    expect(p.pmCost).toBe(2)
  })

  it('Planejamento Marcial: varia 3 PM dia', () => {
    const p = guerreiroPowerById('planejamento-marcial')!
    expect(p.action).toBe('varia')
    expect(p.pmCost).toBe(3)
    expect(p.uses).toBe('dia')
  })

  it('Golpe Pessoal: varia pmCost variavel', () => {
    const p = guerreiroPowerById('golpe-pessoal')!
    expect(p.action).toBe('varia')
    expect(p.pmCost).toBe('variavel')
  })

  it('Ímpeto: livre 1 PM', () => {
    const p = guerreiroPowerById('impeto')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(1)
  })
})

describe('guerreiroPowerById', () => {
  it('miss retorna undefined', () => {
    expect(guerreiroPowerById('inexistente')).toBeUndefined()
  })
})

describe('guerreiroElectives', () => {
  it('retorna todos', () => {
    expect(guerreiroElectives()).toBe(GUERREIRO_ELECTIVES)
  })
})

describe('activeGuerreiroPowers', () => {
  it('exclui passivos', () => {
    for (const p of activeGuerreiroPowers()) {
      expect(p.action).not.toBe('passivo')
    }
  })

  it('exatamente 7 poderes ativos', () => {
    expect(activeGuerreiroPowers().length).toBe(7)
  })

  it('IDs corretos', () => {
    const ids = activeGuerreiroPowers().map((p) => p.id).sort()
    expect(ids).toEqual([
      'ataque-reflexo',
      'golpe-de-raspao',
      'golpe-demolidor',
      'golpe-pessoal',
      'impeto',
      'planejamento-marcial',
      'tornado-de-dor',
    ])
  })
})
