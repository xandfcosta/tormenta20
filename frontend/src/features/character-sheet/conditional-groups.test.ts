import type { ConditionalEffect } from '@tormenta20/t20-data'
import { describe, expect, it } from 'vitest'
import type { ConditionalEntry } from '@/entities/character/derived'
import { groupConditionals, situationalGroups } from './conditional-groups'

function entry(id: string, effect: Partial<ConditionalEffect> = {}, active = false): ConditionalEntry {
  return {
    id,
    active,
    effect: {
      source: 'Espada Rúnica',
      bonusType: 'item',
      amount: 2,
      note: 'em terreno acidentado',
      target: { k: 'defense' },
      ...effect,
    } as ConditionalEffect,
  }
}

describe('groupConditionals', () => {
  it('sem flag, cada entrada é uma linha', () => {
    const groups = groupConditionals([entry('a'), entry('b')])

    expect(groups).toHaveLength(2)
    expect(groups.every((g) => g.kind === 'single')).toBe(true)
  })

  // O grupo homebrew de vários modificadores tem de ligar junto: duas linhas
  // separadas deixariam metade do efeito ligada.
  it('entradas com a mesma flag viram um interruptor só', () => {
    const groups = groupConditionals([
      entry('a', { flag: 'homebrew-1', note: 'Modo brutal' }),
      entry('b', { flag: 'homebrew-1' }),
      entry('c'),
    ])

    const flagGroup = groups.find((g) => g.kind === 'flag')
    expect(flagGroup?.kind === 'flag' && flagGroup.entries).toHaveLength(2)
    expect(flagGroup?.kind === 'flag' && flagGroup.label).toBe('Modo brutal')
    expect(groups.filter((g) => g.kind === 'single')).toHaveLength(1)
  })

  it('flags diferentes não se misturam', () => {
    const groups = groupConditionals([
      entry('a', { flag: 'homebrew-1' }),
      entry('b', { flag: 'homebrew-2' }),
    ])

    expect(groups).toHaveLength(2)
  })

  it('nenhuma entrada some no agrupamento', () => {
    const entries = [entry('a', { flag: 'f' }), entry('b', { flag: 'f' }), entry('c')]

    const shown = groupConditionals(entries).flatMap((g) =>
      g.kind === 'single' ? [g.entry] : g.entries,
    )

    expect(shown.map((e) => e.id).sort()).toEqual(['a', 'b', 'c'])
  })
})

describe('situationalGroups', () => {
  // O interruptor das stances de poder mora nos Poderes (ALE-87); aqui elas
  // sumiriam da aba Situação, mas os toggles homebrew ficam.
  it('tira as stances de poder e mantém os grupos homebrew', () => {
    const groups = situationalGroups([
      entry('furia:1', { flag: 'furia' }),
      entry('hb:1', { flag: 'homebrew-1' }),
      entry('solto'),
    ])

    const flags = groups.map((g) => (g.kind === 'flag' ? g.flag : g.entry.id))
    expect(flags).not.toContain('furia')
    expect(flags).toContain('homebrew-1')
    expect(flags).toContain('solto')
  })
})
