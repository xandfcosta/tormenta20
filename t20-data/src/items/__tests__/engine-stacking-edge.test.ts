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
 * Sibling spec to `engine.test.ts` — pins behaviors the original file
 * leaves implicit, so a refactor to the stacking resolver or the
 * conditional fold cannot silently change semantics.
 *
 * Gaps filled vs `engine.test.ts`:
 *
 *   - `targetKey` covers every branch of the ModifierTarget union (not
 *     just attack-scope + flag). Catches a future case that forgets to
 *     add a switch arm.
 *   - `condition: { c: 'always' }` applies regardless of equip state —
 *     including `'vested'` items emitting `'always'` modifiers.
 *   - `'vested'` condition is suppressed when the item is wielded
 *     (and vice versa) — symmetric counter to the existing wielded-on-vested
 *     test.
 *   - Three modifiers of the same `bonusType` keep only the highest-abs,
 *     dropping both losers — the existing test only had two entries.
 *   - Two different typed `bonusType`s on the same target both survive
 *     (sum). T20 only forbids stacking *within* a type bucket.
 *   - `'context'` condition routes to `conditional` with `note = m.condition.note`,
 *     and `'flagOn'` carries the `flag` prop onto the ConditionalEffect.
 *   - `applyActiveConditionals` re-runs non-stacking after the fold:
 *     a conditional `enhancement +3` displaces a base `enhancement +1`
 *     on the same target.
 *   - `applyActiveConditionals` ignores `flag`-target conditionals (the
 *     engine deliberately leaves no UI for opt-in flags yet).
 *   - `conditionalId` differs by amount, target shape, and bonusType
 *     (existing only tested `source` divergence).
 *   - `equipped: null` suppresses even an `'always'` modifier — the
 *     top-level equip gate dominates.
 *   - Empty input produces a wholly-empty `ItemEffects`.
 */

const defenseTarget: ModifierTarget = { k: 'defense' }
const damageThisTarget: ModifierTarget = { k: 'damage', scope: 'this' }

function vested(source: string, mods: Modifier[]): ActiveItem {
  return { source, equipped: 'vested', modifiers: mods }
}

function wielded(source: string, mods: Modifier[]): ActiveItem {
  return { source, equipped: 'wielded', modifiers: mods }
}

describe('targetKey — full union coverage', () => {
  it.each<[ModifierTarget, string]>([
    [{ k: 'expertise', name: 'Atletismo' }, 'expertise:Atletismo'],
    [{ k: 'expertiseAll' }, 'expertiseAll'],
    [
      { k: 'expertiseRemovePenalty', name: 'Furtividade' },
      'expertiseRemovePenalty:Furtividade',
    ],
    [
      { k: 'expertiseByAttribute', attribute: 'strength' },
      'expertiseByAttribute:strength',
    ],
    [{ k: 'attribute', name: 'wisdom' }, 'attribute:wisdom'],
    [{ k: 'defenseDexCap' }, 'defenseDexCap'],
    [{ k: 'resistance' }, 'resistance'],
    [{ k: 'fearResistance' }, 'fearResistance'],
    [{ k: 'damage', scope: 'all' }, 'damage:all'],
    [{ k: 'critRange' }, 'critRange'],
    [{ k: 'critMult' }, 'critMult'],
    [{ k: 'pmLimit' }, 'pmLimit'],
    [{ k: 'pmCost' }, 'pmCost'],
    [{ k: 'spellDC' }, 'spellDC'],
    [{ k: 'inventorySlots' }, 'inventorySlots'],
    [{ k: 'displacement' }, 'displacement'],
    [{ k: 'armorPenalty' }, 'armorPenalty'],
    [{ k: 'armorPenaltyExpertises' }, 'armorPenaltyExpertises'],
    [{ k: 'tempHp' }, 'tempHp'],
    [{ k: 'tempMp' }, 'tempMp'],
    [{ k: 'maneuver', name: 'derrubar' }, 'maneuver:derrubar'],
  ])('targetKey(%j) → %s', (target, expected) => {
    expect(targetKey(target)).toBe(expected)
  })
})

