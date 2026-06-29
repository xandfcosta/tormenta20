import { describe, expect, it } from 'vitest'
import {
  SPELLCASTER_CLASSES,
  SPELL_PROGRESSION,
  arcanistaProgression,
  canCastCircle,
  feiticeiroSpellsKnown,
  highestCircleAtLevel,
  paladinSpellsViaOrar,
  totalSpellsKnown,
} from '../class-spellcasting'

/**
 * PDF book class tables (p37/44/57/61/81). Pinned:
 *  - 5 classes cast: Arcanista, Bardo, Clérigo, Druida, Paladino
 *  - Arcanista/Clérigo (full): 1° L1, 2° L5, 3° L9, 4° L13, 5° L17
 *  - Bardo/Druida (½): 1° L1, 2° L6, 3° L10, 4° L14, NO 5°
 *  - Paladino: 1° via poder Orar (cap 1°)
 *  - Magias conhecidas: Arc/Cle 3 inicial +1/level; Bardo/Druida 2 inicial +1/par
 *  - Caminho do Arcanista: Bruxo/Mago Int, Feiticeiro Car + cadence ímpar
 *  - Mago starts with 4 magias (grimório bonus)
 *  - Bardo/Druida lock to 3 escolas at L1
 */

describe('SPELLCASTER_CLASSES — coverage', () => {
  it('lists exactly the 5 spellcaster classes', () => {
    expect([...SPELLCASTER_CLASSES].sort()).toEqual([
      'Arcanista',
      'Bardo',
      'Clérigo',
      'Druida',
      'Paladino',
    ])
  })
})

describe('SPELL_PROGRESSION — full caster unlock levels', () => {
  it('Arcanista unlocks 1°/2°/3°/4°/5° at L1/5/9/13/17', () => {
    const u = SPELL_PROGRESSION.Arcanista.unlockLevel
    expect(u[1]).toBe(1)
    expect(u[2]).toBe(5)
    expect(u[3]).toBe(9)
    expect(u[4]).toBe(13)
    expect(u[5]).toBe(17)
  })

  it('Clérigo follows the same full-caster table', () => {
    const u = SPELL_PROGRESSION['Clérigo'].unlockLevel
    expect(u[1]).toBe(1)
    expect(u[2]).toBe(5)
    expect(u[3]).toBe(9)
    expect(u[4]).toBe(13)
    expect(u[5]).toBe(17)
  })
})

describe('SPELL_PROGRESSION — half-caster (Bardo/Druida)', () => {
  it('Bardo unlocks 1°/2°/3°/4° at L1/6/10/14; no 5°', () => {
    const u = SPELL_PROGRESSION.Bardo.unlockLevel
    expect(u[1]).toBe(1)
    expect(u[2]).toBe(6)
    expect(u[3]).toBe(10)
    expect(u[4]).toBe(14)
    expect(u[5]).toBeNull()
  })

  it('Druida shares the Bardo curve', () => {
    const u = SPELL_PROGRESSION.Druida.unlockLevel
    expect(u[1]).toBe(1)
    expect(u[2]).toBe(6)
    expect(u[3]).toBe(10)
    expect(u[4]).toBe(14)
    expect(u[5]).toBeNull()
  })

  it('Bardo and Druida cap at 4° círculo', () => {
    expect(SPELL_PROGRESSION.Bardo.maxCircle).toBe(4)
    expect(SPELL_PROGRESSION.Druida.maxCircle).toBe(4)
  })

  it('Bardo/Druida lock to 3 escolas at L1', () => {
    expect(SPELL_PROGRESSION.Bardo.schoolRestriction).toBe(3)
    expect(SPELL_PROGRESSION.Druida.schoolRestriction).toBe(3)
  })
})

describe('SPELL_PROGRESSION — Paladino', () => {
  it('Paladino has no built-in círculo progression; 1° via Orar poder', () => {
    const prog = SPELL_PROGRESSION.Paladino
    expect(prog.maxCircle).toBe(1)
    expect(prog.unlockLevel[1]).toBe(2)
    expect(prog.unlockLevel[2]).toBeNull()
    expect(prog.learnCadence).toBe('via-power')
    expect(prog.startingSpellsKnown).toBe(0)
  })
})

describe('SPELL_PROGRESSION — atributo-chave & lista', () => {
  it('Arcanista (default Bruxo path): Int + Arcana', () => {
    expect(SPELL_PROGRESSION.Arcanista.attribute).toBe('intelligence')
    expect(SPELL_PROGRESSION.Arcanista.list).toBe('arcana')
  })

  it('Bardo: Car + Arcana', () => {
    expect(SPELL_PROGRESSION.Bardo.attribute).toBe('charisma')
    expect(SPELL_PROGRESSION.Bardo.list).toBe('arcana')
  })

  it('Clérigo + Druida + Paladino: Sab + Divina', () => {
    for (const cls of ['Clérigo', 'Druida', 'Paladino'] as const) {
      expect(SPELL_PROGRESSION[cls].attribute).toBe('wisdom')
      expect(SPELL_PROGRESSION[cls].list).toBe('divina')
    }
  })
})

describe('highestCircleAtLevel — Arcanista (full caster)', () => {
  it('L1 → 1°', () => {
    expect(highestCircleAtLevel('Arcanista', 1)).toBe(1)
  })

  it('L4 → still 1° (2° not yet unlocked)', () => {
    expect(highestCircleAtLevel('Arcanista', 4)).toBe(1)
  })

  it('L5 → 2° unlocks', () => {
    expect(highestCircleAtLevel('Arcanista', 5)).toBe(2)
  })

  it('L9 → 3°', () => {
    expect(highestCircleAtLevel('Arcanista', 9)).toBe(3)
  })

  it('L20 → 5° (cap)', () => {
    expect(highestCircleAtLevel('Arcanista', 20)).toBe(5)
  })
})

