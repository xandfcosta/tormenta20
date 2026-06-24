import { describe, expect, it } from 'vitest'
import { ATTRIBUTE_KEYS } from '../../../attributes'
import { EXPERTISE_NAMES } from '../../../expertises'
import { getGeneralPower } from '../../general-powers'
import type { Prerequisite } from '../../types'
import { CLASS_POWERS_CATALOG } from '../index'

/**
 * Cross-reference integrity for class powers — written as regression tests
 * meant to dictate the catalog wire shape:
 *
 *  - power IDs are unique across all classes
 *  - className matches a known PDF class (Tabela 1-5..1-18)
 *  - level fields (grantedAtLevel / minLevel) stay within 1..20
 *  - every typed prerequisite references a real catalog target:
 *      * power / anyPower — id resolves via CLASS_POWERS_CATALOG or
 *        GENERAL_POWERS_CATALOG
 *      * trained — expertise is in EXPERTISE_NAMES
 *      * attribute — attr is a valid AttributeKey and min is in 1..6
 *      * classChoice — class is a real class, field is devoto|caminho,
 *        label is non-empty
 *
 * If a future catalog edit fat-fingers an id (e.g., "class.bardo.musica-bardica"
 * → "class.bardo.musica-bardia"), one of these specs catches the dangling
 * reference instead of leaving a silent dead gate in the UI.
 */
const BOOK_CLASSES = new Set([
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
])

const ATTR_KEYS = new Set<string>(ATTRIBUTE_KEYS)
const KNOWN_EXPERTISES = new Set<string>(EXPERTISE_NAMES)

const idIndex = new Map(CLASS_POWERS_CATALOG.map((p) => [p.id, p]))

function resolvesAnywhere(id: string): boolean {
  return idIndex.has(id) || getGeneralPower(id) !== undefined
}

function prereqLabel(p: Prerequisite): string {
  if (p.kind === 'classChoice') {
    return `${p.kind}(${p.class}/${p.field})`
  }
  if (p.kind === 'power') return `${p.kind}(${p.id})`
  if (p.kind === 'anyPower') return `${p.kind}(${p.ids.join(',')})`
  if (p.kind === 'trained') return `${p.kind}(${p.expertise})`
  if (p.kind === 'attribute') return `${p.kind}(${p.attr}>=${p.min})`
  return p.kind
}

describe('CLASS_POWERS_CATALOG — id uniqueness', () => {
  it('every power id is unique', () => {
    const seen = new Map<string, string>()
    const dupes: string[] = []
    for (const p of CLASS_POWERS_CATALOG) {
      if (seen.has(p.id)) {
        dupes.push(`${p.id} (${seen.get(p.id)} vs ${p.name})`)
      } else {
        seen.set(p.id, p.name)
      }
    }
    expect(dupes).toEqual([])
  })

  it('every id matches the slug format class.<class>.<slug>', () => {
    const bad = CLASS_POWERS_CATALOG.filter(
      (p) => !/^class\.[a-z0-9-]+\.[a-z0-9-]+$/.test(p.id),
    ).map((p) => p.id)
    expect(bad).toEqual([])
  })
})

describe('CLASS_POWERS_CATALOG — className field', () => {
  it('every entry references a real PDF class', () => {
    const unknown = CLASS_POWERS_CATALOG.filter(
      (p) => !BOOK_CLASSES.has(p.className),
    ).map((p) => `${p.id} → ${p.className}`)
    expect(unknown).toEqual([])
  })
})

describe('CLASS_POWERS_CATALOG — level fields', () => {
  it('grantedAtLevel (when present) is in 1..20', () => {
    const bad = CLASS_POWERS_CATALOG.filter(
      (p) =>
        p.grantedAtLevel !== undefined &&
        (p.grantedAtLevel < 1 || p.grantedAtLevel > 20),
    ).map((p) => `${p.id} grantedAtLevel=${p.grantedAtLevel}`)
    expect(bad).toEqual([])
  })

  it('minLevel (when present) is in 1..20', () => {
    const bad = CLASS_POWERS_CATALOG.filter(
      (p) =>
        p.minLevel !== undefined && (p.minLevel < 1 || p.minLevel > 20),
    ).map((p) => `${p.id} minLevel=${p.minLevel}`)
    expect(bad).toEqual([])
  })

  it('auto-powers (grantedAtLevel set) do not also carry minLevel', () => {
    const bad = CLASS_POWERS_CATALOG.filter(
      (p) => p.grantedAtLevel !== undefined && p.minLevel !== undefined,
    ).map((p) => p.id)
    expect(bad).toEqual([])
  })
})

