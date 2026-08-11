import { describe, expect, it } from 'vitest'
import { classLevelLine } from './class-line'

describe('classLevelLine', () => {
  it('joins multiclass entries with a slash', () => {
    expect(
      classLevelLine([
        { className: 'Guerreiro', level: 3 },
        { className: 'Arcanista', level: 1 },
      ]),
    ).toBe('Guerreiro 3 / Arcanista 1')
  })

  it('returns an empty string for no classes', () => {
    expect(classLevelLine([])).toBe('')
  })
})