describe('highestCircleAtLevel — Bardo (half caster, cap 4°)', () => {
  it('L5 → still 1° (2° at L6)', () => {
    expect(highestCircleAtLevel('Bardo', 5)).toBe(1)
  })

  it('L6 → 2° unlocks', () => {
    expect(highestCircleAtLevel('Bardo', 6)).toBe(2)
  })

  it('L20 → 4° (cap, no 5°)', () => {
    expect(highestCircleAtLevel('Bardo', 20)).toBe(4)
  })
})

describe('highestCircleAtLevel — Paladino', () => {
  it('L1 paladino → 0 (no magia yet — Orar unlocks at L2+)', () => {
    expect(highestCircleAtLevel('Paladino', 1)).toBe(0)
  })

  it('L2+ paladino → 1° (via Orar)', () => {
    expect(highestCircleAtLevel('Paladino', 2)).toBe(1)
    expect(highestCircleAtLevel('Paladino', 20)).toBe(1)
  })
})

describe('highestCircleAtLevel — validation', () => {
  it('rejects level 0', () => {
    expect(() => highestCircleAtLevel('Arcanista', 0)).toThrow(/1..20/)
  })

  it('rejects level 21', () => {
    expect(() => highestCircleAtLevel('Arcanista', 21)).toThrow(/1..20/)
  })
})

describe('canCastCircle', () => {
  it('Arcanista L1 can cast 1° but not 2°', () => {
    expect(canCastCircle('Arcanista', 1, 1)).toBe(true)
    expect(canCastCircle('Arcanista', 1, 2)).toBe(false)
  })

  it('Bardo L14 can cast 4° but not 5° (no 5° access)', () => {
    expect(canCastCircle('Bardo', 14, 4)).toBe(true)
    expect(canCastCircle('Bardo', 20, 5)).toBe(false)
  })

  it('truques (0) always castable when class casts', () => {
    expect(canCastCircle('Arcanista', 1, 0)).toBe(true)
    expect(canCastCircle('Bardo', 1, 0)).toBe(true)
  })
})

describe('totalSpellsKnown — every-level cadence (Arcanista bruxo, Clérigo)', () => {
  it('Arcanista L1 starts with 3', () => {
    expect(totalSpellsKnown('Arcanista', 1)).toBe(3)
  })

  it('Arcanista L5 → 3 + 4 = 7', () => {
    expect(totalSpellsKnown('Arcanista', 5)).toBe(7)
  })

  it('Arcanista L20 → 3 + 19 = 22', () => {
    expect(totalSpellsKnown('Arcanista', 20)).toBe(22)
  })

  it('Clérigo follows the same cadence (3 starting, +1/level)', () => {
    expect(totalSpellsKnown('Clérigo', 1)).toBe(3)
    expect(totalSpellsKnown('Clérigo', 10)).toBe(12)
  })
})

describe('totalSpellsKnown — every-even-level (Bardo, Druida)', () => {
  it('Bardo L1 → 2', () => {
    expect(totalSpellsKnown('Bardo', 1)).toBe(2)
  })

  it('Bardo L2 → 3 (+1 at even level)', () => {
    expect(totalSpellsKnown('Bardo', 2)).toBe(3)
  })

  it('Bardo L3 → still 3', () => {
    expect(totalSpellsKnown('Bardo', 3)).toBe(3)
  })

  it('Bardo L20 → 2 + 10 = 12', () => {
    expect(totalSpellsKnown('Bardo', 20)).toBe(12)
  })

  it('Druida shares Bardo cadence', () => {
    expect(totalSpellsKnown('Druida', 6)).toBe(5)
  })
})

describe('feiticeiroSpellsKnown — odd-level cadence', () => {
  it('L1 → 3 (starting only, no learn yet)', () => {
    expect(feiticeiroSpellsKnown(1)).toBe(3)
  })

  it('L3 → 4 (one odd-level pick)', () => {
    expect(feiticeiroSpellsKnown(3)).toBe(4)
  })

  it('L20 → 3 + 9 = 12', () => {
    expect(feiticeiroSpellsKnown(20)).toBe(12)
  })
})

describe('arcanistaProgression — path overrides', () => {
  it('bruxo: Int + every-level', () => {
    const prog = arcanistaProgression('bruxo')
    expect(prog.attribute).toBe('intelligence')
    expect(prog.learnCadence).toBe('every-level')
  })

  it('feiticeiro: Car + odd-level cadence', () => {
    const prog = arcanistaProgression('feiticeiro')
    expect(prog.attribute).toBe('charisma')
    expect(prog.learnCadence).toBe('every-odd-level')
  })

  it('mago: Int + every-level + 4 starting magias (grimório)', () => {
    const prog = arcanistaProgression('mago')
    expect(prog.attribute).toBe('intelligence')
    expect(prog.startingSpellsKnown).toBe(4)
  })

  it('círculo unlock levels are identical across paths', () => {
    for (const path of ['bruxo', 'feiticeiro', 'mago'] as const) {
      const prog = arcanistaProgression(path)
      expect(prog.unlockLevel[2]).toBe(5)
      expect(prog.unlockLevel[5]).toBe(17)
    }
  })
})

describe('paladinSpellsViaOrar', () => {
  it('0 picks → 0 magias', () => {
    expect(paladinSpellsViaOrar(0)).toBe(0)
  })

  it('5 picks → 5 magias', () => {
    expect(paladinSpellsViaOrar(5)).toBe(5)
  })

  it('rejects negative picks', () => {
    expect(() => paladinSpellsViaOrar(-1)).toThrow(/orarPicks must be ≥ 0/)
  })
})