describe('CLASS_POWERS_CATALOG — prerequisite references', () => {
  it('every `power` prereq id resolves to a real catalog entry', () => {
    const dangling: string[] = []
    for (const p of CLASS_POWERS_CATALOG) {
      for (const req of p.prerequisites ?? []) {
        if (req.kind === 'power' && !resolvesAnywhere(req.id)) {
          dangling.push(`${p.id} → power(${req.id})`)
        }
      }
    }
    expect(dangling).toEqual([])
  })

  it('every `anyPower` prereq id resolves and lists are non-empty', () => {
    const dangling: string[] = []
    const empty: string[] = []
    for (const p of CLASS_POWERS_CATALOG) {
      for (const req of p.prerequisites ?? []) {
        if (req.kind !== 'anyPower') continue
        if (req.ids.length === 0) {
          empty.push(`${p.id} → anyPower([])`)
          continue
        }
        for (const id of req.ids) {
          if (!resolvesAnywhere(id)) {
            dangling.push(`${p.id} → anyPower(${id})`)
          }
        }
      }
    }
    expect(empty).toEqual([])
    expect(dangling).toEqual([])
  })

  it('every `trained` prereq names a real expertise', () => {
    const bad: string[] = []
    for (const p of CLASS_POWERS_CATALOG) {
      for (const req of p.prerequisites ?? []) {
        if (req.kind === 'trained' && !KNOWN_EXPERTISES.has(req.expertise)) {
          bad.push(`${p.id} → trained(${req.expertise})`)
        }
      }
    }
    expect(bad).toEqual([])
  })

  it('every `attribute` prereq has a valid attr key and min in 1..6', () => {
    const bad: string[] = []
    for (const p of CLASS_POWERS_CATALOG) {
      for (const req of p.prerequisites ?? []) {
        if (req.kind !== 'attribute') continue
        if (!ATTR_KEYS.has(req.attr) || req.min < 1 || req.min > 6) {
          bad.push(`${p.id} → ${prereqLabel(req)}`)
        }
      }
    }
    expect(bad).toEqual([])
  })

  it('every `classChoice` prereq targets a real class with a non-empty label', () => {
    const bad: string[] = []
    for (const p of CLASS_POWERS_CATALOG) {
      for (const req of p.prerequisites ?? []) {
        if (req.kind !== 'classChoice') continue
        if (!BOOK_CLASSES.has(req.class)) {
          bad.push(`${p.id} unknown class → ${req.class}`)
        }
        if (req.field !== 'devoto' && req.field !== 'caminho') {
          bad.push(`${p.id} bad field → ${req.field}`)
        }
        if (!req.label || req.label.trim().length === 0) {
          bad.push(`${p.id} empty label`)
        }
      }
    }
    expect(bad).toEqual([])
  })

  it('classChoice allowed/forbidden are non-empty when present (avoid no-op gates)', () => {
    const bad: string[] = []
    for (const p of CLASS_POWERS_CATALOG) {
      for (const req of p.prerequisites ?? []) {
        if (req.kind !== 'classChoice') continue
        if (req.allowed !== undefined && req.allowed.length === 0) {
          bad.push(`${p.id} empty allowed list`)
        }
        if (req.forbidden !== undefined && req.forbidden.length === 0) {
          bad.push(`${p.id} empty forbidden list`)
        }
      }
    }
    expect(bad).toEqual([])
  })
})

describe('CLASS_POWERS_CATALOG — descriptions', () => {
  it('every power has a non-empty name and description', () => {
    const bad = CLASS_POWERS_CATALOG.filter(
      (p) =>
        !p.name ||
        p.name.trim().length === 0 ||
        !p.description ||
        p.description.trim().length === 0,
    ).map((p) => p.id)
    expect(bad).toEqual([])
  })
})
