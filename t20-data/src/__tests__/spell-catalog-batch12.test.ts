import { describe, expect, it } from 'vitest'
import { SPELL_IDS, spellById } from '../spell-catalog'

/**
 * Batch 12 do catálogo — 10 magias verbatim do PDF p201-202.
 */

const BATCH12_IDS = [
  'pele-de-pedra',
  'perdicao',
  'poeira-da-podridao',
  'possessao',
  'potencia-divina',
  'premonicao',
  'primor-atletico',
  'profanar',
  'projetar-consciencia',
  'protecao-contra-magia',
] as const

describe('SPELL_CATALOG batch12 — presença', () => {
  it('as 10 magias estão no catálogo', () => {
    for (const id of BATCH12_IDS) {
      expect(SPELL_IDS).toContain(id)
    }
  })
})

describe('Pele de Pedra — p201', () => {
  const spell = spellById('pele-de-pedra')

  it('círculo 3, transmutação, pessoal, cena, universal 5', () => {
    expect(spell.circle).toBe(3)
    expect(spell.school).toBe('transmutacao')
    expect(spell.classes).toEqual([
      'Arcanista',
      'Bardo',
      'Clérigo',
      'Druida',
      'Paladino',
    ])
    expect(spell.bookPage).toBe(201)
  })

  it('augment +9 gated 5º estátua permanente', () => {
    const aug = spell.augments.find((a) => a.pmCost === 9)
    expect(aug?.requiresCircle).toBe(5)
  })
})

describe('Perdição — p201', () => {
  const spell = spellById('perdicao')

  it('círculo 1, necromancia, Cle/Paladino, sem save', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('necromancia')
    expect(spell.classes).toEqual(['Clérigo', 'Paladino'])
    expect(spell.saveType).toBe('none')
    expect(spell.bookPage).toBe(201)
  })
})

describe('Poeira da Podridão — p201', () => {
  const spell = spellById('poeira-da-podridao')

  it('círculo 3, necromancia, Fortitude metade, Cle/Paladino', () => {
    expect(spell.circle).toBe(3)
    expect(spell.school).toBe('necromancia')
    expect(spell.saveType).toBe('fortitude')
    expect(spell.resistance).toBe('metade')
    expect(spell.classes).toEqual(['Clérigo', 'Paladino'])
    expect(spell.bookPage).toBe(201)
  })
})

describe('Possessão — p201', () => {
  const spell = spellById('possessao')

  it('círculo 5, encantamento, longo, dia, Vontade anula, Arc/Bar', () => {
    expect(spell.circle).toBe(5)
    expect(spell.school).toBe('encantamento')
    expect(spell.range).toBe('longo')
    expect(spell.duration).toBe('dia')
    expect(spell.saveType).toBe('vontade')
    expect(spell.resistance).toBe('anula')
    expect(spell.classes).toEqual(['Arcanista', 'Bardo'])
    expect(spell.bookPage).toBe(201)
  })

  it('3 augments +5 PM', () => {
    const augs5 = spell.augments.filter((a) => a.pmCost === 5)
    expect(augs5.length).toBe(3)
  })
})

describe('Potência Divina — p201', () => {
  const spell = spellById('potencia-divina')

  it('círculo 3, transmutação, pessoal, sustentada, Cle/Paladino', () => {
    expect(spell.circle).toBe(3)
    expect(spell.school).toBe('transmutacao')
    expect(spell.duration).toBe('sustentada')
    expect(spell.classes).toEqual(['Clérigo', 'Paladino'])
    expect(spell.bookPage).toBe(201)
  })
})

describe('Premonição — p201', () => {
  const spell = spellById('premonicao')

  it('círculo 4, adivinhação, pessoal, cena', () => {
    expect(spell.circle).toBe(4)
    expect(spell.school).toBe('adivinhacao')
    expect(spell.range).toBe('pessoal')
    expect(spell.duration).toBe('cena')
    expect(spell.bookPage).toBe(201)
  })
})

describe('Primor Atlético — p201', () => {
  const spell = spellById('primor-atletico')

  it('círculo 1, transmutação, toque, Arc/Bar', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('transmutacao')
    expect(spell.range).toBe('toque')
    expect(spell.classes).toEqual(['Arcanista', 'Bardo'])
    expect(spell.bookPage).toBe(201)
  })

  it('augment +3 PM gated 2º', () => {
    const aug = spell.augments.find((a) => a.pmCost === 3)
    expect(aug?.requiresCircle).toBe(2)
  })
})

describe('Profanar — p202', () => {
  const spell = spellById('profanar')

  it('círculo 1, necromancia, longo, dia, Cle/Paladino', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('necromancia')
    expect(spell.range).toBe('longo')
    expect(spell.duration).toBe('dia')
    expect(spell.classes).toEqual(['Clérigo', 'Paladino'])
    expect(spell.bookPage).toBe(202)
  })
})

describe('Projetar Consciência — p202', () => {
  const spell = spellById('projetar-consciencia')

  it('círculo 5, adivinhação, ilimitado, sustentada, universal 5', () => {
    expect(spell.circle).toBe(5)
    expect(spell.school).toBe('adivinhacao')
    expect(spell.range).toBe('ilimitado')
    expect(spell.duration).toBe('sustentada')
    expect(spell.classes).toEqual([
      'Arcanista',
      'Bardo',
      'Clérigo',
      'Druida',
      'Paladino',
    ])
    expect(spell.bookPage).toBe(202)
  })
})

describe('Proteção Contra Magia — p202', () => {
  const spell = spellById('protecao-contra-magia')

  it('círculo 3, abjuração, toque, cena, Cle/Paladino', () => {
    expect(spell.circle).toBe(3)
    expect(spell.school).toBe('abjuracao')
    expect(spell.range).toBe('toque')
    expect(spell.classes).toEqual(['Clérigo', 'Paladino'])
    expect(spell.bookPage).toBe(202)
  })

  it('augments +4 (2x) gated 4º, +9 gated 5º', () => {
    const c4 = spell.augments.filter((a) => a.requiresCircle === 4)
    const c5 = spell.augments.find((a) => a.requiresCircle === 5)
    expect(c4.length).toBe(2)
    expect(c5?.pmCost).toBe(9)
  })
})
