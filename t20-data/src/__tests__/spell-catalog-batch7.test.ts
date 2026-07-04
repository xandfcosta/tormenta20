import { describe, expect, it } from 'vitest'
import { SPELL_IDS, spellById } from '../spell-catalog'

/**
 * Batch 7 do catálogo — 10 magias verbatim do PDF p190-192.
 */

const BATCH7_IDS = [
  'despertar-consciencia',
  'dificultar-deteccao',
  'dispersar-as-trevas',
  'disfarce-ilusorio',
  'duplicata-ilusoria',
  'engenho-de-mana',
  'enxame-de-pestes',
  'enxame-rubro-de-ichabod',
  'erupcao-glacial',
  'esculpir-sons',
] as const

describe('SPELL_CATALOG batch7 — presença', () => {
  it('as 10 magias estão no catálogo', () => {
    for (const id of BATCH7_IDS) {
      expect(SPELL_IDS).toContain(id)
    }
  })
})

describe('Despertar Consciência — p190', () => {
  const spell = spellById('despertar-consciencia')

  it('círculo 3, encantamento, completa, toque, dia', () => {
    expect(spell.circle).toBe(3)
    expect(spell.school).toBe('encantamento')
    expect(spell.execution).toBe('completa')
    expect(spell.range).toBe('toque')
    expect(spell.duration).toBe('dia')
    expect(spell.bookPage).toBe(190)
  })
})

describe('Dificultar Detecção — p190', () => {
  const spell = spellById('dificultar-deteccao')

  it('círculo 3, abjuração, toque, dia, Arc/Bar', () => {
    expect(spell.circle).toBe(3)
    expect(spell.school).toBe('abjuracao')
    expect(spell.classes).toEqual(['Arcanista', 'Bardo'])
    expect(spell.bookPage).toBe(190)
  })
})

describe('Dispersar as Trevas — p191', () => {
  const spell = spellById('dispersar-as-trevas')

  it('círculo 3, evocação, pessoal, definida', () => {
    expect(spell.circle).toBe(3)
    expect(spell.school).toBe('evocacao')
    expect(spell.duration).toBe('definida')
    expect(spell.durationNote).toBe('veja texto')
    expect(spell.bookPage).toBe(191)
  })

  it('augments +4 gated 4º, +9 gated 5º (círc de magia dissipada)', () => {
    const c4 = spell.augments.find((a) => a.requiresCircle === 4)
    const c5 = spell.augments.find((a) => a.requiresCircle === 5)
    expect(c4?.pmCost).toBe(4)
    expect(c5?.pmCost).toBe(9)
  })
})

describe('Disfarce Ilusório — p191', () => {
  const spell = spellById('disfarce-ilusorio')

  it('círculo 1, ilusão, pessoal, Vontade desacredita', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('ilusao')
    expect(spell.saveType).toBe('vontade')
    expect(spell.resistance).toBe('desacredita')
    expect(spell.bookPage).toBe(191)
  })

  it('augment truque +0 PM', () => {
    const truque = spell.augments.find((a) => a.pmCost === 0 && a.kind === 'muda')
    expect(truque?.description).toMatch(/[Tt]ruque/)
  })
})

describe('Duplicata Ilusória — p191', () => {
  const spell = spellById('duplicata-ilusoria')

  it('círculo 4, ilusão, médio, cena', () => {
    expect(spell.circle).toBe(4)
    expect(spell.school).toBe('ilusao')
    expect(spell.range).toBe('medio')
    expect(spell.duration).toBe('cena')
    expect(spell.bookPage).toBe(191)
  })
})

describe('Engenho de Mana — p192', () => {
  const spell = spellById('engenho-de-mana')

  it('círculo 5, abjuração, médio, sustentada, Arc/Bar', () => {
    expect(spell.circle).toBe(5)
    expect(spell.school).toBe('abjuracao')
    expect(spell.duration).toBe('sustentada')
    expect(spell.classes).toEqual(['Arcanista', 'Bardo'])
    expect(spell.bookPage).toBe(192)
  })
})

describe('Enxame de Pestes — p192', () => {
  const spell = spellById('enxame-de-pestes')

  it('círculo 2, convocação, completa, Fortitude metade', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('convocacao')
    expect(spell.execution).toBe('completa')
    expect(spell.saveType).toBe('fortitude')
    expect(spell.resistance).toBe('metade')
    expect(spell.bookPage).toBe(192)
  })

  it('augments +5 gated 3º, +7 gated 4º', () => {
    const c3 = spell.augments.find((a) => a.pmCost === 5)
    const c4 = spell.augments.find((a) => a.pmCost === 7)
    expect(c3?.requiresCircle).toBe(3)
    expect(c4?.requiresCircle).toBe(4)
  })
})

describe('Enxame Rubro de Ichabod — p192', () => {
  const spell = spellById('enxame-rubro-de-ichabod')

  it('círculo 3, convocação, Reflexos metade, sustentada', () => {
    expect(spell.circle).toBe(3)
    expect(spell.school).toBe('convocacao')
    expect(spell.saveType).toBe('reflexos')
    expect(spell.resistance).toBe('metade')
    expect(spell.duration).toBe('sustentada')
    expect(spell.bookPage).toBe(192)
  })

  it('6 augments', () => {
    expect(spell.augments.length).toBe(6)
  })
})

describe('Erupção Glacial — p192', () => {
  const spell = spellById('erupcao-glacial')

  it('círculo 3, evocação, médio, Reflexos parcial', () => {
    expect(spell.circle).toBe(3)
    expect(spell.school).toBe('evocacao')
    expect(spell.saveType).toBe('reflexos')
    expect(spell.resistance).toBe('parcial')
    expect(spell.bookPage).toBe(192)
  })
})

describe('Esculpir Sons — p192', () => {
  const spell = spellById('esculpir-sons')

  it('círculo 2, ilusão, médio, cena, Vontade anula', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('ilusao')
    expect(spell.range).toBe('medio')
    expect(spell.saveType).toBe('vontade')
    expect(spell.resistance).toBe('anula')
    expect(spell.bookPage).toBe(192)
  })
})
