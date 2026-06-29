import { describe, expect, it } from 'vitest'
import {
  GENERAL_POWERS_CATALOG,
  generalPowersByKinds,
  getGeneralPower,
} from '../abilities/general-powers'

/**
 * Poderes Gerais — PDF book p124-131.
 *
 * Pinned:
 *  - 3 categorias: Combate (p124-128), Destino (p129-131), Magia (p131)
 *  - Tormenta excluded — encoded separately in `tormenta.ts`
 *  - Prereq shapes: attribute / trained / power / note (free-form)
 *  - `minLevel` only on Celebrar Ritual (8º), Ao Sabor do Destino (6º), etc.
 *  - All ids unique and kebab-case
 */

describe('GENERAL_POWERS_CATALOG — shape & invariants', () => {
  it('has at least 38 entries', () => {
    expect(GENERAL_POWERS_CATALOG.length).toBeGreaterThanOrEqual(38)
  })

  it('all ids unique', () => {
    const ids = GENERAL_POWERS_CATALOG.map((p) => p.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('all ids are kebab-case (lowercase, ASCII letters, dashes)', () => {
    for (const p of GENERAL_POWERS_CATALOG) {
      expect(p.id).toMatch(/^[a-z]+(-[a-z]+)*$/)
    }
  })

  it('every entry has non-empty name + description', () => {
    for (const p of GENERAL_POWERS_CATALOG) {
      expect(p.name).toBeTruthy()
      expect(p.description).toBeTruthy()
    }
  })

  it('only encodes the 3 non-Tormenta kinds (combate / destino / magia)', () => {
    for (const p of GENERAL_POWERS_CATALOG) {
      expect(['combate', 'destino', 'magia']).toContain(p.kind)
    }
  })
})

describe('getGeneralPower — lookup', () => {
  it('returns the entry for a known id', () => {
    expect(getGeneralPower('ataque-poderoso')?.name).toBe('Ataque Poderoso')
  })

  it('returns undefined for an unknown id', () => {
    expect(getGeneralPower('not-real')).toBeUndefined()
  })
})

describe('generalPowersByKinds — filter', () => {
  it('Combate filter returns only combat powers', () => {
    const combate = generalPowersByKinds(['combate'])
    expect(combate.length).toBeGreaterThan(0)
    for (const p of combate) {
      expect(p.kind).toBe('combate')
    }
  })

  it('Magia filter returns only magia powers', () => {
    const magia = generalPowersByKinds(['magia'])
    for (const p of magia) {
      expect(p.kind).toBe('magia')
    }
  })

  it('Multi-kind filter unions results', () => {
    const both = generalPowersByKinds(['combate', 'destino'])
    const combate = generalPowersByKinds(['combate']).length
    const destino = generalPowersByKinds(['destino']).length
    expect(both.length).toBe(combate + destino)
  })

  it('Tormenta filter returns nothing (encoded elsewhere)', () => {
    expect(generalPowersByKinds(['tormenta'])).toEqual([])
  })
})

describe('Pinned canonical poderes — PDF integrity', () => {
  it('Ataque Poderoso: combate, For 1', () => {
    const p = getGeneralPower('ataque-poderoso')!
    expect(p.kind).toBe('combate')
    expect(p.prerequisites).toEqual([
      { kind: 'attribute', attr: 'strength', min: 1 },
    ])
  })

  it('Esquiva: combate, Des 1', () => {
    const p = getGeneralPower('esquiva')!
    expect(p.prerequisites).toEqual([
      { kind: 'attribute', attr: 'dexterity', min: 1 },
    ])
  })

  it('Sortudo: destino, no prerequisites', () => {
    const p = getGeneralPower('sortudo')!
    expect(p.kind).toBe('destino')
    expect(p.prerequisites).toBeUndefined()
  })

  it('Surto Heroico: destino, no prerequisites', () => {
    expect(getGeneralPower('surto-heroico')?.kind).toBe('destino')
    expect(getGeneralPower('surto-heroico')?.prerequisites).toBeUndefined()
  })

  it('Foco em Magia: magia kind, note prereq "Lançar magias"', () => {
    const p = getGeneralPower('foco-em-magia')!
    expect(p.kind).toBe('magia')
    expect(p.prerequisites).toEqual([
      { kind: 'note', description: 'Lançar magias' },
    ])
  })

  it('Celebrar Ritual: minLevel 8', () => {
    expect(getGeneralPower('celebrar-ritual')?.minLevel).toBe(8)
  })

  it('Vitalidade: combate kind, Con 1', () => {
    expect(getGeneralPower('vitalidade')?.prerequisites).toEqual([
      { kind: 'attribute', attr: 'constitution', min: 1 },
    ])
  })

  it('Ginete: trained em Cavalgar', () => {
    expect(getGeneralPower('ginete')?.prerequisites).toEqual([
      { kind: 'trained', expertise: 'Cavalgar' },
    ])
  })

  it('Estilo de Duas Mãos: For 2 + Luta', () => {
    expect(getGeneralPower('estilo-de-duas-maos')?.prerequisites).toEqual([
      { kind: 'attribute', attr: 'strength', min: 2 },
      { kind: 'trained', expertise: 'Luta' },
    ])
  })
})

describe('Categoria coverage', () => {
  it('Combate has at least 15 powers', () => {
    expect(generalPowersByKinds(['combate']).length).toBeGreaterThanOrEqual(15)
  })

  it('Destino has at least 10 powers', () => {
    expect(generalPowersByKinds(['destino']).length).toBeGreaterThanOrEqual(10)
  })

  it('Magia has at least 6 powers', () => {
    expect(generalPowersByKinds(['magia']).length).toBeGreaterThanOrEqual(6)
  })
})
