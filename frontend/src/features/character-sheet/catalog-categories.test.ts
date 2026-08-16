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
  // A asserção anterior era `toEqual([...new Set(x)].sort())` — o corpo da
  // função re-escrito no teste. O que importa para a tela é que a lista tem as
  // categorias que o filtro do catálogo oferece, sem repetir.
  it('traz as categorias do catálogo, sem repetir', () => {
    const categories = catalogCategories()

    expect(categories).toContain('shield')
    expect(categories).toContain('weapon-simple')
    expect(new Set(categories).size).toBe(categories.length)
  })
})
