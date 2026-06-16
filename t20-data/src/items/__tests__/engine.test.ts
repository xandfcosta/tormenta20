import { describe, expect, it } from 'vitest'
import {
  type ActiveItem,
  applyActiveConditionals,
  computeItemEffects,
  conditionalId,
  statFor,
  targetKey,
} from '../engine'
import type { Modifier, ModifierTarget } from '../types'

/**
 * Items engine encodes T20 non-stacking rules (PDF Cap 6, plus general
 * rule on p150 sidebar). Pinned:
 *
 *  - Two modifiers with the same bonusType + target → keep the highest
 *    absolute amount; the other is suppressed.
 *  - 'untyped' modifiers stack freely with each other and with typed.
 *  - 'flag' modifiers don't bucket — they switch a boolean on/off.
 *  - Wielded modifiers only apply when item.equipped is 'wielded'/'wielded2'.
 *  - Vested modifiers only apply when item.equipped is 'vested'.
 *  - Conditional modifiers (terrain/against/context/flagOn) are deferred to
 *    the player's opt-in via `applyActiveConditionals`.
 */

const defenseTarget: ModifierTarget = { k: 'defense' }
const damageThisTarget: ModifierTarget = { k: 'damage', scope: 'this' }
const flagFatigue: ModifierTarget = { k: 'flag', name: 'fatigue-on-sleep' }

function vested(source: string, mods: Modifier[]): ActiveItem {
  return { source, equipped: 'vested', modifiers: mods }
}

function wielded(source: string, mods: Modifier[]): ActiveItem {
  return { source, equipped: 'wielded', modifiers: mods }
}

describe('targetKey — stable identity per target shape', () => {
  it('distinguishes attack scopes', () => {
    expect(targetKey({ k: 'attack', scope: 'this' })).not.toBe(
      targetKey({ k: 'attack', scope: 'all' }),
    )
  })

  it('distinguishes flags by name', () => {
    expect(targetKey({ k: 'flag', name: 'fatigue-on-sleep' })).toBe(
      'flag:fatigue-on-sleep',
    )
  })

  it('returns the same key for same shape', () => {
    expect(targetKey({ k: 'defense' })).toBe(targetKey({ k: 'defense' }))
  })
})

describe('computeItemEffects — non-stacking by bonusType', () => {
  it('two "armor"-typed defense modifiers keep only the highest', () => {
    const effects = computeItemEffects([
      vested('cota-malha', [
        {
          target: defenseTarget,
          amount: 6,
          bonusType: 'armor',
          condition: { c: 'vested' },
        },
      ]),
      // Hypothetical second armor source — shouldn't stack.
      vested('escudo-pesado', [
        {
          target: defenseTarget,
          amount: 2,
          bonusType: 'armor',
          condition: { c: 'vested' },
        },
      ]),
    ])
    const def = statFor(effects, defenseTarget)
    expect(def.total).toBe(6)
    expect(def.contributions).toHaveLength(1)
  })

  it('two "untyped" modifiers stack freely', () => {
    const effects = computeItemEffects([
      wielded('a', [
        {
          target: damageThisTarget,
          amount: 1,
          bonusType: 'untyped',
          condition: { c: 'wielded' },
        },
      ]),
      wielded('b', [
        {
          target: damageThisTarget,
          amount: 2,
          bonusType: 'untyped',
          condition: { c: 'wielded' },
        },
      ]),
    ])
    const dmg = statFor(effects, damageThisTarget)
    expect(dmg.total).toBe(3)
    expect(dmg.contributions).toHaveLength(2)
  })

  it('untyped stacks on top of typed', () => {
    const effects = computeItemEffects([
      vested('armor', [
        {
          target: defenseTarget,
          amount: 5,
          bonusType: 'armor',
          condition: { c: 'vested' },
        },
      ]),
      vested('untyped-bonus', [
        {
          target: defenseTarget,
          amount: 1,
          bonusType: 'untyped',
          condition: { c: 'vested' },
        },
      ]),
    ])
    expect(statFor(effects, defenseTarget).total).toBe(6)
  })

  it('keeps the most-negative entry for stacked negatives (penalty resolution)', () => {
    const target: ModifierTarget = { k: 'armorPenalty' }
    const effects = computeItemEffects([
      vested('couro-batido', [
        {
          target,
          amount: -1,
          bonusType: 'armor',
          condition: { c: 'vested' },
        },
      ]),
      vested('cota-malha', [
        {
          target,
          amount: -2,
          bonusType: 'armor',
          condition: { c: 'vested' },
        },
      ]),
    ])
    expect(statFor(effects, target).total).toBe(-2)
  })
})

