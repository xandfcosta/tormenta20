import { describe, expect, it } from 'vitest'
import type { Character } from '@/shared/api/api'
import { grantedSpells } from './granted-spells'

/**
 * grantedSpells: powers whose sub-choice teaches a spell (Bárbaro Totem
 * Espiritual, PDF p42). The pick lives in the `powerChoices` JSON blob keyed
 * by power id ({ 'class.barbaro.totem-espiritual': ['lobo'] }); the option's
 * `note` names the granted spell.
 */

const TOTEM_ID = 'class.barbaro.totem-espiritual'

// Named fake — minimal Character with only the columns grantedSpells reads
// (classPowers + powerChoices); everything else stays neutral.
function fakeBarbaro(over: Partial<Character> = {}): Character {
  return {
    id: 1,
    name: 'Krug',
    level: 4,
    hpMax: 40,
    hpCurrent: 40,
    mpMax: 12,
    mpCurrent: 12,
    strength: 4,
    dexterity: 1,
    constitution: 2,
    intelligence: 0,
    wisdom: 1,
    charisma: 0,
    size: 'M',
    displacement: 9,
    proficiencies: '[]',
    raceAbilityChoices: '[]',
    raceAttributeChoices: '{}',
    secondaryRaceChoices: '[]',
    originChoices: '[]',
    classPowers: JSON.stringify([TOTEM_ID]),
    classChoices: '{}',
    powerChoices: JSON.stringify({ [TOTEM_ID]: ['lobo'] }),
    origin: 'Selvagem',
    god: null,
    expertises: [],
    races: [],
    classes: [{ className: 'Bárbaro', level: 4 }],
    items: [],
    activeEffects: [],
    spells: [],
    ...over,
  } as Character
}

describe('grantedSpells', () => {
  it('resolve o totem escolhido para a magia concedida (Sab)', () => {
    const granted = grantedSpells(fakeBarbaro())
    expect(granted).toHaveLength(1)
    expect(granted[0].spell.id).toBe('concentracao-de-combate')
    expect(granted[0].sourcePower).toBe('Totem Espiritual')
    expect(granted[0].keyAttribute).toBe('wisdom')
  })

  it('cada animal totêmico resolve para uma magia distinta do catálogo', () => {
    const animals = [
      'coruja',
      'corvo',
      'falcao',
      'grifo',
      'lobo',
      'raposa',
      'tartaruga',
      'urso',
    ]
    const ids = animals.map((animal) => {
      const c = fakeBarbaro({
        powerChoices: JSON.stringify({ [TOTEM_ID]: [animal] }),
      })
      return grantedSpells(c)[0]?.spell.id
    })
    expect(ids.every(Boolean)).toBe(true)
    expect(new Set(ids).size).toBe(animals.length)
  })

  it('sem pick armazenado → lista vazia', () => {
    expect(grantedSpells(fakeBarbaro({ powerChoices: '{}' }))).toEqual([])
  })

  it('poder não possuído → lista vazia mesmo com pick órfão', () => {
    const c = fakeBarbaro({ classPowers: '[]' })
    expect(grantedSpells(c)).toEqual([])
  })

  it('pick desconhecido (animal inexistente) → vazio, sem lançar', () => {
    const c = fakeBarbaro({
      powerChoices: JSON.stringify({ [TOTEM_ID]: ['dragao'] }),
    })
    expect(grantedSpells(c)).toEqual([])
  })

  it('blobs malformados degradam para vazio, sem lançar', () => {
    expect(grantedSpells(fakeBarbaro({ powerChoices: 'not json' }))).toEqual([])
    expect(grantedSpells(fakeBarbaro({ powerChoices: '[1,2]' }))).toEqual([])
    expect(
      grantedSpells(fakeBarbaro({ powerChoices: JSON.stringify({ [TOTEM_ID]: 'lobo' }) })),
    ).toEqual([])
  })

  it('poder com choice sem grantsSpellAttribute não concede magia', () => {
    // Especialista em Escola tem choice kind 'school' mas não ensina magia.
    const schoolId = 'class.arcanista.especialista-em-escola'
    const c = fakeBarbaro({
      classPowers: JSON.stringify([schoolId]),
      powerChoices: JSON.stringify({ [schoolId]: ['evocacao'] }),
    })
    expect(grantedSpells(c)).toEqual([])
  })
})
