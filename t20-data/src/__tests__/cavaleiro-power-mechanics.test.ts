import { describe, expect, it } from 'vitest'
import {
  CAVALEIRO_ELECTIVES,
  activeCavaleiroPowers,
  caminhoPowers,
  cavaleiroElectives,
  cavaleiroPowerById,
  posturaPowers,
} from '../cavaleiro-power-mechanics'

/**
 * PDF Cap 1 Cavaleiro p52-55. Pinned:
 *  - 22 eletivos
 *  - 6 Posturas: Ariete/Castigo/Foco/Muralha/Provocação/Torre
 *  - 2 Caminhos: Bastião, Montaria
 *  - 3 active: Autoridade Feudal (varia), Investida Destruidora (livre 2 PM),
 *    Torre Armada (livre 1 PM)
 */

describe('CAVALEIRO_ELECTIVES — shape', () => {
  it('22 eletivos total', () => {
    expect(CAVALEIRO_ELECTIVES.length).toBe(22)
  })

  it('frozen', () => {
    expect(Object.isFrozen(CAVALEIRO_ELECTIVES)).toBe(true)
  })

  it('bookPage entre 53 e 55', () => {
    for (const p of CAVALEIRO_ELECTIVES) {
      expect(p.bookPage).toBeGreaterThanOrEqual(53)
      expect(p.bookPage).toBeLessThanOrEqual(55)
    }
  })

  it('IDs únicos', () => {
    const ids = CAVALEIRO_ELECTIVES.map((p) => p.id)
    expect(new Set(ids).size).toBe(ids.length)
  })
})

describe('CAVALEIRO_ELECTIVES — pinned entries', () => {
  it('Investida Destruidora: livre, 2 PM', () => {
    const p = cavaleiroPowerById('investida-destruidora')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(2)
  })

  it('Torre Armada: livre, 1 PM', () => {
    const p = cavaleiroPowerById('torre-armada')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(1)
  })

  it('Autoridade Feudal: varia, 2 PM', () => {
    const p = cavaleiroPowerById('autoridade-feudal')!
    expect(p.action).toBe('varia')
    expect(p.pmCost).toBe(2)
  })

  it('Armadura da Honra: passivo, uses cena', () => {
    const p = cavaleiroPowerById('armadura-da-honra')!
    expect(p.action).toBe('passivo')
    expect(p.uses).toBe('cena')
  })

  it('Estandarte: passivo, uses cena', () => {
    const p = cavaleiroPowerById('estandarte')!
    expect(p.uses).toBe('cena')
  })
})

describe('posturaPowers', () => {
  it('exatamente 6 Posturas', () => {
    expect(posturaPowers().length).toBe(6)
  })

  it('IDs corretos', () => {
    const ids = posturaPowers().map((p) => p.id).sort()
    expect(ids).toEqual([
      'postura-ariete-implacavel',
      'postura-castigo-de-ferro',
      'postura-foco-de-batalha',
      'postura-muralha-intransponivel',
      'postura-provocacao-petulante',
      'postura-torre-inabalavel',
    ])
  })

  it('todas começam com "Postura:" no nome', () => {
    for (const p of posturaPowers()) {
      expect(p.name.startsWith('Postura:')).toBe(true)
    }
  })
})

describe('caminhoPowers', () => {
  it('exatamente 2 Caminhos', () => {
    expect(caminhoPowers().length).toBe(2)
  })

  it('IDs Bastião + Montaria', () => {
    const ids = caminhoPowers().map((p) => p.id).sort()
    expect(ids).toEqual(['caminho-bastiao', 'caminho-montaria'])
  })
})

describe('cavaleiroPowerById', () => {
  it('miss retorna undefined', () => {
    expect(cavaleiroPowerById('inexistente')).toBeUndefined()
  })
})

describe('cavaleiroElectives', () => {
  it('retorna todos', () => {
    expect(cavaleiroElectives()).toBe(CAVALEIRO_ELECTIVES)
  })
})

describe('activeCavaleiroPowers', () => {
  it('exclui passivos', () => {
    for (const p of activeCavaleiroPowers()) {
      expect(p.action).not.toBe('passivo')
    }
  })

  it('exatamente 3 poderes ativos (Autoridade, Investida Destruidora, Torre Armada)', () => {
    expect(activeCavaleiroPowers().length).toBe(3)
  })
})
