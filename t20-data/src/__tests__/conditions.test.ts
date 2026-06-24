import { describe, expect, it } from 'vitest'
import {
  CONDITIONS,
  CONDITION_IDS,
  CONDITION_TAGS,
  applyCondition,
  conditionUpgradeChain,
  removeCondition,
  type ConditionId,
} from '../conditions'

/**
 * PDF book p394-395 — Lista de Condições. Pinned:
 *   - the 35 condições present in the rulebook
 *   - the four upgrade chains (Abalado→Apavorado, Fraco→Debilitado→
 *     Inconsciente, Frustrado→Esmorecido, Fatigado→Exausto→Inconsciente)
 *   - the *tipo de efeito* tag set
 *
 * Behavioural tests cover the apply / remove helpers + the chain walker.
 */
const BOOK_CONDITIONS: readonly ConditionId[] = [
  'abalado',
  'agarrado',
  'alquebrado',
  'apavorado',
  'atordoado',
  'caido',
  'cego',
  'confuso',
  'debilitado',
  'desprevenido',
  'doente',
  'em-chamas',
  'enfeitiçado',
  'enjoado',
  'enredado',
  'envenenado',
  'esmorecido',
  'exausto',
  'fascinado',
  'fatigado',
  'fraco',
  'frustrado',
  'imovel',
  'inconsciente',
  'indefeso',
  'lento',
  'ofuscado',
  'paralisado',
  'pasmo',
  'petrificado',
  'sangrando',
  'sobrecarregado',
  'surdo',
  'surpreendido',
  'vulneravel',
]

describe('CONDITIONS — catalog completeness vs PDF', () => {
  it('lists 35 condições from book p394-395', () => {
    expect(CONDITION_IDS.length).toBe(35)
    expect([...CONDITION_IDS].sort()).toEqual([...BOOK_CONDITIONS].sort())
  })

  it('every CONDITION_IDS entry has a CONDITIONS record', () => {
    for (const id of CONDITION_IDS) {
      const entry = CONDITIONS[id]
      expect(entry, `missing ${id}`).toBeDefined()
      expect(entry.id).toBe(id)
      expect(entry.name).toBeTruthy()
      expect(entry.description).toBeTruthy()
    }
  })

  it('every tag belongs to the known set', () => {
    const valid = new Set<string>(CONDITION_TAGS)
    const bad: string[] = []
    for (const c of Object.values(CONDITIONS)) {
      for (const tag of c.tags) {
        if (!valid.has(tag)) bad.push(`${c.id}: ${tag}`)
      }
    }
    expect(bad).toEqual([])
  })

  it('every upgradesTo target resolves to a known condition', () => {
    const ids = new Set<ConditionId>(CONDITION_IDS)
    const bad: string[] = []
    for (const c of Object.values(CONDITIONS)) {
      if (c.upgradesTo && !ids.has(c.upgradesTo)) {
        bad.push(`${c.id} → ${c.upgradesTo}`)
      }
    }
    expect(bad).toEqual([])
  })
})

describe('CONDITIONS — upgrade chains pinned vs PDF', () => {
  it('Abalado upgrades to Apavorado', () => {
    expect(CONDITIONS.abalado.upgradesTo).toBe('apavorado')
    expect(CONDITIONS.apavorado.upgradesTo).toBeUndefined()
  })

  it('Fraco → Debilitado → Inconsciente', () => {
    expect(CONDITIONS.fraco.upgradesTo).toBe('debilitado')
    expect(CONDITIONS.debilitado.upgradesTo).toBe('inconsciente')
    expect(CONDITIONS.inconsciente.upgradesTo).toBeUndefined()
  })

  it('Frustrado upgrades to Esmorecido (no further)', () => {
    expect(CONDITIONS.frustrado.upgradesTo).toBe('esmorecido')
    expect(CONDITIONS.esmorecido.upgradesTo).toBeUndefined()
  })

  it('Fatigado → Exausto → Inconsciente', () => {
    expect(CONDITIONS.fatigado.upgradesTo).toBe('exausto')
    expect(CONDITIONS.exausto.upgradesTo).toBe('inconsciente')
  })

  it('no other conditions declare an upgrade chain', () => {
    const upgraders = new Set<ConditionId>([
      'abalado',
      'fraco',
      'debilitado',
      'frustrado',
      'fatigado',
      'exausto',
    ])
    for (const c of Object.values(CONDITIONS)) {
      if (upgraders.has(c.id)) continue
      expect(c.upgradesTo, `${c.id} unexpectedly upgrades`).toBeUndefined()
    }
  })
})

