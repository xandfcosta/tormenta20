import { describe, expect, it } from 'vitest'
import { SPELL_IDS, spellById } from '../spell-catalog'

/**
 * Batch 10 do catálogo — 9 magias verbatim do PDF p196-198.
 * Nota: extract também incluiu Manto de Sombras (p197) mas já estava
 * no seed do catálogo — pulada aqui para evitar colisão.
 */

const BATCH10_IDS = [
  'ligacao-sombria',
  'ligacao-telepatica',
  'localizacao',
  'manto-do-cruzado',
  'mao-poderosa-de-talude',
  'mapear',
  'marca-da-obediencia',
  'mata-dragao',
  'mente-divina',
] as const

describe('SPELL_CATALOG batch10 — presença', () => {
  it('as 9 magias estão no catálogo', () => {
    for (const id of BATCH10_IDS) {
      expect(SPELL_IDS).toContain(id)
    }
  })
})

describe('Ligação Sombria — p196', () => {
  const spell = spellById('ligacao-sombria')

  it('círculo 4, necromancia, longo, dia, Fortitude anula, Cle/Dru/Pal', () => {
    expect(spell.circle).toBe(4)
    expect(spell.school).toBe('necromancia')
    expect(spell.range).toBe('longo')
    expect(spell.duration).toBe('dia')
    expect(spell.saveType).toBe('fortitude')
    expect(spell.resistance).toBe('anula')
    expect(spell.classes).toEqual(['Clérigo', 'Druida', 'Paladino'])
    expect(spell.bookPage).toBe(196)
  })
})

describe('Ligação Telepática — p197', () => {
  const spell = spellById('ligacao-telepatica')

  it('círculo 2, adivinhação, toque, dia, Arc/Bar', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('adivinhacao')
    expect(spell.range).toBe('toque')
    expect(spell.duration).toBe('dia')
    expect(spell.classes).toEqual(['Arcanista', 'Bardo'])
    expect(spell.bookPage).toBe(197)
  })

  it('augment +3 PM gated 3º elo sensorial', () => {
    const aug = spell.augments.find((a) => a.pmCost === 3)
    expect(aug?.requiresCircle).toBe(3)
  })
})

describe('Localização — p197', () => {
  const spell = spellById('localizacao')

  it('círculo 2, adivinhação, pessoal, cena', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('adivinhacao')
    expect(spell.range).toBe('pessoal')
    expect(spell.bookPage).toBe(197)
  })

  it('augment truque +0 PM', () => {
    const truque = spell.augments.find((a) => a.pmCost === 0)
    expect(truque?.description).toMatch(/[Tt]ruque/)
  })
})

describe('Manto do Cruzado — p197', () => {
  const spell = spellById('manto-do-cruzado')

  it('círculo 4, evocação, pessoal, sustentada, Cle/Dru/Pal, sem augments', () => {
    expect(spell.circle).toBe(4)
    expect(spell.school).toBe('evocacao')
    expect(spell.duration).toBe('sustentada')
    expect(spell.classes).toEqual(['Clérigo', 'Druida', 'Paladino'])
    expect(spell.augments).toEqual([])
    expect(spell.bookPage).toBe(197)
  })
})

describe('Mão Poderosa de Talude — p197', () => {
  const spell = spellById('mao-poderosa-de-talude')

  it('círculo 4, convocação, médio, sustentada, Arc/Bar', () => {
    expect(spell.circle).toBe(4)
    expect(spell.school).toBe('convocacao')
    expect(spell.range).toBe('medio')
    expect(spell.duration).toBe('sustentada')
    expect(spell.bookPage).toBe(197)
  })

  it('augment +5 PM gated 5º Misticismo +20', () => {
    const aug = spell.augments.find((a) => a.pmCost === 5)
    expect(aug?.requiresCircle).toBe(5)
  })
})

describe('Mapear — p198', () => {
  const spell = spellById('mapear')

  it('círculo 2, adivinhação, toque, cena', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('adivinhacao')
    expect(spell.bookPage).toBe(198)
  })
})

describe('Marca da Obediência — p198', () => {
  const spell = spellById('marca-da-obediencia')

  it('círculo 2, encantamento, universal 5 classes, Vontade anula', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('encantamento')
    expect(spell.classes).toEqual([
      'Arcanista',
      'Bardo',
      'Clérigo',
      'Druida',
      'Paladino',
    ])
    expect(spell.saveType).toBe('vontade')
    expect(spell.resistance).toBe('anula')
    expect(spell.bookPage).toBe(198)
  })

  it('2 augments +3 PM gated 3º', () => {
    const gated = spell.augments.filter((a) => a.requiresCircle === 3)
    expect(gated.length).toBe(2)
  })
})

describe('Mata-Dragão — p198', () => {
  const spell = spellById('mata-dragao')

  it('círculo 5, evocação, completa, pessoal, Reflexos metade, Arc/Bar', () => {
    expect(spell.circle).toBe(5)
    expect(spell.school).toBe('evocacao')
    expect(spell.execution).toBe('completa')
    expect(spell.range).toBe('pessoal')
    expect(spell.saveType).toBe('reflexos')
    expect(spell.resistance).toBe('metade')
    expect(spell.classes).toEqual(['Arcanista', 'Bardo'])
    expect(spell.bookPage).toBe(198)
  })
})

describe('Mente Divina — p198', () => {
  const spell = spellById('mente-divina')

  it('círculo 2, adivinhação, toque, Cle/Dru/Pal', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('adivinhacao')
    expect(spell.range).toBe('toque')
    expect(spell.classes).toEqual(['Clérigo', 'Druida', 'Paladino'])
    expect(spell.bookPage).toBe(198)
  })

  it('augments gated 3º/4º/5º (paralelo estrutural com Físico Divino)', () => {
    const gates = spell.augments
      .map((a) => a.requiresCircle)
      .filter((c) => c !== undefined)
      .sort()
    expect(gates).toEqual([3, 4, 5])
  })
})
