import { describe, expect, it } from 'vitest'
import {
  NPC_ATTITUDES,
  NPC_ATTITUDE_TABLE,
  NPC_TIERS,
  NPC_TIER_TABLE,
  isPersuasionAutoFail,
  npcSkillBonus,
  persuasionModifierForAttitude,
} from '../npc-stats'

/**
 * PDF Cap 6 p259 — "Regras para NPCs". Two independent artifacts:
 *   1. Categorias de Atitude (5 tiers) → persuasion modifier
 *   2. Tabela 6-1: Estatísticas de NPCs (3 tiers × forte/fraca)
 *
 * Attitude order matches the book text (best-to-worst social disposition).
 */

describe('NPC_ATTITUDES (p259)', () => {
  it('lists all 5 attitude tiers in book order', () => {
    expect(NPC_ATTITUDES).toEqual([
      'prestativo',
      'amistoso',
      'indiferente',
      'inamistoso',
      'hostil',
    ])
  })

  it('table matches the enum 1:1', () => {
    expect(NPC_ATTITUDE_TABLE.map((r) => r.attitude)).toEqual([
      ...NPC_ATTITUDES,
    ])
  })
})

describe('persuasionModifierForAttitude (p259)', () => {
  it('prestativo = +5', () => {
    expect(persuasionModifierForAttitude('prestativo')).toBe(5)
  })

  it('amistoso = 0', () => {
    expect(persuasionModifierForAttitude('amistoso')).toBe(0)
  })

  it('indiferente = 0 (baseline)', () => {
    expect(persuasionModifierForAttitude('indiferente')).toBe(0)
  })

  it('inamistoso = -5', () => {
    expect(persuasionModifierForAttitude('inamistoso')).toBe(-5)
  })

  it('hostil = null (auto-fail)', () => {
    expect(persuasionModifierForAttitude('hostil')).toBeNull()
  })
})

describe('isPersuasionAutoFail (p259)', () => {
  it('only hostil auto-fails', () => {
    expect(isPersuasionAutoFail('hostil')).toBe(true)
    for (const a of ['prestativo', 'amistoso', 'indiferente', 'inamistoso'] as const) {
      expect(isPersuasionAutoFail(a)).toBe(false)
    }
  })
})

describe('NPC_TIERS (Tabela 6-1, p259)', () => {
  it('lists the 3 patamares in ascending power', () => {
    expect(NPC_TIERS).toEqual(['iniciante', 'veterano', 'campeao'])
  })

  it('table matches enum 1:1', () => {
    expect(NPC_TIER_TABLE.map((r) => r.tier)).toEqual([...NPC_TIERS])
  })
})

describe('npcSkillBonus (Tabela 6-1, p259)', () => {
  it('iniciante: forte +5, fraca +0', () => {
    expect(npcSkillBonus('iniciante', 'strong')).toBe(5)
    expect(npcSkillBonus('iniciante', 'weak')).toBe(0)
  })

  it('veterano: forte +10, fraca +3', () => {
    expect(npcSkillBonus('veterano', 'strong')).toBe(10)
    expect(npcSkillBonus('veterano', 'weak')).toBe(3)
  })

  it('campeao: forte +15, fraca +6', () => {
    expect(npcSkillBonus('campeao', 'strong')).toBe(15)
    expect(npcSkillBonus('campeao', 'weak')).toBe(6)
  })

  it('forte column is strictly greater than fraca in every tier', () => {
    for (const row of NPC_TIER_TABLE) {
      expect(row.strongSkill).toBeGreaterThan(row.weakSkill)
    }
  })
})
