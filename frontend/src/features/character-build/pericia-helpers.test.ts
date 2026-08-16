import { describe, expect, it } from 'vitest'
import { periciaBudget, periciaPlan } from './pericia-helpers'

describe('periciaPlan — class band vs free (Int + raça) band', () => {
  it('keeps the base class count free of the INT bonus', () => {
    const plan = periciaPlan('Bárbaro', 3, [])
    // Bárbaro base chooseCount is 4; INT must NOT inflate it.
    expect(plan?.classCount).toBe(4)
    expect(plan?.freeCount).toBe(3)
    expect(plan?.intCount).toBe(3)
    expect(plan?.raceCount).toBe(0)
  })

  it('folds a non-positive INT mod to zero and still counts race grants', () => {
    const plan = periciaPlan('Bárbaro', -1, ['Humano'])
    expect(plan?.intCount).toBe(0)
    expect(plan?.raceCount).toBe(2) // Humano Versátil
    expect(plan?.freeCount).toBe(2)
  })

  it('sums INT + race into the free budget', () => {
    const plan = periciaPlan('Arcanista', 2, ['Kliren'])
    expect(plan?.intCount).toBe(2)
    expect(plan?.raceCount).toBe(1) // Kliren Híbrido
    expect(plan?.freeCount).toBe(3)
  })

  /**
   * Arcanista, transcrito do LIVRO: "Misticismo (Int) e Vontade (Sab), mais 2 a
   * sua escolha entre Conhecimento, Diplomacia, Enganação, Guerra, Iniciativa,
   * Intimidação, Intuição, Investigação, Nobreza, Ofício e Percepção".
   *
   * A asserção anterior era "a união do conjunto com o complemento dele é o
   * universo" — verdade por construção (`freePool` É `todas menos as da classe`),
   * e o mesmo valia para a disjunção. Nenhuma das duas podia falhar.
   */
  it('o plano do Arcanista é o do livro, e o bolo livre é o que sobra dele', () => {
    const plan = periciaPlan('Arcanista', 2, [])
    if (!plan) throw new Error('expected a plan for Arcanista')

    expect(plan.fixed).toEqual(['Misticismo', 'Vontade'])
    expect(plan.classCount).toBe(2)
    expect(plan.eitherOr).toBeNull()
    expect(plan.classPool).toEqual([
      'Conhecimento',
      'Diplomacia',
      'Enganação',
      'Guerra',
      'Iniciativa',
      'Intimidação',
      'Intuição',
      'Investigação',
      'Nobreza',
      'Ofício',
      'Percepção',
    ])
    // O bolo LIVRE é o resto da lista de perícias: uma que a classe não oferece
    // (Furtividade) está lá, e uma que ela oferece (Guerra) não está.
    expect(plan.freePool).toContain('Furtividade')
    expect(plan.freePool).not.toContain('Guerra')
    expect(plan.freePool).not.toContain('Misticismo')
  })

  it('returns null for an unknown class', () => {
    expect(periciaPlan('Hexer', 2, [])).toBeNull()
  })
})

describe('periciaBudget — shared free budget with class overflow', () => {
  const plan = () => {
    const p = periciaPlan('Arcanista', 1, []) // classCount 2, freeCount 1
    if (!p) throw new Error('no plan')
    return p
  }

  it('spends the class cap first, then the free budget on class overflow', () => {
    const p = plan()
    // 3 class-pool picks: 2 fill the class cap, the 3rd draws the free slot.
    const trained = p.classPool.slice(0, 3)
    const b = periciaBudget(p, trained)
    expect(b.classSpent).toBe(2)
    expect(b.freeSpent).toBe(1)
    expect(b.classRemaining).toBe(0)
    expect(b.freeRemaining).toBe(0)
    expect(b.classOverflow).toBe(true)
  })

  it('a free-pool pick consumes the free budget, not the class cap', () => {
    const p = plan()
    const trained = [p.freePool[0]]
    const b = periciaBudget(p, trained)
    expect(b.classSpent).toBe(0)
    expect(b.freeSpent).toBe(1)
    expect(b.freeRemaining).toBe(0)
    expect(b.classOverflow).toBe(false)
  })

  it('counts nothing when only fixed perícias are trained', () => {
    const p = plan()
    const b = periciaBudget(p, [...p.fixed])
    expect(b.classSpent).toBe(0)
    expect(b.freeSpent).toBe(0)
  })
})
