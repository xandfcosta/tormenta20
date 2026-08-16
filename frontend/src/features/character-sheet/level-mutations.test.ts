import { QueryClient } from '@tanstack/solid-query'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { characterQueryOptions } from '@/entities/character/queries'
import type { Character } from '@/shared/api/api'
import {
  bumpClassLevel,
  eligibleClasses,
  levelActions,
  settleClassLevel,
} from './level-mutations'

const CHARACTER_ID = 1

function character(overrides: Partial<Character> = {}): Character {
  return {
    id: CHARACTER_ID,
    name: 'Tanque',
    origin: 'Soldado',
    god: null,
    godPower: '',
    level: 3,
    hpMax: 40,
    hpCurrent: 30,
    mpMax: 10,
    mpCurrent: 10,
    strength: 2,
    dexterity: 1,
    constitution: 2,
    intelligence: 0,
    wisdom: 0,
    charisma: 0,
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
    races: [{ race: 'Humano' }],
    classes: [{ className: 'Guerreiro', level: 3 }],
    expertises: [],
    items: [],
    activeEffects: [],
    spells: [],
    createdAt: '2026-01-01T00:00:00.000Z',
    updatedAt: '2026-01-01T00:00:00.000Z',
    tibar: 0,
    ownerId: 1,
    ...overrides,
  } as Character
}

const multiclass = () =>
  character({
    level: 4,
    classes: [
      { className: 'Guerreiro', level: 3 },
      { className: 'Bardo', level: 1 },
    ],
  })

afterEach(() => vi.restoreAllMocks())

describe('eligibleClasses', () => {
  it('subir vale para qualquer classe abaixo de 20', () => {
    expect(eligibleClasses(multiclass(), 'up').map((c) => c.className)).toEqual([
      'Guerreiro',
      'Bardo',
    ])
  })

  // Descer uma classe de nível 1 a apagaria — isso se edita em outro lugar.
  it('descer ignora a classe que está no nível 1', () => {
    expect(eligibleClasses(multiclass(), 'down').map((c) => c.className)).toEqual(['Guerreiro'])
  })

  it('no teto de 20 não sobra classe para subir', () => {
    const lenda = character({ level: 20, classes: [{ className: 'Guerreiro', level: 20 }] })
    expect(eligibleClasses(lenda, 'up')).toEqual([])
  })
})

describe('bumpClassLevel', () => {
  it('soma na classe pedida e refaz o total', () => {
    const after = bumpClassLevel(multiclass(), 'Bardo', 1)
    expect(after.classes).toEqual([
      { className: 'Guerreiro', level: 3 },
      { className: 'Bardo', level: 2 },
    ])
    expect(after.level).toBe(5)
  })

  /**
   * PV/PM máximos são DERIVADOS do nível de classe: se a barra não sobe junto,
   * o jogador vê o nível novo com o pool velho até um refetch.
   */
  it('os pools acompanham o degrau, não esperam o servidor', () => {
    const after = bumpClassLevel(character(), 'Guerreiro', 1)
    expect(after.hpMax).toBeGreaterThan(40)
    expect(after.hpCurrent).toBeGreaterThan(30)
  })

  // Guerreiro p65: "começa com 20 pontos de vida + Constituição e ganha 5 PV +
  // Constituição por nível". CON 2, nível 3 → 22 + 7 + 7 = 36. O esperado é o
  // LIVRO, não `bumpClassLevel(..., 0)` — comparar a descida com outra chamada
  // da mesma função passaria verde com as duas erradas juntas.
  it('descer devolve o que o nível tinha dado', () => {
    const up = bumpClassLevel(character(), 'Guerreiro', 1)
    const down = bumpClassLevel(up, 'Guerreiro', -1)
    expect(down.hpMax).toBe(36)
    expect(down.level).toBe(3)
  })
})

describe('settleClassLevel', () => {
  it('a resposta do servidor manda no nível e nos pools', () => {
    const after = settleClassLevel(character(), {
      level: 4,
      classes: [{ className: 'Guerreiro', level: 4 }],
      vitals: { hpMax: 50, hpCurrent: 40, mpMax: 12, mpCurrent: 12 },
    })
    expect(after).toMatchObject({ level: 4, hpMax: 50, hpCurrent: 40, mpMax: 12 })
  })
})

function seeded(char: Character = character()) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  client.setQueryData(characterQueryOptions(CHARACTER_ID).queryKey, char)
  return client
}

const cached = (client: QueryClient) =>
  client.getQueryData<Character>(characterQueryOptions(CHARACTER_ID).queryKey)

describe('levelActions', () => {
  it('manda o nível ABSOLUTO da classe, não o degrau', async () => {
    const client = seeded()
    const api = await import('@/shared/api/api')
    const call = vi.spyOn(api.api.characters, 'updateClassLevel').mockResolvedValue({
      level: 4,
      classes: [{ className: 'Guerreiro', level: 4 }],
      vitals: { hpMax: 50, hpCurrent: 40, mpMax: 12, mpCurrent: 12 },
    })

    await levelActions(client, CHARACTER_ID).bump('Guerreiro', 1)

    expect(call).toHaveBeenCalledWith(CHARACTER_ID, { className: 'Guerreiro', level: 4 })
    expect(cached(client)?.hpMax).toBe(50)
  })

  it('falha devolve o personagem inteiro como estava', async () => {
    const client = seeded()
    const api = await import('@/shared/api/api')
    vi.spyOn(api.api.characters, 'updateClassLevel').mockRejectedValue(new Error('500'))

    await expect(levelActions(client, CHARACTER_ID).bump('Guerreiro', 1)).rejects.toThrow('500')

    expect(cached(client)).toMatchObject({ level: 3, hpMax: 40, hpCurrent: 30 })
  })

  it('classe que não é do personagem estoura com o valor ofensivo', async () => {
    const client = seeded()
    await expect(levelActions(client, CHARACTER_ID).bump('Arcanista', 1)).rejects.toThrow(
      /Arcanista/,
    )
  })
})
