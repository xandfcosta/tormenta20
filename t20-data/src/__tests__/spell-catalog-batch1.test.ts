import { describe, expect, it } from 'vitest'
import { SPELL_IDS, spellById } from '../spell-catalog'

/**
 * Batch 1 do catálogo — 10 magias verbatim do PDF p178-179.
 *
 * Pins mecânica (círculo, escola, alcance, duração, save/resistance,
 * classes, augments gating) para prevenir regressão silenciosa.
 */

const BATCH1_IDS = [
  'abencoar-alimentos',
  'acalmar-animal',
  'adaga-mental',
  'alarme',
  'aliado-animal',
  'alterar-destino',
  'alterar-memoria',
  'alterar-tamanho',
  'amarras-etereas',
  'ancora-dimensional',
] as const

describe('SPELL_CATALOG batch1 — presença', () => {
  it('as 10 magias estão no catálogo', () => {
    for (const id of BATCH1_IDS) {
      expect(SPELL_IDS).toContain(id)
    }
  })
})

describe('Abençoar Alimentos — p178', () => {
  const spell = spellById('abencoar-alimentos')

  it('círculo 1, transmutação, curto, cena', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('transmutacao')
    expect(spell.range).toBe('curto')
    expect(spell.duration).toBe('cena')
    expect(spell.saveType).toBe('none')
    expect(spell.resistance).toBeNull()
    expect(spell.bookPage).toBe(178)
  })

  it('lista divina (Clérigo, Druida)', () => {
    expect(spell.classes).toEqual(['Clérigo', 'Druida'])
  })

  it('carrega augment truque (+0 PM muda)', () => {
    const truque = spell.augments.find((a) => a.pmCost === 0 && a.kind === 'muda')
    expect(truque).toBeDefined()
  })
})

describe('Acalmar Animal — p178', () => {
  const spell = spellById('acalmar-animal')

  it('círculo 1, encantamento, Vontade anula', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('encantamento')
    expect(spell.saveType).toBe('vontade')
    expect(spell.resistance).toBe('anula')
    expect(spell.bookPage).toBe(178)
  })

  it('augment +5 PM requer 3º círculo (monstro/espírito)', () => {
    const gated = spell.augments.find((a) => a.pmCost === 5)
    expect(gated?.requiresCircle).toBe(3)
  })
})

describe('Adaga Mental — p178', () => {
  const spell = spellById('adaga-mental')

  it('círculo 1, encantamento, instantânea, Vontade parcial', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('encantamento')
    expect(spell.duration).toBe('instantanea')
    expect(spell.saveType).toBe('vontade')
    expect(spell.resistance).toBe('parcial')
    expect(spell.bookPage).toBe(178)
  })

  it('lista arcana (Arcanista, Bardo)', () => {
    expect(spell.classes).toEqual(['Arcanista', 'Bardo'])
  })

  it('augment +2 PM aumenta dano em +1d6', () => {
    const dmg = spell.augments.find((a) => a.pmCost === 2 && a.kind === 'aumenta')
    expect(dmg?.description).toMatch(/\+1d6/)
  })
})

describe('Alarme — p178', () => {
  const spell = spellById('alarme')

  it('círculo 1, abjuração, duração dia', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('abjuracao')
    expect(spell.duration).toBe('dia')
    expect(spell.saveType).toBe('none')
    expect(spell.bookPage).toBe(178)
  })

  it('3 augments (2, 5, 9 PM)', () => {
    const costs = spell.augments.map((a) => a.pmCost).sort((a, b) => a - b)
    expect(costs).toEqual([2, 5, 9])
  })
})

describe('Aliado Animal — p178', () => {
  const spell = spellById('aliado-animal')

  it('círculo 2, encantamento, duração dia', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('encantamento')
    expect(spell.duration).toBe('dia')
    expect(spell.bookPage).toBe(178)
  })

  it('augment +12 PM requer 4º círculo (2 animais)', () => {
    const gated = spell.augments.find((a) => a.pmCost === 12)
    expect(gated?.requiresCircle).toBe(4)
  })
})

describe('Alterar Destino — p179', () => {
  const spell = spellById('alterar-destino')

  it('círculo 5, adivinhação, reação, pessoal, instantânea', () => {
    expect(spell.circle).toBe(5)
    expect(spell.school).toBe('adivinhacao')
    expect(spell.execution).toBe('reacao')
    expect(spell.range).toBe('pessoal')
    expect(spell.duration).toBe('instantanea')
    expect(spell.bookPage).toBe(179)
  })

  it('sem augments', () => {
    expect(spell.augments).toEqual([])
  })
})

describe('Alterar Memória — p179', () => {
  const spell = spellById('alterar-memoria')

  it('círculo 4, encantamento, toque, Vontade anula', () => {
    expect(spell.circle).toBe(4)
    expect(spell.school).toBe('encantamento')
    expect(spell.range).toBe('toque')
    expect(spell.saveType).toBe('vontade')
    expect(spell.resistance).toBe('anula')
    expect(spell.bookPage).toBe(179)
  })
})

describe('Alterar Tamanho — p179', () => {
  const spell = spellById('alterar-tamanho')

  it('círculo 2, transmutação, duração dia', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('transmutacao')
    expect(spell.duration).toBe('dia')
    expect(spell.bookPage).toBe(179)
  })

  it('augments gated por 3º e 4º círculo', () => {
    const gates = spell.augments.map((a) => a.requiresCircle).filter(Boolean)
    expect(gates).toContain(3)
    expect(gates).toContain(4)
  })
})

describe('Amarras Etéreas — p179', () => {
  const spell = spellById('amarras-etereas')

  it('círculo 2, convocação, médio, Reflexos anula', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('convocacao')
    expect(spell.range).toBe('medio')
    expect(spell.saveType).toBe('reflexos')
    expect(spell.resistance).toBe('anula')
    expect(spell.bookPage).toBe(179)
  })

  it('6 augments', () => {
    expect(spell.augments.length).toBe(6)
  })
})

describe('Âncora Dimensional — p179', () => {
  const spell = spellById('ancora-dimensional')

  it('círculo 3, abjuração, curto, cena', () => {
    expect(spell.circle).toBe(3)
    expect(spell.school).toBe('abjuracao')
    expect(spell.range).toBe('curto')
    expect(spell.duration).toBe('cena')
    expect(spell.bookPage).toBe(179)
  })

  it('sem augments', () => {
    expect(spell.augments).toEqual([])
  })
})
