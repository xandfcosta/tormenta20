import type { Monster } from '@/shared/api/catalog-types'
import { describe, expect, it } from 'vitest'
import { EMPTY_MONSTER_CRITERIA, filterMonsters } from './monster-filter'

const monster = (over: Partial<Monster>): Monster =>
  ({
    id: 'x',
    name: 'Monstro',
    nd: 1,
    tipo: 'monstro',
    size: 'médio',
    hp: 10,
    defesa: 12,
    forca: 0,
    destreza: 0,
    constituicao: 0,
    inteligencia: 0,
    sabedoria: 0,
    carisma: 0,
    fortitude: 0,
    reflexos: 0,
    vontade: 0,
    deslocamento: '9m',
    attacks: [],
    specialAbilities: [],
    bookPage: 1,
    ...over,
  }) as Monster

const BESTIARY = [
  monster({ id: 'goblin', name: 'Goblin Salteador', nd: 0.25, tipo: 'humanoide' }),
  monster({ id: 'lobo', name: 'Lobo', nd: 1, tipo: 'animal' }),
  monster({ id: 'anao-zumbi', name: 'Anão Zumbi', nd: 2, tipo: 'morto-vivo' }),
  monster({ id: 'dragao', name: 'Dragão Vermelho', nd: 15, tipo: 'monstro' }),
]

const criteria = (over: Partial<typeof EMPTY_MONSTER_CRITERIA> = {}) => ({
  ...EMPTY_MONSTER_CRITERIA,
  ...over,
})

describe('filterMonsters', () => {
  it('sem critério nenhum devolve o bestiário inteiro', () => {
    expect(filterMonsters(BESTIARY, criteria())).toHaveLength(4)
  })

  it('ordena por ND e depois por nome — o mestre procura por ameaça', () => {
    expect(filterMonsters(BESTIARY, criteria()).map((m) => m.id)).toEqual([
      'goblin',
      'lobo',
      'anao-zumbi',
      'dragao',
    ])
  })

  it('busca por nome ignorando acento', () => {
    // "dragao" tem de achar "Dragão" — o catálogo é todo acentuado.
    expect(filterMonsters(BESTIARY, criteria({ name: 'dragao' })).map((m) => m.id)).toEqual([
      'dragao',
    ])
  })

  it('filtra por tipo, aceitando mais de um', () => {
    const found = filterMonsters(
      BESTIARY,
      criteria({ tipos: new Set(['animal', 'morto-vivo'] as const) }),
    )

    expect(found.map((m) => m.id)).toEqual(['lobo', 'anao-zumbi'])
  })

  it('tipo vazio significa TODOS, não nenhum', () => {
    expect(filterMonsters(BESTIARY, criteria({ tipos: new Set() }))).toHaveLength(4)
  })

  it('corta pela faixa de ND, incluindo as pontas', () => {
    const found = filterMonsters(BESTIARY, criteria({ ndMin: 1, ndMax: 2 }))

    expect(found.map((m) => m.id)).toEqual(['lobo', 'anao-zumbi'])
  })

  it('combina nome, tipo e faixa', () => {
    const found = filterMonsters(
      BESTIARY,
      criteria({ name: 'a', tipos: new Set(['morto-vivo'] as const), ndMin: 0, ndMax: 20 }),
    )

    expect(found.map((m) => m.id)).toEqual(['anao-zumbi'])
  })

  it('devolve lista vazia quando nada casa, em vez de tudo', () => {
    expect(filterMonsters(BESTIARY, criteria({ name: 'zzzznada' }))).toEqual([])
  })
})
