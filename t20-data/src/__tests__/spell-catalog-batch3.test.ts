import { describe, expect, it } from 'vitest'
import { SPELL_IDS, spellById } from '../spell-catalog'

/**
 * Batch 3 do catálogo — 10 magias verbatim do PDF p182-183.
 */

const BATCH3_IDS = [
  'aura-divina',
  'aviso',
  'banimento',
  'barragem-elemental-de-vectorius',
  'bencao',
  'buraco-negro',
  'campo-antimagia',
  'campo-de-forca',
  'camuflagem-ilusoria',
  'chuva-de-meteoros',
] as const

describe('SPELL_CATALOG batch3 — presença', () => {
  it('as 10 magias estão no catálogo', () => {
    for (const id of BATCH3_IDS) {
      expect(SPELL_IDS).toContain(id)
    }
  })
})

describe('Aura Divina — p182', () => {
  const spell = spellById('aura-divina')

  it('círculo 5, abjuração, Clérigo apenas, Vontade parcial', () => {
    expect(spell.circle).toBe(5)
    expect(spell.school).toBe('abjuracao')
    expect(spell.classes).toEqual(['Clérigo'])
    expect(spell.saveType).toBe('vontade')
    expect(spell.resistance).toBe('parcial')
    expect(spell.bookPage).toBe(182)
  })
})

describe('Aviso — p182', () => {
  const spell = spellById('aviso')

  it('círculo 1, adivinhação, movimento, longo, universal 4 classes', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('adivinhacao')
    expect(spell.execution).toBe('movimento')
    expect(spell.range).toBe('longo')
    expect(spell.classes).toEqual(['Arcanista', 'Bardo', 'Clérigo', 'Druida'])
    expect(spell.bookPage).toBe(182)
  })

  it('4 augments', () => {
    expect(spell.augments.length).toBe(4)
  })
})

describe('Banimento — p182', () => {
  const spell = spellById('banimento')

  it('círculo 3, abjuração, completa (ritual 1d3+1 rodadas), Vontade parcial', () => {
    expect(spell.circle).toBe(3)
    expect(spell.school).toBe('abjuracao')
    expect(spell.execution).toBe('completa')
    expect(spell.saveType).toBe('vontade')
    expect(spell.resistance).toBe('parcial')
    expect(spell.bookPage).toBe(182)
  })

  it('nota de execução ritual em baseEffect', () => {
    expect(spell.baseEffect).toMatch(/1d3\+1 rodadas/)
  })

  it('augment +0 PM muda remove resistência', () => {
    const zero = spell.augments.find((a) => a.pmCost === 0)
    expect(zero?.kind).toBe('muda')
    expect(zero?.description).toMatch(/resistência para nenhum/)
  })
})

describe('Barragem Elemental de Vectorius — p182', () => {
  const spell = spellById('barragem-elemental-de-vectorius')

  it('círculo 5, evocação, longo, Reflexos parcial, Arcanista apenas', () => {
    expect(spell.circle).toBe(5)
    expect(spell.school).toBe('evocacao')
    expect(spell.range).toBe('longo')
    expect(spell.saveType).toBe('reflexos')
    expect(spell.resistance).toBe('parcial')
    expect(spell.classes).toEqual(['Arcanista'])
    expect(spell.bookPage).toBe(182)
  })
})

describe('Bênção — p182', () => {
  const spell = spellById('bencao')

  it('círculo 1, encantamento, curto, cena, Cle/Dru/Paladino', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('encantamento')
    expect(spell.classes).toEqual(['Clérigo', 'Druida', 'Paladino'])
    expect(spell.bookPage).toBe(182)
  })
})

describe('Buraco Negro — p182', () => {
  const spell = spellById('buraco-negro')

  it('círculo 5, convocação, completa, longo, Fortitude parcial, definida 3 rodadas', () => {
    expect(spell.circle).toBe(5)
    expect(spell.school).toBe('convocacao')
    expect(spell.execution).toBe('completa')
    expect(spell.range).toBe('longo')
    expect(spell.saveType).toBe('fortitude')
    expect(spell.resistance).toBe('parcial')
    expect(spell.duration).toBe('definida')
    expect(spell.durationNote).toBe('3 rodadas')
    expect(spell.classes).toEqual(['Arcanista', 'Clérigo'])
    expect(spell.bookPage).toBe(182)
  })
})

describe('Campo Antimagia — p183', () => {
  const spell = spellById('campo-antimagia')

  it('círculo 4, abjuração, sustentada, sem augments', () => {
    expect(spell.circle).toBe(4)
    expect(spell.school).toBe('abjuracao')
    expect(spell.duration).toBe('sustentada')
    expect(spell.augments).toEqual([])
    expect(spell.bookPage).toBe(183)
  })
})

describe('Campo de Força — p183', () => {
  const spell = spellById('campo-de-forca')

  it('círculo 2, abjuração, pessoal, cena', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('abjuracao')
    expect(spell.bookPage).toBe(183)
  })

  it('augments +3 PM 3º círc, +7 PM 4º círc, +9 PM 4º círc', () => {
    const c3 = spell.augments.find((a) => a.pmCost === 3)
    const c4 = spell.augments.filter((a) => a.requiresCircle === 4)
    expect(c3?.requiresCircle).toBe(3)
    expect(c4.length).toBe(3)
  })
})

describe('Camuflagem Ilusória — p183', () => {
  const spell = spellById('camuflagem-ilusoria')

  it('círculo 2, ilusão, toque, Arc/Bar', () => {
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('ilusao')
    expect(spell.range).toBe('toque')
    expect(spell.classes).toEqual(['Arcanista', 'Bardo'])
    expect(spell.bookPage).toBe(183)
  })
})

describe('Chuva de Meteoros — p183', () => {
  const spell = spellById('chuva-de-meteoros')

  it('círculo 5, convocação, completa, longo, Reflexos parcial, Arcanista', () => {
    expect(spell.circle).toBe(5)
    expect(spell.school).toBe('convocacao')
    expect(spell.execution).toBe('completa')
    expect(spell.range).toBe('longo')
    expect(spell.saveType).toBe('reflexos')
    expect(spell.resistance).toBe('parcial')
    expect(spell.classes).toEqual(['Arcanista'])
    expect(spell.bookPage).toBe(183)
  })
})
