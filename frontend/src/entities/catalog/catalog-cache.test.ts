import { beforeEach, describe, expect, it } from 'vitest'
import type { CatalogItem } from '@tormenta20/t20-data'
import {
  getCatalogItem,
  isItemCatalogPrimed,
  primeItemCatalog,
} from './catalog-cache'

function item(id: string, name: string): CatalogItem {
  return { id, name } as CatalogItem
}

describe('catalog-cache (item accessor)', () => {
  beforeEach(() => primeItemCatalog([]))

  it('resolves an item by id once primed', () => {
    primeItemCatalog([item('espada-longa', 'Espada Longa'), item('adaga', 'Adaga')])
    expect(getCatalogItem('adaga')?.name).toBe('Adaga')
    expect(isItemCatalogPrimed()).toBe(true)
  })

  it('returns undefined for an unknown id (mirrors t20-data getCatalogItem)', () => {
    primeItemCatalog([item('adaga', 'Adaga')])
    expect(getCatalogItem('nope')).toBeUndefined()
  })

  it('re-priming replaces the previous catalog', () => {
    primeItemCatalog([item('a', 'A')])
    primeItemCatalog([item('b', 'B')])
    expect(getCatalogItem('a')).toBeUndefined()
    expect(getCatalogItem('b')?.name).toBe('B')
  })
})
