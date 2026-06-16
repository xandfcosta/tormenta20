import { describe, expect, it } from 'vitest'
import {
  GENERAL_POWERS_CATALOG,
  generalPowersByKinds,
  getGeneralPower,
} from '../general-powers'

/**
 * PDF Cap 2 — "Poderes Gerais" (Combate, Destino, Magia, Tormenta) plus the
 * class-specific pools that share the same PowerKind union. Catalog content
 * is *deliberately* empty (per general-powers.ts comment: "until Cap 2 pages
 * are audited"). These tests pin the *helpers* so they remain correct once
 * the catalog is populated in a future PR.
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

  it('returns undefined while catalog is empty (Cap 2 not audited)', () => {
    expect(getGeneralPower('any')).toBeUndefined()
  })
})

describe('generalPowersByKinds', () => {
  it('returns empty array when no kinds are passed', () => {
    expect(generalPowersByKinds([])).toEqual([])
  })

  it('filters by kind set — empty catalog → empty result regardless of kinds', () => {
    expect(generalPowersByKinds(['combate'])).toEqual([])
    expect(generalPowersByKinds(['combate', 'destino', 'magia'])).toEqual([])
  })
})
