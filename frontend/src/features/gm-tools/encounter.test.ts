import { describe, expect, it } from 'vitest'
import { BESTIARY } from '@tormenta20/t20-data'
import { enrichEncounter, encounterDifficulty } from './encounter'

describe('enrichEncounter', () => {
  it('resolves real monster ids to groups with a computed group ND', () => {
    const first = BESTIARY[0]!
    const groups = enrichEncounter(
      [{ monsterId: first.id, quantity: 3 }],
      BESTIARY,
    )
    expect(groups).toHaveLength(1)
    expect(groups[0]!.monster.id).toBe(first.id)
    expect(groups[0]!.quantity).toBe(3)
    expect(typeof groups[0]!.groupNd).toBe('number')
  })

  it('drops entries whose monster no longer exists', () => {
    expect(
      enrichEncounter([{ monsterId: 'does-not-exist', quantity: 1 }], BESTIARY),
    ).toEqual([])
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

  // Regressão ALE-25: gaps fracionários (monstros ND < 1, escalonamento log2)
  // arredondam ao passo de ND mais próximo. Antes, um gap pequeno e negativo
  // caía direto em "Difícil".
  it('a small negative gap é fácil, não difícil (1 monstro ND 1/4 vs grupo Nv 1)', () => {
    // encounterNd 0.25 − partyLevel 1 = −0.75 → arredonda p/ −1 → Fácil
    expect(encounterDifficulty(-0.75).label).toBe('Fácil')
  })

  it('a small positive gap é médio (5 monstros ND 1/4 ≈ ND 1.25 vs grupo Nv 1)', () => {
    // 1.25 − 1 = 0.25 → arredonda p/ 0 → Médio
    expect(encounterDifficulty(0.25).label).toBe('Médio')
  })

  it('arredonda para cima nos limites (gap 1.5 → Difícil, −1.5 → Fácil)', () => {
    expect(encounterDifficulty(1.5).label).toBe('Difícil')
    expect(encounterDifficulty(-1.5).label).toBe('Fácil')
  })
})
