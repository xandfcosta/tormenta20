import { describe, expect, it } from 'vitest'
import { fuzzyMatches, matchesQuery } from './fuzzy-filter'

describe('fuzzyMatches', () => {
  it('busca vazia passa tudo', () => {
    expect(fuzzyMatches('Necromante', '')).toBe(true)
    expect(fuzzyMatches('Necromante', '   ')).toBe(true)
  })

  it('casa por prefixo e por trecho', () => {
    expect(fuzzyMatches('Necromante Nv12 Magias', 'necro')).toBe(true)
    expect(fuzzyMatches('Necromante Nv12 Magias', 'magias')).toBe(true)
  })

  // O motivo de usar match-sorter em vez de includes(): os catálogos são em
  // português e ninguém digita acento na busca.
  it('ignora acento — "anao" acha "Anão"', () => {
    expect(fuzzyMatches('Anão', 'anao')).toBe(true)
    expect(fuzzyMatches('Curandeira Divina', 'divina')).toBe(true)
  })

  it('não casa o que não tem nada a ver', () => {
    expect(fuzzyMatches('Necromante', 'guerreiro')).toBe(false)
  })
})

describe('matchesQuery', () => {
  const fields = ['Tanque Placas Nv10', 'Guerreiro', 'Soldado', 'Anão']

  it('passa se QUALQUER campo casar', () => {
    expect(matchesQuery(fields, 'guerreiro')).toBe(true)
    expect(matchesQuery(fields, 'soldado')).toBe(true)
    expect(matchesQuery(fields, 'anao')).toBe(true)
  })

  it('falha quando nenhum campo casa', () => {
    expect(matchesQuery(fields, 'necromante')).toBe(false)
  })

  it('busca vazia não filtra nada', () => {
    expect(matchesQuery(fields, '')).toBe(true)
  })
})
