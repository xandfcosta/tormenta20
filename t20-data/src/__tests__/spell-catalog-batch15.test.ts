import { describe, expect, it } from 'vitest'
import { SPELL_IDS, spellById } from '../spell-catalog'

/**
 * Batch 15 do catálogo — 10 magias verbatim do PDF p206-207.
 */

const BATCH15_IDS = [
  'servo-morto-vivo',
  'servos-invisiveis',
  'seta-infalivel-de-talude',
  'silencio',
  'soco-de-arsenal',
  'sombra-assassina',
  'sonho',
  'sopro-da-salvacao',
  'suporte-ambiental',
  'sussurros-insanos',
] as const

describe('SPELL_CATALOG batch15 — presença', () => {
  it('as 10 magias estão no catálogo', () => {
    for (const id of BATCH15_IDS) {
      expect(SPELL_IDS).toContain(id)
    }
  })
})

describe('Servo Morto-Vivo — p206', () => {
  const spell = spellById('servo-morto-vivo')

  it('círculo 3, necromancia, completa, componente material, universal 5', () => {
    expect(spell.circle).toBe(3)
    expect(spell.school).toBe('necromancia')
    expect(spell.execution).toBe('completa')
    expect(spell.components).toContain('material')
    expect(spell.classes).toEqual([
      'Arcanista',
      'Bardo',
      'Clérigo',
      'Druida',
      'Paladino',
    ])
    expect(spell.bookPage).toBe(206)
  })

  it('augment +7 PM múmia gated 4º', () => {
    const aug = spell.augments.find((a) => a.pmCost === 7)
    expect(aug?.requiresCircle).toBe(4)
  })
})

describe('Servos Invisíveis — p206', () => {
  const spell = spellById('servos-invisiveis')

  it('círculo 2, convocação, longo, cena, Arc/Bar', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('convocacao')
    expect(spell.range).toBe('longo')
    expect(spell.classes).toEqual(['Arcanista', 'Bardo'])
    expect(spell.bookPage).toBe(206)
  })
})

describe('Seta Infalível de Talude — p206', () => {
  const spell = spellById('seta-infalivel-de-talude')

  it('círculo 1, evocação, médio, sem save', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('evocacao')
    expect(spell.range).toBe('medio')
    expect(spell.saveType).toBe('none')
    expect(spell.bookPage).toBe(206)
  })

  it('augments escalados 2º/4º círc', () => {
    const c2 = spell.augments.filter((a) => a.requiresCircle === 2)
    const c4 = spell.augments.find((a) => a.requiresCircle === 4)
    expect(c2.length).toBe(2)
    expect(c4?.pmCost).toBe(9)
  })
})

describe('Silêncio — p206', () => {
  const spell = spellById('silencio')

  it('círculo 2, ilusão, médio, sustentada, Cle/Dru/Pal', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('ilusao')
    expect(spell.duration).toBe('sustentada')
    expect(spell.classes).toEqual(['Clérigo', 'Druida', 'Paladino'])
    expect(spell.bookPage).toBe(206)
  })
})

describe('Soco de Arsenal — p206', () => {
  const spell = spellById('soco-de-arsenal')

  it('círculo 2, convocação, médio, Fortitude parcial, Cle/Dru/Pal', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('convocacao')
    expect(spell.saveType).toBe('fortitude')
    expect(spell.resistance).toBe('parcial')
    expect(spell.classes).toEqual(['Clérigo', 'Druida', 'Paladino'])
    expect(spell.bookPage).toBe(206)
  })
})

describe('Sombra Assassina — p207', () => {
  const spell = spellById('sombra-assassina')

  it('círculo 5, ilusão, Vontade parcial, cena, Arc/Bar', () => {
    expect(spell.circle).toBe(5)
    expect(spell.school).toBe('ilusao')
    expect(spell.saveType).toBe('vontade')
    expect(spell.resistance).toBe('parcial')
    expect(spell.duration).toBe('cena')
    expect(spell.bookPage).toBe(207)
  })
})

describe('Sonho — p207', () => {
  const spell = spellById('sonho')

  it('círculo 4, adivinhação, completa, ilimitado, definida', () => {
    expect(spell.circle).toBe(4)
    expect(spell.school).toBe('adivinhacao')
    expect(spell.execution).toBe('completa')
    expect(spell.range).toBe('ilimitado')
    expect(spell.duration).toBe('definida')
    expect(spell.durationNote).toBe('veja texto')
    expect(spell.bookPage).toBe(207)
  })
})

describe('Sopro da Salvação — p207', () => {
  const spell = spellById('sopro-da-salvacao')

  it('círculo 3, evocação, pessoal, instantânea, Cle/Dru/Pal', () => {
    expect(spell.circle).toBe(3)
    expect(spell.school).toBe('evocacao')
    expect(spell.range).toBe('pessoal')
    expect(spell.duration).toBe('instantanea')
    expect(spell.classes).toEqual(['Clérigo', 'Druida', 'Paladino'])
    expect(spell.bookPage).toBe(207)
  })
})

describe('Suporte Ambiental — p207', () => {
  const spell = spellById('suporte-ambiental')

  it('círculo 1, abjuração, toque, dia, Cle/Dru/Pal', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('abjuracao')
    expect(spell.duration).toBe('dia')
    expect(spell.classes).toEqual(['Clérigo', 'Druida', 'Paladino'])
    expect(spell.bookPage).toBe(207)
  })
})

describe('Sussurros Insanos — p207', () => {
  const spell = spellById('sussurros-insanos')

  it('círculo 2, encantamento, Vontade anula, Arc/Bar', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('encantamento')
    expect(spell.saveType).toBe('vontade')
    expect(spell.resistance).toBe('anula')
    expect(spell.classes).toEqual(['Arcanista', 'Bardo'])
    expect(spell.bookPage).toBe(207)
  })

  it('augment +12 PM gated 5º criaturas escolhidas', () => {
    const aug = spell.augments.find((a) => a.pmCost === 12)
    expect(aug?.requiresCircle).toBe(5)
  })
})
