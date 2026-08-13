import { initials } from '@/shared/lib/initials'
import { describe, expect, it } from 'vitest'
import type { Character, RaceDefinition } from '@/shared/api/api'
import {characterFlavor, isCreateSlot, primaryClass, primaryRole, raceAbilityBlurbs, stepRosterIndex} from './select-helpers'

function character(overrides: Partial<Character> = {}): Character {
  return {
    id: 1,
    ownerId: 1,
    name: 'Tanque Placas',
    origin: 'Soldado',
    god: 'Khalmyr',
    godPower: '',
    tibar: 0,
    level: 10,
    hpMax: 137,
    hpCurrent: 137,
    mpMax: 30,
    mpCurrent: 30,
    strength: 18,
    dexterity: 12,
    constitution: 16,
    intelligence: 10,
    wisdom: 12,
    charisma: 8,
    size: 'Médio',
    displacement: 9,
    proficiencies: '[]',
    raceAbilityChoices: '[]',
    activeConditions: '[]',
    raceAttributeChoices: '{}',
    secondaryRaceChoices: '[]',
    originChoices: '[]',
    classPowers: '[]',
    classChoices: '{}',
    powerChoices: '{}',
    createdAt: '2026-01-01T00:00:00.000Z',
    updatedAt: '2026-01-01T00:00:00.000Z',
    races: [{ race: 'Anão' }],
    classes: [{ className: 'Guerreiro', level: 10 }],
    expertises: [],
    items: [],
    activeEffects: [],
    spells: [],
    ...overrides,
  }
}

describe('primaryRole', () => {
  it('é a classe principal + nível, em caixa alta', () => {
    expect(primaryRole(character())).toBe('GUERREIRO 10')
  })

  // Personagem recém-criado ainda não tem classe; a origem é o que sobra.
  it('cai na origem quando não há classe', () => {
    expect(primaryRole(character({ classes: [] }))).toBe('SOLDADO')
  })

  it('usa a PRIMEIRA classe num multiclasse', () => {
    const multi = character({
      classes: [
        { className: 'Bárbaro', level: 2 },
        { className: 'Arcanista', level: 1 },
      ],
    })
    expect(primaryRole(multi)).toBe('BÁRBARO 2')
  })
})

describe('primaryClass', () => {
  it('mantém a caixa original', () => {
    expect(primaryClass(character())).toBe('Guerreiro 10')
  })
})

describe('characterFlavor', () => {
  // Não existe campo de bio livre — a linha é composta dos campos estruturados.
  it('compõe raça, origem, devoção, tamanho e nível', () => {
    expect(characterFlavor(character())).toBe(
      'Anão • Soldado • devoto de Khalmyr • Médio • Nível 10',
    )
  })

  it('some com a devoção quando não há deus', () => {
    expect(characterFlavor(character({ god: null }))).toBe('Anão • Soldado • Médio • Nível 10')
  })
})

describe('raceAbilityBlurbs', () => {
  const raceDefs: RaceDefinition[] = [
    {
      id: 'anao',
      name: 'Anão',
      attributeBonuses: {},
      abilities: [
        { id: 'a1', name: 'Conhecimento das Rochas', description: 'Visão no escuro.' },
        { id: 'a2', name: 'Devagar e Sempre', description: 'Deslocamento 6m.' },
        { id: 'a3', name: 'Duro como Pedra', description: '+1 PV por nível.' },
      ],
    } as RaceDefinition,
  ]

  it('devolve as habilidades da raça do personagem', () => {
    const blurbs = raceAbilityBlurbs(raceDefs, character(), 8)
    expect(blurbs.map((b) => b.name)).toEqual([
      'Conhecimento das Rochas',
      'Devagar e Sempre',
      'Duro como Pedra',
    ])
  })

  it('respeita o limite', () => {
    expect(raceAbilityBlurbs(raceDefs, character(), 2)).toHaveLength(2)
  })

  // O catálogo chega por fetch: enquanto não chegou, a lista é vazia — não um
  // erro. É o que deixa o dossiê abrir antes do catálogo carregar.
  it('lista vazia quando o catálogo ainda não chegou', () => {
    expect(raceAbilityBlurbs([], character(), 8)).toEqual([])
  })

  it('lista vazia pra raça desconhecida', () => {
    expect(raceAbilityBlurbs(raceDefs, character({ races: [{ race: 'Golem' }] }), 8)).toEqual([])
  })

  it('sem raça, sem habilidades', () => {
    expect(raceAbilityBlurbs(raceDefs, character({ races: [] }), 8)).toEqual([])
  })
})

/** ALE-98: o "+" da criação virou uma POSIÇÃO do cursor, não só um link. */
describe('cursor do roster com o slot de criação', () => {
  const CINCO_HEROIS = 5

  it('avança de herói em herói dentro do elenco', () => {
    expect(stepRosterIndex(0, 1, CINCO_HEROIS)).toBe(1)
    expect(stepRosterIndex(3, -1, CINCO_HEROIS)).toBe(2)
  })

  it('do último herói, → cai no slot de criação em vez de parar', () => {
    expect(stepRosterIndex(4, 1, CINCO_HEROIS)).toBe(5)
    expect(isCreateSlot(5, CINCO_HEROIS)).toBe(true)
  })

  it('o slot de criação é o fim da linha — → de novo não passa dele', () => {
    expect(stepRosterIndex(5, 1, CINCO_HEROIS)).toBe(5)
  })

  it('← do slot de criação volta pro último herói', () => {
    const voltou = stepRosterIndex(5, -1, CINCO_HEROIS)

    expect(voltou).toBe(4)
    expect(isCreateSlot(voltou, CINCO_HEROIS)).toBe(false)
  })

  it('não anda pra antes do primeiro herói', () => {
    expect(stepRosterIndex(0, -1, CINCO_HEROIS)).toBe(0)
  })

  // O salto de 5 em 5 (bumper) não pode pular POR CIMA do slot e sumir.
  it('um salto grande para no slot de criação, não além', () => {
    expect(stepRosterIndex(2, 5, CINCO_HEROIS)).toBe(5)
  })

  // Elenco vazio: a EmptyStage já convida a criar, então o cursor nasce no slot.
  it('sem heróis, a única posição é o slot de criação', () => {
    expect(stepRosterIndex(0, 1, 0)).toBe(0)
    expect(isCreateSlot(0, 0)).toBe(true)
  })
})

describe('initials', () => {
  it('duas iniciais do nome', () => {
    expect(initials('Thal, o Errante')).toBe('TO')
  })

  it('nome de uma palavra rende uma letra', () => {
    expect(initials('Zumbi')).toBe('Z')
  })

  // A vaga vazia e o nome em branco caem no mesmo símbolo do card de criação.
  it('nome vazio vira "?"', () => {
    expect(initials('   ')).toBe('?')
  })
})
