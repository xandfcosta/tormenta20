import { describe, expect, it } from 'vitest'
import {
  AUTORIDADE_FEUDAL_MIN_LEVEL,
  CAVALEIRO_PARCEIRO_POWERS,
  MONTARIA_MESTRE_LEVEL,
  MONTARIA_UNLOCK_LEVEL,
  TITULO_MIN_LEVEL,
  cavaleiroParceiroPowerById,
  hasParceiroLimitException,
  unlockedCavaleiroParceiroPowers,
} from '../cavaleiro-parceiro-rules'

/**
 * PDF Cavaleiro p54-55. Pinned:
 *  - Autoridade Feudal L6, iniciante, conta contra limite
 *  - Escudeiro sem gate, exceção ao limite, retrain 1 mês
 *  - Pajem sem gate, exceção ao limite, retrain 1 semana
 *  - Título L10, veterano, prereq Autoridade Feudal + terras
 *  - Caminho Montaria L5 unlock (veterano), mestre L11, requer compra
 */

describe('level gate constants', () => {
  it('Autoridade Feudal = 6', () => {
    expect(AUTORIDADE_FEUDAL_MIN_LEVEL).toBe(6)
  })

  it('Título = 10', () => {
    expect(TITULO_MIN_LEVEL).toBe(10)
  })

  it('Caminho Montaria unlock = 5', () => {
    expect(MONTARIA_UNLOCK_LEVEL).toBe(5)
  })

  it('Caminho Montaria mestre = 11', () => {
    expect(MONTARIA_MESTRE_LEVEL).toBe(11)
  })
})

describe('CAVALEIRO_PARCEIRO_POWERS — cobertura', () => {
  it('5 poderes', () => {
    expect(CAVALEIRO_PARCEIRO_POWERS.length).toBe(5)
  })

  it('frozen', () => {
    expect(Object.isFrozen(CAVALEIRO_PARCEIRO_POWERS)).toBe(true)
  })

  it('IDs únicos', () => {
    const ids = CAVALEIRO_PARCEIRO_POWERS.map((p) => p.id)
    expect(new Set(ids).size).toBe(5)
  })

  it('todos com bookPage 54 ou 55', () => {
    for (const p of CAVALEIRO_PARCEIRO_POWERS) {
      expect([54, 55]).toContain(p.bookPage)
    }
  })
})

describe('Escudeiro', () => {
  it('sem level gate (minLevel 1)', () => {
    expect(cavaleiroParceiroPowerById('escudeiro')!.minLevel).toBe(1)
  })

  it('exceção ao limite de parceiros', () => {
    expect(cavaleiroParceiroPowerById('escudeiro')!.countsAgainstLimit).toBe(false)
  })

  it('grantedTier null (não é parceiro típico)', () => {
    expect(cavaleiroParceiroPowerById('escudeiro')!.grantedTier).toBeNull()
  })

  it('retrain 1 mês', () => {
    expect(cavaleiroParceiroPowerById('escudeiro')!.retrain).toEqual({
      unit: 'mes',
      count: 1,
    })
  })

  it('sem compra prévia', () => {
    expect(cavaleiroParceiroPowerById('escudeiro')!.requiresPurchase).toBe(false)
  })
})

describe('Pajem', () => {
  it('sem level gate', () => {
    expect(cavaleiroParceiroPowerById('pajem')!.minLevel).toBe(1)
  })

  it('exceção ao limite de parceiros', () => {
    expect(cavaleiroParceiroPowerById('pajem')!.countsAgainstLimit).toBe(false)
  })

  it('retrain 1 semana', () => {
    expect(cavaleiroParceiroPowerById('pajem')!.retrain).toEqual({
      unit: 'semana',
      count: 1,
    })
  })
})

