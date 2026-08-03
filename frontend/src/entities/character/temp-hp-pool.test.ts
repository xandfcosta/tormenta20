import { describe, expect, it } from 'vitest'
import type { ActiveEffect, Character } from '@/shared/api/api'
import {
  applyDrainToEffects,
  applyPoolResult,
  drainPlan,
  reconcileDamageResult,
  routeDamage,
  tempHpPool,
  type TempHpSlice,
} from './temp-hp-pool'

// Named fakes — minimal rows exercising exactly the parsing/routing math.
function fakeTempHpEffect(
  id: number,
  amount: number,
  catalogId = 'class.barbaro.alma-de-bronze',
): ActiveEffect {
  return {
    id,
    catalogId,
    scope: 'scene',
    modifiers: JSON.stringify([
      { target: { k: 'tempHp' }, amount, bonusType: 'untyped', note: 'PV temporários' },
    ]),
    createdAt: '2026-08-01T00:00:00Z',
  }
}

function fakeDefenseEffect(id: number): ActiveEffect {
  return {
    id,
    catalogId: 'armadura-arcana',
    scope: 'scene',
    modifiers: JSON.stringify([
      { target: { k: 'defense' }, amount: 5, bonusType: 'armor' },
    ]),
    createdAt: '2026-08-01T00:00:00Z',
  }
}

function fakeCharacterWith(effects: ActiveEffect[]): Character {
  return { activeEffects: effects } as unknown as Character
}

describe('tempHpPool', () => {
  it('sums only tempHp modifiers across active effects', () => {
    const character = fakeCharacterWith([
      fakeTempHpEffect(1, 10),
      fakeDefenseEffect(2),
      fakeTempHpEffect(3, 30, 'campo-de-forca'),
    ])
    const pool = tempHpPool(character)
    expect(pool.total).toBe(40)
    expect(pool.slices.map((s) => s.effectId)).toEqual([1, 3])
  })

  it('labels slices with the power name from the activation registry', () => {
    const pool = tempHpPool(fakeCharacterWith([fakeTempHpEffect(1, 10)]))
    expect(pool.slices[0]?.label).toBe('Alma de Bronze')
  })

  it('labels the GM manual pool as PV temporários (manual)', () => {
    const pool = tempHpPool(
      fakeCharacterWith([fakeTempHpEffect(1, 12, 'manual-temp-hp')]),
    )
    expect(pool.slices[0]?.label).toBe('PV temporários (manual)')
  })

  it('is empty without effects (characters with no pool)', () => {
    expect(tempHpPool(fakeCharacterWith([])).total).toBe(0)
    expect(tempHpPool(fakeCharacterWith([fakeDefenseEffect(1)])).total).toBe(0)
  })

  it('ignores drained (amount 0) and malformed modifier blobs', () => {
    const broken: ActiveEffect = { ...fakeTempHpEffect(1, 5), modifiers: '{oops' }
    const zero = fakeTempHpEffect(2, 0)
    expect(tempHpPool(fakeCharacterWith([broken, zero])).total).toBe(0)
  })
})

describe('routeDamage — temp-first routing math', () => {
  it('soaks fully in the pool when it covers the damage', () => {
    expect(routeDamage(3, 10)).toEqual({ toPool: 3, toHp: 0 })
  })

  it('splits overflow damage to hp', () => {
    expect(routeDamage(7, 5)).toEqual({ toPool: 5, toHp: 2 })
  })

  it('sends everything to hp with an empty pool', () => {
    expect(routeDamage(4, 0)).toEqual({ toPool: 0, toHp: 4 })
  })

  it('healing (non-positive damage) never touches the pool', () => {
    expect(routeDamage(0, 10)).toEqual({ toPool: 0, toHp: 0 })
    expect(routeDamage(-5, 10)).toEqual({ toPool: 0, toHp: 0 })
  })
})

describe('drainPlan — highest pool first (mirrors POST :id/damage)', () => {
  const slices: TempHpSlice[] = [
    { effectId: 1, amount: 4, label: 'Alma de Bronze' },
    { effectId: 2, amount: 30, label: 'Campo de Força' },
  ]

  it('drains the biggest slice partially', () => {
    expect(drainPlan(slices, 3)).toEqual([
      { effectId: 2, drained: 3, remaining: 27 },
    ])
  })

  it('spills into the next slice when the biggest empties', () => {
    expect(drainPlan(slices, 32)).toEqual([
      { effectId: 2, drained: 30, remaining: 0 },
      { effectId: 1, drained: 2, remaining: 2 },
    ])
  })

  it('is empty for a non-positive amount', () => {
    expect(drainPlan(slices, 0)).toEqual([])
  })
})

