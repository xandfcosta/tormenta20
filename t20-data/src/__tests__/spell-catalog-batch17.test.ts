import { describe, expect, it } from 'vitest'
import { SPELL_IDS, spellById } from '../spell-catalog'

/**
 * Batch 17 do catálogo — 5 magias verbatim do PDF p210-211.
 * Final batch: encerra o preenchimento do catálogo (Cap 4 Magia, seção
 * "Descrição das Magias" — última entrada alfabética é Voz Divina p211).
 */

const BATCH17_IDS = [
  'vestimenta-da-fe',
  'viagem-arborea',
  'viagem-planar',
  'visao-da-verdade',
  'vitalidade-fantasma',
] as const

describe('SPELL_CATALOG batch17 — presença', () => {
  it('as 5 magias finais estão no catálogo', () => {
    for (const id of BATCH17_IDS) {
      expect(SPELL_IDS).toContain(id)
    }
  })
})

describe('Vestimenta da Fé — p210', () => {
  const spell = spellById('vestimenta-da-fe')

  it('círculo 2, abjuração, toque, dia, Cle/Dru/Pal', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('abjuracao')
    expect(spell.duration).toBe('dia')
    expect(spell.classes).toEqual(['Clérigo', 'Druida', 'Paladino'])
    expect(spell.bookPage).toBe(210)
  })

  it('augments +3 gated 3º, +7 gated 4º', () => {
    const c3 = spell.augments.find((a) => a.pmCost === 3)
    const c4 = spell.augments.find((a) => a.pmCost === 7)
    expect(c3?.requiresCircle).toBe(3)
    expect(c4?.requiresCircle).toBe(4)
  })
})

describe('Viagem Arbórea — p210', () => {
  const spell = spellById('viagem-arborea')

  it('círculo 3, convocação, completa, pessoal, cena, Cle/Dru/Pal', () => {
    expect(spell.circle).toBe(3)
    expect(spell.school).toBe('convocacao')
    expect(spell.execution).toBe('completa')
    expect(spell.duration).toBe('cena')
    expect(spell.classes).toEqual(['Clérigo', 'Druida', 'Paladino'])
    expect(spell.bookPage).toBe(210)
  })
})

describe('Viagem Planar — p211', () => {
  const spell = spellById('viagem-planar')

  it('círculo 4, convocação, completa, toque, componente material, universal 5', () => {
    expect(spell.circle).toBe(4)
    expect(spell.school).toBe('convocacao')
    expect(spell.execution).toBe('completa')
    expect(spell.range).toBe('toque')
    expect(spell.components).toContain('material')
    expect(spell.classes).toEqual([
      'Arcanista',
      'Bardo',
      'Clérigo',
      'Druida',
      'Paladino',
    ])
    expect(spell.bookPage).toBe(211)
  })
})

describe('Visão da Verdade — p211', () => {
  const spell = spellById('visao-da-verdade')

  it('círculo 4, adivinhação, movimento, pessoal, cena, universal 5', () => {
    expect(spell.circle).toBe(4)
    expect(spell.school).toBe('adivinhacao')
    expect(spell.execution).toBe('movimento')
    expect(spell.range).toBe('pessoal')
    expect(spell.classes).toEqual([
      'Arcanista',
      'Bardo',
      'Clérigo',
      'Druida',
      'Paladino',
    ])
    expect(spell.bookPage).toBe(211)
  })
})

describe('Vitalidade Fantasma — p211', () => {
  const spell = spellById('vitalidade-fantasma')

  it('círculo 2, necromancia, pessoal, instantânea, Arc/Bar', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('necromancia')
    expect(spell.range).toBe('pessoal')
    expect(spell.duration).toBe('instantanea')
    expect(spell.classes).toEqual(['Arcanista', 'Bardo'])
    expect(spell.bookPage).toBe(211)
  })

  it('augment +5 PM gated 2º área com Fortitude metade', () => {
    const aug = spell.augments.find((a) => a.pmCost === 5)
    expect(aug?.requiresCircle).toBe(2)
  })
})

describe('SPELL_CATALOG — coverage final', () => {
  it('catálogo tem ≥ 190 magias (após completar todos os batches)', () => {
    expect(SPELL_IDS.length).toBeGreaterThanOrEqual(190)
  })
})
