import { describe, expect, it } from 'vitest'
import {
  NOBRE_AUTORIDADE_FEUDAL_MIN_LEVEL,
  NOBRE_PARCEIRO_POWERS,
  NOBRE_TITULO_MIN_LEVEL,
  nobreParceiroPowerById,
  unlockedNobreParceiroPowers,
} from '../nobre-parceiro-rules'

/**
 * PDF Nobre p79-80. Pinned:
 *  - 2 poderes: Autoridade Feudal (L6, iniciante) + Título (sem gate, veterano)
 *  - Ambos contam contra limite
 *  - Título Nobre SEM prereqs (assimetria vs Cavaleiro)
 */

describe('constantes', () => {
  it('Autoridade Feudal Nobre = L6', () => {
    expect(NOBRE_AUTORIDADE_FEUDAL_MIN_LEVEL).toBe(6)
  })

  it('Título Nobre sem gate (L1)', () => {
    expect(NOBRE_TITULO_MIN_LEVEL).toBe(1)
  })
})

describe('NOBRE_PARCEIRO_POWERS', () => {
  it('2 poderes', () => {
    expect(NOBRE_PARCEIRO_POWERS.length).toBe(2)
  })

  it('frozen', () => {
    expect(Object.isFrozen(NOBRE_PARCEIRO_POWERS)).toBe(true)
  })

  it('IDs únicos', () => {
    const ids = NOBRE_PARCEIRO_POWERS.map((p) => p.id)
    expect(new Set(ids).size).toBe(2)
  })
})

describe('Autoridade Feudal (Nobre)', () => {
  const p = () => nobreParceiroPowerById('autoridade-feudal')!

  it('L6', () => {
    expect(p().minLevel).toBe(6)
  })

  it('parceiro iniciante', () => {
    expect(p().grantedTier).toBe('iniciante')
  })

  it('duração até fim da aventura', () => {
    expect(p().duration).toBe('ate-fim-aventura')
  })

  it('custo 2 PM', () => {
    expect(p().pmCost).toBe(2)
  })

  it('bookPage 79', () => {
    expect(p().bookPage).toBe(79)
  })

  it('conta contra limite', () => {
    expect(p().countsAgainstLimit).toBe(true)
  })

  it('prereq inclui influência local', () => {
    expect(p().additionalPrereqs[0]).toMatch(/influência/)
  })
})

describe('Título (Nobre)', () => {
  const p = () => nobreParceiroPowerById('titulo')!

  it('sem prereq de nível (L1)', () => {
    expect(p().minLevel).toBe(1)
  })

  it('parceiro veterano (membro da corte)', () => {
    expect(p().grantedTier).toBe('veterano')
  })

  it('duração início-aventura', () => {
    expect(p().duration).toBe('inicio-aventura')
  })

  it('sem PM cost', () => {
    expect(p().pmCost).toBeNull()
  })

  it('bookPage 80', () => {
    expect(p().bookPage).toBe(80)
  })

  it('sem prereqs adicionais (assimetria vs Cavaleiro Título)', () => {
    expect(p().additionalPrereqs.length).toBe(0)
  })
})

describe('unlockedNobreParceiroPowers', () => {
  it('L1: apenas Título', () => {
    const ids = unlockedNobreParceiroPowers(1).map((p) => p.id)
    expect(ids).toEqual(['titulo'])
  })

  it('L5: apenas Título', () => {
    expect(unlockedNobreParceiroPowers(5).map((p) => p.id)).toEqual(['titulo'])
  })

  it('L6: Autoridade Feudal + Título', () => {
    const ids = unlockedNobreParceiroPowers(6).map((p) => p.id).sort()
    expect(ids).toEqual(['autoridade-feudal', 'titulo'])
  })

  it('L20: ambos', () => {
    expect(unlockedNobreParceiroPowers(20).length).toBe(2)
  })

  it('throws se nobreLevel < 1', () => {
    expect(() => unlockedNobreParceiroPowers(0)).toThrow(/nobreLevel/)
  })
})
