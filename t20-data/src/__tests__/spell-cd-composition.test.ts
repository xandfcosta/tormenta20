import { describe, expect, it } from 'vitest'
import type { SchoolSpecialization } from '../school-specialization'
import {
  FORTALECIMENTO_ADVANCED_BONUS,
  FORTALECIMENTO_BASIC_BONUS,
  MAGIA_PUNGENTE_CD_BONUS,
  MAGIA_PUNGENTE_PM_EXTRA,
  MITRAL_ESOTERICO_CD_BONUS,
  MITRAL_ESOTERICO_PM_EXTRA,
  PODEROSO_CD_BONUS,
  extraPmFromCdBoosts,
  totalSpellSaveCd,
} from '../spell-cd-composition'

/**
 * PDF Cap 2 p38 (Arcanista), Cap 3 p167 (Mitral), Cap 3 p165
 * (Poderoso). Pinned:
 *  - Poderoso +1 CD static
 *  - Fortalecimento Arcano: 1-4 → +1; com 4º círculo → +2
 *  - Magia Pungente +2 CD por 1 PM
 *  - Mitral esotérico +2 CD por 2 PM
 *  - Especialista +2 CD por escola
 */

describe('constants', () => {
  it('PODEROSO_CD_BONUS = 1', () => {
    expect(PODEROSO_CD_BONUS).toBe(1)
  })

  it('FORTALECIMENTO_BASIC_BONUS = 1', () => {
    expect(FORTALECIMENTO_BASIC_BONUS).toBe(1)
  })

  it('FORTALECIMENTO_ADVANCED_BONUS = 2', () => {
    expect(FORTALECIMENTO_ADVANCED_BONUS).toBe(2)
  })

  it('MAGIA_PUNGENTE_CD_BONUS = 2, PM extra 1', () => {
    expect(MAGIA_PUNGENTE_CD_BONUS).toBe(2)
    expect(MAGIA_PUNGENTE_PM_EXTRA).toBe(1)
  })

  it('MITRAL_ESOTERICO_CD_BONUS = 2, PM extra 2', () => {
    expect(MITRAL_ESOTERICO_CD_BONUS).toBe(2)
    expect(MITRAL_ESOTERICO_PM_EXTRA).toBe(2)
  })
})

describe('totalSpellSaveCd — no bonuses', () => {
  it('sem bumps → base CD', () => {
    expect(
      totalSpellSaveCd({ baseCd: 15, spellSchool: 'evocacao' }),
    ).toBe(15)
  })
})

describe('totalSpellSaveCd — Especialista em Escola', () => {
  it('Especialista evocação → +2 pra magia evocação', () => {
    const specs: SchoolSpecialization[] = [
      { school: 'evocacao', kind: 'especialista' },
    ]
    expect(
      totalSpellSaveCd({
        baseCd: 15,
        spellSchool: 'evocacao',
        specializations: specs,
      }),
    ).toBe(17)
  })

  it('Especialista em escola diferente → sem bump', () => {
    const specs: SchoolSpecialization[] = [
      { school: 'ilusao', kind: 'especialista' },
    ]
    expect(
      totalSpellSaveCd({
        baseCd: 15,
        spellSchool: 'evocacao',
        specializations: specs,
      }),
    ).toBe(15)
  })
})

describe('totalSpellSaveCd — Poderoso stacks', () => {
  it('1 item Poderoso → +1', () => {
    expect(
      totalSpellSaveCd({
        baseCd: 15,
        spellSchool: 'evocacao',
        poderosoStacks: 1,
      }),
    ).toBe(16)
  })

  it('2 itens Poderoso → +2', () => {
    expect(
      totalSpellSaveCd({
        baseCd: 15,
        spellSchool: 'evocacao',
        poderosoStacks: 2,
      }),
    ).toBe(17)
  })

  it('throws se stacks negativo', () => {
    expect(() =>
      totalSpellSaveCd({
        baseCd: 15,
        spellSchool: 'evocacao',
        poderosoStacks: -1,
      }),
    ).toThrow(/poderosoStacks/)
  })
})

describe('totalSpellSaveCd — Fortalecimento Arcano', () => {
  it('basic → +1', () => {
    expect(
      totalSpellSaveCd({
        baseCd: 15,
        spellSchool: 'evocacao',
        fortalecimentoArcano: 'basic',
      }),
    ).toBe(16)
  })

  it('advanced (4º círculo) → +2', () => {
    expect(
      totalSpellSaveCd({
        baseCd: 15,
        spellSchool: 'evocacao',
        fortalecimentoArcano: 'advanced',
      }),
    ).toBe(17)
  })

  it('none → +0', () => {
    expect(
      totalSpellSaveCd({
        baseCd: 15,
        spellSchool: 'evocacao',
        fortalecimentoArcano: 'none',
      }),
    ).toBe(15)
  })
})

describe('totalSpellSaveCd — Magia Pungente opt-in', () => {
  it('paid → +2', () => {
    expect(
      totalSpellSaveCd({
        baseCd: 15,
        spellSchool: 'evocacao',
        magiaPungentePaid: true,
      }),
    ).toBe(17)
  })

  it('not paid → +0', () => {
    expect(
      totalSpellSaveCd({
        baseCd: 15,
        spellSchool: 'evocacao',
        magiaPungentePaid: false,
      }),
    ).toBe(15)
  })
})

describe('totalSpellSaveCd — Mitral esotérico opt-in', () => {
  it('paid → +2', () => {
    expect(
      totalSpellSaveCd({
        baseCd: 15,
        spellSchool: 'evocacao',
        mitralEsotericoPaid: true,
      }),
    ).toBe(17)
  })
})

describe('totalSpellSaveCd — composição completa', () => {
  it('Especialista + Poderoso + Fortalecimento adv + Magia Pungente + Mitral → +9 CD', () => {
    // Base 15 + 2 (esp) + 1 (pod) + 2 (fort-adv) + 2 (pung) + 2 (mitral) = 24
    const specs: SchoolSpecialization[] = [
      { school: 'evocacao', kind: 'especialista' },
    ]
    expect(
      totalSpellSaveCd({
        baseCd: 15,
        spellSchool: 'evocacao',
        specializations: specs,
        poderosoStacks: 1,
        fortalecimentoArcano: 'advanced',
        magiaPungentePaid: true,
        mitralEsotericoPaid: true,
      }),
    ).toBe(24)
  })

  it('Mestre em Escola sozinho NÃO afeta CD (só afeta PM)', () => {
    const specs: SchoolSpecialization[] = [
      { school: 'evocacao', kind: 'mestre' },
    ]
    expect(
      totalSpellSaveCd({
        baseCd: 15,
        spellSchool: 'evocacao',
        specializations: specs,
      }),
    ).toBe(15)
  })
})

describe('extraPmFromCdBoosts', () => {
  it('sem opt-ins → 0', () => {
    expect(extraPmFromCdBoosts(false, false)).toBe(0)
    expect(extraPmFromCdBoosts()).toBe(0)
  })

  it('só Pungente → 1', () => {
    expect(extraPmFromCdBoosts(true, false)).toBe(1)
  })

  it('só Mitral → 2', () => {
    expect(extraPmFromCdBoosts(false, true)).toBe(2)
  })

  it('ambos → 3', () => {
    expect(extraPmFromCdBoosts(true, true)).toBe(3)
  })
})
