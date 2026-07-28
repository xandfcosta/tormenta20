import { describe, expect, it } from 'vitest'
import { EXPERTISE_NAMES } from '../expertises'
import {
  RACE_FREE_EXPERTISES,
  raceFreeExpertiseCount,
} from '../race-expertises'

describe('raceFreeExpertiseCount', () => {
  it('grants Humano 2 free perícias (Versátil)', () => {
    expect(raceFreeExpertiseCount(['Humano'])).toBe(2)
  })

  it('grants Kliren 1 free perícia (Híbrido)', () => {
    expect(raceFreeExpertiseCount(['Kliren'])).toBe(1)
  })

  it('sums across multiple races and ignores non-granting ones', () => {
    expect(raceFreeExpertiseCount(['Humano', 'Anão', 'Kliren'])).toBe(3)
  })

  it('returns 0 for a race with no free-perícia grant', () => {
    expect(raceFreeExpertiseCount(['Anão'])).toBe(0)
    expect(raceFreeExpertiseCount([])).toBe(0)
  })
})

describe('RACE_FREE_EXPERTISES table', () => {
  it('uses positive counts within the perícia list size', () => {
    for (const [race, count] of Object.entries(RACE_FREE_EXPERTISES)) {
      expect(count, race).toBeGreaterThan(0)
      expect(count, race).toBeLessThanOrEqual(EXPERTISE_NAMES.length)
    }
  })
})
