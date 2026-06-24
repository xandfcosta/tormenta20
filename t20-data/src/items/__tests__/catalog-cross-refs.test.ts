import { describe, expect, it } from 'vitest'
import { ATTRIBUTE_KEYS } from '../../attributes'
import { EXPERTISE_NAMES } from '../../expertises'
import { CATALOG_ITEMS } from '../catalog/index'
import type {
  BonusType,
  ItemCategory,
  Modifier,
  ModifierCondition,
} from '../types'

/**
 * Cross-reference + wire-shape integrity for CATALOG_ITEMS. Sibling spec to
 * abilities/__tests__/catalog-cross-refs.test.ts — pins the parts the engine
 * relies on so future catalog edits can't silently break a reference.
 *
 *  - modifier targets resolve to real expertise names / attribute keys
 *  - modifier amounts are finite (no NaN) and in a sane range (-20..+50);
 *    catches a stray string or accidental zero default that would hide a
 *    real effect
 *  - modifier bonusType belongs to the known stacking-resolution set
 *  - modifier condition `c` is a known shape
 *  - consumable scope is 'instant' | 'scene' | 'day'
 *  - overlay categories (improvement / material) declare `appliesTo`
 *  - non-overlay categories do NOT declare `appliesTo` (it's a no-op there)
 *  - id matches kebab slug; name non-empty; price/slots ≥ 0
 */
const KNOWN_EXPERTISES = new Set<string>(EXPERTISE_NAMES)
const ATTR_KEYS = new Set<string>(ATTRIBUTE_KEYS)

const BONUS_TYPES = new Set<BonusType>([
  'armor',
  'item',
  'training',
  'morale',
  'enhancement',
  'untyped',
])

const CONDITION_KINDS = new Set<ModifierCondition['c']>([
  'always',
  'wielded',
  'vested',
  'terrain',
  'against',
  'context',
  'flagOn',
])

const OVERLAY_CATEGORIES = new Set<ItemCategory>(['improvement', 'material'])

function targetIssues(mod: Modifier, where: string): string[] {
  const t = mod.target
  const issues: string[] = []
  if (t.k === 'expertise' || t.k === 'expertiseRemovePenalty') {
    if (!KNOWN_EXPERTISES.has(t.name)) {
      issues.push(`${where} → ${t.k}(${t.name})`)
    }
  }
  if (t.k === 'attribute' && !ATTR_KEYS.has(t.name)) {
    issues.push(`${where} → attribute(${t.name})`)
  }
  if (t.k === 'expertiseByAttribute' && !ATTR_KEYS.has(t.attribute)) {
    issues.push(`${where} → expertiseByAttribute(${t.attribute})`)
  }
  return issues
}

describe('CATALOG_ITEMS — id + naming', () => {
  it('every id is kebab-case slug ([a-z0-9-]+)', () => {
    const bad = CATALOG_ITEMS.filter(
      (it) => !/^[a-z0-9][a-z0-9-]*$/.test(it.id),
    ).map((it) => it.id)
    expect(bad).toEqual([])
  })

  it('every item has a non-empty name', () => {
    const bad = CATALOG_ITEMS.filter((it) => !it.name?.trim()).map(
      (it) => it.id,
    )
    expect(bad).toEqual([])
  })

  it('price and slots are non-negative finite numbers', () => {
    const bad: string[] = []
    for (const it of CATALOG_ITEMS) {
      if (!Number.isFinite(it.price) || it.price < 0) {
        bad.push(`${it.id} price=${it.price}`)
      }
      if (!Number.isFinite(it.slots) || it.slots < 0) {
        bad.push(`${it.id} slots=${it.slots}`)
      }
    }
    expect(bad).toEqual([])
  })
})

