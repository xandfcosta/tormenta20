import { describe, expect, it } from 'vitest'
import { DEFAULT_TOOL, GM_TOOLS, isToolSlug, toolLabel } from './gm-tools'

describe('isToolSlug — slug vindo da URL', () => {
  it('aceita uma ferramenta real', () => {
    expect(isToolSlug('bestiario')).toBe(true)
  })

  it('recusa qualquer outra coisa', () => {
    expect(isToolSlug('bestiary')).toBe(false)
    expect(isToolSlug('')).toBe(false)
  })

  // As tabelas e o gerador de masmorra viraram uma ferramenta só.
  it('não conhece mais tabelas e masmorras separadas', () => {
    expect(isToolSlug('tabelas')).toBe(false)
    expect(isToolSlug('masmorras')).toBe(false)
  })
})

describe('GM_TOOLS', () => {
  it('toda ferramenta tem rótulo e explicação', () => {
    for (const tool of GM_TOOLS) {
      expect(tool.label).toBeTruthy()
      expect(tool.hint).toBeTruthy()
    }
  })

  it('os slugs são únicos — a rota resolve por eles', () => {
    const slugs = GM_TOOLS.map((tool) => tool.slug)
    expect(new Set(slugs).size).toBe(slugs.length)
  })

  it('a ferramenta padrão é uma das listadas', () => {
    expect(isToolSlug(DEFAULT_TOOL)).toBe(true)
  })
})

describe('toolLabel', () => {
  it('nomeia a ferramenta para o cabeçalho', () => {
    expect(toolLabel('catalogos')).toBe('Catálogos')
  })
})
