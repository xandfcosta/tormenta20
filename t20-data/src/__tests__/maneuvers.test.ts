import { describe, expect, it } from 'vitest'
import {
  MANEUVERS,
  MANEUVER_IDS,
  maneuverOutcome,
  type ManeuverId,
} from '../maneuvers'

/**
 * PDF book p233-234 — Manobras de Combate.
 *
 * Core mechanic: ação padrão, opposed Luta test. Cannot be performed
 * with ranged weapons; even a ranged-armed defender opposes with Luta.
 *
 * Pinned:
 *   - 5 core manobras: Agarrar, Derrubar, Desarmar, Empurrar, Quebrar
 *   - all are ação padrão
 *   - Derrubar / Desarmar gain a bonus effect "vencer por 5 pontos ou
 *     mais"; Agarrar / Empurrar / Quebrar do NOT
 *   - Empurrar uses a margin formula but its base success doesn't have
 *     a discrete 5-over breakpoint (the formula is continuous: +1.5m
 *     per 5 of margin)
 */
const BOOK_MANEUVERS: readonly ManeuverId[] = [
  'agarrar',
  'derrubar',
  'desarmar',
  'empurrar',
  'quebrar',
]

describe('MANEUVERS — catalog completeness', () => {
  it('covers the 5 core PDF manobras', () => {
    expect([...MANEUVER_IDS].sort()).toEqual([...BOOK_MANEUVERS].sort())
  })

  it('every id resolves to a record with a non-empty name + effect', () => {
    for (const id of MANEUVER_IDS) {
      const entry = MANEUVERS[id]
      expect(entry.id).toBe(id)
      expect(entry.name).toBeTruthy()
      expect(entry.successEffect).toBeTruthy()
    }
  })

  it('every manobra is ação padrão (PDF: "ação padrão para fazer ataque corpo a corpo")', () => {
    for (const id of MANEUVER_IDS) {
      expect(MANEUVERS[id].action).toBe('padrao')
    }
  })

  it('every manobra opposes Luta (defender uses Luta even with ranged weapon)', () => {
    for (const id of MANEUVER_IDS) {
      expect(MANEUVERS[id].defenderRoll).toBe('luta')
    }
  })
})

describe('MANEUVERS — five-over bonus flags vs PDF', () => {
  it('Derrubar grants a five-over bonus (push 1 square)', () => {
    expect(MANEUVERS.derrubar.hasFiveOverBonus).toBe(true)
  })

  it('Desarmar grants a five-over bonus (fling 1 square)', () => {
    expect(MANEUVERS.desarmar.hasFiveOverBonus).toBe(true)
  })

  it('Agarrar does NOT have a discrete five-over bonus', () => {
    expect(MANEUVERS.agarrar.hasFiveOverBonus).toBe(false)
  })

  it('Empurrar uses a continuous margin formula (no discrete 5-over flag)', () => {
    expect(MANEUVERS.empurrar.hasFiveOverBonus).toBe(false)
  })

  it('Quebrar does NOT have a five-over bonus', () => {
    expect(MANEUVERS.quebrar.hasFiveOverBonus).toBe(false)
  })
})

describe('maneuverOutcome — opposed test resolution', () => {
  it('attacker total higher → success with positive margin', () => {
    const out = maneuverOutcome('derrubar', 15, 10)
    expect(out.success).toBe(true)
    expect(out.margin).toBe(5)
  })

  it('attacker total lower → failure with margin 0', () => {
    const out = maneuverOutcome('derrubar', 8, 12)
    expect(out.success).toBe(false)
    expect(out.margin).toBe(0)
  })

  it('tie → failure (PDF says reroll; caller treats as failure for now)', () => {
    // PDF rule: highest Luta bonus wins, then reroll. Caller has
    // already applied the bonus-tiebreak before passing the totals
    // here, so an equal-total result means "go reroll".
    const out = maneuverOutcome('derrubar', 12, 12)
    expect(out.success).toBe(false)
  })

  it('Derrubar: success with margin exactly 5 triggers the bonus', () => {
    expect(maneuverOutcome('derrubar', 15, 10).fiveOverBonus).toBe(true)
  })

  it('Derrubar: success with margin 4 does NOT trigger the bonus', () => {
    expect(maneuverOutcome('derrubar', 14, 10).fiveOverBonus).toBe(false)
  })

  it('Desarmar: bonus triggers at margin ≥ 5 (PDF p234)', () => {
    expect(maneuverOutcome('desarmar', 18, 10).fiveOverBonus).toBe(true)
    expect(maneuverOutcome('desarmar', 14, 10).fiveOverBonus).toBe(false)
  })

  it('Agarrar: no five-over bonus regardless of margin', () => {
    expect(maneuverOutcome('agarrar', 25, 10).fiveOverBonus).toBe(false)
  })

  it('Empurrar: no discrete bonus flag (continuous +1.5m per 5 margin)', () => {
    expect(maneuverOutcome('empurrar', 30, 10).fiveOverBonus).toBe(false)
  })

  it('failure cannot trigger the five-over bonus even with large gap', () => {
    // Defensive: success guard takes precedence over margin check.
    expect(maneuverOutcome('derrubar', 5, 20).fiveOverBonus).toBe(false)
  })
})