describe('CONDITIONS — selected effect tags', () => {
  it('Abalado / Apavorado carry the Medo tag', () => {
    expect(CONDITIONS.abalado.tags).toContain('medo')
    expect(CONDITIONS.apavorado.tags).toContain('medo')
  })

  it('Fatigado / Exausto carry the Cansaço tag', () => {
    expect(CONDITIONS.fatigado.tags).toContain('cansaco')
    expect(CONDITIONS.exausto.tags).toContain('cansaco')
  })

  it('Sangrando carries the Metabolismo tag', () => {
    expect(CONDITIONS.sangrando.tags).toContain('metabolismo')
  })

  it('Cego / Surdo / Ofuscado carry the Sentidos tag', () => {
    expect(CONDITIONS.cego.tags).toContain('sentidos')
    expect(CONDITIONS.surdo.tags).toContain('sentidos')
    expect(CONDITIONS.ofuscado.tags).toContain('sentidos')
  })
})

describe('applyCondition — first application + upgrade', () => {
  it('adds a fresh condition to an empty set', () => {
    const out = applyCondition(new Set(), 'abalado')
    expect([...out]).toEqual(['abalado'])
  })

  it('returns a new Set (no mutation of the input)', () => {
    const before = new Set<ConditionId>()
    const after = applyCondition(before, 'abalado')
    expect(before.size).toBe(0)
    expect(after).not.toBe(before)
  })

  it('upgrades on re-apply: Abalado + Abalado → Apavorado', () => {
    let s = applyCondition(new Set(), 'abalado')
    s = applyCondition(s, 'abalado')
    expect(s.has('abalado')).toBe(false)
    expect(s.has('apavorado')).toBe(true)
  })

  it('walks the full Fraco→Debilitado→Inconsciente ladder', () => {
    let s = applyCondition(new Set(), 'fraco')
    s = applyCondition(s, 'fraco') // → debilitado
    expect(s.has('debilitado')).toBe(true)
    s = applyCondition(s, 'debilitado') // → inconsciente
    expect(s.has('inconsciente')).toBe(true)
    expect(s.has('debilitado')).toBe(false)
  })

  it('re-applying a top-of-chain condition is a no-op (already worst)', () => {
    // Apavorado has no upgrade; a second apavorado leaves the set alone.
    const s = applyCondition(new Set(['apavorado']), 'apavorado')
    expect([...s]).toEqual(['apavorado'])
  })

  it('re-applying a condition without an upgrade is also a no-op', () => {
    const s = applyCondition(new Set(['caido']), 'caido')
    expect([...s]).toEqual(['caido'])
  })
})

describe('removeCondition', () => {
  it('removes the condition when present', () => {
    const s = removeCondition(new Set(['fraco', 'caido']), 'fraco')
    expect([...s]).toEqual(['caido'])
  })

  it('is a no-op for an absent condition', () => {
    const s = removeCondition(new Set(['caido']), 'fraco')
    expect([...s]).toEqual(['caido'])
  })
})

describe('conditionUpgradeChain', () => {
  it('returns the full Fraco ladder', () => {
    expect(conditionUpgradeChain('fraco')).toEqual([
      'fraco',
      'debilitado',
      'inconsciente',
    ])
  })

  it('returns the full Fatigado ladder', () => {
    expect(conditionUpgradeChain('fatigado')).toEqual([
      'fatigado',
      'exausto',
      'inconsciente',
    ])
  })

  it('returns Abalado → Apavorado (no further)', () => {
    expect(conditionUpgradeChain('abalado')).toEqual([
      'abalado',
      'apavorado',
    ])
  })

  it('returns just the id for a condition without upgrades', () => {
    expect(conditionUpgradeChain('caido')).toEqual(['caido'])
  })
})