describe('computeItemEffects — `always` condition', () => {
  it('applies regardless of equip state (vested item, always condition)', () => {
    const effects = computeItemEffects([
      vested('relic', [
        {
          target: defenseTarget,
          amount: 1,
          bonusType: 'untyped',
          condition: { c: 'always' },
        },
      ]),
    ])
    expect(statFor(effects, defenseTarget).total).toBe(1)
  })

  it('still applies on a wielded item', () => {
    const effects = computeItemEffects([
      wielded('amulet', [
        {
          target: defenseTarget,
          amount: 1,
          bonusType: 'untyped',
          condition: { c: 'always' },
        },
      ]),
    ])
    expect(statFor(effects, defenseTarget).total).toBe(1)
  })

  it('is suppressed when item.equipped is null (top-level gate dominates)', () => {
    const effects = computeItemEffects([
      {
        source: 'pack',
        equipped: null,
        modifiers: [
          {
            target: defenseTarget,
            amount: 1,
            bonusType: 'untyped',
            condition: { c: 'always' },
          },
        ],
      },
    ])
    expect(statFor(effects, defenseTarget).total).toBe(0)
  })

  it('falls back to `always` when modifier carries no condition at all', () => {
    const effects = computeItemEffects([
      vested('ring', [
        { target: defenseTarget, amount: 1, bonusType: 'untyped' },
      ]),
    ])
    expect(statFor(effects, defenseTarget).total).toBe(1)
  })
})

describe('computeItemEffects — equip gating (symmetric)', () => {
  it('ignores `vested` modifiers when the item is wielded', () => {
    const effects = computeItemEffects([
      wielded('vest-and-blade', [
        {
          target: defenseTarget,
          amount: 2,
          bonusType: 'armor',
          condition: { c: 'vested' },
        },
      ]),
    ])
    expect(statFor(effects, defenseTarget).total).toBe(0)
  })
})

describe('computeItemEffects — non-stacking edge cases', () => {
  it('three same-bonusType entries: keeps only the highest-abs', () => {
    const effects = computeItemEffects([
      vested('a', [
        {
          target: defenseTarget,
          amount: 2,
          bonusType: 'enhancement',
          condition: { c: 'vested' },
        },
      ]),
      vested('b', [
        {
          target: defenseTarget,
          amount: 4,
          bonusType: 'enhancement',
          condition: { c: 'vested' },
        },
      ]),
      vested('c', [
        {
          target: defenseTarget,
          amount: 3,
          bonusType: 'enhancement',
          condition: { c: 'vested' },
        },
      ]),
    ])
    const def = statFor(effects, defenseTarget)
    expect(def.total).toBe(4)
    expect(def.contributions).toHaveLength(1)
    expect(def.contributions[0]!.source).toBe('b')
  })

  it('different typed bonusTypes on the same target both apply (sum)', () => {
    const effects = computeItemEffects([
      vested('cota-malha', [
        {
          target: defenseTarget,
          amount: 5,
          bonusType: 'armor',
          condition: { c: 'vested' },
        },
      ]),
      vested('aco-rubi', [
        {
          target: defenseTarget,
          amount: 1,
          bonusType: 'enhancement',
          condition: { c: 'vested' },
        },
      ]),
    ])
    const def = statFor(effects, defenseTarget)
    expect(def.total).toBe(6)
    expect(def.contributions).toHaveLength(2)
  })
})