describe('applyDrainToEffects — optimistic cache patch', () => {
  it('rewrites the tempHp amount and drops emptied rows', () => {
    const effects = [
      fakeTempHpEffect(1, 4),
      fakeTempHpEffect(2, 30, 'campo-de-forca'),
      fakeDefenseEffect(3),
    ]
    const next = applyDrainToEffects(effects, [
      { effectId: 1, drained: 4, remaining: 0 },
      { effectId: 2, drained: 2, remaining: 28 },
    ])
    expect(next.map((e) => e.id)).toEqual([2, 3])
    expect(JSON.parse(next[0]!.modifiers)[0].amount).toBe(28)
  })

  it('leaves untouched effects verbatim', () => {
    const effects = [fakeDefenseEffect(3)]
    expect(applyDrainToEffects(effects, [])).toEqual(effects)
  })

  it('keeps an emptied MIXED buff row (Heroísmo) with its pool zeroed', () => {
    const next = applyDrainToEffects(
      [fakeMixedEffect(9, 40)],
      [{ effectId: 9, drained: 40, remaining: 0 }],
    )
    expect(next).toHaveLength(1)
    const modifiers = JSON.parse(next[0]!.modifiers)
    expect(modifiers[0].amount).toBe(0)
    expect(modifiers[1].amount).toBe(4)
  })
})

/** Mixed buff — tempHp + attack, like Heroísmo (p383). */
function fakeMixedEffect(id: number, amount: number): ActiveEffect {
  return {
    id,
    catalogId: 'heroismo',
    scope: 'scene',
    modifiers: JSON.stringify([
      { target: { k: 'tempHp' }, amount, bonusType: 'untyped', note: 'Heroísmo' },
      { target: { k: 'attack', scope: 'all' }, amount: 4, bonusType: 'untyped' },
    ]),
    createdAt: '2026-08-01T00:00:00Z',
  }
}

describe('applyPoolResult — POST :id/active-effects cache merge', () => {
  const row = fakeTempHpEffect(10, 12, 'manual-temp-hp')

  it('upserts a plain ActiveEffect row (spell buffs, verbatim grants)', () => {
    const prev = fakeCharacterWith([fakeDefenseEffect(3)])
    const next = applyPoolResult(prev, row)
    expect(next.activeEffects.map((e) => e.id)).toEqual([3, 10])
  })

  it('replaces a stale row for the same catalogId+scope', () => {
    const prev = fakeCharacterWith([fakeTempHpEffect(8, 5, 'manual-temp-hp')])
    const next = applyPoolResult(prev, { effect: row, displaced: [] })
    expect(next.activeEffects).toEqual([row])
  })

  it('drops removed pools and zeroes kept (mixed) ones under vale-o-maior', () => {
    const prev = fakeCharacterWith([fakeTempHpEffect(1, 4), fakeMixedEffect(9, 8)])
    const next = applyPoolResult(prev, {
      effect: row,
      displaced: [
        { effectId: 1, removed: true },
        { effectId: 9, removed: false },
      ],
    })
    expect(next.activeEffects.map((e) => e.id)).toEqual([9, 10])
    expect(JSON.parse(next.activeEffects[0]!.modifiers)[0].amount).toBe(0)
  })

  it('superseded leaves the character untouched', () => {
    const prev = fakeCharacterWith([fakeTempHpEffect(1, 30)])
    expect(
      applyPoolResult(prev, { superseded: true, keptEffectId: 1, keptAmount: 30 }),
    ).toBe(prev)
  })

  it('cleared drops the manual pool rows by id', () => {
    const prev = fakeCharacterWith([
      fakeTempHpEffect(8, 5, 'manual-temp-hp'),
      fakeDefenseEffect(3),
    ])
    const next = applyPoolResult(prev, { cleared: true, removedEffectIds: [8] })
    expect(next.activeEffects.map((e) => e.id)).toEqual([3])
  })
})

describe('reconcileDamageResult — POST :id/damage cache merge', () => {
  it('applies hpCurrent + drops/rewrites drained pools from the response', () => {
    const prev = {
      ...fakeCharacterWith([fakeTempHpEffect(1, 4), fakeTempHpEffect(2, 30, 'campo-de-forca')]),
      hpCurrent: 20,
    } as Character
    const next = reconcileDamageResult(prev, {
      hpCurrent: 20,
      tempHpRemaining: 27,
      drained: [
        { effectId: 2, newAmount: 27, removed: false },
        { effectId: 1, newAmount: 0, removed: true },
      ],
    })
    expect(next.hpCurrent).toBe(20)
    expect(next.activeEffects.map((e) => e.id)).toEqual([2])
    expect(JSON.parse(next.activeEffects[0]!.modifiers)[0].amount).toBe(27)
  })

  it('lowers hp on passthrough damage with no drained pools', () => {
    const prev = { ...fakeCharacterWith([]), hpCurrent: 20 } as Character
    const next = reconcileDamageResult(prev, {
      hpCurrent: 15,
      tempHpRemaining: 0,
      drained: [],
    })
    expect(next.hpCurrent).toBe(15)
  })
})
