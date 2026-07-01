import { describe, expect, it } from 'vitest'
import {
  applySaveResult,
  armorArcaneConcentrationCd,
  damageAfterSave,
  damageConcentrationCd,
  desacreditaExtraSaveAllowed,
  resolveCounterspell,
} from '../spell-save-resolution'

/**
 * PDF Cap 4 p170-173 + Cap 5 p227. Pinned:
 *  - dano CD = dano recebido (p170)
 *  - armor arcane CD = 20 + PM + armor penalty (p171)
 *  - anula pass → none; metade pass → half; parcial pass → reduced;
 *    desacredita pass → revealed
 *  - Dissipar Magia: Misticismo vs max(Mist, Vontade)
 */

describe('damageConcentrationCd — p170', () => {
  it('dano 15 → CD 15', () => {
    expect(damageConcentrationCd(15)).toBe(15)
  })

  it('dano 0 → CD 0 (sem teste)', () => {
    expect(damageConcentrationCd(0)).toBe(0)
  })

  it('dano negativo (cura) → CD 0', () => {
    expect(damageConcentrationCd(-5)).toBe(0)
  })
})

describe('armorArcaneConcentrationCd — p171', () => {
  it('PM 3 + penalidade -4 → CD 27', () => {
    expect(armorArcaneConcentrationCd(3, 4)).toBe(27)
  })

  it('PM 0 + sem penalidade → CD 20', () => {
    expect(armorArcaneConcentrationCd(0, 0)).toBe(20)
  })

  it('throws se PM negativo', () => {
    expect(() => armorArcaneConcentrationCd(-1, 0)).toThrow(/spellTotalPm/)
  })

  it('throws se penalidade negativa', () => {
    expect(() => armorArcaneConcentrationCd(0, -1)).toThrow(/armorPenalty/)
  })
})

describe('applySaveResult — p227', () => {
  it('anula + fail → full', () => {
    expect(applySaveResult('anula', false)).toBe('full')
  })

  it('anula + pass → none', () => {
    expect(applySaveResult('anula', true)).toBe('none')
  })

  it('metade + pass → half', () => {
    expect(applySaveResult('metade', true)).toBe('half')
  })

  it('metade + fail → full', () => {
    expect(applySaveResult('metade', false)).toBe('full')
  })

  it('parcial + pass → reduced', () => {
    expect(applySaveResult('parcial', true)).toBe('reduced')
  })

  it('desacredita + pass → revealed', () => {
    expect(applySaveResult('desacredita', true)).toBe('revealed')
  })

  it('desacredita + fail → full (não percebe ilusão)', () => {
    expect(applySaveResult('desacredita', false)).toBe('full')
  })
})

describe('damageAfterSave — aplicação numérica', () => {
  it('anula + pass → 0 (bola de fogo evitada)', () => {
    expect(damageAfterSave(30, 'anula', true)).toBe(0)
  })

  it('anula + fail → dano integral', () => {
    expect(damageAfterSave(30, 'anula', false)).toBe(30)
  })

  it('metade + pass → floor(dano/2)', () => {
    expect(damageAfterSave(31, 'metade', true)).toBe(15)
  })

  it('metade + fail → dano integral', () => {
    expect(damageAfterSave(31, 'metade', false)).toBe(31)
  })

  it('parcial + pass → dano integral (per-spell text define redução)', () => {
    expect(damageAfterSave(20, 'parcial', true)).toBe(20)
  })

  it('desacredita + pass → 0 (ilusão)', () => {
    expect(damageAfterSave(10, 'desacredita', true)).toBe(0)
  })

  it('desacredita + fail → dano integral', () => {
    expect(damageAfterSave(10, 'desacredita', false)).toBe(10)
  })

  it('throws se dano negativo', () => {
    expect(() => damageAfterSave(-1, 'anula', true)).toThrow(/damage/)
  })

  it('dano 0 = 0 em qualquer combo', () => {
    for (const r of ['anula', 'metade', 'parcial', 'desacredita'] as const) {
      for (const pass of [true, false]) {
        expect(damageAfterSave(0, r, pass)).toBe(0)
      }
    }
  })
})

describe('desacreditaExtraSaveAllowed — p227', () => {
  it('interação → true', () => {
    expect(desacreditaExtraSaveAllowed(true)).toBe(true)
  })

  it('sem interação → false', () => {
    expect(desacreditaExtraSaveAllowed(false)).toBe(false)
  })
})

describe('resolveCounterspell — p173', () => {
  it('same-spell sempre succeeds', () => {
    const r = resolveCounterspell({ kind: 'same-spell', spellId: 'bola-de-fogo' })
    expect(r.succeeds).toBe(true)
    expect(r.detail).toContain('bola-de-fogo')
  })

  it('dispel: dispeler total > caster total → succeeds', () => {
    const r = resolveCounterspell(
      { kind: 'dispel', dispelerMisticismo: 10, casterOpposedMod: 8 },
      15,
      15,
    )
    // 15+10=25 vs 15+8=23 → succeeds
    expect(r.succeeds).toBe(true)
  })

  it('dispel: empate → falha (need >, not ≥)', () => {
    const r = resolveCounterspell(
      { kind: 'dispel', dispelerMisticismo: 10, casterOpposedMod: 10 },
      15,
      15,
    )
    // 25 vs 25 → tie = not >
    expect(r.succeeds).toBe(false)
  })

  it('dispel: dispeler total < caster total → falha', () => {
    const r = resolveCounterspell(
      { kind: 'dispel', dispelerMisticismo: 5, casterOpposedMod: 12 },
      12,
      18,
    )
    // 12+5=17 vs 18+12=30 → falha
    expect(r.succeeds).toBe(false)
  })

  it('dispel: throws se rolls faltando', () => {
    expect(() =>
      resolveCounterspell({
        kind: 'dispel',
        dispelerMisticismo: 10,
        casterOpposedMod: 8,
      }),
    ).toThrow(/dispelerRoll/)
  })

  it('dispel detail inclui totais', () => {
    const r = resolveCounterspell(
      { kind: 'dispel', dispelerMisticismo: 10, casterOpposedMod: 8 },
      15,
      15,
    )
    expect(r.detail).toContain('25')
    expect(r.detail).toContain('23')
  })
})
