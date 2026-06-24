import { describe, expect, it } from 'vitest'
import { raceModifiers } from '../catalog'
import { originModifiers } from '../origins'
import type {
  OriginBenefit,
  OriginDefinition,
  RaceAbility,
  RaceDefinition,
} from '../types'

/**
 * `raceModifiers` and `originModifiers` are the projection layer between
 * the catalog (which stores race/origin definitions) and the items engine
 * (which expects a flat `Modifier[]`). They run for every character on
 * every derived recalc, so any drift here changes the sheet immediately.
 *
 * These specs use fabricated `RaceDefinition` / `OriginDefinition`
 * fixtures rather than walking the real catalog — keeping behavior
 * assertions independent of future catalog edits.
 *
 *  - `raceModifiers`:
 *      * attributeBonuses emit one `attribute` target per non-zero entry,
 *        with `race.name` as the modifier note
 *      * zero (or missing) bonuses do NOT emit
 *      * ability.modifiers are folded in directly
 *      * for a variant ability, only the chosen variant's modifiers are
 *        included (variantChoices set drives the pick)
 *      * when no variant is chosen, variant-only modifiers are excluded
 *      * abilities without modifiers contribute nothing
 *
 *  - `originModifiers`:
 *      * only benefits whose id is in `choiceSet` contribute
 *      * the poderUnico is reachable via choiceSet just like a normal benefit
 *      * benefits without modifiers (perícia-only) contribute nothing
 *      * empty choiceSet → `[]`
 */

function ability(over: Partial<RaceAbility> = {}): RaceAbility {
  return {
    id: 'ability-x',
    raceId: 'RaceX',
    name: 'Ability X',
    description: 'd',
    ...over,
  }
}

function race(over: Partial<RaceDefinition> = {}): RaceDefinition {
  return {
    id: 'RaceX',
    name: 'Raça X',
    attributeBonuses: {},
    abilities: [],
    ...over,
  }
}

function benefit(over: Partial<OriginBenefit> = {}): OriginBenefit {
  return {
    id: 'benefit-x',
    name: 'Benefit X',
    kind: 'poder',
    description: 'd',
    ...over,
  }
}

function origin(over: Partial<OriginDefinition> = {}): OriginDefinition {
  return {
    id: 'OriginX',
    name: 'Origem X',
    benefits: [],
    poderUnico: benefit({ id: 'poderUnico-x', name: 'Único' }),
    ...over,
  }
}

describe('raceModifiers — attributeBonuses projection', () => {
  it('emits one `attribute` target per non-zero entry', () => {
    const result = raceModifiers(
      race({
        attributeBonuses: { strength: 2, wisdom: 1, dexterity: -1 },
      }),
      new Set(),
    )
    expect(result).toHaveLength(3)
    const byAttr = Object.fromEntries(
      result.map((m) => [
        m.target.k === 'attribute' ? m.target.name : 'none',
        m.amount,
      ]),
    )
    expect(byAttr).toEqual({ strength: 2, wisdom: 1, dexterity: -1 })
  })

  it('uses race.name as the modifier note', () => {
    const result = raceModifiers(
      race({ name: 'Minotauro', attributeBonuses: { strength: 2 } }),
      new Set(),
    )
    expect(result[0]!.note).toBe('Minotauro')
  })

  it('skips bonuses with value 0 (no zero-mod noise)', () => {
    const result = raceModifiers(
      race({
        attributeBonuses: { strength: 2, dexterity: 0, charisma: -1 },
      }),
      new Set(),
    )
    const attrs = result.map((m) =>
      m.target.k === 'attribute' ? m.target.name : null,
    )
    expect(attrs).toEqual(expect.arrayContaining(['strength', 'charisma']))
    expect(attrs).not.toContain('dexterity')
  })

  it('emits everything as bonusType=untyped (race modifiers stack with items)', () => {
    const result = raceModifiers(
      race({ attributeBonuses: { strength: 2 } }),
      new Set(),
    )
    expect(result[0]!.bonusType).toBe('untyped')
  })
})

