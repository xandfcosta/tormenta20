import { describe, expect, it } from 'vitest'
import {
  HEAVY_ARMOR_SLEEP_APPLIES_FATIGADO,
  MIN_SLEEP_HOURS,
  REST_CONDITIONS,
  REST_CONDITION_LABELS,
  SONO_FORT_CD_BASE,
  descansoNaturalOverride,
  noSleepRecovery,
  restRecoveryAmount,
  restRecoveryWithCare,
  sonoForcadoFortCd,
  type RestCondition,
} from '../rest'

/**
 * PDF p106 — "Recuperando PV e PM". One 8h sleep restores PV and PM
 * (same amount) by `nível × multiplier(condition)`:
 *   Ruim         → ½
 *   Normal       → 1
 *   Confortável  → 2
 *   Luxuosa      → 3
 *
 * PDF caps recovery at "never more than lost" — caller clamps to
 * current max. These tests pin the *recovery amount* formula, not the
 * clamp (it's a simple Math.min at the call site).
 *
 * Cuidados Prolongados (book p117 — Cura, treinado only): +1 per
 * caregiver level per day on top of the lodging tier. One caregiver may
 * tend to up to `level` patients per day; the resolver here does not
 * enforce the patient-count cap (call site decides whether care was
 * given).
 */
describe('REST_CONDITIONS — vocabulary', () => {
  it('lists the four PDF p106 tiers in canonical order', () => {
    expect([...REST_CONDITIONS]).toEqual([
      'ruim',
      'normal',
      'confortavel',
      'luxuosa',
    ])
  })

  it('every tier has a UI label', () => {
    for (const cond of REST_CONDITIONS) {
      expect(REST_CONDITION_LABELS[cond]).toBeTruthy()
    }
  })
})

describe('restRecoveryAmount — PDF p106 formula', () => {
  it.each<[number, RestCondition, number]>([
    // PDF example: "Helior, elfo caçador de 7º nível, recupera 7 PV e 7 PM
    // numa estalagem [Normal]. Dormindo ao relento, se acostumou a
    // recuperar apenas 3 [Ruim]." (book p106)
    [7, 'normal', 7],
    [7, 'ruim', 3], // floor(7/2) = 3
    [7, 'confortavel', 14],
    [7, 'luxuosa', 21],
    // Edge: L1
    [1, 'ruim', 0], // floor(1/2) = 0
    [1, 'normal', 1],
    [1, 'confortavel', 2],
    [1, 'luxuosa', 3],
    // Edge: L20
    [20, 'ruim', 10],
    [20, 'normal', 20],
    [20, 'confortavel', 40],
    [20, 'luxuosa', 60],
    // Edge: even/odd ½-level rounding
    [4, 'ruim', 2],
    [5, 'ruim', 2], // floor(5/2) = 2
    [6, 'ruim', 3],
  ])('L%i %s → %i', (level, condition, expected) => {
    expect(restRecoveryAmount(level, condition)).toBe(expected)
  })

  it('returns 0 for level < 1 (defensive)', () => {
    expect(restRecoveryAmount(0, 'normal')).toBe(0)
    expect(restRecoveryAmount(-3, 'luxuosa')).toBe(0)
  })
})

describe('restRecoveryWithCare — Cuidados Prolongados (PDF p117)', () => {
  it('adds the caregiver level on top of the base tier amount', () => {
    // L5 patient in Normal lodging cared for by a L7 caregiver:
    // base 5 + caregiver 7 = 12.
    expect(restRecoveryWithCare(5, 'normal', 7)).toBe(12)
  })

  it('stacks the bonus even on Ruim conditions', () => {
    // L4 patient in Ruim (base = 2) + L5 caregiver = 7.
    expect(restRecoveryWithCare(4, 'ruim', 5)).toBe(7)
  })

  it('returns the base amount when caregiver level < 1', () => {
    expect(restRecoveryWithCare(5, 'normal', 0)).toBe(5)
    expect(restRecoveryWithCare(5, 'normal', -2)).toBe(5)
  })

  it('caregiver bonus applies to luxury rest too (no implicit cap)', () => {
    // PDF does not cap care at the tier — it's additive.
    expect(restRecoveryWithCare(10, 'luxuosa', 10)).toBe(40)
  })
})

describe('MIN_SLEEP_HOURS + HEAVY_ARMOR_SLEEP_APPLIES_FATIGADO', () => {
  it('minimum sleep window = 8h (PDF p106)', () => {
    expect(MIN_SLEEP_HOURS).toBe(8)
  })

  it('heavy armor sleep applies fatigado (PDF p152)', () => {
    expect(HEAVY_ARMOR_SLEEP_APPLIES_FATIGADO).toBe(true)
  })
})

describe('descansoNaturalOverride — Devoto de Allihanna (PDF p133)', () => {
  it('promotes ruim → confortavel', () => {
    expect(descansoNaturalOverride('ruim')).toBe('confortavel')
  })

  it('leaves normal untouched', () => {
    expect(descansoNaturalOverride('normal')).toBe('normal')
  })

  it('leaves confortavel untouched', () => {
    expect(descansoNaturalOverride('confortavel')).toBe('confortavel')
  })

  it('leaves luxuosa untouched', () => {
    expect(descansoNaturalOverride('luxuosa')).toBe('luxuosa')
  })
})

describe('sono forçado — PDF p318', () => {
  it('SONO_FORT_CD_BASE = 15', () => {
    expect(SONO_FORT_CD_BASE).toBe(15)
  })

  it('primeira noite após limite: CD 15', () => {
    expect(sonoForcadoFortCd(0)).toBe(15)
  })

  it('segunda: CD 16', () => {
    expect(sonoForcadoFortCd(1)).toBe(16)
  })

  it('quinta: CD 19', () => {
    expect(sonoForcadoFortCd(4)).toBe(19)
  })

  it('throws se previousChecks negativo', () => {
    expect(() => sonoForcadoFortCd(-1)).toThrow(/previousChecks/)
  })

  it('noSleepRecovery sempre 0', () => {
    expect(noSleepRecovery()).toBe(0)
  })
})
