import { describe, expect, it } from 'vitest'
import {
  CACADOR_ELECTIVES,
  activeCacadorPowers,
  armadilhaPowers,
  cacadorElectives,
  cacadorPowerById,
} from '../cacador-power-mechanics'

/**
 * PDF Cap 1 Caçador p48-51. Pinned:
 *  - 23 eletivos
 *  - 4 Armadilha: Arataca, Espinhos, Laço, Rede (isArmadilha=true)
 *  - Bote/Ímpeto: livre 1 PM
 *  - Camuflagem: livre 2 PM
 *  - Chuva de Lâminas: livre 2 PM rodada
 *  - Emboscar: livre 2 PM rodada (primeira rodada)
 *  - Ervas Curativas: completa pmCost variavel
 *  - Elo com a Natureza: varia pmCost variavel
 */

describe('CACADOR_ELECTIVES — shape', () => {
  it('23 eletivos total', () => {
    expect(CACADOR_ELECTIVES.length).toBe(23)
  })

  it('frozen', () => {
    expect(Object.isFrozen(CACADOR_ELECTIVES)).toBe(true)
  })

  it('bookPage entre 50 e 51', () => {
    for (const p of CACADOR_ELECTIVES) {
      expect(p.bookPage).toBeGreaterThanOrEqual(50)
      expect(p.bookPage).toBeLessThanOrEqual(51)
    }
  })

  it('IDs únicos', () => {
    const ids = CACADOR_ELECTIVES.map((p) => p.id)
    expect(new Set(ids).size).toBe(ids.length)
  })
})

describe('CACADOR_ELECTIVES — pinned entries', () => {
  it('Bote: livre, 1 PM', () => {
    const p = cacadorPowerById('bote')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(1)
  })

  it('Camuflagem: livre, 2 PM', () => {
    const p = cacadorPowerById('camuflagem')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(2)
  })

  it('Chuva de Lâminas: livre, 2 PM, rodada', () => {
    const p = cacadorPowerById('chuva-de-laminas')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(2)
    expect(p.uses).toBe('rodada')
  })

  it('Emboscar: livre, 2 PM', () => {
    const p = cacadorPowerById('emboscar')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(2)
  })

  it('Ímpeto: livre, 1 PM', () => {
    const p = cacadorPowerById('impeto')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(1)
  })

  it('Ervas Curativas: completa, pmCost variavel', () => {
    const p = cacadorPowerById('ervas-curativas')!
    expect(p.action).toBe('completa')
    expect(p.pmCost).toBe('variavel')
  })

  it('Elo com a Natureza: varia, pmCost variavel', () => {
    const p = cacadorPowerById('elo-com-a-natureza')!
    expect(p.action).toBe('varia')
    expect(p.pmCost).toBe('variavel')
  })
})

describe('armadilhaPowers', () => {
  it('exatamente 4 armadilhas', () => {
    expect(armadilhaPowers().length).toBe(4)
  })

  it('IDs corretos', () => {
    const ids = armadilhaPowers().map((p) => p.id).sort()
    expect(ids).toEqual([
      'armadilha-arataca',
      'armadilha-espinhos',
      'armadilha-laco',
      'armadilha-rede',
    ])
  })

  it('todas action padrao', () => {
    for (const p of armadilhaPowers()) {
      expect(p.action).toBe('padrao')
    }
  })
})

describe('cacadorPowerById', () => {
  it('miss retorna undefined', () => {
    expect(cacadorPowerById('inexistente')).toBeUndefined()
  })
})

describe('cacadorElectives', () => {
  it('retorna todos', () => {
    expect(cacadorElectives()).toBe(CACADOR_ELECTIVES)
  })
})

describe('activeCacadorPowers', () => {
  it('exclui passivos', () => {
    for (const p of activeCacadorPowers()) {
      expect(p.action).not.toBe('passivo')
    }
  })

  it('inclui armadilhas + bote + camuflagem + etc', () => {
    const ids = activeCacadorPowers().map((p) => p.id)
    expect(ids).toContain('bote')
    expect(ids).toContain('camuflagem')
    expect(ids).toContain('armadilha-arataca')
    expect(ids).toContain('ervas-curativas')
  })
})
