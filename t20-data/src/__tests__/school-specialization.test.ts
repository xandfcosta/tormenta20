import { describe, expect, it } from 'vitest'
import {
  ESPECIALISTA_CD_BONUS,
  MESTRE_PM_DISCOUNT,
  invalidMestreWithoutEspecialista,
  spellPmCostWithSpecialization,
  spellSaveCdWithSpecialization,
  type SchoolSpecialization,
} from '../school-specialization'

/**
 * PDF Cap 2 p38 (Especialista/Mestre em Escola). Pinned:
 *  - Especialista: +2 CD para resistir a magias da escola.
 *  - Mestre: -1 PM em magias da escola (min 0).
 *  - Mestre requer Especialista da mesma escola (prereq).
 */

describe('constants', () => {
  it('ESPECIALISTA_CD_BONUS = 2', () => {
    expect(ESPECIALISTA_CD_BONUS).toBe(2)
  })

  it('MESTRE_PM_DISCOUNT = 1', () => {
    expect(MESTRE_PM_DISCOUNT).toBe(1)
  })
})

describe('spellSaveCdWithSpecialization', () => {
  it('sem specs → CD base', () => {
    expect(spellSaveCdWithSpecialization(15, 'evocacao', [])).toBe(15)
  })

  it('Especialista em Evocação → CD +2 pra magia de Evocação', () => {
    const specs: SchoolSpecialization[] = [
      { school: 'evocacao', kind: 'especialista' },
    ]
    expect(spellSaveCdWithSpecialization(15, 'evocacao', specs)).toBe(17)
  })

  it('Especialista em Evocação NÃO afeta magia de Ilusão', () => {
    const specs: SchoolSpecialization[] = [
      { school: 'evocacao', kind: 'especialista' },
    ]
    expect(spellSaveCdWithSpecialization(15, 'ilusao', specs)).toBe(15)
  })

  it('duplicados na mesma escola contam 1x (dedupe via Set)', () => {
    const specs: SchoolSpecialization[] = [
      { school: 'evocacao', kind: 'especialista' },
      { school: 'evocacao', kind: 'especialista' },
    ]
    expect(spellSaveCdWithSpecialization(15, 'evocacao', specs)).toBe(17)
  })

  it('Mestre em Escola sozinho NÃO adiciona CD (só Especialista)', () => {
    const specs: SchoolSpecialization[] = [
      { school: 'evocacao', kind: 'mestre' },
    ]
    expect(spellSaveCdWithSpecialization(15, 'evocacao', specs)).toBe(15)
  })

  it('Especialista + Mestre juntos → só +2 CD (do Especialista)', () => {
    const specs: SchoolSpecialization[] = [
      { school: 'evocacao', kind: 'especialista' },
      { school: 'evocacao', kind: 'mestre' },
    ]
    expect(spellSaveCdWithSpecialization(15, 'evocacao', specs)).toBe(17)
  })
})

describe('spellPmCostWithSpecialization', () => {
  it('sem specs → PM base', () => {
    expect(spellPmCostWithSpecialization(3, 'evocacao', [])).toBe(3)
  })

  it('Mestre em Evocação → -1 PM em magia de Evocação', () => {
    const specs: SchoolSpecialization[] = [
      { school: 'evocacao', kind: 'mestre' },
    ]
    expect(spellPmCostWithSpecialization(3, 'evocacao', specs)).toBe(2)
  })

  it('Mestre em Evocação NÃO afeta magia de Ilusão', () => {
    const specs: SchoolSpecialization[] = [
      { school: 'evocacao', kind: 'mestre' },
    ]
    expect(spellPmCostWithSpecialization(3, 'ilusao', specs)).toBe(3)
  })

  it('Especialista sozinho NÃO reduz PM', () => {
    const specs: SchoolSpecialization[] = [
      { school: 'evocacao', kind: 'especialista' },
    ]
    expect(spellPmCostWithSpecialization(3, 'evocacao', specs)).toBe(3)
  })

  it('reduz até 0 mínimo (não fica negativo)', () => {
    const specs: SchoolSpecialization[] = [
      { school: 'evocacao', kind: 'mestre' },
    ]
    expect(spellPmCostWithSpecialization(0, 'evocacao', specs)).toBe(0)
    expect(spellPmCostWithSpecialization(1, 'evocacao', specs)).toBe(0)
  })

  it('throws se basePm < 0', () => {
    expect(() =>
      spellPmCostWithSpecialization(-1, 'evocacao', []),
    ).toThrow(/basePm/)
  })
})

describe('invalidMestreWithoutEspecialista', () => {
  it('sem Mestre → array vazio', () => {
    const specs: SchoolSpecialization[] = [
      { school: 'evocacao', kind: 'especialista' },
    ]
    expect(invalidMestreWithoutEspecialista(specs)).toEqual([])
  })

  it('Mestre com Especialista → válido', () => {
    const specs: SchoolSpecialization[] = [
      { school: 'evocacao', kind: 'especialista' },
      { school: 'evocacao', kind: 'mestre' },
    ]
    expect(invalidMestreWithoutEspecialista(specs)).toEqual([])
  })

  it('Mestre sem Especialista → escola listada', () => {
    const specs: SchoolSpecialization[] = [
      { school: 'necromancia', kind: 'mestre' },
    ]
    expect(invalidMestreWithoutEspecialista(specs)).toEqual(['necromancia'])
  })

  it('multiplas escolas invalidadas', () => {
    const specs: SchoolSpecialization[] = [
      { school: 'evocacao', kind: 'especialista' },
      { school: 'ilusao', kind: 'mestre' },
      { school: 'necromancia', kind: 'mestre' },
    ]
    const invalid = [...invalidMestreWithoutEspecialista(specs)].sort()
    expect(invalid).toEqual(['ilusao', 'necromancia'])
  })

  it('Especialista + Mestre em escolas diferentes → Mestre inválido', () => {
    const specs: SchoolSpecialization[] = [
      { school: 'evocacao', kind: 'especialista' },
      { school: 'transmutacao', kind: 'mestre' },
    ]
    expect(invalidMestreWithoutEspecialista(specs)).toEqual(['transmutacao'])
  })
})
