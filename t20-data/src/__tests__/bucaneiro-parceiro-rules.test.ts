import { describe, expect, it } from 'vitest'
import {
  AMIGOS_NO_PORTO_ACTIVATION_CD,
  AMIGOS_NO_PORTO_MIN_LEVEL,
  BUCANEIRO_PARCEIRO_POWERS,
  bucaneiroParceiroPowerById,
  unlockedBucaneiroParceiroPowers,
} from '../bucaneiro-parceiro-rules'

/**
 * PDF Bucaneiro p47 "Amigos no Porto". Pinned:
 *  - L6, veterano (não iniciante), 1 dia (não aventura)
 *  - CD 10 Carisma para ativar; prereq Car 1 + comunidade portuária
 */

describe('constantes', () => {
  it('L6 mínimo', () => {
    expect(AMIGOS_NO_PORTO_MIN_LEVEL).toBe(6)
  })

  it('CD ativação 10', () => {
    expect(AMIGOS_NO_PORTO_ACTIVATION_CD).toBe(10)
  })
})

describe('BUCANEIRO_PARCEIRO_POWERS', () => {
  it('1 poder (Amigos no Porto)', () => {
    expect(BUCANEIRO_PARCEIRO_POWERS.length).toBe(1)
  })

  it('frozen', () => {
    expect(Object.isFrozen(BUCANEIRO_PARCEIRO_POWERS)).toBe(true)
  })
})

describe('Amigos no Porto', () => {
  const p = () => bucaneiroParceiroPowerById('amigos-no-porto')!

  it('L6', () => {
    expect(p().minLevel).toBe(6)
  })

  it('parceiro veterano', () => {
    expect(p().grantedTier).toBe('veterano')
  })

  it('duração 1 dia', () => {
    expect(p().duration).toBe('dia')
  })

  it('bookPage 47', () => {
    expect(p().bookPage).toBe(47)
  })

  it('conta contra limite', () => {
    expect(p().countsAgainstLimit).toBe(true)
  })

  it('teste Carisma CD 10', () => {
    expect(p().activationCd).toBe(10)
    expect(p().activationSkill).toBe('Carisma')
  })

  it('prereqs: Car 1 + comunidade portuária', () => {
    expect(p().additionalPrereqs).toEqual(['Car 1', 'comunidade portuária'])
  })
})

describe('unlockedBucaneiroParceiroPowers', () => {
  it('L1-5: vazio', () => {
    expect(unlockedBucaneiroParceiroPowers(1)).toEqual([])
    expect(unlockedBucaneiroParceiroPowers(5)).toEqual([])
  })

  it('L6: Amigos no Porto', () => {
    expect(unlockedBucaneiroParceiroPowers(6).map((p) => p.id)).toEqual([
      'amigos-no-porto',
    ])
  })

  it('L20: Amigos no Porto', () => {
    expect(unlockedBucaneiroParceiroPowers(20).length).toBe(1)
  })

  it('throws se bucaneiroLevel < 1', () => {
    expect(() => unlockedBucaneiroParceiroPowers(0)).toThrow(/bucaneiroLevel/)
  })
})
