import { describe, expect, it } from 'vitest'
import { SPELL_IDS, spellById } from '../spell-catalog'

/**
 * Batch 11 do catálogo — 10 magias verbatim do PDF p199-200.
 */

const BATCH11_IDS = [
  'miasma-mefitico',
  'miragem',
  'missao-divina',
  'montaria-arcana',
  'muralha-de-ossos',
  'muralha-elemental',
  'nevoa',
  'oracao',
  'orientacao',
  'palavra-primordial',
] as const

describe('SPELL_CATALOG batch11 — presença', () => {
  it('as 10 magias estão no catálogo', () => {
    for (const id of BATCH11_IDS) {
      expect(SPELL_IDS).toContain(id)
    }
  })
})

describe('Miasma Mefítico — p199', () => {
  const spell = spellById('miasma-mefitico')

  it('círculo 2, necromancia, Fortitude metade, Cle/Dru/Pal', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('necromancia')
    expect(spell.saveType).toBe('fortitude')
    expect(spell.resistance).toBe('metade')
    expect(spell.classes).toEqual(['Clérigo', 'Druida', 'Paladino'])
    expect(spell.bookPage).toBe(199)
  })

  it('carrega augment truque +0 PM', () => {
    const truque = spell.augments.find((a) => a.pmCost === 0)
    expect(truque?.description).toMatch(/[Tt]ruque/)
  })
})

describe('Miragem — p199', () => {
  const spell = spellById('miragem')

  it('círculo 3, ilusão, longo, dia, Vontade desacredita, Arc/Bar', () => {
    expect(spell.circle).toBe(3)
    expect(spell.school).toBe('ilusao')
    expect(spell.range).toBe('longo')
    expect(spell.duration).toBe('dia')
    expect(spell.saveType).toBe('vontade')
    expect(spell.resistance).toBe('desacredita')
    expect(spell.bookPage).toBe(199)
  })

  it('augment +9 PM permanente gated 4º', () => {
    const aug = spell.augments.find((a) => a.pmCost === 9)
    expect(aug?.requiresCircle).toBe(4)
  })
})

describe('Missão Divina — p199', () => {
  const spell = spellById('missao-divina')

  it('círculo 3, encantamento, Vontade anula, definida', () => {
    expect(spell.circle).toBe(3)
    expect(spell.school).toBe('encantamento')
    expect(spell.saveType).toBe('vontade')
    expect(spell.resistance).toBe('anula')
    expect(spell.duration).toBe('definida')
    expect(spell.bookPage).toBe(199)
  })
})

describe('Montaria Arcana — p199', () => {
  const spell = spellById('montaria-arcana')

  it('círculo 2, convocação, curto, dia, Arc/Bar', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('convocacao')
    expect(spell.duration).toBe('dia')
    expect(spell.classes).toEqual(['Arcanista', 'Bardo'])
    expect(spell.bookPage).toBe(199)
  })
})

describe('Muralha de Ossos — p199', () => {
  const spell = spellById('muralha-de-ossos')

  it('círculo 4, necromancia, Reflexos anula, universal 5', () => {
    expect(spell.circle).toBe(4)
    expect(spell.school).toBe('necromancia')
    expect(spell.saveType).toBe('reflexos')
    expect(spell.resistance).toBe('anula')
    expect(spell.classes).toEqual([
      'Arcanista',
      'Bardo',
      'Clérigo',
      'Druida',
      'Paladino',
    ])
    expect(spell.bookPage).toBe(199)
  })
})

describe('Muralha Elemental — p200', () => {
  const spell = spellById('muralha-elemental')

  it('círculo 3, evocação, Reflexos metade, Arc/Bar', () => {
    expect(spell.circle).toBe(3)
    expect(spell.school).toBe('evocacao')
    expect(spell.saveType).toBe('reflexos')
    expect(spell.resistance).toBe('metade')
    expect(spell.classes).toEqual(['Arcanista', 'Bardo'])
    expect(spell.bookPage).toBe(200)
  })

  it('augment +4 PM muralha Força gated 4º', () => {
    const aug = spell.augments.find((a) => a.pmCost === 4)
    expect(aug?.requiresCircle).toBe(4)
  })
})

describe('Névoa — p200', () => {
  const spell = spellById('nevoa')

  it('círculo 1, convocação, curto, cena, universal 5', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('convocacao')
    expect(spell.classes).toEqual([
      'Arcanista',
      'Bardo',
      'Clérigo',
      'Druida',
      'Paladino',
    ])
    expect(spell.bookPage).toBe(200)
  })

  it('6 augments', () => {
    expect(spell.augments.length).toBe(6)
  })
})

describe('Oração — p200', () => {
  const spell = spellById('oracao')

  it('círculo 2, encantamento, sustentada, Cle/Dru/Pal, componente material', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('encantamento')
    expect(spell.duration).toBe('sustentada')
    expect(spell.classes).toEqual(['Clérigo', 'Druida', 'Paladino'])
    expect(spell.components).toContain('material')
    expect(spell.bookPage).toBe(200)
  })

  it('augments +7 gated 3º, +12 gated 4º', () => {
    const c3 = spell.augments.find((a) => a.pmCost === 7)
    const c4 = spell.augments.find((a) => a.pmCost === 12)
    expect(c3?.requiresCircle).toBe(3)
    expect(c4?.requiresCircle).toBe(4)
  })
})

describe('Orientação — p200', () => {
  const spell = spellById('orientacao')

  it('círculo 1, adivinhação, curto, definida 1 rodada', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('adivinhacao')
    expect(spell.duration).toBe('definida')
    expect(spell.durationNote).toBe('1 rodada')
    expect(spell.bookPage).toBe(200)
  })

  it('augments gated 2º/3º', () => {
    const gates = spell.augments
      .map((a) => a.requiresCircle)
      .filter((c) => c !== undefined)
      .sort()
    expect(gates).toEqual([2, 3, 3])
  })
})

describe('Palavra Primordial — p200', () => {
  const spell = spellById('palavra-primordial')

  it('círculo 5, encantamento, Vontade parcial, universal 5', () => {
    expect(spell.circle).toBe(5)
    expect(spell.school).toBe('encantamento')
    expect(spell.saveType).toBe('vontade')
    expect(spell.resistance).toBe('parcial')
    expect(spell.classes).toEqual([
      'Arcanista',
      'Bardo',
      'Clérigo',
      'Druida',
      'Paladino',
    ])
    expect(spell.bookPage).toBe(200)
  })
})