describe('CATALOG_ITEMS — modifier wire shape', () => {
  it('every modifier target resolves (expertise/attribute name in catalog)', () => {
    const bad: string[] = []
    for (const it of CATALOG_ITEMS) {
      for (const mod of it.modifiers) {
        bad.push(...targetIssues(mod, it.id))
      }
      for (const mod of it.consumable?.modifiers ?? []) {
        bad.push(...targetIssues(mod, `${it.id}/consumable`))
      }
    }
    expect(bad).toEqual([])
  })

  it('every modifier amount is a finite number in -20..+50', () => {
    const bad: string[] = []
    for (const it of CATALOG_ITEMS) {
      const all = [...it.modifiers, ...(it.consumable?.modifiers ?? [])]
      for (const mod of all) {
        if (
          !Number.isFinite(mod.amount) ||
          mod.amount < -20 ||
          mod.amount > 50
        ) {
          bad.push(`${it.id} amount=${mod.amount}`)
        }
      }
    }
    expect(bad).toEqual([])
  })

  it('every modifier bonusType is in the known stacking set', () => {
    const bad: string[] = []
    for (const it of CATALOG_ITEMS) {
      const all = [...it.modifiers, ...(it.consumable?.modifiers ?? [])]
      for (const mod of all) {
        if (!BONUS_TYPES.has(mod.bonusType)) {
          bad.push(`${it.id} bonusType=${mod.bonusType}`)
        }
      }
    }
    expect(bad).toEqual([])
  })

  it('every modifier condition.c (when present) is a known shape', () => {
    const bad: string[] = []
    for (const it of CATALOG_ITEMS) {
      const all = [...it.modifiers, ...(it.consumable?.modifiers ?? [])]
      for (const mod of all) {
        if (mod.condition && !CONDITION_KINDS.has(mod.condition.c)) {
          bad.push(`${it.id} condition.c=${mod.condition.c}`)
        }
      }
    }
    expect(bad).toEqual([])
  })
})

describe('CATALOG_ITEMS — consumable scope', () => {
  it("consumable.scope is 'instant' | 'scene' | 'day'", () => {
    const valid = new Set(['instant', 'scene', 'day'])
    const bad: string[] = []
    for (const it of CATALOG_ITEMS) {
      if (!it.consumable) continue
      if (!valid.has(it.consumable.scope)) {
        bad.push(`${it.id} scope=${it.consumable.scope}`)
      }
    }
    expect(bad).toEqual([])
  })

  it('items in category=consumable / meal / catalyst all carry a consumable spec', () => {
    const consumableCats = new Set<ItemCategory>([
      'consumable',
      'meal',
      'catalyst',
    ])
    const bad = CATALOG_ITEMS.filter(
      (it) => consumableCats.has(it.category) && !it.consumable,
    ).map((it) => it.id)
    expect(bad).toEqual([])
  })
})

describe('CATALOG_ITEMS — overlay categories (improvement / material)', () => {
  it('every overlay item declares appliesTo with a non-empty family list', () => {
    const bad: string[] = []
    for (const it of CATALOG_ITEMS) {
      if (!OVERLAY_CATEGORIES.has(it.category)) continue
      if (!it.appliesTo || it.appliesTo.length === 0) {
        bad.push(it.id)
      }
    }
    expect(bad).toEqual([])
  })

  it('non-overlay items do NOT declare appliesTo (no-op there)', () => {
    const bad = CATALOG_ITEMS.filter(
      (it) => !OVERLAY_CATEGORIES.has(it.category) && it.appliesTo,
    ).map((it) => it.id)
    expect(bad).toEqual([])
  })
})

describe('CATALOG_ITEMS — equip + hands', () => {
  it("equip is 'vested' | 'wielded' | 'either'", () => {
    const valid = new Set(['vested', 'wielded', 'either'])
    const bad = CATALOG_ITEMS.filter((it) => !valid.has(it.equip)).map(
      (it) => `${it.id} equip=${it.equip}`,
    )
    expect(bad).toEqual([])
  })

  it('hands (when present) is 1 or 2', () => {
    const bad = CATALOG_ITEMS.filter(
      (it) => it.hands !== undefined && it.hands !== 1 && it.hands !== 2,
    ).map((it) => `${it.id} hands=${it.hands}`)
    expect(bad).toEqual([])
  })
})
