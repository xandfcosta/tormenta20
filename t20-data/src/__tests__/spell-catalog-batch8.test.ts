import { describe, expect, it } from 'vitest'
import { SPELL_IDS, spellById } from '../spell-catalog'

/**
 * Batch 8 do catálogo — 10 magias verbatim do PDF p193-194.
 */

const BATCH8_IDS = [
  'escuridao',
  'explosao-caleidoscopica',
  'ferver-sangue',
  'fisico-divino',
  'flecha-acida',
  'forma-eterea',
  'furia-do-panteao',
  'globo-da-verdade-de-gwen',
  'globo-de-invulnerabilidade',
  'guardiao-divino',
] as const

describe('SPELL_CATALOG batch8 — presença', () => {
  it('as 10 magias estão no catálogo', () => {
    for (const id of BATCH8_IDS) {
      expect(SPELL_IDS).toContain(id)
    }
  })
})

describe('Escuridão — p193', () => {
  const spell = spellById('escuridao')

  it('círculo 1, necromancia, Vontade anula, universal 4 classes (sem Paladino)', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('necromancia')
    expect(spell.saveType).toBe('vontade')
    expect(spell.resistance).toBe('anula')
    expect(spell.classes).toEqual(['Arcanista', 'Bardo', 'Clérigo', 'Druida'])
    expect(spell.bookPage).toBe(193)
  })
})

describe('Explosão Caleidoscópica — p193', () => {
  const spell = spellById('explosao-caleidoscopica')

  it('círculo 4, ilusão, Fortitude parcial, Arcanista apenas, sem augments', () => {
    expect(spell.circle).toBe(4)
    expect(spell.school).toBe('ilusao')
    expect(spell.saveType).toBe('fortitude')
    expect(spell.resistance).toBe('parcial')
    expect(spell.classes).toEqual(['Arcanista'])
    expect(spell.augments).toEqual([])
    expect(spell.bookPage).toBe(193)
  })
})

describe('Ferver Sangue — p193', () => {
  const spell = spellById('ferver-sangue')

  it('círculo 3, necromancia, sustentada, Fortitude parcial', () => {
    expect(spell.circle).toBe(3)
    expect(spell.school).toBe('necromancia')
    expect(spell.duration).toBe('sustentada')
    expect(spell.saveType).toBe('fortitude')
    expect(spell.bookPage).toBe(193)
  })

  it('augment +9 PM gated 5º círc', () => {
    const aug = spell.augments.find((a) => a.pmCost === 9)
    expect(aug?.requiresCircle).toBe(5)
  })
})

describe('Físico Divino — p193', () => {
  const spell = spellById('fisico-divino')

  it('círculo 2, transmutação, toque, cena, Cle/Dru/Pal', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('transmutacao')
    expect(spell.range).toBe('toque')
    expect(spell.classes).toEqual(['Clérigo', 'Druida', 'Paladino'])
    expect(spell.bookPage).toBe(193)
  })

  it('augments gated 3º/4º/5º', () => {
    const gates = spell.augments
      .map((a) => a.requiresCircle)
      .filter((c) => c !== undefined)
      .sort()
    expect(gates).toEqual([3, 4, 5])
  })
})

describe('Flecha Ácida — p193', () => {
  const spell = spellById('flecha-acida')

  it('círculo 2, evocação, médio, Reflexos parcial', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('evocacao')
    expect(spell.range).toBe('medio')
    expect(spell.saveType).toBe('reflexos')
    expect(spell.resistance).toBe('parcial')
    expect(spell.bookPage).toBe(193)
  })
})

describe('Forma Etérea — p193', () => {
  const spell = spellById('forma-eterea')

  it('círculo 4, transmutação, completa, sustentada, Arcanista apenas', () => {
    expect(spell.circle).toBe(4)
    expect(spell.school).toBe('transmutacao')
    expect(spell.execution).toBe('completa')
    expect(spell.duration).toBe('sustentada')
    expect(spell.classes).toEqual(['Arcanista'])
    expect(spell.bookPage).toBe(193)
  })

  it('augment +5 PM gated 5º', () => {
    const aug = spell.augments.find((a) => a.pmCost === 5)
    expect(aug?.requiresCircle).toBe(5)
  })
})

describe('Fúria do Panteão — p194', () => {
  const spell = spellById('furia-do-panteao')

  it('círculo 5, evocação, completa, longo, sustentada, Cle/Dru', () => {
    expect(spell.circle).toBe(5)
    expect(spell.school).toBe('evocacao')
    expect(spell.execution).toBe('completa')
    expect(spell.range).toBe('longo')
    expect(spell.duration).toBe('sustentada')
    expect(spell.classes).toEqual(['Clérigo', 'Druida'])
    expect(spell.augments).toEqual([])
    expect(spell.bookPage).toBe(194)
  })
})

describe('Globo da Verdade de Gwen — p194', () => {
  const spell = spellById('globo-da-verdade-de-gwen')

  it('círculo 2, adivinhação, curto, cena, Cle/Dru/Pal', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('adivinhacao')
    expect(spell.classes).toEqual(['Clérigo', 'Druida', 'Paladino'])
    expect(spell.bookPage).toBe(194)
  })
})

describe('Globo de Invulnerabilidade — p194', () => {
  const spell = spellById('globo-de-invulnerabilidade')

  it('círculo 3, abjuração, pessoal, sustentada', () => {
    expect(spell.circle).toBe(3)
    expect(spell.school).toBe('abjuracao')
    expect(spell.range).toBe('pessoal')
    expect(spell.duration).toBe('sustentada')
    expect(spell.bookPage).toBe(194)
  })

  it('augments +4 gated 4º, +9 gated 5º', () => {
    const c4 = spell.augments.find((a) => a.pmCost === 4)
    const c5 = spell.augments.find((a) => a.pmCost === 9)
    expect(c4?.requiresCircle).toBe(4)
    expect(c5?.requiresCircle).toBe(5)
  })
})

describe('Guardião Divino — p194', () => {
  const spell = spellById('guardiao-divino')

  it('círculo 4, convocação, curto, definida, Cle/Dru', () => {
    expect(spell.circle).toBe(4)
    expect(spell.school).toBe('convocacao')
    expect(spell.range).toBe('curto')
    expect(spell.duration).toBe('definida')
    expect(spell.durationNote).toBeTruthy()
    expect(spell.classes).toEqual(['Clérigo', 'Druida'])
    expect(spell.bookPage).toBe(194)
  })
})
