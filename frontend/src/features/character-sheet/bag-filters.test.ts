import { describe, expect, it } from 'vitest'
import type { CharacterItem } from '@/shared/api/api'
import { filterStowed, matchesBagFilter } from './bag-filters'

function item(overrides: Partial<CharacterItem> = {}): CharacterItem {
  return {
    id: 1,
    catalogId: null,
    name: 'Item',
    quantity: 1,
    slots: 1,
    equipped: null,
    improvements: '[]',
    material: null,
    ...overrides,
  }
}

const espada = item({ id: 1, catalogId: 'espada-longa', name: 'Espada longa' })
const escudo = item({ id: 2, catalogId: 'escudo-leve', name: 'Escudo leve' })
const balsamo = item({ id: 3, catalogId: 'balsamo-restaurador', name: 'Bálsamo restaurador' })
const corda = item({ id: 4, name: 'Corda de cânhamo' })

describe('matchesBagFilter', () => {
  it('"tudo" não filtra nada', () => {
    expect([espada, escudo, balsamo, corda].every((i) => matchesBagFilter(i, 'all'))).toBe(true)
  })

  it('separa armas, defesa e consumo pela categoria do catálogo', () => {
    expect(matchesBagFilter(espada, 'weapons')).toBe(true)
    expect(matchesBagFilter(escudo, 'defense')).toBe(true)
    expect(matchesBagFilter(balsamo, 'consumables')).toBe(true)
    expect(matchesBagFilter(espada, 'defense')).toBe(false)
  })

  // Item inventado não tem categoria; ele não pode sumir da mochila — cai em
  // "outros" e continua visível em "tudo".
  it('item custom conta como "outros"', () => {
    expect(matchesBagFilter(corda, 'other')).toBe(true)
    expect(matchesBagFilter(corda, 'weapons')).toBe(false)
    expect(matchesBagFilter(escudo, 'other')).toBe(false)
  })
})

describe('filterStowed', () => {
  const all = [espada, escudo, balsamo, corda]

  it('sem busca nem filtro, devolve tudo', () => {
    expect(filterStowed(all, '  ', 'all')).toHaveLength(4)
  })

  // Ninguém digita acento no meio da mesa.
  it('acha ignorando acento e caixa', () => {
    expect(filterStowed(all, 'balsamo', 'all')).toEqual([balsamo])
    expect(filterStowed(all, 'CANHAMO', 'all')).toEqual([corda])
  })

  it('cruza busca e categoria', () => {
    expect(filterStowed(all, 'e', 'defense')).toEqual([escudo])
  })
})
