import { describe, expect, it } from 'vitest'
import { BESTIARY } from '@tormenta20/t20-data'
import { enrichEncounter, encounterDifficulty } from './encounter'

describe('enrichEncounter', () => {
  it('resolves real monster ids to groups with a computed group ND', () => {
    const first = BESTIARY[0]!
    const groups = enrichEncounter([{ monsterId: first.id, quantity: 3 }])
    expect(groups).toHaveLength(1)
    expect(groups[0]!.monster.id).toBe(first.id)
    expect(groups[0]!.quantity).toBe(3)
    expect(typeof groups[0]!.groupNd).toBe('number')
  })

  it('drops entries whose monster no longer exists', () => {
    expect(enrichEncounter([{ monsterId: 'does-not-exist', quantity: 1 }])).toEqual([])
  })
})

describe('encounterDifficulty', () => {
  it('bands the party-vs-encounter ND gap', () => {
    expect(encounterDifficulty(-4).label).toBe('Trivial')
    expect(encounterDifficulty(-1).label).toBe('Fácil')
    expect(encounterDifficulty(0).label).toBe('Médio')
    expect(encounterDifficulty(2).label).toBe('Difícil')
    expect(encounterDifficulty(5).label).toBe('Mortal')
  })
})
