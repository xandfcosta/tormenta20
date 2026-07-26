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

/**
 * Passive general powers whose flat bonuses map onto existing modifier
 * targets. Regression for the `derived.ts` bug where general-power modifiers
 * never applied (it filtered on a `general.` id prefix the picker never
 * writes). Saves are perícias in T20, so Reflexos/Fortitude/Vontade are
 * `expertise` targets.
 */
describe('passive general-power modifiers', () => {
  const expected: Record<string, number> = {
    esquiva: 2, // defense + Reflexos
    atletico: 2, // Atletismo + displacement
    'saque-rapido': 1, // Iniciativa
    investigador: 1, // Investigação
    vitalidade: 1, // Fortitude
    'vontade-de-ferro': 1, // Vontade
    'sentidos-agucados': 1, // Percepção
  }

  for (const [id, count] of Object.entries(expected)) {
    it(`${id} carries ${count} computed modifier(s)`, () => {
      expect(getGeneralPower(id)?.modifiers).toHaveLength(count)
    })
  }

  it('esquiva grants +2 Defesa and +2 Reflexos', () => {
    const mods = getGeneralPower('esquiva')?.modifiers ?? []
    expect(mods).toContainEqual({
      target: { k: 'defense' },
      amount: 2,
      bonusType: 'untyped',
    })
    expect(mods).toContainEqual({
      target: { k: 'expertise', name: 'Reflexos' },
      amount: 2,
      bonusType: 'untyped',
    })
  })

  it('atletico grants +3m displacement', () => {
    expect(getGeneralPower('atletico')?.modifiers).toContainEqual({
      target: { k: 'displacement' },
      amount: 3,
      bonusType: 'untyped',
    })
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
