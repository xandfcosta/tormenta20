import { describe, expect, it } from 'vitest'
import { SPELL_IDS, spellById } from '../spell-catalog'

/**
 * Batch 5 do catálogo — 10 magias verbatim do PDF p185-188.
 */

const BATCH5_IDS = [
  'conjurar-elemental',
  'consagrar',
  'contato-extraplanar',
  'controlar-a-gravidade',
  'controlar-agua',
  'controlar-fogo',
  'controlar-madeira',
  'controlar-o-clima',
  'controlar-o-tempo',
  'controlar-plantas',
] as const

describe('SPELL_CATALOG batch5 — presença', () => {
  it('as 10 magias estão no catálogo', () => {
    for (const id of BATCH5_IDS) {
      expect(SPELL_IDS).toContain(id)
    }
  })
})

describe('Conjurar Elemental — p185', () => {
  const spell = spellById('conjurar-elemental')

  it('círculo 4, convocação, completa, médio, sustentada, Arc/Bar', () => {
    expect(spell.circle).toBe(4)
    expect(spell.school).toBe('convocacao')
    expect(spell.execution).toBe('completa')
    expect(spell.range).toBe('medio')
    expect(spell.duration).toBe('sustentada')
    expect(spell.classes).toEqual(['Arcanista', 'Bardo'])
    expect(spell.bookPage).toBe(185)
  })

  it('augment +5 PM 5º círc convoca todos elementais', () => {
    const gated = spell.augments.find((a) => a.requiresCircle === 5)
    expect(gated?.pmCost).toBe(5)
  })
})

describe('Consagrar — p186', () => {
  const spell = spellById('consagrar')

  it('círculo 1, evocação, longo, dia', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('evocacao')
    expect(spell.range).toBe('longo')
    expect(spell.duration).toBe('dia')
    expect(spell.bookPage).toBe(186)
  })

  it('augment +9 PM permanente gated 4º círc', () => {
    const aug = spell.augments.find((a) => a.pmCost === 9)
    expect(aug?.requiresCircle).toBe(4)
    expect(aug?.description).toMatch(/permanente/)
  })
})

describe('Contato Extraplanar — p186', () => {
  const spell = spellById('contato-extraplanar')

  it('círculo 2, adivinhação, completa, pessoal, dia', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('adivinhacao')
    expect(spell.execution).toBe('completa')
    expect(spell.range).toBe('pessoal')
    expect(spell.duration).toBe('dia')
    expect(spell.bookPage).toBe(186)
  })
})

describe('Controlar a Gravidade — p186', () => {
  const spell = spellById('controlar-a-gravidade')

  it('círculo 4, transmutação, sustentada, sem augments', () => {
    expect(spell.circle).toBe(4)
    expect(spell.school).toBe('transmutacao')
    expect(spell.duration).toBe('sustentada')
    expect(spell.augments).toEqual([])
    expect(spell.bookPage).toBe(186)
  })
})

describe('Controlar Água — p186', () => {
  const spell = spellById('controlar-agua')

  it('círculo 3, transmutação, longo, cena', () => {
    expect(spell.circle).toBe(3)
    expect(spell.school).toBe('transmutacao')
    expect(spell.range).toBe('longo')
    expect(spell.duration).toBe('cena')
    expect(spell.bookPage).toBe(186)
  })
})

describe('Controlar Fogo — p187', () => {
  const spell = spellById('controlar-fogo')

  it('círculo 2, evocação, curto, cena', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('evocacao')
    expect(spell.bookPage).toBe(187)
  })

  it('3 augments', () => {
    expect(spell.augments.length).toBe(3)
  })
})

describe('Controlar Madeira — p187', () => {
  const spell = spellById('controlar-madeira')

  it('círculo 2, transmutação, médio, cena', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('transmutacao')
    expect(spell.range).toBe('medio')
    expect(spell.bookPage).toBe(187)
  })

  it('augments +7 gated 3º, +12 gated 4º', () => {
    const c3 = spell.augments.find((a) => a.pmCost === 7)
    const c4 = spell.augments.find((a) => a.pmCost === 12)
    expect(c3?.requiresCircle).toBe(3)
    expect(c4?.requiresCircle).toBe(4)
  })
})

describe('Controlar o Clima — p187', () => {
  const spell = spellById('controlar-o-clima')

  it('círculo 4, transmutação, completa, longo, definida 4d12 horas', () => {
    expect(spell.circle).toBe(4)
    expect(spell.school).toBe('transmutacao')
    expect(spell.execution).toBe('completa')
    expect(spell.range).toBe('longo')
    expect(spell.duration).toBe('definida')
    expect(spell.durationNote).toBe('4d12 horas')
    expect(spell.bookPage).toBe(187)
  })

  it('augment +1 PM apenas druidas', () => {
    const aug = spell.augments.find((a) => a.pmCost === 1)
    expect(aug?.classOnly).toBe('druidas')
  })
})

describe('Controlar o Tempo — p187', () => {
  const spell = spellById('controlar-o-tempo')

  it('círculo 5, transmutação, pessoal, Arc/Bar, sem augments', () => {
    expect(spell.circle).toBe(5)
    expect(spell.school).toBe('transmutacao')
    expect(spell.range).toBe('pessoal')
    expect(spell.classes).toEqual(['Arcanista', 'Bardo'])
    expect(spell.augments).toEqual([])
    expect(spell.bookPage).toBe(187)
  })
})

describe('Controlar Plantas — p188', () => {
  const spell = spellById('controlar-plantas')

  it('círculo 1, transmutação, curto, cena, Reflexos anula', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('transmutacao')
    expect(spell.range).toBe('curto')
    expect(spell.duration).toBe('cena')
    expect(spell.saveType).toBe('reflexos')
    expect(spell.resistance).toBe('anula')
    expect(spell.bookPage).toBe(188)
  })

  it('carrega augment truque +0 PM', () => {
    const truque = spell.augments.find((a) => a.pmCost === 0 && a.kind === 'muda')
    expect(truque?.description).toMatch(/[Tt]ruque/)
  })
})