describe('Autoridade Feudal', () => {
  it('L6 mínimo', () => {
    expect(cavaleiroParceiroPowerById('autoridade-feudal')!.minLevel).toBe(6)
  })

  it('concede parceiro iniciante', () => {
    expect(cavaleiroParceiroPowerById('autoridade-feudal')!.grantedTier).toBe(
      'iniciante',
    )
  })

  it('conta contra limite (padrão)', () => {
    expect(
      cavaleiroParceiroPowerById('autoridade-feudal')!.countsAgainstLimit,
    ).toBe(true)
  })

  it('sem retrain (parceiro por aventura)', () => {
    expect(cavaleiroParceiroPowerById('autoridade-feudal')!.retrain).toBeNull()
  })
})

describe('Título', () => {
  it('L10 mínimo', () => {
    expect(cavaleiroParceiroPowerById('titulo')!.minLevel).toBe(10)
  })

  it('concede parceiro veterano', () => {
    expect(cavaleiroParceiroPowerById('titulo')!.grantedTier).toBe('veterano')
  })

  it('conta contra limite', () => {
    expect(cavaleiroParceiroPowerById('titulo')!.countsAgainstLimit).toBe(true)
  })

  it('prereq inclui Autoridade Feudal + terras/suserano', () => {
    const p = cavaleiroParceiroPowerById('titulo')!
    expect(p.additionalPrereqs).toContain('Autoridade Feudal')
    expect(p.additionalPrereqs.length).toBe(2)
  })
})

describe('Caminho da Montaria', () => {
  it('L5 unlock', () => {
    expect(cavaleiroParceiroPowerById('caminho-montaria')!.minLevel).toBe(5)
  })

  it('concede parceiro veterano baseline', () => {
    expect(cavaleiroParceiroPowerById('caminho-montaria')!.grantedTier).toBe(
      'veterano',
    )
  })

  it('requer compra prévia', () => {
    expect(
      cavaleiroParceiroPowerById('caminho-montaria')!.requiresPurchase,
    ).toBe(true)
  })

  it('retrain 1 semana', () => {
    expect(cavaleiroParceiroPowerById('caminho-montaria')!.retrain).toEqual({
      unit: 'semana',
      count: 1,
    })
  })
})

describe('unlockedCavaleiroParceiroPowers', () => {
  it('L1: Escudeiro + Pajem apenas', () => {
    const ids = unlockedCavaleiroParceiroPowers(1).map((p) => p.id).sort()
    expect(ids).toEqual(['escudeiro', 'pajem'])
  })

  it('L5: +Caminho Montaria', () => {
    const ids = unlockedCavaleiroParceiroPowers(5).map((p) => p.id).sort()
    expect(ids).toEqual(['caminho-montaria', 'escudeiro', 'pajem'])
  })

  it('L6: +Autoridade Feudal', () => {
    const ids = unlockedCavaleiroParceiroPowers(6).map((p) => p.id).sort()
    expect(ids).toEqual([
      'autoridade-feudal',
      'caminho-montaria',
      'escudeiro',
      'pajem',
    ])
  })

  it('L10: todos os 5', () => {
    expect(unlockedCavaleiroParceiroPowers(10).length).toBe(5)
  })

  it('L20: todos os 5', () => {
    expect(unlockedCavaleiroParceiroPowers(20).length).toBe(5)
  })

  it('throws se cavaleiroLevel < 1', () => {
    expect(() => unlockedCavaleiroParceiroPowers(0)).toThrow(/cavaleiroLevel/)
  })
})

describe('hasParceiroLimitException', () => {
  it('Escudeiro → true', () => {
    expect(hasParceiroLimitException('escudeiro')).toBe(true)
  })

  it('Pajem → true', () => {
    expect(hasParceiroLimitException('pajem')).toBe(true)
  })

  it('Autoridade Feudal → false', () => {
    expect(hasParceiroLimitException('autoridade-feudal')).toBe(false)
  })

  it('Título → false', () => {
    expect(hasParceiroLimitException('titulo')).toBe(false)
  })

  it('Caminho Montaria → false', () => {
    expect(hasParceiroLimitException('caminho-montaria')).toBe(false)
  })
})
