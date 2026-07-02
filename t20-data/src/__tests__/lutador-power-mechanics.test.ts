import { describe, expect, it } from 'vitest'
import {
  LUTADOR_ELECTIVES,
  activeLutadorPowers,
  lutadorElectives,
  lutadorPowerById,
} from '../lutador-power-mechanics'

/**
 * PDF Cap 1 Lutador p76-78. Pinned:
 *  - 23 eletivos
 *  - Cabeçada + Golpe Baixo: livre 2 PM cena
 *  - Rasteira + Voadora + Trocação Tumultuosa: livre 2 PM
 *  - Trocação: livre pmCost variavel
 *  - Sequência Destruidora: livre 2 PM
 *  - Nome na Arena: completa cena
 *  - Imobilização: completa
 */

describe('LUTADOR_ELECTIVES — shape', () => {
  it('23 eletivos total', () => {
    expect(LUTADOR_ELECTIVES.length).toBe(23)
  })

  it('frozen', () => {
    expect(Object.isFrozen(LUTADOR_ELECTIVES)).toBe(true)
  })

  it('bookPage entre 76 e 78', () => {
    for (const p of LUTADOR_ELECTIVES) {
      expect(p.bookPage).toBeGreaterThanOrEqual(76)
      expect(p.bookPage).toBeLessThanOrEqual(78)
    }
  })

  it('IDs únicos', () => {
    const ids = LUTADOR_ELECTIVES.map((p) => p.id)
    expect(new Set(ids).size).toBe(ids.length)
  })
})

describe('LUTADOR_ELECTIVES — pinned entries', () => {
  it('Cabeçada: livre 2 PM cena', () => {
    const p = lutadorPowerById('cabecada')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(2)
    expect(p.uses).toBe('cena')
  })

  it('Golpe Baixo: livre 2 PM cena', () => {
    const p = lutadorPowerById('golpe-baixo')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(2)
    expect(p.uses).toBe('cena')
  })

  it('Rasteira: livre 2 PM', () => {
    const p = lutadorPowerById('rasteira')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(2)
  })

  it('Voadora: livre 2 PM', () => {
    const p = lutadorPowerById('voadora')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(2)
  })

  it('Trocação: livre pmCost variavel', () => {
    const p = lutadorPowerById('trocacao')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe('variavel')
  })

  it('Trocação Tumultuosa: livre 2 PM', () => {
    const p = lutadorPowerById('trocacao-tumultuosa')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(2)
  })

  it('Sequência Destruidora: livre 2 PM', () => {
    const p = lutadorPowerById('sequencia-destruidora')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(2)
  })

  it('Nome na Arena: completa cena', () => {
    const p = lutadorPowerById('nome-na-arena')!
    expect(p.action).toBe('completa')
    expect(p.uses).toBe('cena')
  })

  it('Imobilização: completa', () => {
    const p = lutadorPowerById('imobilizacao')!
    expect(p.action).toBe('completa')
  })

  it('Língua dos Becos: livre 1 PM', () => {
    const p = lutadorPowerById('lingua-dos-becos')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(1)
  })
})

describe('lutadorPowerById', () => {
  it('miss retorna undefined', () => {
    expect(lutadorPowerById('inexistente')).toBeUndefined()
  })
})

describe('lutadorElectives', () => {
  it('retorna todos', () => {
    expect(lutadorElectives()).toBe(LUTADOR_ELECTIVES)
  })
})

describe('activeLutadorPowers', () => {
  it('exclui passivos', () => {
    for (const p of activeLutadorPowers()) {
      expect(p.action).not.toBe('passivo')
    }
  })

  it('exatamente 10 poderes ativos', () => {
    expect(activeLutadorPowers().length).toBe(10)
  })
})
