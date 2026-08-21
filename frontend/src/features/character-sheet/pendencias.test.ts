import { beforeEach, describe, expect, it } from 'vitest'
import type { Character } from '@/shared/api/api'
import type { Raca, RaceDefinition } from '@/shared/api/catalog-types'
import { primeAbilities } from '@/shared/lib/abilities-cache'
import { primeRacas } from '@/shared/lib/racas-cache'
import { computePendencias } from './pendencias'

/**
 * As pendências que a ficha promete resolver (ALE-169).
 *
 * O passo de Resumo da forja diz, por escrito, "dá para criar assim e terminar
 * na ficha", e lista "Bônus de atributo de raça não colocado". A ficha então
 * mostrava UMA pendência — a variante de habilidade — e o +1 sumia: nem
 * aparecia na lista nem tinha onde ser colocado. O personagem ficava ilegal
 * pelo livro ("Sua raça modifica seus atributos", p18) sem caminho de conserto
 * que não fosse refazer a forja inteira.
 */
const HUMANO_DEFINICAO = {
  id: 'humano',
  name: 'Humano',
  attributeBonuses: {},
  abilities: [
    {
      id: 'versatil',
      name: 'Versátil',
      description: 'Duas perícias, ou uma e um poder.',
      variants: [
        { id: 'versatil-pericias', name: 'Duas perícias' },
        { id: 'versatil-poder', name: 'Uma perícia e um poder' },
      ],
    },
  ],
} as unknown as RaceDefinition

const HUMANO_REGRA = {
  id: 'humano',
  name: 'Humano',
  tier: 'comum',
  atributoMod: { kind: 'floating', count: 3, value: 1 },
} as unknown as Raca

const ANAO_REGRA = {
  id: 'anao',
  name: 'Anão',
  tier: 'comum',
  atributoMod: {
    kind: 'fixed',
    mods: { constitution: 2, wisdom: 1, dexterity: -1 },
  },
} as unknown as Raca

const ANAO_DEFINICAO = {
  id: 'anao',
  name: 'Anão',
  attributeBonuses: { constitution: 2 },
  abilities: [],
} as unknown as RaceDefinition

function personagem(over: Partial<Character>): Character {
  return {
    id: 1,
    name: 'Sonda',
    races: [{ race: 'humano', applied: true }],
    raceAbilityChoices: '{}',
    raceAttributeChoices: '{}',
    origin: '',
    originChoices: '[]',
    classes: [],
    classChoices: '{}',
    ...over,
  } as unknown as Character
}

describe('computePendencias — o bônus de atributo de raça', () => {
  beforeEach(() => {
    primeAbilities({
      races: [HUMANO_DEFINICAO, ANAO_DEFINICAO],
      origins: [],
      classPowers: [],
      generalPowers: [],
      deuses: [],
      grantedPowers: [],
    })
    primeRacas({ humano: HUMANO_REGRA, anao: ANAO_REGRA }, {})
  })

  it('cobra o +1 que a forja deixou sem colocar', () => {
    const pendencias = computePendencias(personagem({ raceAttributeChoices: '{}' }))

    expect(pendencias.map((p) => p.label)).toContainEqual(
      expect.stringContaining('atributo'),
    )
  })

  it('para de cobrar quando os três estão colocados', () => {
    const escolhido = personagem({
      raceAttributeChoices: JSON.stringify({
        floatingPicks: ['strength', 'dexterity', 'constitution'],
      }),
    })

    expect(computePendencias(escolhido).map((p) => p.label)).not.toContainEqual(
      expect.stringContaining('atributo'),
    )
  })

  /** Uma raça de modificador fixo nunca deve uma escolha de atributo. */
  it('não cobra nada de uma raça sem escolha', () => {
    const anao = personagem({
      races: [{ race: 'anao', applied: true }] as unknown as Character['races'],
    })

    expect(computePendencias(anao).map((p) => p.label)).not.toContainEqual(
      expect.stringContaining('atributo'),
    )
  })

  it('aponta para o cartão da raça, para o clique saber onde abrir', () => {
    const pendencia = computePendencias(personagem({})).find((p) =>
      p.label.includes('atributo'),
    )

    expect(pendencia?.cardId).toBe('raca:humano')
    expect(pendencia?.source).toBe('raca')
  })
})
