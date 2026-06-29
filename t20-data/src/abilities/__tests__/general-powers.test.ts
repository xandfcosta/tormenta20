import { describe, expect, it } from 'vitest'
import {
  GENERAL_POWERS_CATALOG,
  generalPowersByKinds,
  getGeneralPower,
} from '../general-powers'

/**
 * PDF Cap 2 — "Poderes Gerais" (Combate, Destino, Magia, Tormenta) plus the
 * class-specific pools that share the same PowerKind union. These tests pin
 * catalog *plumbing* (id uniqueness, kind validity, helper edge cases).
 * Content-level assertions live in `__tests__/general-powers.test.ts`.
 */
describe('GENERAL_POWERS_CATALOG plumbing', () => {
  it('every id is globally unique', () => {
    const ids = GENERAL_POWERS_CATALOG.map((p) => p.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('every entry declares a known PowerKind', () => {
    const valid = new Set([
      'combate',
      'destino',
      'magia',
      'tormenta',
      'arcanista',
      'barbaro',
      'bardo',
      'bucaneiro',
      'cacador',
      'cavaleiro',
      'clerigo',
      'druida',
      'guerreiro',
      'inventor',
      'ladino',
      'lutador',
      'nobre',
      'paladino',
    ])
    for (const p of GENERAL_POWERS_CATALOG) expect(valid.has(p.kind)).toBe(true)
  })
})

describe('getGeneralPower', () => {
  it('returns undefined for unknown id', () => {
    expect(getGeneralPower('ghost.id')).toBeUndefined()
  })
})

describe('generalPowersByKinds', () => {
  it('returns empty array when no kinds are passed', () => {
    expect(generalPowersByKinds([])).toEqual([])
  })

  it('filtering by a class kind not in the catalog returns empty', () => {
    expect(generalPowersByKinds(['barbaro'])).toEqual([])
  })
})
