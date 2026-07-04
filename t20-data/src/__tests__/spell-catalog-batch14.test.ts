import { describe, expect, it } from 'vitest'
import { SPELL_IDS, spellById } from '../spell-catalog'

/**
 * Batch 14 do catálogo — 10 magias verbatim do PDF p204-205.
 */

const BATCH14_IDS = [
  'resistencia-a-energia',
  'rogar-maldicao',
  'roubar-a-alma',
  'runa-de-protecao',
  'salto-dimensional',
  'santuario',
  'segunda-chance',
  'selo-de-mana',
  'semiplano',
  'servo-divino',
] as const

describe('SPELL_CATALOG batch14 — presença', () => {
  it('as 10 magias estão no catálogo', () => {
    for (const id of BATCH14_IDS) {
      expect(SPELL_IDS).toContain(id)
    }
  })
})

describe('Resistência a Energia — p204', () => {
  const spell = spellById('resistencia-a-energia')

  it('círculo 1, abjuração, toque, cena, universal 5', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('abjuracao')
    expect(spell.classes).toEqual([
      'Arcanista',
      'Bardo',
      'Clérigo',
      'Druida',
      'Paladino',
    ])
    expect(spell.bookPage).toBe(204)
  })
})

describe('Rogar Maldição — p204', () => {
  const spell = spellById('rogar-maldicao')

  it('círculo 2, necromancia, sustentada, Fortitude anula, Cle/Paladino', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('necromancia')
    expect(spell.duration).toBe('sustentada')
    expect(spell.saveType).toBe('fortitude')
    expect(spell.resistance).toBe('anula')
    expect(spell.classes).toEqual(['Clérigo', 'Paladino'])
    expect(spell.bookPage).toBe(204)
  })

  it('augments +3 gated 3º, +7 gated 4º', () => {
    const c3 = spell.augments.find((a) => a.pmCost === 3)
    const c4 = spell.augments.find((a) => a.pmCost === 7)
    expect(c3?.requiresCircle).toBe(3)
    expect(c4?.requiresCircle).toBe(4)
  })
})

describe('Roubar a Alma — p204', () => {
  const spell = spellById('roubar-a-alma')

  it('círculo 5, necromancia, toque, permanente, Vontade anula, universal 5', () => {
    expect(spell.circle).toBe(5)
    expect(spell.school).toBe('necromancia')
    expect(spell.duration).toBe('permanente')
    expect(spell.saveType).toBe('vontade')
    expect(spell.resistance).toBe('anula')
    expect(spell.classes).toEqual([
      'Arcanista',
      'Bardo',
      'Clérigo',
      'Druida',
      'Paladino',
    ])
    expect(spell.bookPage).toBe(204)
  })
})

describe('Runa de Proteção — p204', () => {
  const spell = spellById('runa-de-protecao')

  it('círculo 2, abjuração, completa, componente material, universal 5', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('abjuracao')
    expect(spell.execution).toBe('completa')
    expect(spell.components).toContain('material')
    expect(spell.classes).toEqual([
      'Arcanista',
      'Bardo',
      'Clérigo',
      'Druida',
      'Paladino',
    ])
    expect(spell.bookPage).toBe(204)
  })

  it('augment +3 PM gated 3º runa magia 2º círc', () => {
    const aug = spell.augments.find((a) => a.pmCost === 3)
    expect(aug?.requiresCircle).toBe(3)
  })
})

describe('Salto Dimensional — p205', () => {
  const spell = spellById('salto-dimensional')

  it('círculo 2, convocação, curto, instantânea, Arc/Bar', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('convocacao')
    expect(spell.range).toBe('curto')
    expect(spell.duration).toBe('instantanea')
    expect(spell.classes).toEqual(['Arcanista', 'Bardo'])
    expect(spell.bookPage).toBe(205)
  })

  it('4 augments', () => {
    expect(spell.augments.length).toBe(4)
  })
})

describe('Santuário — p205', () => {
  const spell = spellById('santuario')

  it('círculo 1, abjuração, toque, cena, Vontade anula, Cle/Paladino', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('abjuracao')
    expect(spell.saveType).toBe('vontade')
    expect(spell.resistance).toBe('anula')
    expect(spell.classes).toEqual(['Clérigo', 'Paladino'])
    expect(spell.bookPage).toBe(205)
  })
})

describe('Segunda Chance — p205', () => {
  const spell = spellById('segunda-chance')

  it('círculo 5, evocação, toque, instantânea, Cle/Paladino', () => {
    expect(spell.circle).toBe(5)
    expect(spell.school).toBe('evocacao')
    expect(spell.range).toBe('toque')
    expect(spell.duration).toBe('instantanea')
    expect(spell.classes).toEqual(['Clérigo', 'Paladino'])
    expect(spell.bookPage).toBe(205)
  })
})

describe('Selo de Mana — p205', () => {
  const spell = spellById('selo-de-mana')

  it('círculo 3, encantamento, toque, cena, Vontade anula, universal 5', () => {
    expect(spell.circle).toBe(3)
    expect(spell.school).toBe('encantamento')
    expect(spell.classes).toEqual([
      'Arcanista',
      'Bardo',
      'Clérigo',
      'Druida',
      'Paladino',
    ])
    expect(spell.bookPage).toBe(205)
  })
})

describe('Semiplano — p205', () => {
  const spell = spellById('semiplano')

  it('círculo 5, convocação, completa, dia, Arc/Bar', () => {
    expect(spell.circle).toBe(5)
    expect(spell.school).toBe('convocacao')
    expect(spell.execution).toBe('completa')
    expect(spell.duration).toBe('dia')
    expect(spell.classes).toEqual(['Arcanista', 'Bardo'])
    expect(spell.bookPage).toBe(205)
  })
})

describe('Servo Divino — p205', () => {
  const spell = spellById('servo-divino')

  it('círculo 3, convocação, definida, componente material, Cle/Paladino', () => {
    expect(spell.circle).toBe(3)
    expect(spell.school).toBe('convocacao')
    expect(spell.duration).toBe('definida')
    expect(spell.components).toContain('material')
    expect(spell.classes).toEqual(['Clérigo', 'Paladino'])
    expect(spell.bookPage).toBe(205)
  })
})
