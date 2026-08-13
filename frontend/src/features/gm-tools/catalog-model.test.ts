import type { CatalogSpell, Condition } from '@/shared/api/catalog-types'
import type { CatalogItem } from '@/shared/api/item-types'
import { describe, expect, it } from 'vitest'
import {
  type SearchableCatalogs,
  catalogSearchRows,
  matchesAllTerms,
} from './catalog-model'

const condition = (id: string, name: string, description = '') =>
  ({ id, name, description, tags: [] }) as unknown as Condition
const spell = (id: string, name: string, baseEffect = '') =>
  ({ id, name, baseEffect }) as unknown as CatalogSpell
const item = (id: string, name: string, category = 'geral') =>
  ({ id, name, category }) as unknown as CatalogItem

const CATALOGS: SearchableCatalogs = {
  conditions: [condition('abalado', 'Abalado', '−2 em testes de perícia')],
  spells: [spell('bola-de-fogo', 'Bola de Fogo', 'dano de fogo em área')],
  powers: [{ id: 'p1', name: 'Ataque Poderoso', source: 'Geral · combate', description: '' }],
  items: [item('adaga', 'Adaga', 'weapon-simple')],
}

describe('matchesAllTerms', () => {
  it('exige TODOS os termos, não qualquer um', () => {
    expect(matchesAllTerms(['Bola de Fogo', 'dano de fogo'], 'bola fogo')).toBe(true)
    expect(matchesAllTerms(['Bola de Fogo'], 'bola gelo')).toBe(false)
  })

  it('busca vazia casa com tudo', () => {
    expect(matchesAllTerms(['qualquer coisa'], '')).toBe(true)
    expect(matchesAllTerms(['qualquer coisa'], '   ')).toBe(true)
  })

  it('ignora acento e caixa — o catálogo é todo acentuado', () => {
    expect(matchesAllTerms(['Condições'], 'condicoes')).toBe(true)
    expect(matchesAllTerms(['Abalado'], 'ABALADO')).toBe(true)
  })

  it('casa por pedaço de palavra', () => {
    expect(matchesAllTerms(['Bola de Fogo'], 'fog')).toBe(true)
  })
})

describe('catalogSearchRows', () => {
  it('agrupa os achados por catálogo, com cabeçalho e contagem', () => {
    const rows = catalogSearchRows('fogo', CATALOGS)

    expect(rows[0]).toMatchObject({ kind: 'header', label: 'Magias', count: 1 })
    expect(rows[1]).toMatchObject({ kind: 'spell' })
  })

  it('omite o catálogo sem acerto — cabeçalho vazio é ruído', () => {
    const labels = catalogSearchRows('fogo', CATALOGS)
      .filter((row) => row.kind === 'header')
      .map((row) => (row.kind === 'header' ? row.label : ''))

    expect(labels).toEqual(['Magias'])
  })

  it('atravessa os quatro catálogos numa busca só', () => {
    const kinds = new Set(catalogSearchRows('', CATALOGS).map((row) => row.kind))

    expect(kinds).toEqual(new Set(['header', 'condition', 'spell', 'power', 'item']))
  })

  it('busca sem acerto nenhum devolve lista vazia', () => {
    expect(catalogSearchRows('zzzznada', CATALOGS)).toEqual([])
  })

  it('cada linha tem chave única — a lista virtual reconcilia por ela', () => {
    const keys = catalogSearchRows('', CATALOGS).map((row) => row.key)

    expect(new Set(keys).size).toBe(keys.length)
  })
})
