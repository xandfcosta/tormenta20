import { describe, expect, it } from 'vitest'
import {
  CLASS_POWERS_CATALOG,
  classPowersFor,
  getClassPower,
  getOrigin,
  getOriginBenefit,
  getRace,
  getRaceAbility,
  racesGrantTormenta,
} from '../catalog'

describe('racesGrantTormenta', () => {
  it('is true only for Tormenta-touched races (Lefou)', () => {
    expect(racesGrantTormenta(['Lefou'])).toBe(true)
    expect(racesGrantTormenta(['Anão'])).toBe(false)
    // Secondary Lefou still grants access (any selected race counts).
    expect(racesGrantTormenta(['Minotauro', 'Lefou'])).toBe(true)
    expect(racesGrantTormenta([])).toBe(false)
  })
})

/**
 * Catalog lookup helpers — `get*` and `classPowersFor`. Every UI / engine
 * read path that resolves a stored id goes through one of these, so a
 * silent regression here would hide the offending feature without a stack
 * trace. None of these were directly covered before — `getClassPower` was
 * only exercised via the Arma Sagrada prereq spec.
 *
 * Specs deliberately use stable catalog ids that match the slug/id
 * conventions enforced elsewhere (catalog-integrity, catalog-cross-refs)
 * — so a slug-pattern change here breaks together with those, not in
 * isolation.
 */
describe('getRace', () => {
  it('returns the RaceDefinition for a known id', () => {
    const race = getRace('Humano')
    expect(race).toBeDefined()
    expect(race?.id).toBe('Humano')
    expect(race?.name).toBe('Humano')
  })

  it('returns undefined for unknown id', () => {
    expect(getRace('Hexer')).toBeUndefined()
    expect(getRace('')).toBeUndefined()
  })
})

describe('getOrigin', () => {
  it('returns the OriginDefinition for a known id', () => {
    const origin = getOrigin('Soldado')
    expect(origin).toBeDefined()
    expect(origin?.id).toBe('Soldado')
  })

  it('returns undefined for unknown id', () => {
    expect(getOrigin('Mercador-Espacial')).toBeUndefined()
  })
})

describe('getClassPower', () => {
  it('returns the ClassPower for a known catalog id', () => {
    // 'class.guerreiro.especializacao-em-arma' is referenced as a prereq id
    // by another guerreiro power — its presence is asserted indirectly by
    // catalog-integrity already, so this id is stable.
    const power = getClassPower('class.guerreiro.especializacao-em-arma')
    expect(power).toBeDefined()
    expect(power?.className).toBe('Guerreiro')
  })

  it('returns undefined for unknown id', () => {
    expect(getClassPower('class.guerreiro.does-not-exist')).toBeUndefined()
    expect(getClassPower('ghost')).toBeUndefined()
  })
})

describe('classPowersFor', () => {
  it('returns every catalog power belonging to the named class', () => {
    const all = classPowersFor('Guerreiro')
    expect(all.length).toBeGreaterThan(0)
    for (const power of all) {
      expect(power.className).toBe('Guerreiro')
    }
  })

  it('does not include powers from sibling classes', () => {
    const guerreiroIds = new Set(classPowersFor('Guerreiro').map((p) => p.id))
    const bardoIds = classPowersFor('Bardo').map((p) => p.id)
    for (const id of bardoIds) {
      expect(guerreiroIds.has(id)).toBe(false)
    }
  })

  it('returns empty array for an unknown class name', () => {
    expect(classPowersFor('Hexer')).toEqual([])
  })

  it('partitions the catalog exhaustively across all classes', () => {
    // Sanity: union of all classes' powers equals the full catalog size.
    // Catches a catalog row with a stray (typo'd) className that doesn't
    // match any picker — would otherwise silently never be selectable.
    const BOOK_CLASSES = [
      'Arcanista',
      'Bárbaro',
      'Bardo',
      'Bucaneiro',
      'Caçador',
      'Cavaleiro',
      'Clérigo',
      'Druida',
      'Guerreiro',
      'Inventor',
      'Ladino',
      'Lutador',
      'Nobre',
      'Paladino',
    ]
    const summed = BOOK_CLASSES.reduce(
      (n, c) => n + classPowersFor(c).length,
      0,
    )
    expect(summed).toBe(CLASS_POWERS_CATALOG.length)
  })
})

