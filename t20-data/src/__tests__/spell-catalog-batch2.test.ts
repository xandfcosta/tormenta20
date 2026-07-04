import { describe, expect, it } from 'vitest'
import { SPELL_IDS, spellById } from '../spell-catalog'

/**
 * Batch 2 do catálogo — 10 magias verbatim do PDF p179-181.
 */

const BATCH2_IDS = [
  'animar-objetos',
  'anular-a-luz',
  'aparencia-perfeita',
  'aprisionamento',
  'area-escorregadia',
  'arma-espiritual',
  'arma-magica',
  'armamento-da-natureza',
  'assassino-fantasmagorico',
  'augurio',
] as const

describe('SPELL_CATALOG batch2 — presença', () => {
  it('as 10 magias estão no catálogo', () => {
    for (const id of BATCH2_IDS) {
      expect(SPELL_IDS).toContain(id)
    }
  })
})

describe('Animar Objetos — p179', () => {
  const spell = spellById('animar-objetos')

  it('círculo 4, transmutação, médio, cena', () => {
    expect(spell.circle).toBe(4)
    expect(spell.school).toBe('transmutacao')
    expect(spell.range).toBe('medio')
    expect(spell.duration).toBe('cena')
    expect(spell.bookPage).toBe(179)
  })

  it('augment +5 PM muda para permanente', () => {
    const aug = spell.augments.find((a) => a.pmCost === 5)
    expect(aug?.kind).toBe('muda')
    expect(aug?.description).toMatch(/permanente/)
  })
})

describe('Anular a Luz — p180', () => {
  const spell = spellById('anular-a-luz')

  it('círculo 3, necromancia, duração definida com nota', () => {
    expect(spell.circle).toBe(3)
    expect(spell.school).toBe('necromancia')
    expect(spell.duration).toBe('definida')
    expect(spell.durationNote).toBeTruthy()
    expect(spell.bookPage).toBe(180)
  })

  it('augments +4 e +9 PM gated 4º e 5º círculo', () => {
    const c4 = spell.augments.find((a) => a.pmCost === 4)
    const c5 = spell.augments.find((a) => a.pmCost === 9)
    expect(c4?.requiresCircle).toBe(4)
    expect(c5?.requiresCircle).toBe(5)
  })
})

describe('Aparência Perfeita — p180', () => {
  const spell = spellById('aparencia-perfeita')

  it('círculo 2, ilusão, pessoal, cena', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('ilusao')
    expect(spell.range).toBe('pessoal')
    expect(spell.duration).toBe('cena')
    expect(spell.bookPage).toBe(180)
  })
})

describe('Aprisionamento — p180', () => {
  const spell = spellById('aprisionamento')

  it('círculo 5, abjuração, completa, permanente, Vontade anula', () => {
    expect(spell.circle).toBe(5)
    expect(spell.school).toBe('abjuracao')
    expect(spell.execution).toBe('completa')
    expect(spell.duration).toBe('permanente')
    expect(spell.saveType).toBe('vontade')
    expect(spell.resistance).toBe('anula')
    expect(spell.bookPage).toBe(180)
  })

  it('exige componente material', () => {
    expect(spell.components).toContain('material')
  })

  it('sem augments', () => {
    expect(spell.augments).toEqual([])
  })
})

describe('Área Escorregadia — p180', () => {
  const spell = spellById('area-escorregadia')

  it('círculo 1, convocação, Reflexos anula', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('convocacao')
    expect(spell.saveType).toBe('reflexos')
    expect(spell.resistance).toBe('anula')
    expect(spell.bookPage).toBe(180)
  })
})

describe('Arma Espiritual — p180', () => {
  const spell = spellById('arma-espiritual')

  it('círculo 1, convocação, pessoal, cena, Clérigo/Druida', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('convocacao')
    expect(spell.range).toBe('pessoal')
    expect(spell.classes).toEqual(['Clérigo', 'Druida'])
    expect(spell.bookPage).toBe(180)
  })

  it('augment +5 PM gated 3º círculo (2 armas)', () => {
    const aug = spell.augments.find((a) => a.pmCost === 5)
    expect(aug?.requiresCircle).toBe(3)
  })

  it('augment +2 PM essência gated 2º círculo', () => {
    const essencia = spell.augments.find(
      (a) => a.pmCost === 2 && a.description.includes('essência'),
    )
    expect(essencia?.requiresCircle).toBe(2)
  })
})

describe('Arma Mágica — p181', () => {
  const spell = spellById('arma-magica')

  it('círculo 1, transmutação, toque, todas 4 classes conjuradoras', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('transmutacao')
    expect(spell.range).toBe('toque')
    expect(spell.classes).toEqual(['Arcanista', 'Bardo', 'Clérigo', 'Druida'])
    expect(spell.bookPage).toBe(181)
  })
})

describe('Armamento da Natureza — p181', () => {
  const spell = spellById('armamento-da-natureza')

  it('círculo 1, transmutação, Clérigo/Druida', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('transmutacao')
    expect(spell.classes).toEqual(['Clérigo', 'Druida'])
    expect(spell.bookPage).toBe(181)
  })

  it('4 augments', () => {
    expect(spell.augments.length).toBe(4)
  })
})

describe('Assassino Fantasmagórico — p181', () => {
  const spell = spellById('assassino-fantasmagorico')

  it('círculo 4, necromancia, longo, Vontade anula, definida', () => {
    expect(spell.circle).toBe(4)
    expect(spell.school).toBe('necromancia')
    expect(spell.range).toBe('longo')
    expect(spell.saveType).toBe('vontade')
    expect(spell.resistance).toBe('anula')
    expect(spell.duration).toBe('definida')
    expect(spell.durationNote).toBeTruthy()
    expect(spell.bookPage).toBe(181)
  })

  it('sem augments', () => {
    expect(spell.augments).toEqual([])
  })
})

describe('Augúrio — p181', () => {
  const spell = spellById('augurio')

  it('círculo 2, adivinhação, completa, pessoal, instantânea', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('adivinhacao')
    expect(spell.execution).toBe('completa')
    expect(spell.range).toBe('pessoal')
    expect(spell.duration).toBe('instantanea')
    expect(spell.bookPage).toBe(181)
  })

  it('augments +3 e +7 PM gated 3º e 4º círculo', () => {
    const c3 = spell.augments.find((a) => a.pmCost === 3)
    const c4 = spell.augments.find(
      (a) => a.pmCost === 7 && a.requiresCircle === 4,
    )
    expect(c3?.requiresCircle).toBe(3)
    expect(c4).toBeDefined()
  })
})
