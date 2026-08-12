import { describe, expect, it } from 'vitest'
import { originSwitchPatch } from './origin-switch'
import { origemRolledMoneySum } from './starting-equipment'
import { wizardDefaults } from './wizard-steps'

describe('originSwitchPatch', () => {
  it('sets the new origin and clears the previous picks', () => {
    const patch = originSwitchPatch(
      {
        ...wizardDefaults,
        origin: 'Acólito',
        originChoices: ['acolito-cura-leve'],
        originItemPicks: { 'Símbolo sagrado': 'simbolo-sagrado' },
      },
      'Batedor',
    )
    expect(patch.origin).toBe('Batedor')
    expect(patch.originChoices).toEqual([])
    expect(patch.originItemPicks).toEqual({})
  })

  it('gives back the T$ the previous origin had already rolled', () => {
    const picks = { 'T$ 2d6': '7' }
    const patch = originSwitchPatch(
      { ...wizardDefaults, origin: 'Batedor', originItemPicks: picks, tibar: 12 },
      'Artesão',
    )
    // Whatever the previous origin rolled leaves with it — money must not leak
    // from an origin the character no longer has.
    expect(patch.tibar).toBe(12 - origemRolledMoneySum('Batedor', picks))
  })

  it('never drives tibar negative', () => {
    const patch = originSwitchPatch(
      { ...wizardDefaults, origin: 'Batedor', originItemPicks: { 'T$ 2d6': '9' }, tibar: 0 },
      'Artesão',
    )
    expect(patch.tibar).toBe(0)
  })

  it('drops the power picked for a benefit of the origin being left', () => {
    const patch = originSwitchPatch(
      {
        ...wizardDefaults,
        origin: 'Soldado',
        originChoices: ['origin-soldado-poder-poder-de-combate-escolha'],
        powerChoices: { 'origin-soldado-poder-poder-de-combate-escolha': ['ataque-poderoso'], 'racial-x': ['dentes-afiados'] },
      },
      'Artesão',
    )
    expect(patch.powerChoices['origin-soldado-poder-poder-de-combate-escolha']).toBeUndefined()
    // Picks that belong to anything else (race, class) are not this step's to erase.
    expect(patch.powerChoices['racial-x']).toEqual(['dentes-afiados'])
  })

  it('is a no-op on tibar when there was no origin yet', () => {
    const patch = originSwitchPatch({ ...wizardDefaults, tibar: 5 }, 'Acólito')
    expect(patch.tibar).toBe(5)
    expect(patch.origin).toBe('Acólito')
  })
})
