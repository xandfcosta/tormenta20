import type { Monster } from '@tormenta20/t20-data'
import { describe, expect, it } from 'vitest'
import {
  INITIATIVE_MAX_ENTRIES,
  encounterDifficulty,
  encounterInitiativeLabels,
  encounterNd,
  enrichEncounter,
} from './encounter'

const monster = (id: string, name: string, nd: number) =>
  ({ id, name, nd }) as Monster

const BESTIARY = [
  monster('goblin', 'Goblin', 0.25),
  monster('ogro', 'Ogro', 2),
  monster('dragao', 'Dragão', 15),
]

describe('enrichEncounter', () => {
  it('resolve as entradas e calcula o ND de cada grupo', () => {
    const [group] = enrichEncounter([{ monsterId: 'goblin', quantity: 4 }], BESTIARY)

    expect(group.monster.name).toBe('Goblin')
    expect(group.groupNd).toBe(1) // quatro de ND 1/4
  })

  it('descarta entrada cujo monstro sumiu do bestiário', () => {
    // Um id órfão renderizaria uma linha vazia com quantidade viva.
    const groups = enrichEncounter(
      [{ monsterId: 'nao-existe', quantity: 3 }, { monsterId: 'ogro', quantity: 1 }],
      BESTIARY,
    )

    expect(groups.map((g) => g.monster.id)).toEqual(['ogro'])
  })

  it('bestiário ainda não carregado devolve nada, não explode', () => {
    expect(enrichEncounter([{ monsterId: 'ogro', quantity: 1 }], [])).toEqual([])
  })
})

describe('encounterNd', () => {
  it('soma o ND dos grupos', () => {
    const groups = enrichEncounter(
      [{ monsterId: 'goblin', quantity: 4 }, { monsterId: 'ogro', quantity: 1 }],
      BESTIARY,
    )

    expect(encounterNd(groups)).toBe(3) // 1 + 2
  })

  it('encontro vazio é ND 0', () => {
    expect(encounterNd([])).toBe(0)
  })
})

describe('encounterDifficulty', () => {
  it('ND igual ao nível do grupo é um combate justo (p281)', () => {
    expect(encounterDifficulty(0).label).toBe('Médio')
  })

  it('escala para os dois lados', () => {
    expect(encounterDifficulty(-4).label).toBe('Trivial')
    expect(encounterDifficulty(-1).label).toBe('Fácil')
    expect(encounterDifficulty(2).label).toBe('Difícil')
    expect(encounterDifficulty(5).label).toBe('Mortal')
  })

  // Regressão ALE-25: gap fracionário pequeno escapava dos testes `<= -1` e
  // `=== 0` e caía direto em "Difícil".
  it('arredonda o gap fracionário antes de bandar', () => {
    expect(encounterDifficulty(-0.75).label).toBe('Fácil')
    expect(encounterDifficulty(0.3).label).toBe('Médio')
  })
})

describe('encounterInitiativeLabels', () => {
  it('numera as cópias — o mestre acompanha o PV de cada uma', () => {
    const groups = enrichEncounter([{ monsterId: 'goblin', quantity: 3 }], BESTIARY)

    expect(encounterInitiativeLabels(groups).labels).toEqual([
      'Goblin 1',
      'Goblin 2',
      'Goblin 3',
    ])
  })

  it('monstro único não ganha número', () => {
    const groups = enrichEncounter([{ monsterId: 'dragao', quantity: 1 }], BESTIARY)

    expect(encounterInitiativeLabels(groups).labels).toEqual(['Dragão'])
  })

  it('corta no teto do servidor e diz quantos ficaram de fora', () => {
    const groups = enrichEncounter([{ monsterId: 'goblin', quantity: 60 }], BESTIARY)
    const { labels, dropped } = encounterInitiativeLabels(groups)

    expect(labels).toHaveLength(INITIATIVE_MAX_ENTRIES)
    expect(dropped).toBe(10)
  })

  it('conta o que já está no rastreador antes de cortar', () => {
    const groups = enrichEncounter([{ monsterId: 'goblin', quantity: 5 }], BESTIARY)
    const { labels, dropped } = encounterInitiativeLabels(groups, 48)

    expect(labels).toHaveLength(2)
    expect(dropped).toBe(3)
  })
})
