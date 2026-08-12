import { describe, expect, it } from 'vitest'
import type { SheetSearchEntry } from './sheet-search-index'
import { groupSheetEntries, rankSheetEntries } from './sheet-search-groups'

const entry = (name: string, source: string, tab = 'expertises'): SheetSearchEntry => ({
  name,
  source,
  tab,
  detail: `detalhe de ${name}`,
})

describe('rankSheetEntries', () => {
  it('sem busca, devolve tudo na ordem do índice', () => {
    const index = [entry('Furtividade', 'Perícia'), entry('Adaga', 'Item')]

    expect(rankSheetEntries(index, '')).toEqual(index)
  })

  it('some com o que não casa', () => {
    const index = [entry('Furtividade', 'Perícia'), entry('Adaga', 'Item')]

    expect(rankSheetEntries(index, 'adaga').map((e) => e.name)).toEqual(['Adaga'])
  })

  // A busca é sem acento: quem digita "furia" no meio da mesa acha "Fúria".
  it('acha sem acento', () => {
    const index = [entry('Fúria', 'Classe')]

    expect(rankSheetEntries(index, 'furia')).toHaveLength(1)
  })

  // O prefixo vale mais que o meio da palavra, senão a resposta óbvia afunda.
  it('prefixo vem antes de casamento no meio', () => {
    const index = [entry('Grande Fúria', 'Classe'), entry('Fúria', 'Classe')]

    expect(rankSheetEntries(index, 'furia').map((e) => e.name)).toEqual([
      'Fúria',
      'Grande Fúria',
    ])
  })

  it('busca também casa pela fonte', () => {
    const index = [entry('Adaga', 'Item'), entry('Furtividade', 'Perícia')]

    expect(rankSheetEntries(index, 'item').map((e) => e.name)).toEqual(['Adaga'])
  })
})

describe('groupSheetEntries', () => {
  it('agrupa na ordem de leitura da ficha', () => {
    const index = [
      entry('Adaga', 'Item'),
      entry('Bola de Fogo', 'Magia'),
      entry('Furtividade', 'Perícia'),
    ]

    expect(groupSheetEntries(index).map((g) => g.source)).toEqual([
      'Perícia',
      'Item',
      'Magia',
    ])
  })

  // Raça, Origem, Classe e Poder geral são a MESMA pergunta pro jogador
  // ("que poder é esse?"), então caem num balde só.
  it('as fontes de poder viram um grupo só', () => {
    const index = [entry('Totem', 'Raça'), entry('Ataque Especial', 'Classe')]

    const groups = groupSheetEntries(index)

    expect(groups).toHaveLength(1)
    expect(groups[0].source).toBe('Poderes & habilidades')
    expect(groups[0].entries).toHaveLength(2)
  })

  it('índice vazio não gera grupo', () => {
    expect(groupSheetEntries([])).toEqual([])
  })
})

describe('groupSheetEntries — com busca', () => {
  // Sem isso o cursor da paleta pousa no primeiro grupo da ordem fixa, e o
  // Enter escolhe um casamento fraco por subsequência em vez do exato.
  it('o grupo da melhor resposta vem primeiro', () => {
    const index = [entry('Adestramento', 'Perícia'), entry('Adaga Rúnica', 'Item', 'inventory')]

    const ranked = rankSheetEntries(index, 'adaga')
    const groups = groupSheetEntries(ranked, { ranked: true })

    expect(groups[0].source).toBe('Item')
    expect(groups[0].entries[0].name).toBe('Adaga Rúnica')
  })

  it('sem busca, mantém a ordem de leitura da ficha', () => {
    const index = [entry('Adaga', 'Item'), entry('Furtividade', 'Perícia')]

    expect(groupSheetEntries(index).map((g) => g.source)).toEqual(['Perícia', 'Item'])
  })
})
