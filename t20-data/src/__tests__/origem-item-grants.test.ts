import { describe, expect, it } from 'vitest'
import {
  origemItemGrantsByName,
  parseOrigemItem,
} from '../origem-item-grants'

describe('parseOrigemItem — classifica entradas de itensIniciais', () => {
  it('armas por proficiência viram picker', () => {
    expect(parseOrigemItem('Arma simples')).toEqual({
      kind: 'weapon',
      categories: ['weapon-simple'],
      label: 'Arma simples',
    })
    expect(parseOrigemItem('Arma marcial ou exótica')).toMatchObject({
      kind: 'weapon',
      categories: ['weapon-martial', 'weapon-exotic'],
    })
  })

  it('"até T$ N" vira anyItem com teto', () => {
    expect(parseOrigemItem('Um item estrangeiro (até T$ 100)')).toEqual({
      kind: 'anyItem',
      maxPrice: 100,
      label: 'Um item estrangeiro (até T$ 100)',
    })
    expect(parseOrigemItem('Um item fabricado (até T$ 50)')).toMatchObject({
      maxPrice: 50,
    })
  })

  it('" OU " maiúsculo vira oneOf; "ou" minúsculo em parênteses não', () => {
    expect(parseOrigemItem('Estojo de disfarces OU gazua')).toEqual({
      kind: 'oneOf',
      options: ['Estojo de disfarces', 'gazua'],
      label: 'Estojo de disfarces OU gazua',
    })
    expect(
      parseOrigemItem('Ferramenta pesada (stats de maça ou lança)'),
    ).toEqual({ kind: 'fixed', name: 'Ferramenta pesada (stats de maça ou lança)' })
  })

  it('"(escolha)" enumera opções', () => {
    expect(
      parseOrigemItem('Cão de caça, cavalo, pônei ou trobo (escolha)'),
    ).toEqual({
      kind: 'oneOf',
      options: ['Cão de caça', 'Cavalo', 'Pônei', 'Trobo'],
      label: 'Cão de caça, cavalo, pônei ou trobo (escolha)',
    })
  })

  it('"T$ 2d6" vira bônus de dinheiro, não item', () => {
    expect(parseOrigemItem('T$ 2d6 (último salário)')).toEqual({
      kind: 'money',
      dice: '2d6',
      label: 'T$ 2d6 (último salário)',
    })
  })

  it('demais entradas ficam fixas', () => {
    expect(parseOrigemItem('Uniforme militar')).toEqual({
      kind: 'fixed',
      name: 'Uniforme militar',
    })
  })
})

describe('origemItemGrantsByName', () => {
  it('Refugiado: item estrangeiro é escolha com teto T$ 100', () => {
    const grants = origemItemGrantsByName('Refugiado')
    expect(grants.some((g) => g.kind === 'anyItem' && g.maxPrice === 100)).toBe(
      true,
    )
  })

  it('Soldado: arma marcial é picker', () => {
    const grants = origemItemGrantsByName('Soldado')
    expect(grants.some((g) => g.kind === 'weapon')).toBe(true)
  })

  it('origem desconhecida → vazio', () => {
    expect(origemItemGrantsByName('Nada')).toEqual([])
  })
})
