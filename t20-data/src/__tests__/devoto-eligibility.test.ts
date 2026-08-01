import { describe, expect, it } from 'vitest'
import { DEUS_BY_ID } from '../abilities/deuses'
import { devotoEligible } from '../devoto-eligibility'

describe('devotoEligible — p96 (raça/classe na lista de Devotos)', () => {
  it('Anão pode ser devoto de Arsenal (Anões listados)', () => {
    expect(devotoEligible(DEUS_BY_ID.arsenal, ['Anão'], ['Bardo'])).toBe(true)
  })

  it('classe listada basta (Guerreiro → Arsenal)', () => {
    expect(devotoEligible(DEUS_BY_ID.arsenal, ['Elfo'], ['Guerreiro'])).toBe(true)
  })

  it('fora da lista → false (Elfo Bardo → Arsenal)', () => {
    expect(devotoEligible(DEUS_BY_ID.arsenal, ['Elfo'], ['Bardo'])).toBe(false)
  })

  it('Humano é exceção: qualquer deus', () => {
    expect(devotoEligible(DEUS_BY_ID.arsenal, ['Humano'], ['Bardo'])).toBe(true)
  })

  it('Clérigo é exceção: qualquer deus', () => {
    expect(devotoEligible(DEUS_BY_ID.arsenal, ['Elfo'], ['Clérigo'])).toBe(true)
  })

  it('"Quaisquer" (Aharadak) aceita todos', () => {
    expect(devotoEligible(DEUS_BY_ID.aharadak, ['Golem'], ['Inventor'])).toBe(true)
  })

  it('Aggelus/Sulfure mapeiam para Suraggel', () => {
    // Azgher lista Aggelus; Suraggel conta.
    expect(devotoEligible(DEUS_BY_ID.azgher, ['Suraggel'], ['Ladino'])).toBe(true)
  })
})
