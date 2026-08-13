import type { Monster } from '@/shared/api/catalog-types'
import { describe, expect, it } from 'vitest'
import {
  INITIATIVE_MAX_ENTRIES,
  encounterDifficulty,
  encounterInitiativeEntries,
  encounterNd,
  enrichEncounter,
} from './encounter'

const monster = (id: string, name: string, nd: number, hp: number) =>
  ({ id, name, nd, hp }) as Monster

const BESTIARY = [
  monster('goblin', 'Goblin', 0.25, 4),
  monster('ogro', 'Ogro', 2, 30),
  monster('dragao', 'Dragão', 15, 800),
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

describe('encounterInitiativeEntries', () => {
  it('numera as cópias — o mestre acompanha cada uma separado', () => {
    const groups = enrichEncounter([{ monsterId: 'goblin', quantity: 3 }], BESTIARY)

    expect(encounterInitiativeEntries(groups).entries.map((e) => e.label)).toEqual([
      'Goblin 1',
      'Goblin 2',
      'Goblin 3',
    ])
  })

  // Sem PV a linha entra no rastreador sem barra e o mestre não tem o que
  // gastar — o "adicionar monstro" avulso já semeava, o encontro não.
  it('cada cópia leva o PV do bloco de estatísticas', () => {
    const groups = enrichEncounter([{ monsterId: 'ogro', quantity: 2 }], BESTIARY)

    for (const entry of encounterInitiativeEntries(groups).entries) {
      expect(entry.hpCurrent).toBe(30)
      expect(entry.hpMax).toBe(30)
    }
  })

  it('monstro único não ganha número', () => {
    const groups = enrichEncounter([{ monsterId: 'dragao', quantity: 1 }], BESTIARY)

    expect(encounterInitiativeEntries(groups).entries.map((e) => e.label)).toEqual(['Dragão'])
  })

  it('corta no teto do servidor e diz quantos ficaram de fora', () => {
    const groups = enrichEncounter([{ monsterId: 'goblin', quantity: 60 }], BESTIARY)
    const { entries, dropped } = encounterInitiativeEntries(groups)

    expect(entries).toHaveLength(INITIATIVE_MAX_ENTRIES)
    expect(dropped).toBe(10)
  })

  it('conta o que já está no rastreador antes de cortar', () => {
    const groups = enrichEncounter([{ monsterId: 'goblin', quantity: 5 }], BESTIARY)
    const { entries, dropped } = encounterInitiativeEntries(groups, 48)

    expect(entries).toHaveLength(2)
    expect(dropped).toBe(3)
  })
})
