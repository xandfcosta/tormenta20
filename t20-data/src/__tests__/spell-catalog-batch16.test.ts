import { describe, expect, it } from 'vitest'
import { SPELL_IDS, spellById } from '../spell-catalog'

/**
 * Batch 16 do catálogo — 10 magias verbatim do PDF p207-210.
 */

const BATCH16_IDS = [
  'talho-invisivel-de-edauros',
  'teia',
  'telecinesia',
  'tentaculos-de-trevas',
  'terremoto',
  'toque-chocante',
  'toque-da-morte',
  'tranquilidade',
  'transformacao-de-guerra',
  'transmutar-objetos',
] as const

describe('SPELL_CATALOG batch16 — presença', () => {
  it('as 10 magias estão no catálogo', () => {
    for (const id of BATCH16_IDS) {
      expect(SPELL_IDS).toContain(id)
    }
  })
})

describe('Talho Invisível de Edauros — p207', () => {
  const spell = spellById('talho-invisivel-de-edauros')

  it('círculo 4, evocação, pessoal, Fortitude parcial, Arc/Bar', () => {
    expect(spell.circle).toBe(4)
    expect(spell.school).toBe('evocacao')
    expect(spell.range).toBe('pessoal')
    expect(spell.saveType).toBe('fortitude')
    expect(spell.resistance).toBe('parcial')
    expect(spell.classes).toEqual(['Arcanista', 'Bardo'])
    expect(spell.bookPage).toBe(207)
  })
})

describe('Teia — p208', () => {
  const spell = spellById('teia')

  it('círculo 1, convocação, curto, cena, Reflexos anula, Arc/Bar', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('convocacao')
    expect(spell.saveType).toBe('reflexos')
    expect(spell.resistance).toBe('anula')
    expect(spell.bookPage).toBe(208)
  })

  it('augment +2 PM re-teste gated 2º', () => {
    const c2 = spell.augments.find((a) => a.requiresCircle === 2)
    expect(c2?.pmCost).toBe(2)
  })
})

describe('Telecinesia — p208', () => {
  const spell = spellById('telecinesia')

  it('círculo 3, transmutação, médio, sustentada, Arc/Bar', () => {
    expect(spell.circle).toBe(3)
    expect(spell.school).toBe('transmutacao')
    expect(spell.duration).toBe('sustentada')
    expect(spell.classes).toEqual(['Arcanista', 'Bardo'])
    expect(spell.bookPage).toBe(208)
  })
})

describe('Tentáculos de Trevas — p209', () => {
  const spell = spellById('tentaculos-de-trevas')

  it('círculo 3, necromancia, médio, cena', () => {
    expect(spell.circle).toBe(3)
    expect(spell.school).toBe('necromancia')
    expect(spell.range).toBe('medio')
    expect(spell.duration).toBe('cena')
    expect(spell.bookPage).toBe(209)
  })
})

describe('Terremoto — p209', () => {
  const spell = spellById('terremoto')

  it('círculo 4, evocação, longo, definida 1 rodada, Cle/Dru/Pal, sem augments', () => {
    expect(spell.circle).toBe(4)
    expect(spell.school).toBe('evocacao')
    expect(spell.range).toBe('longo')
    expect(spell.duration).toBe('definida')
    expect(spell.durationNote).toBe('1 rodada')
    expect(spell.classes).toEqual(['Clérigo', 'Druida', 'Paladino'])
    expect(spell.augments).toEqual([])
    expect(spell.bookPage).toBe(209)
  })
})

describe('Toque Chocante — p209', () => {
  const spell = spellById('toque-chocante')

  it('círculo 1, evocação, toque, Fortitude metade, Arc/Bar', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('evocacao')
    expect(spell.range).toBe('toque')
    expect(spell.saveType).toBe('fortitude')
    expect(spell.resistance).toBe('metade')
    expect(spell.bookPage).toBe(209)
  })
})

describe('Toque da Morte — p209', () => {
  const spell = spellById('toque-da-morte')

  it('círculo 5, necromancia, toque, Fortitude parcial, universal 5', () => {
    expect(spell.circle).toBe(5)
    expect(spell.school).toBe('necromancia')
    expect(spell.saveType).toBe('fortitude')
    expect(spell.resistance).toBe('parcial')
    expect(spell.classes).toEqual([
      'Arcanista',
      'Bardo',
      'Clérigo',
      'Druida',
      'Paladino',
    ])
    expect(spell.bookPage).toBe(209)
  })
})

describe('Tranquilidade — p210', () => {
  const spell = spellById('tranquilidade')

  it('círculo 1, encantamento, Vontade parcial, Cle/Dru/Pal', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('encantamento')
    expect(spell.saveType).toBe('vontade')
    expect(spell.resistance).toBe('parcial')
    expect(spell.classes).toEqual(['Clérigo', 'Druida', 'Paladino'])
    expect(spell.bookPage).toBe(210)
  })

  it('augment +5 PM gated 3º', () => {
    const aug = spell.augments.find((a) => a.pmCost === 5)
    expect(aug?.requiresCircle).toBe(3)
  })
})

describe('Transformação de Guerra — p210', () => {
  const spell = spellById('transformacao-de-guerra')

  it('círculo 3, transmutação, pessoal, sustentada, Arc/Bar', () => {
    expect(spell.circle).toBe(3)
    expect(spell.school).toBe('transmutacao')
    expect(spell.duration).toBe('sustentada')
    expect(spell.classes).toEqual(['Arcanista', 'Bardo'])
    expect(spell.bookPage).toBe(210)
  })
})

describe('Transmutar Objetos — p210', () => {
  const spell = spellById('transmutar-objetos')

  it('círculo 1, transmutação, toque, cena, Arc/Bar', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('transmutacao')
    expect(spell.range).toBe('toque')
    expect(spell.classes).toEqual(['Arcanista', 'Bardo'])
    expect(spell.bookPage).toBe(210)
  })

  it('augment truque +0 PM', () => {
    const truque = spell.augments.find((a) => a.pmCost === 0)
    expect(truque?.description).toMatch(/[Tt]ruque/)
  })

  it('augment +5 PM gated 3º restaura destruído', () => {
    const aug = spell.augments.find((a) => a.pmCost === 5)
    expect(aug?.requiresCircle).toBe(3)
  })
})
