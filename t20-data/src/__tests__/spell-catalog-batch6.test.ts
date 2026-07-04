import { describe, expect, it } from 'vitest'
import { SPELL_IDS, spellById } from '../spell-catalog'

/**
 * Batch 6 do catálogo — 10 magias verbatim do PDF p188-190.
 */

const BATCH6_IDS = [
  'controlar-terra',
  'convocacao-instantanea',
  'cranio-voador-de-vladislav',
  'criar-elementos',
  'cupula-de-repulsao',
  'deflagracao-de-mana',
  'desejo',
  'desespero-esmagador',
  'desintegrar',
  'despedacar',
] as const

describe('SPELL_CATALOG batch6 — presença', () => {
  it('as 10 magias estão no catálogo', () => {
    for (const id of BATCH6_IDS) {
      expect(SPELL_IDS).toContain(id)
    }
  })
})

describe('Controlar Terra — p188', () => {
  const spell = spellById('controlar-terra')

  it('círculo 3, transmutação, longo, Reflexos metade', () => {
    expect(spell.circle).toBe(3)
    expect(spell.school).toBe('transmutacao')
    expect(spell.range).toBe('longo')
    expect(spell.saveType).toBe('reflexos')
    expect(spell.resistance).toBe('metade')
    expect(spell.bookPage).toBe(188)
  })
})

describe('Convocação Instantânea — p188', () => {
  const spell = spellById('convocacao-instantanea')

  it('círculo 3, convocação, ilimitado, Arc/Bar', () => {
    expect(spell.circle).toBe(3)
    expect(spell.school).toBe('convocacao')
    expect(spell.range).toBe('ilimitado')
    expect(spell.classes).toEqual(['Arcanista', 'Bardo'])
    expect(spell.bookPage).toBe(188)
  })

  it('4 augments', () => {
    expect(spell.augments.length).toBe(4)
  })
})

describe('Crânio Voador de Vladislav — p188', () => {
  const spell = spellById('cranio-voador-de-vladislav')

  it('círculo 2, necromancia, Fortitude parcial', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('necromancia')
    expect(spell.saveType).toBe('fortitude')
    expect(spell.resistance).toBe('parcial')
    expect(spell.bookPage).toBe(188)
  })
})

describe('Criar Elementos — p188', () => {
  const spell = spellById('criar-elementos')

  it('círculo 1, convocação, Cle/Dru/Pal', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('convocacao')
    expect(spell.classes).toEqual(['Clérigo', 'Druida', 'Paladino'])
    expect(spell.bookPage).toBe(188)
  })
})

describe('Cúpula de Repulsão — p189', () => {
  const spell = spellById('cupula-de-repulsao')

  it('círculo 4, abjuração, completa, sustentada, Vontade anula', () => {
    expect(spell.circle).toBe(4)
    expect(spell.school).toBe('abjuracao')
    expect(spell.execution).toBe('completa')
    expect(spell.duration).toBe('sustentada')
    expect(spell.saveType).toBe('vontade')
    expect(spell.resistance).toBe('anula')
    expect(spell.bookPage).toBe(189)
  })

  it('augment +5 PM gated 5º círc', () => {
    const aug = spell.augments.find((a) => a.pmCost === 5)
    expect(aug?.requiresCircle).toBe(5)
  })
})

describe('Deflagração de Mana — p189', () => {
  const spell = spellById('deflagracao-de-mana')

  it('círculo 5, evocação, completa, pessoal, Fortitude parcial, Arc/Bar', () => {
    expect(spell.circle).toBe(5)
    expect(spell.school).toBe('evocacao')
    expect(spell.execution).toBe('completa')
    expect(spell.range).toBe('pessoal')
    expect(spell.saveType).toBe('fortitude')
    expect(spell.resistance).toBe('parcial')
    expect(spell.classes).toEqual(['Arcanista', 'Bardo'])
    expect(spell.bookPage).toBe(189)
  })
})

describe('Desejo — p190', () => {
  const spell = spellById('desejo')

  it('círculo 5, transmutação, completa, longo, definida "veja texto", Arc/Bar', () => {
    expect(spell.circle).toBe(5)
    expect(spell.school).toBe('transmutacao')
    expect(spell.execution).toBe('completa')
    expect(spell.range).toBe('longo')
    expect(spell.duration).toBe('definida')
    expect(spell.durationNote).toBe('veja texto')
    expect(spell.classes).toEqual(['Arcanista', 'Bardo'])
    expect(spell.bookPage).toBe(190)
  })

  it('sem augments', () => {
    expect(spell.augments).toEqual([])
  })
})

describe('Desespero Esmagador — p190', () => {
  const spell = spellById('desespero-esmagador')

  it('círculo 2, encantamento, pessoal, Vontade parcial', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('encantamento')
    expect(spell.range).toBe('pessoal')
    expect(spell.saveType).toBe('vontade')
    expect(spell.resistance).toBe('parcial')
    expect(spell.bookPage).toBe(190)
  })

  it('augment +3 PM aumenta gated 3º', () => {
    const aug = spell.augments.find((a) => a.pmCost === 3 && a.kind === 'aumenta')
    expect(aug?.requiresCircle).toBe(3)
  })
})

describe('Desintegrar — p190', () => {
  const spell = spellById('desintegrar')

  it('círculo 4, transmutação, médio, Fortitude parcial, Arc/Bar', () => {
    expect(spell.circle).toBe(4)
    expect(spell.school).toBe('transmutacao')
    expect(spell.range).toBe('medio')
    expect(spell.saveType).toBe('fortitude')
    expect(spell.resistance).toBe('parcial')
    expect(spell.classes).toEqual(['Arcanista', 'Bardo'])
    expect(spell.bookPage).toBe(190)
  })
})

describe('Despedaçar — p190', () => {
  const spell = spellById('despedacar')

  it('círculo 1, evocação, Fortitude parcial, Cle/Dru/Pal', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('evocacao')
    expect(spell.saveType).toBe('fortitude')
    expect(spell.resistance).toBe('parcial')
    expect(spell.classes).toEqual(['Clérigo', 'Druida', 'Paladino'])
    expect(spell.bookPage).toBe(190)
  })

  it('augments gated 2º/3º/4º/5º por tamanho de objeto', () => {
    const gates = spell.augments
      .map((a) => a.requiresCircle)
      .filter((c) => c !== undefined)
      .sort()
    expect(gates).toEqual([2, 3, 4, 5])
  })
})
