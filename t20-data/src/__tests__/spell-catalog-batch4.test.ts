import { describe, expect, it } from 'vitest'
import { SPELL_IDS, spellById } from '../spell-catalog'

/**
 * Batch 4 do catálogo — 10 magias verbatim do PDF p183-185.
 */

const BATCH4_IDS = [
  'circulo-da-justica',
  'circulo-da-restauracao',
  'colera-de-azgher',
  'coluna-de-chamas',
  'comando',
  'compreensao',
  'comunhao-com-a-natureza',
  'conceder-milagre',
  'concentracao-de-combate',
  'condicao',
] as const

describe('SPELL_CATALOG batch4 — presença', () => {
  it('as 10 magias estão no catálogo', () => {
    for (const id of BATCH4_IDS) {
      expect(SPELL_IDS).toContain(id)
    }
  })
})

describe('Círculo da Justiça — p183', () => {
  const spell = spellById('circulo-da-justica')

  it('círculo 2, abjuração, completa, dia, Vontade parcial', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('abjuracao')
    expect(spell.execution).toBe('completa')
    expect(spell.duration).toBe('dia')
    expect(spell.saveType).toBe('vontade')
    expect(spell.resistance).toBe('parcial')
    expect(spell.bookPage).toBe(183)
  })

  it('augment +3 PM gated 4º círculo', () => {
    const aug = spell.augments.find((a) => a.pmCost === 3)
    expect(aug?.requiresCircle).toBe(4)
  })
})

describe('Círculo da Restauração — p184', () => {
  const spell = spellById('circulo-da-restauracao')

  it('círculo 4, evocação, definida 5 rodadas', () => {
    expect(spell.circle).toBe(4)
    expect(spell.school).toBe('evocacao')
    expect(spell.duration).toBe('definida')
    expect(spell.durationNote).toBe('5 rodadas')
    expect(spell.bookPage).toBe(184)
  })
})

describe('Cólera de Azgher — p184', () => {
  const spell = spellById('colera-de-azgher')

  it('círculo 4, evocação, médio, Reflexos parcial', () => {
    expect(spell.circle).toBe(4)
    expect(spell.school).toBe('evocacao')
    expect(spell.range).toBe('medio')
    expect(spell.saveType).toBe('reflexos')
    expect(spell.resistance).toBe('parcial')
    expect(spell.bookPage).toBe(184)
  })

  it('augment +5 PM dissipa necromancia gated 5º círculo', () => {
    const aug = spell.augments.find((a) => a.pmCost === 5)
    expect(aug?.requiresCircle).toBe(5)
    expect(aug?.description).toMatch(/necromancia/)
  })
})

describe('Coluna de Chamas — p184', () => {
  const spell = spellById('coluna-de-chamas')

  it('círculo 3, evocação, longo, Reflexos metade', () => {
    expect(spell.circle).toBe(3)
    expect(spell.school).toBe('evocacao')
    expect(spell.range).toBe('longo')
    expect(spell.saveType).toBe('reflexos')
    expect(spell.resistance).toBe('metade')
    expect(spell.bookPage).toBe(184)
  })
})

describe('Comando — p184', () => {
  const spell = spellById('comando')

  it('círculo 1, encantamento, Vontade anula, definida 1 rodada', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('encantamento')
    expect(spell.saveType).toBe('vontade')
    expect(spell.resistance).toBe('anula')
    expect(spell.duration).toBe('definida')
    expect(spell.durationNote).toBe('1 rodada')
    expect(spell.bookPage).toBe(184)
  })
})

describe('Compreensão — p184', () => {
  const spell = spellById('compreensao')

  it('círculo 1, adivinhação, universal 5 classes', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('adivinhacao')
    expect(spell.classes).toEqual([
      'Arcanista',
      'Bardo',
      'Clérigo',
      'Druida',
      'Paladino',
    ])
    expect(spell.bookPage).toBe(184)
  })

  it('augments +2 PM gated 2º, +5 PM gated 3º', () => {
    const c2 = spell.augments.find(
      (a) => a.pmCost === 2 && a.requiresCircle === 2,
    )
    const c3 = spell.augments.find(
      (a) => a.pmCost === 5 && a.requiresCircle === 3,
    )
    expect(c2).toBeDefined()
    expect(c3).toBeDefined()
  })
})

describe('Comunhão com a Natureza — p184', () => {
  const spell = spellById('comunhao-com-a-natureza')

  it('círculo 3, adivinhação, completa, pessoal, dia', () => {
    expect(spell.circle).toBe(3)
    expect(spell.school).toBe('adivinhacao')
    expect(spell.execution).toBe('completa')
    expect(spell.range).toBe('pessoal')
    expect(spell.duration).toBe('dia')
    expect(spell.bookPage).toBe(184)
  })

  it('4 augments', () => {
    expect(spell.augments.length).toBe(4)
  })
})

describe('Conceder Milagre — p184', () => {
  const spell = spellById('conceder-milagre')

  it('círculo 4, encantamento, toque, definida com nota', () => {
    expect(spell.circle).toBe(4)
    expect(spell.school).toBe('encantamento')
    expect(spell.range).toBe('toque')
    expect(spell.duration).toBe('definida')
    expect(spell.durationNote).toBeTruthy()
    expect(spell.bookPage).toBe(184)
  })
})

describe('Concentração de Combate — p185', () => {
  const spell = spellById('concentracao-de-combate')

  it('círculo 1, adivinhação, livre, pessoal, definida 1 rodada, Arc/Bar', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('adivinhacao')
    expect(spell.execution).toBe('livre')
    expect(spell.range).toBe('pessoal')
    expect(spell.duration).toBe('definida')
    expect(spell.durationNote).toBe('1 rodada')
    expect(spell.classes).toEqual(['Arcanista', 'Bardo'])
    expect(spell.bookPage).toBe(185)
  })

  it('augments gated 2º, 3º, 4º, 5º círculo', () => {
    const gates = spell.augments.map((a) => a.requiresCircle).sort()
    expect(gates).toEqual([2, 3, 4, 5])
  })
})

describe('Condição — p185', () => {
  const spell = spellById('condicao')

  it('círculo 2, adivinhação, curto, cena', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('adivinhacao')
    expect(spell.range).toBe('curto')
    expect(spell.duration).toBe('cena')
    expect(spell.bookPage).toBe(185)
  })
})
