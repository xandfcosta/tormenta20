import type { CatalogSpell } from '@tormenta20/t20-data'
import { describe, expect, it } from 'vitest'
import { augmentPicksFrom, augmentPmFor, isAugmentLocked } from './spell-augments'

const augments: CatalogSpell['augments'] = [
  { pmCost: 2, kind: 'aumenta', description: 'mais dano' },
  { pmCost: 5, kind: 'muda', description: 'vira gelo' },
  { pmCost: 3, kind: 'aumenta', description: 'só no 3º', requiresCircle: 3 },
]

describe('augmentPicksFrom', () => {
  it('vira uma lista de escolhas, ignorando o que está em zero', () => {
    const picks = augmentPicksFrom(new Map([[0, 2], [1, 0]]))

    expect(picks).toEqual([{ augmentIndex: 0, stacks: 2 }])
  })

  it('sem escolha nenhuma, lista vazia', () => {
    expect(augmentPicksFrom(new Map())).toEqual([])
  })

  // O servidor recusa índice duplicado com 400 — a fonte é um Map, então cada
  // aprimoramento só pode sair uma vez.
  it('cada aprimoramento aparece uma vez só', () => {
    const picks = augmentPicksFrom(new Map([[0, 1], [2, 3]]))

    expect(picks.map((p) => p.augmentIndex)).toEqual([0, 2])
  })
})

describe('augmentPmFor', () => {
  it('multiplica o custo pelos degraus escolhidos', () => {
    expect(augmentPmFor(augments, [{ augmentIndex: 0, stacks: 3 }])).toBe(6)
  })

  it('soma aprimoramentos diferentes', () => {
    const picks = [
      { augmentIndex: 0, stacks: 1 },
      { augmentIndex: 1, stacks: 1 },
    ]

    expect(augmentPmFor(augments, picks)).toBe(7)
  })

  it('índice fora da lista não derruba a conta', () => {
    expect(augmentPmFor(augments, [{ augmentIndex: 99, stacks: 1 }])).toBe(0)
  })
})

describe('isAugmentLocked', () => {
  // p42/p171: sem acesso ao círculo, sem o aprimoramento dele — é o caso do
  // Bárbaro com Totem, que conjura a magia concedida e nada além dela.
  it('trava o aprimoramento acima do círculo alcançável', () => {
    expect(isAugmentLocked(augments[2], 2)).toBe(true)
    expect(isAugmentLocked(augments[2], 3)).toBe(false)
  })

  it('aprimoramento sem exigência de círculo nunca trava', () => {
    expect(isAugmentLocked(augments[0], 0)).toBe(false)
  })
})
