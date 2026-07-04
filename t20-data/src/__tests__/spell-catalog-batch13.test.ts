import { describe, expect, it } from 'vitest'
import { SPELL_IDS, spellById } from '../spell-catalog'

/**
 * Batch 13 do catálogo — 9 magias verbatim do PDF p202-204.
 * Nota: extract também incluiu Queda Suave (p202) mas já estava no seed
 * — pulada aqui para evitar colisão.
 */

const BATCH13_IDS = [
  'protecao-divina',
  'purificacao',
  'raio-do-enfraquecimento',
  'raio-polar',
  'raio-solar',
  'reanimacao-impura',
  'refugio',
  'relampago-flamejante-de-reynard',
  'requiem',
] as const

describe('SPELL_CATALOG batch13 — presença', () => {
  it('as 9 magias estão no catálogo', () => {
    for (const id of BATCH13_IDS) {
      expect(SPELL_IDS).toContain(id)
    }
  })
})

describe('Proteção Divina — p202', () => {
  const spell = spellById('protecao-divina')

  it('círculo 1, abjuração, toque, cena, Cle/Dru/Pal', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('abjuracao')
    expect(spell.range).toBe('toque')
    expect(spell.classes).toEqual(['Clérigo', 'Druida', 'Paladino'])
    expect(spell.bookPage).toBe(202)
  })

  it('augment +5 PM gated 3º imune mental', () => {
    const aug = spell.augments.find((a) => a.pmCost === 5)
    expect(aug?.requiresCircle).toBe(3)
  })
})

describe('Purificação — p202', () => {
  const spell = spellById('purificacao')

  it('círculo 2, evocação, toque, instantânea', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('evocacao')
    expect(spell.duration).toBe('instantanea')
    expect(spell.bookPage).toBe(202)
  })

  it('augment +7 PM gated 3º dissipa encantamento/necromancia/transmutação', () => {
    const aug = spell.augments.find((a) => a.pmCost === 7)
    expect(aug?.requiresCircle).toBe(3)
  })
})

describe('Raio do Enfraquecimento — p202', () => {
  const spell = spellById('raio-do-enfraquecimento')

  it('círculo 1, necromancia, Fortitude parcial, Arc/Bar', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('necromancia')
    expect(spell.saveType).toBe('fortitude')
    expect(spell.resistance).toBe('parcial')
    expect(spell.classes).toEqual(['Arcanista', 'Bardo'])
    expect(spell.bookPage).toBe(202)
  })

  it('augment truque +0 PM', () => {
    const truque = spell.augments.find((a) => a.pmCost === 0)
    expect(truque?.description).toMatch(/[Tt]ruque/)
  })
})

describe('Raio Polar — p203', () => {
  const spell = spellById('raio-polar')

  it('círculo 4, evocação, médio, Fortitude parcial', () => {
    expect(spell.circle).toBe(4)
    expect(spell.school).toBe('evocacao')
    expect(spell.range).toBe('medio')
    expect(spell.saveType).toBe('fortitude')
    expect(spell.resistance).toBe('parcial')
    expect(spell.bookPage).toBe(203)
  })
})

describe('Raio Solar — p203', () => {
  const spell = spellById('raio-solar')

  it('círculo 2, evocação, pessoal, Reflexos metade, Cle/Dru/Pal', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('evocacao')
    expect(spell.range).toBe('pessoal')
    expect(spell.saveType).toBe('reflexos')
    expect(spell.resistance).toBe('metade')
    expect(spell.classes).toEqual(['Clérigo', 'Druida', 'Paladino'])
    expect(spell.bookPage).toBe(203)
  })

  it('augment truque +0 PM', () => {
    const truque = spell.augments.find((a) => a.pmCost === 0)
    expect(truque?.description).toMatch(/[Tt]ruque/)
  })
})

describe('Reanimação Impura — p203', () => {
  const spell = spellById('reanimacao-impura')

  it('círculo 5, necromancia, completa, toque, cena, sem augments', () => {
    expect(spell.circle).toBe(5)
    expect(spell.school).toBe('necromancia')
    expect(spell.execution).toBe('completa')
    expect(spell.range).toBe('toque')
    expect(spell.duration).toBe('cena')
    expect(spell.augments).toEqual([])
    expect(spell.bookPage).toBe(203)
  })
})

describe('Refúgio — p203', () => {
  const spell = spellById('refugio')

  it('círculo 2, abjuração, completa, curto, dia, Arc/Bar', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('abjuracao')
    expect(spell.execution).toBe('completa')
    expect(spell.duration).toBe('dia')
    expect(spell.classes).toEqual(['Arcanista', 'Bardo'])
    expect(spell.bookPage).toBe(203)
  })

  it('augments gated 3º (extradimensional) e 4º (mansão)', () => {
    const c3 = spell.augments.find((a) => a.requiresCircle === 3)
    const c4 = spell.augments.find((a) => a.requiresCircle === 4)
    expect(c3?.pmCost).toBe(3)
    expect(c4?.pmCost).toBe(9)
  })
})

describe('Relâmpago Flamejante de Reynard — p203', () => {
  const spell = spellById('relampago-flamejante-de-reynard')

  it('círculo 4, evocação, completa, curto, sustentada, Reflexos metade', () => {
    expect(spell.circle).toBe(4)
    expect(spell.school).toBe('evocacao')
    expect(spell.execution).toBe('completa')
    expect(spell.duration).toBe('sustentada')
    expect(spell.saveType).toBe('reflexos')
    expect(spell.resistance).toBe('metade')
    expect(spell.bookPage).toBe(203)
  })
})

describe('Réquiem — p204', () => {
  const spell = spellById('requiem')

  it('círculo 5, ilusão, completa, curto, sustentada, Vontade anula', () => {
    expect(spell.circle).toBe(5)
    expect(spell.school).toBe('ilusao')
    expect(spell.execution).toBe('completa')
    expect(spell.duration).toBe('sustentada')
    expect(spell.saveType).toBe('vontade')
    expect(spell.resistance).toBe('anula')
    expect(spell.augments).toEqual([])
    expect(spell.bookPage).toBe(204)
  })
})
