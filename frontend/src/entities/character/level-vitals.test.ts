import { describe, expect, it } from 'vitest'
import type { Character } from '@/shared/api/api'
import { characterEffects } from './derived'
import { optimisticLevelVitals } from './level-vitals'

function character(over: Partial<Character> = {}): Character {
  return {
    id: 1,
    name: 'X',
    level: 8,
    hpMax: 87,
    hpCurrent: 68,
    mpMax: 24,
    mpCurrent: 24,
    strength: 4,
    dexterity: 3,
    constitution: 4,
    intelligence: 2,
    wisdom: 2,
    charisma: 1,
    size: 'M',
    displacement: 9,
    proficiencies: '[]',
    raceAbilityChoices: '[]',
    raceAttributeChoices: '{}',
    secondaryRaceChoices: '[]',
    originChoices: '[]',
    classPowers: '[]',
    classChoices: '{}',
    origin: 'Soldado',
    god: null,
    expertises: [],
    races: [],
    classes: [{ className: 'Guerreiro', level: 8 }],
    items: [],
    activeEffects: [],
    ...over,
  } as Character
}

/**
 * Optimistic mirror of the server's level vitals sync (`levelVitalsPatch`):
 * pools recomputed from the SHARED engine helpers, currents shifted by the
 * max delta (level up heals the gained PV/PM, level down takes them back).
 * The server response still reconciles in onSuccess — this only removes the
 * visual lag on the VIDA/MANA bars.
 */
describe('optimisticLevelVitals', () => {
  it('level up shifts max and current together (68/87 → 77/96, CON 4)', () => {
    const c = character()
    const next = [{ className: 'Guerreiro', level: 9 }]
    expect(optimisticLevelVitals(c, characterEffects(c), next)).toEqual({
      hpMax: 96,
      hpCurrent: 77,
      mpMax: 27,
      mpCurrent: 27,
    })
  })

  it('level down walks the delta back, floored at 0', () => {
    const c = character({ hpCurrent: 4, mpCurrent: 1 })
    const next = [{ className: 'Guerreiro', level: 7 }]
    const v = optimisticLevelVitals(c, characterEffects(c), next)
    expect(v.hpMax).toBe(78)
    expect(v.hpCurrent).toBe(0)
    expect(v.mpMax).toBe(21)
    expect(v.mpCurrent).toBe(0)
  })

  it('multiclasse: novo 1º nível de outra classe dá PV subsequente (p35)', () => {
    // Guerreiro 8 → +Arcanista 1: PV += max(1, 2+4) = 6; PM += 6.
    const c = character()
    const next = [
      { className: 'Guerreiro', level: 8 },
      { className: 'Arcanista', level: 1 },
    ]
    expect(optimisticLevelVitals(c, characterEffects(c), next)).toEqual({
      hpMax: 93,
      hpCurrent: 74,
      mpMax: 30,
      mpCurrent: 30,
    })
  })

  it('grants por nível acompanham (caminho mago: +Int fixo, PM só do nível)', () => {
    // Arcanista mago Int 2: L1 pm = 6+2 = 8; L2 = 12+2 = 14 → delta +6.
    const c = character({
      level: 1,
      hpMax: 12,
      hpCurrent: 12,
      mpMax: 8,
      mpCurrent: 8,
      classes: [{ className: 'Arcanista', level: 1 }],
      classChoices: JSON.stringify({ Arcanista: { caminho: 'mago' } }),
    })
    const next = [{ className: 'Arcanista', level: 2 }]
    const v = optimisticLevelVitals(c, characterEffects(c), next)
    expect(v.mpMax).toBe(14)
    expect(v.mpCurrent).toBe(14)
  })
})
