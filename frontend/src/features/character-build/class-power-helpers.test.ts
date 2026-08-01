import { describe, expect, it } from 'vitest'
import {
  type PowerOption,
  powerChoiceOptions,
  tormentaPowerOptions,
  usedSlots,
} from './class-power-helpers'

describe('tormentaPowerOptions', () => {
  it('returns the 22 poderes da Tormenta, all source "tormenta"', () => {
    const opts = tormentaPowerOptions()
    expect(opts).toHaveLength(22)
    expect(opts.every((o) => o.source === 'tormenta')).toBe(true)
  })

  it('encodes a specific-power prereq (Larva Explosiva ← Dentes Afiados)', () => {
    const larva = tormentaPowerOptions().find((o) => o.id === 'larva-explosiva')
    expect(larva?.prerequisites).toContainEqual({
      kind: 'power',
      id: 'dentes-afiados',
    })
  })
})

const opt = (id: string, repeatable?: boolean): PowerOption => ({
  id,
  name: id,
  description: '',
  minLevel: 1,
  prerequisites: [],
  source: 'class',
  choice: repeatable
    ? { kind: 'attribute', label: 'Atributo', repeatable: true }
    : undefined,
})

describe('usedSlots — repeatable powers count per sub-choice', () => {
  const byId = new Map([
    ['rep', opt('rep', true)],
    ['single', opt('single')],
  ])

  it('a plain power uses one slot', () => {
    expect(usedSlots(['single'], {}, byId)).toBe(1)
  })

  it('a repeatable power uses one slot per pick', () => {
    expect(usedSlots(['rep'], { rep: ['strength', 'dexterity', 'wisdom'] }, byId)).toBe(3)
  })

  it('a repeatable power with no picks yet uses zero slots', () => {
    expect(usedSlots(['rep'], {}, byId)).toBe(0)
  })

  it('sums plain + repeatable', () => {
    expect(usedSlots(['single', 'rep'], { rep: ['strength', 'charisma'] }, byId)).toBe(3)
  })
})

describe('powerChoiceOptions', () => {
  it('returns the inline options for enumerated kinds', () => {
    const opts = powerChoiceOptions({
      kind: 'attribute',
      label: 'Atributo',
      options: [{ id: 'strength', name: 'Força' }],
    })
    expect(opts).toEqual([{ id: 'strength', name: 'Força' }])
  })

  it('sources weapons from the catalog when options are omitted', () => {
    const opts = powerChoiceOptions({ kind: 'weapon', label: 'Arma' })
    expect(opts.length).toBeGreaterThan(0)
    expect(opts[0]).toHaveProperty('name')
  })

  it('attaches a spell effect as desc when a totem note names a spell', () => {
    const [urso] = powerChoiceOptions({
      kind: 'totem',
      label: 'Animal totêmico',
      options: [{ id: 'urso', name: 'Urso', note: 'Vitalidade Fantasma' }],
    })
    expect(urso.note).toBe('Vitalidade Fantasma')
    expect(urso.desc).toBeTruthy()
  })

  it('leaves desc undefined when a note names no catalog spell', () => {
    const [o] = powerChoiceOptions({
      kind: 'totem',
      label: 'X',
      options: [{ id: 'x', name: 'X', note: 'Não Existe Magia' }],
    })
    expect(o.desc).toBeUndefined()
  })
})
