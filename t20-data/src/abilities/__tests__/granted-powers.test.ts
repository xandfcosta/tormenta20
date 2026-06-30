import { describe, expect, it } from 'vitest'
import {
  GRANTED_POWERS,
  grantedPowerById,
  grantedPowerByName,
  grantedPowersByDeus,
  grantedPowersByKind,
  type GrantedPower,
  type GrantedPowerKind,
} from '../granted-powers'
import { DEUSES } from '../deuses'

/**
 * Poderes Concedidos catalog (PDF Cap 2, p132-136).
 *
 * Cross-reference invariant:
 *  - Every (deus, poder) pair in this catalog must be present as a
 *    member of the matching `Deus.poderesConcedidos[]`.
 *
 * The reverse (every `poderesConcedidos[]` entry has a catalog match)
 * is NOT asserted yet — we only encode ~36 of the ~70 unique poderes
 * concedidos referenced.
 */

const KINDS: readonly GrantedPowerKind[] = [
  'ataque',
  'defesa',
  'utilidade',
  'sentido',
  'social',
  'movimento',
  'magia',
]

describe('GRANTED_POWERS — shape & invariants', () => {
  it('has between 30 and 40 entries (target catalog size)', () => {
    expect(GRANTED_POWERS.length).toBeGreaterThanOrEqual(30)
    expect(GRANTED_POWERS.length).toBeLessThanOrEqual(40)
  })

  it('all ids unique', () => {
    const ids = GRANTED_POWERS.map((p) => p.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('all names unique', () => {
    const names = GRANTED_POWERS.map((p) => p.name)
    expect(new Set(names).size).toBe(names.length)
  })

  it('every entry has non-empty effect and at least 1 deus', () => {
    for (const p of GRANTED_POWERS) {
      expect(p.effect.length).toBeGreaterThan(20)
      expect(p.deuses.length).toBeGreaterThan(0)
    }
  })

  it('every kind is one of the known GrantedPowerKind values', () => {
    for (const p of GRANTED_POWERS) {
      expect(KINDS).toContain(p.kind)
    }
  })

  it('every bookPage falls in the PDF "Poderes Concedidos" range 132-136', () => {
    for (const p of GRANTED_POWERS) {
      expect(p.bookPage).toBeGreaterThanOrEqual(132)
      expect(p.bookPage).toBeLessThanOrEqual(136)
    }
  })

  it('catalog is frozen', () => {
    expect(Object.isFrozen(GRANTED_POWERS)).toBe(true)
  })
})

describe('GRANTED_POWERS — cross-reference vs DEUSES', () => {
  const deusesByName = new Map(DEUSES.map((d) => [d.name, d]))

  it('every deus referenced is a known major deus', () => {
    for (const p of GRANTED_POWERS) {
      for (const dName of p.deuses) {
        const deus = deusesByName.get(dName)
        expect(deus, `deus ${dName} (poder ${p.name})`).toBeDefined()
        expect(deus!.major).toBe(true)
      }
    }
  })

  it('every (deus, poder) pair exists in Deus.poderesConcedidos[]', () => {
    for (const p of GRANTED_POWERS) {
      for (const dName of p.deuses) {
        const deus = deusesByName.get(dName)!
        expect(
          deus.poderesConcedidos,
          `deus ${dName} missing poderesConcedidos`,
        ).toBeDefined()
        expect(deus.poderesConcedidos).toContain(p.name)
      }
    }
  })

  it('Coragem Total granted by 4 deuses (Arsenal/Khalmyr/Lin-Wu/Valkaria)', () => {
    const p = grantedPowerByName('Coragem Total')!
    expect(p.deuses).toEqual(['Arsenal', 'Khalmyr', 'Lin-Wu', 'Valkaria'])
  })

  it('Êxtase da Loucura granted by Aharadak + Nimb', () => {
    const p = grantedPowerByName('Êxtase da Loucura')!
    expect(p.deuses).toEqual(['Aharadak', 'Nimb'])
  })

  it('Presas Primordiais granted by Kallyadranoch + Megalokk', () => {
    const p = grantedPowerByName('Presas Primordiais')!
    expect(p.deuses).toEqual(['Kallyadranoch', 'Megalokk'])
  })

  it('Ataque Piedoso granted by Lena + Thyatis', () => {
    const p = grantedPowerByName('Ataque Piedoso')!
    expect(p.deuses).toEqual(['Lena', 'Thyatis'])
  })
})

describe('GRANTED_POWERS — pinned canonical entries (PDF integrity)', () => {
  it('Espada Justiceira: Khalmyr, ataque, 1 PM, p133', () => {
    const p = grantedPowerById('espada-justiceira')!
    expect(p.deuses).toEqual(['Khalmyr'])
    expect(p.kind).toBe('ataque')
    expect(p.effect).toMatch(/1 PM/)
    expect(p.bookPage).toBe(133)
  })

  it('Dedo Verde: Allihanna, magia, learns Controlar Plantas', () => {
    const p = grantedPowerById('dedo-verde')!
    expect(p.deuses).toEqual(['Allihanna'])
    expect(p.kind).toBe('magia')
    expect(p.effect).toMatch(/Controlar Plantas/)
  })

  it('Bênção do Mana: Wynna, magia, +1 PM cada nível ímpar', () => {
    const p = grantedPowerById('bencao-do-mana')!
    expect(p.deuses).toEqual(['Wynna'])
    expect(p.kind).toBe('magia')
    expect(p.effect).toMatch(/\+1 PM/)
  })

  it('Forma de Macaco: Hyninn, movimento (transformação)', () => {
    const p = grantedPowerById('forma-de-macaco')!
    expect(p.deuses).toEqual(['Hyninn'])
    expect(p.kind).toBe('movimento')
    expect(p.effect).toMatch(/macaco/)
  })

  it('Curandeira Perfeita: Lena, utilidade, "escolher 10" em Cura', () => {
    const p = grantedPowerById('curandeira-perfeita')!
    expect(p.deuses).toEqual(['Lena'])
    expect(p.kind).toBe('utilidade')
    expect(p.effect).toMatch(/escolher 10/)
  })

  it('Servos do Dragão: Kallyadranoch, invoca 2d4+1 kobolds', () => {
    const p = grantedPowerById('servos-do-dragao')!
    expect(p.deuses).toEqual(['Kallyadranoch'])
    expect(p.effect).toMatch(/2d4\+1 kobolds/)
  })
})

describe('GRANTED_POWERS — lookup helpers', () => {
  it('grantedPowerById returns the matching entry', () => {
    expect(grantedPowerById('coragem-total')?.name).toBe('Coragem Total')
  })

  it('grantedPowerById returns undefined for unknown id', () => {
    expect(grantedPowerById('not-a-power')).toBeUndefined()
  })

  it('grantedPowerByName returns the matching entry', () => {
    expect(grantedPowerByName('Espada Justiceira')?.id).toBe(
      'espada-justiceira',
    )
  })

  it('grantedPowersByDeus("Khalmyr") returns 4 poderes', () => {
    const powers = grantedPowersByDeus('Khalmyr')
    expect(powers.length).toBe(4)
    const ids = powers.map((p) => p.id).sort()
    expect(ids).toEqual([
      'coragem-total',
      'dom-da-verdade',
      'espada-justiceira',
      'reparar-injustica',
    ])
  })

  it('grantedPowersByDeus("Allihanna") returns 4 poderes (all encoded)', () => {
    expect(grantedPowersByDeus('Allihanna').length).toBe(4)
  })

  it('grantedPowersByDeus returns [] for a deus with no encoded poderes', () => {
    expect(grantedPowersByDeus('Oceano')).toEqual([])
    expect(grantedPowersByDeus('Sszzaas')).toEqual([])
  })

  it('grantedPowersByKind returns only entries of that kind', () => {
    for (const kind of KINDS) {
      const slice = grantedPowersByKind(kind)
      for (const p of slice) {
        expect(p.kind).toBe(kind)
      }
    }
  })

  it('kind distribution covers every GrantedPowerKind', () => {
    const present = new Set<GrantedPower['kind']>()
    for (const p of GRANTED_POWERS) present.add(p.kind)
    for (const kind of KINDS) {
      expect(present.has(kind)).toBe(true)
    }
  })
})
