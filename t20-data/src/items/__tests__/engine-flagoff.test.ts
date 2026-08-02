import { describe, expect, it } from 'vitest'
import { computeItemEffects, statFor } from '../engine'
import type { ActiveItem } from '../engine'

const peleDeFerro: ActiveItem = {
  source: 'Classe: Bárbaro',
  equipped: 'vested',
  modifiers: [
    {
      target: { k: 'defense' },
      amount: 4,
      bonusType: 'untyped',
      condition: {
        c: 'flagOff',
        flag: 'armadura-pesada',
        label: 'sem armadura pesada',
      },
    },
  ],
}

const brunea: ActiveItem = {
  source: 'Brunea',
  equipped: 'vested',
  modifiers: [
    {
      target: { k: 'defense' },
      amount: 5,
      bonusType: 'armor',
      condition: { c: 'vested' },
    },
    {
      target: { k: 'flag', name: 'armadura-pesada' },
      amount: 1,
      bonusType: 'untyped',
      condition: { c: 'vested' },
    },
  ],
}

describe('flagOff — condição auto-avaliada (Pele de Ferro p42)', () => {
  it('aplica quando a flag está ausente', () => {
    const effects = computeItemEffects([peleDeFerro])
    expect(statFor(effects, { k: 'defense' }).total).toBe(4)
  })

  it('desliga com armadura pesada vestida', () => {
    const effects = computeItemEffects([peleDeFerro, brunea])
    expect(statFor(effects, { k: 'defense' }).total).toBe(5) // só a brunea
  })

  it('independe da ordem dos itens (pré-pass de flags)', () => {
    const effects = computeItemEffects([brunea, peleDeFerro])
    expect(statFor(effects, { k: 'defense' }).total).toBe(5)
  })

  it('não vira conditional togglável', () => {
    const effects = computeItemEffects([peleDeFerro])
    expect(effects.conditional).toHaveLength(0)
  })
})
