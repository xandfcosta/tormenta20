import { describe, expect, it } from 'vitest'
import {
  isValid,
  firstErrorMessage,
  validateEquipChange,
  validateTotalLevel,
  validateConsumeQuantity,
  validateLearnSpell,
  validateSpellLearned,
  validateApplyBuff,
  validateCast,
} from '../rules'

describe('validateEquipChange — equip caps', () => {
  it('allows a 4th vested item, rejects a 5th', () => {
    expect(isValid(validateEquipChange(['vested', 'vested', 'vested'], 'vested'))).toBe(true)
    expect(
      firstErrorMessage(validateEquipChange(['vested', 'vested', 'vested', 'vested'], 'vested')),
    ).toBe('Limite de 4 itens vestidos atingido')
  })

  it('counts hand-slots: wielded2 + wielded exceeds 2 hands', () => {
    expect(isValid(validateEquipChange(['wielded'], 'wielded'))).toBe(true)
    expect(firstErrorMessage(validateEquipChange(['wielded2'], 'wielded'))).toBe(
      'Limite de 2 mãos atingido',
    )
  })

  it('vested items do not consume hand-slots', () => {
    expect(isValid(validateEquipChange(['vested', 'vested'], 'wielded2'))).toBe(true)
  })
})

describe('validateTotalLevel — class-level sum ≤ 20', () => {
  it('accepts a multiclass summing to 20', () => {
    expect(
      isValid(
        validateTotalLevel([
          { className: 'Guerreiro', level: 15 },
          { className: 'Ladino', level: 5 },
        ]),
      ),
    ).toBe(true)
  })

  it('rejects a sum over 20', () => {
    expect(
      firstErrorMessage(validateTotalLevel([{ className: 'Guerreiro', level: 21 }])),
    ).toContain('excede o máximo de 20')
  })

  it('rejects a level below 1', () => {
    expect(isValid(validateTotalLevel([{ className: 'Bardo', level: 0 }]))).toBe(false)
  })
})

describe('validateConsumeQuantity', () => {
  it('requires at least one', () => {
    expect(isValid(validateConsumeQuantity(1))).toBe(true)
    expect(firstErrorMessage(validateConsumeQuantity(0))).toBe('Item esgotado')
  })
})

describe('spell rules', () => {
  // The validators are data-free (A): the caller resolves the catalog lookup and
  // passes the boolean. 'bencao' carries a buff block (Phase-1 SpellBuff).
  it('validateLearnSpell rejects unknown + duplicate, accepts new', () => {
    expect(isValid(validateLearnSpell([], 'bencao', true))).toBe(true)
    expect(firstErrorMessage(validateLearnSpell([], 'not-a-spell', false))).toContain('não existe')
    expect(firstErrorMessage(validateLearnSpell(['bencao'], 'bencao', true))).toBe('Magia já conhecida')
  })

  it('validateSpellLearned requires the spell in the book', () => {
    expect(isValid(validateSpellLearned(['bencao'], 'bencao'))).toBe(true)
    expect(firstErrorMessage(validateSpellLearned([], 'bencao'))).toBe('Magia não aprendida')
  })

  it('validateApplyBuff requires a buff block', () => {
    expect(isValid(validateApplyBuff(true))).toBe(true)
    expect(firstErrorMessage(validateApplyBuff(false))).toBe('Magia sem efeito aplicável')
  })

  it('validateCast enforces prep, PM limit, and current PM', () => {
    const base = { circle: 1, totalPm: 2, pmLimit: 3, mpCurrent: 10, needsPrep: false, prepared: false }
    expect(isValid(validateCast(base))).toBe(true)
    expect(firstErrorMessage(validateCast({ ...base, needsPrep: true }))).toBe(
      'Magia precisa estar preparada',
    )
    expect(firstErrorMessage(validateCast({ ...base, totalPm: 4 }))).toBe('Limite PM excedido (3)')
    expect(firstErrorMessage(validateCast({ ...base, totalPm: 3, mpCurrent: 2 }))).toBe(
      'Sem PM suficiente',
    )
  })

  it('truques (circle 0) skip the PM-limit check', () => {
    expect(
      isValid(validateCast({ circle: 0, totalPm: 0, pmLimit: 0, mpCurrent: 0, needsPrep: false, prepared: false })),
    ).toBe(true)
  })
})