describe('raceModifiers — abilities + variants', () => {
  it('folds ability.modifiers directly into the output', () => {
    const result = raceModifiers(
      race({
        abilities: [
          ability({
            modifiers: [
              {
                target: { k: 'defense' },
                amount: 1,
                bonusType: 'untyped',
              },
            ],
          }),
        ],
      }),
      new Set(),
    )
    expect(result).toHaveLength(1)
    expect(result[0]!.target).toEqual({ k: 'defense' })
  })

  it('includes only the chosen variant when a variant is picked', () => {
    const result = raceModifiers(
      race({
        abilities: [
          ability({
            variants: [
              {
                id: 'aggelus',
                name: 'Aggelus',
                description: 'd',
                modifiers: [
                  {
                    target: { k: 'attribute', name: 'wisdom' },
                    amount: 2,
                    bonusType: 'untyped',
                  },
                ],
              },
              {
                id: 'sulfure',
                name: 'Sulfure',
                description: 'd',
                modifiers: [
                  {
                    target: { k: 'attribute', name: 'dexterity' },
                    amount: 2,
                    bonusType: 'untyped',
                  },
                ],
              },
            ],
          }),
        ],
      }),
      new Set(['aggelus']),
    )
    expect(result).toHaveLength(1)
    expect(result[0]!.target).toEqual({ k: 'attribute', name: 'wisdom' })
  })

  it('excludes variant modifiers entirely when no variant is chosen', () => {
    const result = raceModifiers(
      race({
        abilities: [
          ability({
            variants: [
              {
                id: 'aggelus',
                name: 'Aggelus',
                description: 'd',
                modifiers: [
                  {
                    target: { k: 'attribute', name: 'wisdom' },
                    amount: 2,
                    bonusType: 'untyped',
                  },
                ],
              },
            ],
          }),
        ],
      }),
      new Set(),
    )
    expect(result).toEqual([])
  })

  it('combines ability.modifiers AND chosen-variant.modifiers when both present', () => {
    const result = raceModifiers(
      race({
        abilities: [
          ability({
            modifiers: [
              {
                target: { k: 'defense' },
                amount: 1,
                bonusType: 'untyped',
              },
            ],
            variants: [
              {
                id: 'v1',
                name: 'V1',
                description: 'd',
                modifiers: [
                  {
                    target: { k: 'attribute', name: 'strength' },
                    amount: 1,
                    bonusType: 'untyped',
                  },
                ],
              },
            ],
          }),
        ],
      }),
      new Set(['v1']),
    )
    expect(result).toHaveLength(2)
  })

  it('contributes nothing for abilities that have no modifiers at all', () => {
    const result = raceModifiers(
      race({
        abilities: [ability({ name: 'Lore only', description: 'd' })],
      }),
      new Set(),
    )
    expect(result).toEqual([])
  })
})

describe('originModifiers — choice gating', () => {
  it('includes only benefits whose id is in choiceSet', () => {
    const def = origin({
      benefits: [
        benefit({
          id: 'b1',
          modifiers: [
            {
              target: { k: 'defense' },
              amount: 1,
              bonusType: 'untyped',
            },
          ],
        }),
        benefit({
          id: 'b2',
          modifiers: [
            {
              target: { k: 'defense' },
              amount: 5,
              bonusType: 'untyped',
            },
          ],
        }),
      ],
    })
    const result = originModifiers(def, new Set(['b1']))
    expect(result).toHaveLength(1)
    expect(result[0]!.amount).toBe(1)
  })

  it('returns [] when choiceSet is empty', () => {
    const def = origin({
      benefits: [
        benefit({
          id: 'b1',
          modifiers: [
            { target: { k: 'defense' }, amount: 1, bonusType: 'untyped' },
          ],
        }),
      ],
    })
    expect(originModifiers(def, new Set())).toEqual([])
  })

  it('contributes the poderUnico modifiers when its id is in choiceSet', () => {
    const def = origin({
      benefits: [],
      poderUnico: benefit({
        id: 'unico-1',
        kind: 'poder',
        modifiers: [
          {
            target: { k: 'attribute', name: 'charisma' },
            amount: 2,
            bonusType: 'untyped',
          },
        ],
      }),
    })
    const result = originModifiers(def, new Set(['unico-1']))
    expect(result).toHaveLength(1)
    expect(result[0]!.amount).toBe(2)
  })

  it('ignores chosen benefits that have no modifiers (perícia-only)', () => {
    const def = origin({
      benefits: [
        benefit({
          id: 'pericia-1',
          kind: 'pericia',
          expertise: 'Atletismo',
        }),
        benefit({
          id: 'poder-1',
          kind: 'poder',
          modifiers: [
            {
              target: { k: 'defense' },
              amount: 1,
              bonusType: 'untyped',
            },
          ],
        }),
      ],
    })
    const result = originModifiers(def, new Set(['pericia-1', 'poder-1']))
    expect(result).toHaveLength(1)
    expect(result[0]!.target).toEqual({ k: 'defense' })
  })

  it('preserves the order: benefits first, poderUnico last', () => {
    const def = origin({
      benefits: [
        benefit({
          id: 'b1',
          modifiers: [
            { target: { k: 'defense' }, amount: 1, bonusType: 'untyped' },
          ],
        }),
      ],
      poderUnico: benefit({
        id: 'u1',
        modifiers: [
          { target: { k: 'defense' }, amount: 2, bonusType: 'untyped' },
        ],
      }),
    })
    const result = originModifiers(def, new Set(['b1', 'u1']))
    expect(result.map((m) => m.amount)).toEqual([1, 2])
  })
})
