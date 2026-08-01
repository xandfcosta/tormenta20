import { describe, expect, it } from 'vitest'
import { DIVINE_POWERS, grantedPowerOptionsFor } from '../divine-power-mechanics'
import { DIVINE_POWER_DESCRIPTIONS } from '../divine-power-descriptions'

describe('DIVINE_POWER_DESCRIPTIONS — catálogo completo (book p132-136)', () => {
  it('cobre todos os 72 poderes concedidos únicos', () => {
    expect(Object.keys(DIVINE_POWER_DESCRIPTIONS)).toHaveLength(72)
  })

  it('todo poder em DIVINE_POWERS tem descrição', () => {
    for (const p of DIVINE_POWERS) {
      expect(DIVINE_POWER_DESCRIPTIONS[p.name], p.name).toBeTruthy()
    }
  })

  it('sem descrições órfãs (nome fora do catálogo mecânico)', () => {
    const known = new Set(DIVINE_POWERS.map((p) => p.name))
    for (const name of Object.keys(DIVINE_POWER_DESCRIPTIONS)) {
      expect(known.has(name), name).toBe(true)
    }
  })

  it('amostras pinadas do livro', () => {
    expect(DIVINE_POWER_DESCRIPTIONS['Coragem Total']).toMatch(
      /imune a efeitos de medo/i,
    )
    expect(DIVINE_POWER_DESCRIPTIONS['Bênção do Mana']).toMatch(
      /\+1 PM a cada nível ímpar/i,
    )
    expect(DIVINE_POWER_DESCRIPTIONS['Urro Divino']).toMatch(/Constituição/)
  })
})

describe('grantedPowerOptionsFor — opções do painel de devoção', () => {
  it('Khalmyr: 4 poderes com descrição', () => {
    const opts = grantedPowerOptionsFor('khalmyr')
    expect(opts.map((o) => o.name).sort()).toEqual([
      'Coragem Total',
      'Dom da Verdade',
      'Espada Justiceira',
      'Reparar Injustiça',
    ])
    for (const o of opts) expect(o.description).toBeTruthy()
  })

  it('deus desconhecido → lista vazia', () => {
    expect(grantedPowerOptionsFor('cthulhu')).toEqual([])
  })
})
