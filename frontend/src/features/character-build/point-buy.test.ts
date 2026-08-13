import type { AttributeKey } from '@/shared/api/attribute-keys'
import { describe, expect, it } from 'vitest'
import { pointBuyStatusFor } from './point-buy'

const attrs = (patch: Partial<Record<AttributeKey, number>> = {}) => ({
  strength: 0,
  dexterity: 0,
  constitution: 0,
  intelligence: 0,
  wisdom: 0,
  charisma: 0,
  ...patch,
})

describe('pointBuyStatusFor — pontos de atributo (p17)', () => {
  it('todos em 0 não custam nada', () => {
    expect(pointBuyStatusFor(attrs()).spent).toBe(0)
  })

  it('cobra a Tabela 1-1 (1/2/4/7) — a escalada começa no +3', () => {
    expect(pointBuyStatusFor(attrs({ strength: 1 })).spent).toBe(1)
    expect(pointBuyStatusFor(attrs({ strength: 2 })).spent).toBe(2)
    expect(pointBuyStatusFor(attrs({ strength: 3 })).spent).toBe(4)
    expect(pointBuyStatusFor(attrs({ strength: 4 })).spent).toBe(7)
  })

  it('reduzir um atributo a −1 devolve 1 ponto', () => {
    expect(pointBuyStatusFor(attrs({ charisma: -1 })).spent).toBe(-1)
  })

  it('estourar os 10 pontos vira aviso', () => {
    const { warnings } = pointBuyStatusFor(attrs({ strength: 4, dexterity: 4 }))

    expect(warnings.join(' ')).toContain('excedem o limite')
  })

  it('só UM atributo pode cair a −1 (p17)', () => {
    const { warnings } = pointBuyStatusFor(attrs({ charisma: -1, strength: -1 }))

    expect(warnings.join(' ')).toContain('apenas UM atributo')
  })

  it('valor fora da tabela → spent null, não exceção (modo livre)', () => {
    // Modo livre deixa passar valores que a compra por pontos não precifica; a
    // Forja mostra "—" em vez de derrubar o passo de atributos.
    expect(pointBuyStatusFor(attrs({ strength: 9 })).spent).toBeNull()
  })

  it('avisos são conselho, nunca bloqueio', () => {
    expect(Array.isArray(pointBuyStatusFor(attrs()).warnings)).toBe(true)
  })
})