describe('computeItemEffects — equip gating', () => {
  it('ignores wielded modifiers when item is vested only', () => {
    const effects = computeItemEffects([
      vested('weapon-on-belt', [
        {
          target: damageThisTarget,
          amount: 1,
          bonusType: 'untyped',
          condition: { c: 'wielded' },
        },
      ]),
    ])
    expect(statFor(effects, damageThisTarget).total).toBe(0)
  })

  it('ignores items with equipped=null', () => {
    const effects = computeItemEffects([
      {
        source: 'pack',
        equipped: null,
        modifiers: [
          {
            target: defenseTarget,
            amount: 5,
            bonusType: 'armor',
            condition: { c: 'vested' },
          },
        ],
      },
    ])
    expect(statFor(effects, defenseTarget).total).toBe(0)
  })

  it('wielded2 counts as wielded for condition matching', () => {
    const effects = computeItemEffects([
      {
        source: 'off-hand',
        equipped: 'wielded2',
        modifiers: [
          {
            target: damageThisTarget,
            amount: 1,
            bonusType: 'untyped',
            condition: { c: 'wielded' },
          },
        ],
      },
    ])
    expect(statFor(effects, damageThisTarget).total).toBe(1)
  })
})

describe('computeItemEffects — flags', () => {
  it('flag modifiers populate the flags set, not byTarget', () => {
    const effects = computeItemEffects([
      vested('cota-malha', [
        {
          target: flagFatigue,
          amount: 1,
          bonusType: 'untyped',
          condition: { c: 'vested' },
        },
      ]),
    ])
    expect(effects.flags.has('fatigue-on-sleep')).toBe(true)
    expect(effects.byTarget['flag:fatigue-on-sleep']).toBeUndefined()
  })
})

describe('computeItemEffects — conditional opt-ins', () => {
  it('"against" condition is deferred to conditional list', () => {
    const effects = computeItemEffects([
      wielded('material-aco-rubi', [
        {
          target: damageThisTarget,
          amount: 2,
          bonusType: 'enhancement',
          condition: { c: 'against', trait: 'vivos' },
          note: '+2 dano vs vivos',
        },
      ]),
    ])
    expect(effects.conditional).toHaveLength(1)
    expect(statFor(effects, damageThisTarget).total).toBe(0)
  })

  it('"terrain" condition is deferred to conditional list', () => {
    const effects = computeItemEffects([
      vested('explorador-boots', [
        {
          target: { k: 'expertise', name: 'Sobrevivência' },
          amount: 2,
          bonusType: 'item',
          condition: { c: 'terrain', type: 'floresta' },
        },
      ]),
    ])
    expect(effects.conditional).toHaveLength(1)
    expect(effects.conditional[0]?.note).toBe('terreno: floresta')
  })
})

describe('applyActiveConditionals', () => {
  it('folds active conditional ids back into byTarget', () => {
    const base = computeItemEffects([
      wielded('material-aco-rubi', [
        {
          target: damageThisTarget,
          amount: 2,
          bonusType: 'enhancement',
          condition: { c: 'against', trait: 'vivos' },
          note: '+2 dano vs vivos',
        },
      ]),
    ])
    const id = conditionalId(base.conditional[0]!)
    const next = applyActiveConditionals(base, new Set([id]))
    expect(statFor(next, damageThisTarget).total).toBe(2)
    expect(next.conditional).toHaveLength(0)
  })

  it('returns the original effects when no ids are active', () => {
    const base = computeItemEffects([
      wielded('material-aco-rubi', [
        {
          target: damageThisTarget,
          amount: 2,
          bonusType: 'enhancement',
          condition: { c: 'against', trait: 'vivos' },
        },
      ]),
    ])
    const next = applyActiveConditionals(base, new Set())
    expect(next).toBe(base)
  })

  it('leaves unmatched conditionals in the remaining list', () => {
    const base = computeItemEffects([
      wielded('a', [
        {
          target: damageThisTarget,
          amount: 2,
          bonusType: 'enhancement',
          condition: { c: 'against', trait: 'vivos' },
        },
        {
          target: damageThisTarget,
          amount: 1,
          bonusType: 'untyped',
          condition: { c: 'terrain', type: 'urbano' },
        },
      ]),
    ])
    const firstId = conditionalId(base.conditional[0]!)
    const next = applyActiveConditionals(base, new Set([firstId]))
    expect(next.conditional).toHaveLength(1)
  })
})

describe('statFor', () => {
  it('returns zeroed default for absent target', () => {
    const effects = computeItemEffects([])
    const result = statFor(effects, defenseTarget)
    expect(result.total).toBe(0)
    expect(result.contributions).toEqual([])
  })
})

describe('conditionalId', () => {
  it('is stable across calls for the same effect shape', () => {
    const effect = {
      source: 'a',
      bonusType: 'untyped' as const,
      amount: 2,
      note: 'n',
      target: defenseTarget,
    }
    expect(conditionalId(effect)).toBe(conditionalId(effect))
  })

  it('differs when source differs', () => {
    const a = {
      source: 'a',
      bonusType: 'untyped' as const,
      amount: 2,
      note: 'n',
      target: defenseTarget,
    }
    const b = { ...a, source: 'b' }
    expect(conditionalId(a)).not.toBe(conditionalId(b))
  })
})