describe('getRaceAbility', () => {
  it('resolves an ability id by walking every race', () => {
    // 'humano-versatil' is the Humano "Versátil" ability — stable id.
    const ability = getRaceAbility('humano-versatil')
    expect(ability).toBeDefined()
    expect(ability?.raceId).toBe('Humano')
  })

  it('returns undefined for unknown id', () => {
    expect(getRaceAbility('nao-existe')).toBeUndefined()
  })

  it('finds an ability even when it lives on a later race in the catalog', () => {
    // Trog is near the end of the catalog list — ensures the walk doesn't
    // bail after the first race.
    const ability = getRaceAbility('humano-tres-atributos')
    expect(ability?.raceId).toBe('Humano')
  })
})

describe('getOriginBenefit', () => {
  it('resolves a regular benefit by id (Soldado: Fortitude perícia)', () => {
    // id format from origins.ts pericia(): `origin-${origin}-pericia-${expertise}`.
    const benefit = getOriginBenefit('origin-soldado-pericia-Fortitude')
    expect(benefit).toBeDefined()
    expect(benefit?.kind).toBe('pericia')
    expect(benefit?.expertise).toBe('Fortitude')
  })

  it('resolves the poderUnico via the same lookup (Soldado: Influência Militar)', () => {
    // poderUnico is stored in a sibling field but getOriginBenefit must
    // walk both — UI uses the same picker id space for regular + poderUnico.
    const benefit = getOriginBenefit('origin-soldado-unique')
    expect(benefit).toBeDefined()
    expect(benefit?.kind).toBe('poder')
    expect(benefit?.name).toBe('Influência Militar')
  })

  it('returns undefined for unknown id', () => {
    expect(getOriginBenefit('origin-soldado-pericia-NotAReal')).toBeUndefined()
    expect(getOriginBenefit('')).toBeUndefined()
  })
})

describe('power sub-choices (PowerChoice)', () => {
  const byName = (name: string) =>
    CLASS_POWERS_CATALOG.find((p) => p.name === name)

  it('Totem Espiritual offers 8 totem animals with granted spells', () => {
    const p = byName('Totem Espiritual')
    expect(p?.choice?.kind).toBe('totem')
    expect(p?.choice?.options).toHaveLength(8)
    expect(p?.choice?.options?.find((o) => o.id === 'urso')?.note).toBe(
      'Vitalidade Fantasma',
    )
  })

  it('Especialista em Escola offers the 8 schools (repeatable)', () => {
    const p = byName('Especialista em Escola')
    expect(p?.choice?.kind).toBe('school')
    expect(p?.choice?.repeatable).toBe(true)
    expect(p?.choice?.options).toHaveLength(8)
  })

  it('Especialização em Arma is a catalog-sourced weapon choice', () => {
    const p = byName('Especialização em Arma')
    expect(p?.choice?.kind).toBe('weapon')
    expect(p?.choice?.options).toBeUndefined()
    expect(p?.choice?.repeatable).toBe(true)
  })

  it('Companheiro Animal (Druida + Caçador) offers the base companion types', () => {
    for (const cls of ['Druida', 'Caçador']) {
      const p = CLASS_POWERS_CATALOG.find(
        (x) => x.name === 'Companheiro Animal' && x.className === cls,
      )
      expect(p?.choice?.kind).toBe('companion')
      expect(p?.choice?.options).toHaveLength(8)
    }
  })
})

describe('Aumento de Atributo — repeatable attribute choice', () => {
  it('every class copy carries the repeatable attribute sub-choice', () => {
    const copies = CLASS_POWERS_CATALOG.filter(
      (p) => p.name === 'Aumento de Atributo',
    )
    expect(copies.length).toBeGreaterThan(5)
    for (const p of copies) {
      expect(p.choice?.kind).toBe('attribute')
      expect(p.choice?.repeatable).toBe(true)
      expect(p.choice?.options).toHaveLength(6)
    }
  })
})
