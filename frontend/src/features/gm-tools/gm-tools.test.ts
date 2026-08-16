import { describe, expect, it } from 'vitest'
import { GM_TOOLS, isToolSlug } from './gm-tools'

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
  // Slug repetido deixa uma ferramenta INALCANÇÁVEL — a rota resolve pela
  // primeira e a segunda some sem erro nenhum. É o único caso deste arquivo que
  // o typechecker não pega (rótulo preenchido e "o padrão é uma das listadas"
  // eram garantidos pelo tipo e pela própria construção).
  it('os slugs são únicos — a rota resolve por eles', () => {
    const slugs = GM_TOOLS.map((tool) => tool.slug)
    expect(new Set(slugs).size).toBe(slugs.length)
  })
})
