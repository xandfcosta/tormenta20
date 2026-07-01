import { describe, expect, it } from 'vitest'
import type { SchoolSpecialization } from '../school-specialization'
import {
  focoTotalPicks,
  totalSpellPmCost,
  type FocoEmMagia,
} from '../spell-pm-composition'

/**
 * Composição PM: base + aprimoramento - Foco em Magia - Mestre em Escola.
 * Todas reduções cumulativas, floor 0.
 *
 * Referência PDF (`SPELL_BASE_PM_COST` from spells.ts p171):
 *  - Círculos: 0=0, 1=1, 2=3, 3=6, 4=10, 5=15.
 *  - Foco em Magia -1 PM cumulativo (p131).
 *  - Mestre em Escola -1 PM (p38).
 */

describe('focoTotalPicks', () => {
  it('vazio → 0', () => {
    expect(focoTotalPicks('bola-de-fogo', [])).toBe(0)
  })

  it('1 pick → 1', () => {
    const focos: FocoEmMagia[] = [{ spellId: 'bola-de-fogo', picks: 1 }]
    expect(focoTotalPicks('bola-de-fogo', focos)).toBe(1)
  })

  it('múltiplas entries somam', () => {
    const focos: FocoEmMagia[] = [
      { spellId: 'bola-de-fogo', picks: 1 },
      { spellId: 'bola-de-fogo', picks: 2 },
    ]
    expect(focoTotalPicks('bola-de-fogo', focos)).toBe(3)
  })

  it('ignora spellIds diferentes', () => {
    const focos: FocoEmMagia[] = [
      { spellId: 'bola-de-fogo', picks: 2 },
      { spellId: 'raio', picks: 3 },
    ]
    expect(focoTotalPicks('bola-de-fogo', focos)).toBe(2)
    expect(focoTotalPicks('raio', focos)).toBe(3)
  })
})

describe('totalSpellPmCost — base cost puro (sem reduções)', () => {
  it('círculo 3 sem aprimoramento = 6 PM', () => {
    expect(
      totalSpellPmCost({
        circle: 3,
        spellId: 'bola-de-fogo',
        spellSchool: 'evocacao',
      }),
    ).toBe(6)
  })

  it('círculo 1 + augment 2 PM = 3 PM', () => {
    expect(
      totalSpellPmCost({
        circle: 1,
        augmentPm: 2,
        spellId: 'raio',
        spellSchool: 'evocacao',
      }),
    ).toBe(3)
  })

  it('truque (círculo 0) = 0 PM', () => {
    expect(
      totalSpellPmCost({
        circle: 0,
        spellId: 'faisca',
        spellSchool: 'evocacao',
      }),
    ).toBe(0)
  })
})

describe('totalSpellPmCost — Foco em Magia', () => {
  it('círculo 3 + 1 pick foco = 5 PM', () => {
    expect(
      totalSpellPmCost({
        circle: 3,
        spellId: 'bola-de-fogo',
        spellSchool: 'evocacao',
        focos: [{ spellId: 'bola-de-fogo', picks: 1 }],
      }),
    ).toBe(5)
  })

  it('círculo 3 + 2 picks foco = 4 PM (cumulativo)', () => {
    expect(
      totalSpellPmCost({
        circle: 3,
        spellId: 'bola-de-fogo',
        spellSchool: 'evocacao',
        focos: [{ spellId: 'bola-de-fogo', picks: 2 }],
      }),
    ).toBe(4)
  })

  it('foco em outra magia não afeta', () => {
    expect(
      totalSpellPmCost({
        circle: 3,
        spellId: 'bola-de-fogo',
        spellSchool: 'evocacao',
        focos: [{ spellId: 'raio', picks: 2 }],
      }),
    ).toBe(6)
  })
})

describe('totalSpellPmCost — Mestre em Escola', () => {
  it('círculo 3 + Mestre evocação = 5 PM', () => {
    const specs: SchoolSpecialization[] = [
      { school: 'evocacao', kind: 'mestre' },
    ]
    expect(
      totalSpellPmCost({
        circle: 3,
        spellId: 'bola-de-fogo',
        spellSchool: 'evocacao',
        specializations: specs,
      }),
    ).toBe(5)
  })

  it('Mestre em escola diferente não afeta', () => {
    const specs: SchoolSpecialization[] = [
      { school: 'ilusao', kind: 'mestre' },
    ]
    expect(
      totalSpellPmCost({
        circle: 3,
        spellId: 'bola-de-fogo',
        spellSchool: 'evocacao',
        specializations: specs,
      }),
    ).toBe(6)
  })
})

describe('totalSpellPmCost — Foco + Mestre empilham', () => {
  it('círculo 3 + Foco 2 + Mestre = 3 PM', () => {
    // base 6 - foco 2 = 4 - mestre 1 = 3
    const specs: SchoolSpecialization[] = [
      { school: 'evocacao', kind: 'mestre' },
    ]
    expect(
      totalSpellPmCost({
        circle: 3,
        spellId: 'bola-de-fogo',
        spellSchool: 'evocacao',
        focos: [{ spellId: 'bola-de-fogo', picks: 2 }],
        specializations: specs,
      }),
    ).toBe(3)
  })

  it('reduções levam a 0 mínimo (nunca negativo)', () => {
    const specs: SchoolSpecialization[] = [
      { school: 'evocacao', kind: 'mestre' },
    ]
    // círculo 1 = 1 PM base. Foco 3 + Mestre = -4 PM reduction. Floor 0.
    expect(
      totalSpellPmCost({
        circle: 1,
        spellId: 'faisca',
        spellSchool: 'evocacao',
        focos: [{ spellId: 'faisca', picks: 3 }],
        specializations: specs,
      }),
    ).toBe(0)
  })

  it('composição completa: círculo 4 + augment 2 + Foco 1 + Mestre = 10 PM', () => {
    // Base 10 + augment 2 = 12. Foco 1 → 11. Mestre 1 → 10.
    const specs: SchoolSpecialization[] = [
      { school: 'evocacao', kind: 'mestre' },
    ]
    expect(
      totalSpellPmCost({
        circle: 4,
        augmentPm: 2,
        spellId: 'bola-de-fogo',
        spellSchool: 'evocacao',
        focos: [{ spellId: 'bola-de-fogo', picks: 1 }],
        specializations: specs,
      }),
    ).toBe(10)
  })
})

describe('totalSpellPmCost — validation', () => {
  it('throws se spellId vazio', () => {
    expect(() =>
      totalSpellPmCost({
        circle: 1,
        spellId: '',
        spellSchool: 'evocacao',
      }),
    ).toThrow(/spellId/)
  })

  it('Especialista sozinho não reduz PM (só afeta CD)', () => {
    const specs: SchoolSpecialization[] = [
      { school: 'evocacao', kind: 'especialista' },
    ]
    expect(
      totalSpellPmCost({
        circle: 3,
        spellId: 'bola-de-fogo',
        spellSchool: 'evocacao',
        specializations: specs,
      }),
    ).toBe(6)
  })
})
