import { describe, expect, it } from 'vitest'
import { ATTRIBUTE_KEYS } from '../../attributes'
import { EXPERTISE_NAMES } from '../../expertises'
import { ORIGINS_CATALOG } from '../origins'
import { RACES_CATALOG } from '../races'
import type { Modifier } from '../../items/types'

/**
 * Cross-reference integrity for race & origin catalogs — regression net for
 * the parts the UI relies on but no other spec verifies:
 *
 *  - race ability ids are globally unique and each ability.raceId matches
 *    its parent race (catches copy-paste bugs)
 *  - race ability variant ids are unique within their ability (picker
 *    relies on these as React keys)
 *  - origin benefit ids are unique within an origin (picker selection key)
 *  - poderUnico ids are globally unique across origins (used as picker key
 *    when the player chooses their unique benefit)
 *  - every `kind=pericia` benefit's `expertise` resolves to a real
 *    EXPERTISE_NAMES entry
 *  - every Modifier on a race ability / origin benefit / poderUnico targets
 *    a valid expertise name or attribute key (so engine fold doesn't drop
 *    silently)
 *
 * If a future catalog edit fat-fingers an expertise name or duplicates an
 * id, one of these specs fails immediately instead of leaving a silent
 * dead reference for the UI to fail on later.
 */
const ATTR_KEYS = new Set<string>(ATTRIBUTE_KEYS)
const KNOWN_EXPERTISES = new Set<string>(EXPERTISE_NAMES)

function targetIssues(mod: Modifier, where: string): string[] {
  const t = mod.target
  const issues: string[] = []
  if (t.k === 'expertise' || t.k === 'expertiseRemovePenalty') {
    if (!KNOWN_EXPERTISES.has(t.name)) {
      issues.push(`${where} → ${t.k}(${t.name})`)
    }
  }
  if (t.k === 'attribute') {
    if (!ATTR_KEYS.has(t.name)) {
      issues.push(`${where} → attribute(${t.name})`)
    }
  }
  if (t.k === 'expertiseByAttribute') {
    if (!ATTR_KEYS.has(t.attribute)) {
      issues.push(`${where} → expertiseByAttribute(${t.attribute})`)
    }
  }
  return issues
}

describe('RACES_CATALOG — ability ids', () => {
  it('every ability id is globally unique across the catalog', () => {
    const seen = new Map<string, string>()
    const dupes: string[] = []
    for (const race of RACES_CATALOG) {
      for (const ability of race.abilities) {
        if (seen.has(ability.id)) {
          dupes.push(`${ability.id} (${seen.get(ability.id)} vs ${race.id})`)
        } else {
          seen.set(ability.id, race.id)
        }
      }
    }
    expect(dupes).toEqual([])
  })

  it('every ability.raceId matches its parent race id', () => {
    const bad: string[] = []
    for (const race of RACES_CATALOG) {
      for (const ability of race.abilities) {
        if (ability.raceId !== race.id) {
          bad.push(`${ability.id}: raceId=${ability.raceId} parent=${race.id}`)
        }
      }
    }
    expect(bad).toEqual([])
  })

  it('every ability has a non-empty name and description', () => {
    const bad: string[] = []
    for (const race of RACES_CATALOG) {
      for (const ability of race.abilities) {
        if (!ability.name?.trim() || !ability.description?.trim()) {
          bad.push(`${race.id}/${ability.id}`)
        }
      }
    }
    expect(bad).toEqual([])
  })
})

describe('RACES_CATALOG — ability variants', () => {
  it('variant ids are unique within their parent ability', () => {
    const dupes: string[] = []
    for (const race of RACES_CATALOG) {
      for (const ability of race.abilities) {
        if (!ability.variants) continue
        const ids = ability.variants.map((v) => v.id)
        if (new Set(ids).size !== ids.length) {
          dupes.push(`${race.id}/${ability.id}: ${ids.join(',')}`)
        }
      }
    }
    expect(dupes).toEqual([])
  })

  it('every variant has a non-empty name and description', () => {
    const bad: string[] = []
    for (const race of RACES_CATALOG) {
      for (const ability of race.abilities) {
        for (const variant of ability.variants ?? []) {
          if (!variant.name?.trim() || !variant.description?.trim()) {
            bad.push(`${race.id}/${ability.id}/${variant.id}`)
          }
        }
      }
    }
    expect(bad).toEqual([])
  })
})

