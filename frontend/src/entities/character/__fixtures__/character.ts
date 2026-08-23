import type { Character } from '@/shared/api/api'

/**
 * A complete, valid Character for tests.
 *
 * Complete on purpose: the derived engine walks races, classes and items, so a
 * partial literal blows up somewhere far from the test's subject
 * (`activeItemsFor` reading a race that isn't there). Override only the field
 * under test.
 *
 * @example makeCharacter({ activeConditions: '["caido"]' })
 */
export function makeCharacter(overrides: Partial<Character> = {}): Character {
  return {
    id: 1,
    ownerId: 1,
    name: 'Tanque Placas',
    origin: 'Soldado',
    god: null,
    godPower: '',
    tibar: 0,
    level: 3,
    hpMax: 20,
    hpCurrent: 20,
    mpMax: 10,
    mpCurrent: 10,
    strength: 1,
    dexterity: 2,
    constitution: 2,
    intelligence: 3,
    wisdom: 1,
    charisma: 1,
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
    races: [{ race: 'Humano' }],
    classes: [{ className: 'Bardo', level: 3 }],
    expertises: [],
    items: [],
    activeEffects: [],
    spells: [],
    // O estado de JOGO (ALE-222). Vazio por padrão: um personagem de teste
    // nasce sem nada ligado, e quem testa o situacional o passa por override.
    conditionals: [],
    powerUses: [],
    stances: [],
    ...overrides,
  }
}