describe('computeItemEffects — conditional condition shapes', () => {
  it('`context` condition uses the note from the condition payload', () => {
    const effects = computeItemEffects([
      vested('item', [
        {
          target: defenseTarget,
          amount: 1,
          bonusType: 'untyped',
          condition: { c: 'context', note: 'ao usar manobra' },
        },
      ]),
    ])
    expect(effects.conditional).toHaveLength(1)
    expect(effects.conditional[0]!.note).toBe('ao usar manobra')
  })

  it('`flagOn` condition carries the flag prop onto the ConditionalEffect', () => {
    const effects = computeItemEffects([
      wielded('amulet', [
        {
          target: damageThisTarget,
          amount: 2,
          bonusType: 'untyped',
          condition: { c: 'flagOn', flag: 'enraged', label: 'Enfurecido' },
        },
      ]),
    ])
    expect(effects.conditional[0]!.flag).toBe('enraged')
    expect(effects.conditional[0]!.note).toBe('Enfurecido')
  })

  it('falls back to modifier.note when describeCondition returns empty', () => {
    const effects = computeItemEffects([
      vested('item', [
        {
          target: defenseTarget,
          amount: 1,
          bonusType: 'untyped',
          condition: { c: 'against', trait: '' },
          note: 'fallback',
        },
      ]),
    ])
    expect(effects.conditional[0]!.note).toBe('contra: ')
  })
})

describe('applyActiveConditionals — re-resolves stacking after the fold', () => {
  it('conditional enhancement +3 displaces a base enhancement +1', () => {
    const base = computeItemEffects([
      vested('base', [
        {
          target: defenseTarget,
          amount: 1,
          bonusType: 'enhancement',
          condition: { c: 'vested' },
        },
      ]),
      wielded('blade', [
        {
          target: defenseTarget,
          amount: 3,
          bonusType: 'enhancement',
          condition: { c: 'against', trait: 'mortos-vivos' },
        },
      ]),
    ])
    expect(statFor(base, defenseTarget).total).toBe(1)

    const id = conditionalId(base.conditional[0]!)
    const next = applyActiveConditionals(base, new Set([id]))
    expect(statFor(next, defenseTarget).total).toBe(3)
    const def = statFor(next, defenseTarget)
    expect(def.contributions).toHaveLength(1)
    expect(def.contributions[0]!.amount).toBe(3)
  })

  it('ignores flag-target conditionals on fold (no UI for opt-in flags)', () => {
    const base = computeItemEffects([
      vested('weird', [
        {
          target: { k: 'flag', name: 'fatigue-on-sleep' },
          amount: 1,
          bonusType: 'untyped',
          condition: { c: 'context', note: 'durante a noite' },
        },
      ]),
    ])
    const id = conditionalId(base.conditional[0]!)
    const next = applyActiveConditionals(base, new Set([id]))
    expect(next.flags.has('fatigue-on-sleep')).toBe(false)
    expect(next.byTarget['flag:fatigue-on-sleep']).toBeUndefined()
  })
})

describe('conditionalId — divergence axes', () => {
  it('differs when target shape differs', () => {
    const a = {
      source: 'a',
      bonusType: 'untyped' as const,
      amount: 2,
      note: 'n',
      target: defenseTarget,
    }
    const b = { ...a, target: damageThisTarget }
    expect(conditionalId(a)).not.toBe(conditionalId(b))
  })

  it('differs when amount differs', () => {
    const a = {
      source: 'a',
      bonusType: 'untyped' as const,
      amount: 2,
      note: 'n',
      target: defenseTarget,
    }
    const b = { ...a, amount: 3 }
    expect(conditionalId(a)).not.toBe(conditionalId(b))
  })

  it('differs when bonusType differs', () => {
    const a = {
      source: 'a',
      bonusType: 'untyped' as const,
      amount: 2,
      note: 'n',
      target: defenseTarget,
    }
    const b = { ...a, bonusType: 'enhancement' as const }
    expect(conditionalId(a)).not.toBe(conditionalId(b))
  })
})

describe('computeItemEffects — empty / trivial inputs', () => {
  it('empty item list produces a wholly empty ItemEffects', () => {
    const effects = computeItemEffects([])
    expect(effects.byTarget).toEqual({})
    expect(effects.flags.size).toBe(0)
    expect(effects.conditional).toEqual([])
  })

  it('item with no modifiers contributes nothing', () => {
    const effects = computeItemEffects([vested('empty', [])])
    expect(effects.byTarget).toEqual({})
    expect(effects.flags.size).toBe(0)
    expect(effects.conditional).toEqual([])
  })
})
