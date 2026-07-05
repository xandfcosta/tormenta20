import { describe, expect, it } from 'vitest'
import {
  DUNGEON_IDEA_TABLE,
  DUNGEON_SIZES,
  DUNGEON_SIZE_TABLE,
  ROOMS_PER_THREAT,
  classifyDungeonSize,
  dungeonIdeaFromRoll,
  dungeonSizeRow,
  plannedThreats,
} from '../dungeon-design'

/**
 * PDF Cap 6 p263 — "Ambientes de aventura → masmorras".
 * Two artifacts:
 *   - dungeon size table (rooms + pacing + objectives)
 *   - Tabela 6-2: d20 → 20 ideas
 */

describe('DUNGEON_SIZES (p263)', () => {
  it('lists 3 sizes in ascending order', () => {
    expect(DUNGEON_SIZES).toEqual(['pequena', 'media', 'grande'])
  })

  it('table matches enum 1:1', () => {
    expect(DUNGEON_SIZE_TABLE.map((r) => r.size)).toEqual([...DUNGEON_SIZES])
  })
})

describe('dungeon size rooms (p263)', () => {
  it('pequena: 3-6 salas', () => {
    const r = dungeonSizeRow('pequena')
    expect(r.minRooms).toBe(3)
    expect(r.maxRooms).toBe(6)
  })

  it('media: 7-20 salas', () => {
    const r = dungeonSizeRow('media')
    expect(r.minRooms).toBe(7)
    expect(r.maxRooms).toBe(20)
  })

  it('grande: 21-50 salas', () => {
    const r = dungeonSizeRow('grande')
    expect(r.minRooms).toBe(21)
    expect(r.maxRooms).toBe(50)
  })

  it('room ranges are contiguous (no gaps between tiers)', () => {
    for (let i = 1; i < DUNGEON_SIZE_TABLE.length; i++) {
      const prev = DUNGEON_SIZE_TABLE[i - 1]!
      const curr = DUNGEON_SIZE_TABLE[i]!
      expect(curr.minRooms).toBe(prev.maxRooms + 1)
    }
  })
})

describe('dungeon pacing + objectives (p263)', () => {
  it('pequena = parte-de-sessao, sem secundário, 1 opcional', () => {
    const r = dungeonSizeRow('pequena')
    expect(r.pacing).toBe('parte-de-sessao')
    expect(r.maxSecondaryObjectives).toBe(0)
    expect(r.optionalObjectives).toBe(1)
  })

  it('media = sessao-inteira, até 1 secundário, 2 opcionais', () => {
    const r = dungeonSizeRow('media')
    expect(r.pacing).toBe('sessao-inteira')
    expect(r.maxSecondaryObjectives).toBe(1)
    expect(r.optionalObjectives).toBe(2)
  })

  it('grande = aventura-inteira, até 3 secundários, 3 opcionais', () => {
    const r = dungeonSizeRow('grande')
    expect(r.pacing).toBe('aventura-inteira')
    expect(r.maxSecondaryObjectives).toBe(3)
    expect(r.optionalObjectives).toBe(3)
  })
})

describe('plannedThreats (1 ameaça por 3 salas, p263)', () => {
  it('constant is 3', () => {
    expect(ROOMS_PER_THREAT).toBe(3)
  })

  it('3 salas → 1 ameaça (limite inferior de pequena)', () => {
    expect(plannedThreats(3)).toBe(1)
  })

  it('6 salas → 2 ameaças (limite superior de pequena)', () => {
    expect(plannedThreats(6)).toBe(2)
  })

  it('7 salas → 3 ameaças (limite inferior de média — round up)', () => {
    expect(plannedThreats(7)).toBe(3)
  })

  it('20 salas → 7 ameaças (limite superior de média)', () => {
    expect(plannedThreats(20)).toBe(7)
  })

  it('50 salas → 17 ameaças (limite superior de grande)', () => {
    expect(plannedThreats(50)).toBe(17)
  })

  it('rejects non-positive', () => {
    expect(() => plannedThreats(0)).toThrow()
    expect(() => plannedThreats(-1)).toThrow()
  })
})

describe('classifyDungeonSize (p263)', () => {
  it('boundaries snap to their tier', () => {
    expect(classifyDungeonSize(3)).toBe('pequena')
    expect(classifyDungeonSize(6)).toBe('pequena')
    expect(classifyDungeonSize(7)).toBe('media')
    expect(classifyDungeonSize(20)).toBe('media')
    expect(classifyDungeonSize(21)).toBe('grande')
    expect(classifyDungeonSize(50)).toBe('grande')
  })

  it('null for < 3 rooms (below smallest tier)', () => {
    expect(classifyDungeonSize(1)).toBeNull()
    expect(classifyDungeonSize(2)).toBeNull()
  })

  it('null for > 50 rooms (livro recomenda não crescer além)', () => {
    expect(classifyDungeonSize(51)).toBeNull()
    expect(classifyDungeonSize(100)).toBeNull()
  })

  it('rejects < 1', () => {
    expect(() => classifyDungeonSize(0)).toThrow()
    expect(() => classifyDungeonSize(-5)).toThrow()
  })
})

describe('Tabela 6-2: Ideias de Masmorras (p263)', () => {
  it('has exactly 20 entries (d20)', () => {
    expect(DUNGEON_IDEA_TABLE).toHaveLength(20)
  })

  it('rolls are 1..20 unique + ordered', () => {
    expect(DUNGEON_IDEA_TABLE.map((r) => r.roll)).toEqual(
      Array.from({ length: 20 }, (_, i) => i + 1),
    )
  })

  it('every label is non-empty', () => {
    for (const row of DUNGEON_IDEA_TABLE) {
      expect(row.label.length).toBeGreaterThan(0)
    }
  })

  it('roll 1 = Complexo de cavernas subterrâneas', () => {
    expect(dungeonIdeaFromRoll(1).label).toBe(
      'Complexo de cavernas subterrâneas',
    )
  })

  it('roll 12 = Prisão da cidade', () => {
    expect(dungeonIdeaFromRoll(12).label).toBe('Prisão da cidade')
  })

  it('roll 20 = Castelo nas nuvens', () => {
    expect(dungeonIdeaFromRoll(20).label).toBe('Castelo nas nuvens')
  })

  it('rejects out-of-range d20', () => {
    expect(() => dungeonIdeaFromRoll(0)).toThrow()
    expect(() => dungeonIdeaFromRoll(21)).toThrow()
    expect(() => dungeonIdeaFromRoll(1.5)).toThrow()
  })
})
