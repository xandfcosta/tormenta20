import { describe, expect, it } from 'vitest'
import { deriveDraftDefense } from './draft-defense'

type V = Parameters<typeof deriveDraftDefense>[0]

const base = (o: Partial<V> = {}): V => ({
  classes: [{ className: 'Guerreiro', level: 1 }],
  races: [],
  dexterity: 2,
  startingArmor: '',
  startingShield: false,
  ...o,
})

describe('deriveDraftDefense', () => {
  it('sem equipamento: 10 + Destreza', () => {
    expect(deriveDraftDefense(base(), {})).toBe(12) // 10 + Des 2
  })

  it('conta a Defesa da armadura leve equipada', () => {
    // armadura-couro = +2 Defesa. 10 + Des 2 + 2 = 14.
    expect(deriveDraftDefense(base({ startingArmor: 'armadura-couro' }), {})).toBe(14)
  })

  it('conta o escudo leve equipado (+1)', () => {
    expect(deriveDraftDefense(base({ startingShield: true }), {})).toBe(13)
  })

  it('soma armadura + escudo', () => {
    const v = base({ startingArmor: 'armadura-couro', startingShield: true })
    expect(deriveDraftDefense(v, {})).toBe(15) // 10 + 2 + 2 + 1
  })

  // Regressão ALE-26: Anão Guerreiro, Des base 2 −1 (raça) = 1, escudo leve
  // marcado → ficha abre com DEF 12. O preview antes mostrava 11 (ignorava o
  // escudo). Agora bate com a ficha.
  it('Anão Guerreiro com escudo leve = DEF 12 (repro do bug)', () => {
    const v = base({ races: ['Anão'], startingShield: true })
    expect(deriveDraftDefense(v, {})).toBe(12) // 10 + (2−1) + 1
  })

  it('Arcanista ignora armadura/escudo (kit sem eles) mesmo se marcados', () => {
    const v = base({
      classes: [{ className: 'Arcanista', level: 1 }],
      startingArmor: 'armadura-couro',
      startingShield: true,
    })
    expect(deriveDraftDefense(v, {})).toBe(12) // 10 + Des 2, sem equip
  })
})
