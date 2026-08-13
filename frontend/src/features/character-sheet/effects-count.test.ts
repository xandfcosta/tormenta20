import type { ConditionalEffect } from '@/shared/api/item-types'
import { describe, expect, it } from 'vitest'
import type { ConditionalEntry } from '@/entities/character/derived'
import type { ActiveEffect, Character } from '@/shared/api/api'
import { effectsShownCount } from './effects-count'

function entry(id: string, flag: string | undefined, active: boolean): ConditionalEntry {
  return {
    id,
    active,
    effect: {
      source: 'x',
      bonusType: 'item',
      amount: 1,
      note: 'n',
      target: { k: 'defense' },
      flag,
    } as ConditionalEffect,
  }
}

function character(overrides: Partial<Character> = {}): Character {
  return { activeConditions: '[]', activeEffects: [], ...overrides } as Character
}

const buff: ActiveEffect = {
  id: 1,
  catalogId: 'pocao',
  scope: 'scene',
  modifiers: '[]',
  createdAt: '2026-08-11',
}

describe('effectsShownCount', () => {
  it('conta condição do livro, efeito ativo e condicional ligado', () => {
    const char = character({ activeConditions: '["caido"]', activeEffects: [buff] })

    expect(effectsShownCount(char, [entry('solto', undefined, true)])).toBe(3)
  })

  it('condicional desligado não conta', () => {
    expect(effectsShownCount(character(), [entry('solto', undefined, false)])).toBe(0)
  })

  // A Fúria entra no motor como 8 modificadores de degrau; contá-los cru dava
  // "11" para uma Fúria sozinha com dois toggles.
  it('as entradas de uma mesma flag contam como UMA coisa', () => {
    const furia = [
      entry('furia:1', 'furia', true),
      entry('furia:2', 'furia', true),
      entry('furia:3', 'furia', true),
    ]

    expect(effectsShownCount(character(), furia)).toBe(1)
  })

  it('grupo homebrew ligado também conta uma vez', () => {
    const grupo = [entry('hb:1', 'homebrew-1', true), entry('hb:2', 'homebrew-1', true)]

    expect(effectsShownCount(character(), grupo)).toBe(1)
  })

  it('condição desconhecida no blob não infla a conta', () => {
    const char = character({ activeConditions: '["nao-existe"]' })

    expect(effectsShownCount(char, [])).toBe(0)
  })
})
