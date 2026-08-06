import {
  CLASS_POWERS_CATALOG,
  DEUSES,
  devotoOptionsFor as srcDevotoOptionsFor,
  GENERAL_POWERS_CATALOG,
  getOrigin as srcGetOrigin,
  getRace as srcGetRace,
  GRANTED_POWERS,
  ORIGINS_CATALOG,
  ownedClassPowers as srcOwnedClassPowers,
  RACES_CATALOG,
} from '@tormenta20/t20-data'
import { beforeAll, describe, expect, it } from 'vitest'
import {
  allGeneralPowers,
  classPowersFor,
  devotoOptionsFor,
  getOrigin,
  getRace,
  grantedPowers,
  isAbilitiesPrimed,
  ownedClassPowers,
  primeAbilities,
  racesGrantTormenta,
} from './abilities-cache'

/**
 * The cache must be a FAITHFUL mirror of the t20-data lookups — the front now
 * runs these against fetched-and-cached data instead of the bundled catalog,
 * so any drift is a correctness bug. We prime with the real catalogs (Node
 * test → no bundle concern) and compare against t20-data's own functions.
 */
describe('abilities-cache', () => {
  beforeAll(() => {
    primeAbilities({
      races: RACES_CATALOG,
      origins: ORIGINS_CATALOG,
      classPowers: CLASS_POWERS_CATALOG,
      generalPowers: GENERAL_POWERS_CATALOG,
      deuses: DEUSES,
      grantedPowers: GRANTED_POWERS,
    })
  })

  it('reports primed after prime', () => {
    expect(isAbilitiesPrimed()).toBe(true)
  })

  it('getRace mirrors the source for a known race', () => {
    expect(getRace('Lefou')).toBe(srcGetRace('Lefou'))
    expect(getRace('nope-not-a-race')).toBeUndefined()
  })

  it('getOrigin mirrors the source', () => {
    const anyOriginId = ORIGINS_CATALOG[0].id
    expect(getOrigin(anyOriginId)).toBe(srcGetOrigin(anyOriginId))
  })

  it('racesGrantTormenta detects Lefou (grantsTormentaPowers)', () => {
    expect(racesGrantTormenta(['Lefou'])).toBe(true)
    expect(racesGrantTormenta(['Humano'])).toBe(false)
  })

  it('ownedClassPowers matches the source rule (auto + elective)', () => {
    const chosen = new Set<string>()
    const mine = ownedClassPowers('Bárbaro', 6, chosen)
    expect(mine).toEqual(srcOwnedClassPowers('Bárbaro', 6, chosen))
    expect(mine.length).toBeGreaterThan(0)
  })

  it('devotoOptionsFor matches the source per class', () => {
    expect(devotoOptionsFor('Clérigo')).toEqual(srcDevotoOptionsFor('Clérigo'))
    expect(devotoOptionsFor('Guerreiro')).toBeNull()
  })

  it('classPowersFor filters by className', () => {
    const powers = classPowersFor('Bárbaro')
    expect(powers.length).toBeGreaterThan(0)
    expect(powers.every((p) => p.className === 'Bárbaro')).toBe(true)
  })

  it('allGeneralPowers excludes tormenta powers', () => {
    expect(allGeneralPowers().every((p) => p.kind !== 'tormenta')).toBe(true)
  })

  it('grantedPowers exposes the primed granted-power list', () => {
    expect(grantedPowers()).toBe(GRANTED_POWERS)
  })
})
