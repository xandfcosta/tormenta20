import { describe, expect, it } from 'vitest'
import { catalogCategories, categoryLabel } from './catalog-categories'

describe('categoryLabel', () => {
  it('traduz as categorias do catálogo', () => {
    expect(categoryLabel('weapon-martial')).toBe('Arma marcial')
    expect(categoryLabel('armor-heavy')).toBe('Armadura pesada')
  })

  // Categoria nova no catálogo tem de aparecer feia, não sumir do filtro.
  it('deixa passar um id que ainda não tem tradução', () => {
    expect(categoryLabel('relic-unknown')).toBe('relic-unknown')
  })
})

describe('catalogCategories', () => {
  it('lista as categorias sem repetir e em ordem', () => {
    const categories = catalogCategories()
    expect(categories).toEqual([...new Set(categories)].sort())
    expect(categories).toContain('shield')
  })
})
