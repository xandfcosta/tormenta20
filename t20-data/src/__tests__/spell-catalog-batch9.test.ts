import { describe, expect, it } from 'vitest'
import { SPELL_IDS, spellById } from '../spell-catalog'

/**
 * Batch 9 do catálogo — 10 magias verbatim do PDF p195-196.
 */

const BATCH9_IDS = [
  'ilusao-lacerante',
  'imobilizar',
  'infligir-ferimentos',
  'intervencao-divina',
  'lagrimas-de-wynna',
  'lanca-ignea-de-aleph',
  'legiao',
  'lendas-e-historias',
  'leque-cromatico',
  'libertacao',
] as const

describe('SPELL_CATALOG batch9 — presença', () => {
  it('as 10 magias estão no catálogo', () => {
    for (const id of BATCH9_IDS) {
      expect(SPELL_IDS).toContain(id)
    }
  })
})

describe('Ilusão Lacerante — p195', () => {
  const spell = spellById('ilusao-lacerante')

  it('círculo 3, ilusão, sustentada, Vontade anula', () => {
    expect(spell.circle).toBe(3)
    expect(spell.school).toBe('ilusao')
    expect(spell.duration).toBe('sustentada')
    expect(spell.saveType).toBe('vontade')
    expect(spell.resistance).toBe('anula')
    expect(spell.bookPage).toBe(195)
  })

  it('augment +4 PM gated 4º cubo 90m', () => {
    const aug = spell.augments.find((a) => a.pmCost === 4)
    expect(aug?.requiresCircle).toBe(4)
  })
})

describe('Imobilizar — p195', () => {
  const spell = spellById('imobilizar')

  it('círculo 3, encantamento, universal 4 classes', () => {
    expect(spell.circle).toBe(3)
    expect(spell.school).toBe('encantamento')
    expect(spell.classes).toEqual(['Arcanista', 'Bardo', 'Clérigo', 'Druida'])
    expect(spell.bookPage).toBe(195)
  })

  it('augment +3 PM gated 4º alvo qualquer criatura', () => {
    const aug = spell.augments.find((a) => a.pmCost === 3)
    expect(aug?.requiresCircle).toBe(4)
  })
})

describe('Infligir Ferimentos — p195', () => {
  const spell = spellById('infligir-ferimentos')

  it('círculo 1, necromancia, toque, Fortitude metade, Cle/Dru/Pal', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('necromancia')
    expect(spell.saveType).toBe('fortitude')
    expect(spell.resistance).toBe('metade')
    expect(spell.classes).toEqual(['Clérigo', 'Druida', 'Paladino'])
    expect(spell.bookPage).toBe(195)
  })
})

describe('Intervenção Divina — p195', () => {
  const spell = spellById('intervencao-divina')

  it('círculo 5, convocação, completa, ilimitado, definida "veja texto", Cle/Dru', () => {
    expect(spell.circle).toBe(5)
    expect(spell.school).toBe('convocacao')
    expect(spell.execution).toBe('completa')
    expect(spell.range).toBe('ilimitado')
    expect(spell.duration).toBe('definida')
    expect(spell.durationNote).toBe('veja texto')
    expect(spell.classes).toEqual(['Clérigo', 'Druida'])
    expect(spell.augments).toEqual([])
    expect(spell.bookPage).toBe(195)
  })
})

describe('Lágrimas de Wynna — p196', () => {
  const spell = spellById('lagrimas-de-wynna')

  it('círculo 5, abjuração, Vontade parcial, Cle/Dru', () => {
    expect(spell.circle).toBe(5)
    expect(spell.school).toBe('abjuracao')
    expect(spell.saveType).toBe('vontade')
    expect(spell.resistance).toBe('parcial')
    expect(spell.classes).toEqual(['Clérigo', 'Druida'])
    expect(spell.bookPage).toBe(196)
  })
})

describe('Lança Ígnea de Aleph — p196', () => {
  const spell = spellById('lanca-ignea-de-aleph')

  it('círculo 3, evocação, médio, Reflexos parcial', () => {
    expect(spell.circle).toBe(3)
    expect(spell.school).toBe('evocacao')
    expect(spell.range).toBe('medio')
    expect(spell.saveType).toBe('reflexos')
    expect(spell.resistance).toBe('parcial')
    expect(spell.bookPage).toBe(196)
  })
})

describe('Legião — p196', () => {
  const spell = spellById('legiao')

  it('círculo 5, encantamento, sustentada, Vontade parcial, Arcanista apenas', () => {
    expect(spell.circle).toBe(5)
    expect(spell.school).toBe('encantamento')
    expect(spell.duration).toBe('sustentada')
    expect(spell.saveType).toBe('vontade')
    expect(spell.classes).toEqual(['Arcanista'])
    expect(spell.bookPage).toBe(196)
  })
})

describe('Lendas e Histórias — p196', () => {
  const spell = spellById('lendas-e-historias')

  it('círculo 3, adivinhação, toque, sustentada, universal 4 classes', () => {
    expect(spell.circle).toBe(3)
    expect(spell.school).toBe('adivinhacao')
    expect(spell.range).toBe('toque')
    expect(spell.duration).toBe('sustentada')
    expect(spell.classes).toEqual(['Arcanista', 'Bardo', 'Clérigo', 'Druida'])
    expect(spell.bookPage).toBe(196)
  })
})

describe('Leque Cromático — p196', () => {
  const spell = spellById('leque-cromatico')

  it('círculo 1, ilusão, pessoal, Vontade parcial', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('ilusao')
    expect(spell.range).toBe('pessoal')
    expect(spell.saveType).toBe('vontade')
    expect(spell.resistance).toBe('parcial')
    expect(spell.bookPage).toBe(196)
  })

  it('augments gated 2º/3º para expandir tipos afetados', () => {
    const c2 = spell.augments.find((a) => a.requiresCircle === 2)
    const c3 = spell.augments.find((a) => a.requiresCircle === 3)
    expect(c2).toBeDefined()
    expect(c3).toBeDefined()
  })
})

describe('Libertação — p196', () => {
  const spell = spellById('libertacao')

  it('círculo 4, abjuração, curto, cena, universal 4 classes', () => {
    expect(spell.circle).toBe(4)
    expect(spell.school).toBe('abjuracao')
    expect(spell.classes).toEqual(['Arcanista', 'Bardo', 'Clérigo', 'Druida'])
    expect(spell.bookPage).toBe(196)
  })

  it('5 augments', () => {
    expect(spell.augments.length).toBe(5)
  })
})
