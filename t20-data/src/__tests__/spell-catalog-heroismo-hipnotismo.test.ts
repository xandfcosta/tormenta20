import { describe, expect, it } from 'vitest'
import { spellById, SPELL_IDS } from '../spell-catalog'

/**
 * Cap 4 residual audit 2026-07-05: index p177 (divina 3 Encantamento
 * "Heroísmo") + p174 (arcana 1 Encantamento "Hipnotismo") were missing
 * from the catalog. Both live in Descrição das Magias p194. Pin them
 * so a future cleanup doesn't drop them silently.
 */

describe('Heroísmo (Divina 3, Cap 4 p194)', () => {
  const spell = spellById('heroismo')

  it('is registered under id "heroismo"', () => {
    expect(SPELL_IDS).toContain('heroismo')
  })

  it('circle 3, school encantamento', () => {
    expect(spell.circle).toBe(3)
    expect(spell.school).toBe('encantamento')
  })

  it('touch range, cena duration, no save', () => {
    expect(spell.range).toBe('toque')
    expect(spell.duration).toBe('cena')
    expect(spell.saveType).toBe('none')
  })

  it('classes: Clérigo + Paladino', () => {
    expect(spell.classes).toEqual(['Clérigo', 'Paladino'])
  })

  it('augment +2 PM muda bônus para +6', () => {
    const aug = spell.augments.find((a) => a.pmCost === 2)
    expect(aug).toBeDefined()
    expect(aug?.description).toMatch(/\+6/)
  })

  it('book page = 194', () => {
    expect(spell.bookPage).toBe(194)
  })
})

describe('Hipnotismo (Arcana 1, Cap 4 p194)', () => {
  const spell = spellById('hipnotismo')

  it('is registered under id "hipnotismo"', () => {
    expect(SPELL_IDS).toContain('hipnotismo')
  })

  it('circle 1, school encantamento', () => {
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('encantamento')
  })

  it('curto range, definida duration 1d4 rodadas, vontade anula', () => {
    expect(spell.range).toBe('curto')
    expect(spell.duration).toBe('definida')
    expect(spell.durationNote).toBe('1d4 rodadas')
    expect(spell.saveType).toBe('vontade')
    expect(spell.resistance).toBe('anula')
  })

  it('classes: Arcanista + Bardo', () => {
    expect(spell.classes).toEqual(['Arcanista', 'Bardo'])
  })

  it('has 5 augments (1 PM, 2 PM alvo, 2 PM sustentada, 2 PM espíritos, 5 PM construtos)', () => {
    expect(spell.augments).toHaveLength(5)
  })

  it('augment 5 PM requires circle 3', () => {
    const aug = spell.augments.find((a) => a.pmCost === 5)
    expect(aug?.requiresCircle).toBe(3)
  })

  it('book page = 194', () => {
    expect(spell.bookPage).toBe(194)
  })
})