describe('RACES_CATALOG — modifier targets', () => {
  it('every ability modifier targets a valid expertise/attribute', () => {
    const bad: string[] = []
    for (const race of RACES_CATALOG) {
      for (const ability of race.abilities) {
        for (const mod of ability.modifiers ?? []) {
          bad.push(...targetIssues(mod, `${race.id}/${ability.id}`))
        }
        for (const variant of ability.variants ?? []) {
          for (const mod of variant.modifiers ?? []) {
            bad.push(
              ...targetIssues(mod, `${race.id}/${ability.id}/${variant.id}`),
            )
          }
        }
      }
    }
    expect(bad).toEqual([])
  })

  it('attributeBonuses keys are all valid attribute keys', () => {
    const bad: string[] = []
    for (const race of RACES_CATALOG) {
      for (const key of Object.keys(race.attributeBonuses)) {
        if (!ATTR_KEYS.has(key)) {
          bad.push(`${race.id} → ${key}`)
        }
      }
    }
    expect(bad).toEqual([])
  })
})

describe('ORIGINS_CATALOG — benefit ids', () => {
  it('benefit ids are unique within each origin', () => {
    const dupes: string[] = []
    for (const origin of ORIGINS_CATALOG) {
      const ids = origin.benefits.map((b) => b.id)
      if (new Set(ids).size !== ids.length) {
        const seen = new Set<string>()
        for (const id of ids) {
          if (seen.has(id)) dupes.push(`${origin.id}: ${id}`)
          seen.add(id)
        }
      }
    }
    expect(dupes).toEqual([])
  })

  it('poderUnico ids are globally unique across origins', () => {
    const seen = new Map<string, string>()
    const dupes: string[] = []
    for (const origin of ORIGINS_CATALOG) {
      const id = origin.poderUnico.id
      if (seen.has(id)) {
        dupes.push(`${id} (${seen.get(id)} vs ${origin.id})`)
      } else {
        seen.set(id, origin.id)
      }
    }
    expect(dupes).toEqual([])
  })

  it('every benefit has a non-empty name and description', () => {
    const bad: string[] = []
    for (const origin of ORIGINS_CATALOG) {
      for (const benefit of origin.benefits) {
        if (!benefit.name?.trim() || !benefit.description?.trim()) {
          bad.push(`${origin.id}/${benefit.id}`)
        }
      }
      const u = origin.poderUnico
      if (!u.name?.trim() || !u.description?.trim()) {
        bad.push(`${origin.id}/poderUnico=${u.id}`)
      }
    }
    expect(bad).toEqual([])
  })
})

describe('ORIGINS_CATALOG — perícia benefits', () => {
  it('every kind=pericia benefit names a real EXPERTISE_NAMES entry', () => {
    const bad: string[] = []
    for (const origin of ORIGINS_CATALOG) {
      for (const benefit of origin.benefits) {
        if (benefit.kind !== 'pericia') continue
        if (!benefit.expertise || !KNOWN_EXPERTISES.has(benefit.expertise)) {
          bad.push(`${origin.id}/${benefit.id} → ${benefit.expertise}`)
        }
      }
    }
    expect(bad).toEqual([])
  })

  it('poderUnico is always kind=poder (never a perícia)', () => {
    const bad = ORIGINS_CATALOG.filter(
      (o) => o.poderUnico.kind !== 'poder',
    ).map((o) => `${o.id} → ${o.poderUnico.kind}`)
    expect(bad).toEqual([])
  })
})

describe('ORIGINS_CATALOG — modifier targets', () => {
  it('every benefit modifier targets a valid expertise/attribute', () => {
    const bad: string[] = []
    for (const origin of ORIGINS_CATALOG) {
      for (const benefit of origin.benefits) {
        for (const mod of benefit.modifiers ?? []) {
          bad.push(...targetIssues(mod, `${origin.id}/${benefit.id}`))
        }
      }
      for (const mod of origin.poderUnico.modifiers ?? []) {
        bad.push(...targetIssues(mod, `${origin.id}/poderUnico`))
      }
    }
    expect(bad).toEqual([])
  })
})
