import { describe, expect, it } from 'vitest'
import {
  FORTITUDE_ARMOR_PENALTY,
  FORTITUDE_FOLEGO_CD_BASE,
  FORTITUDE_FOLEGO_CD_INCREMENT,
  FORTITUDE_TRAINED_ONLY,
  FORTITUDE_USAGES,
  REFLEXOS_ARMOR_PENALTY,
  REFLEXOS_TRAINED_ONLY,
  REFLEXOS_USAGES,
  SAVING_THROW_KEY_ATTRIBUTE,
  SAVING_THROW_MODIFIER_APPLIES_TO_ALL,
  VONTADE_ARMOR_PENALTY,
  VONTADE_TRAINED_ONLY,
  VONTADE_USAGES,
  folegoCd,
  fortitudeUsageByKind,
  reflexosUsageByKind,
  savingThrowKeyAttribute,
  savingThrowModifierAppliesTo,
  vontadeUsageByKind,
} from '../saving-throws-skill-usages'

/**
 * PDF livro p119 (sidebar + Fortitude), p122 (Reflexos), p123 (Vontade).
 *
 * Sidebar: bônus em "testes de resistência" aplica às 3 perícias.
 * Fortitude — CON, resistir + fôlego CD 15+1/anterior.
 * Reflexos — DES, resistir + evitar fintas (oposto Enganação).
 * Vontade — SAB, resistir.
 */

describe('SAVING_THROW_KEY_ATTRIBUTE', () => {
  it('frozen', () => {
    expect(Object.isFrozen(SAVING_THROW_KEY_ATTRIBUTE)).toBe(true)
  })

  it('atributos-chave verbatim', () => {
    expect(SAVING_THROW_KEY_ATTRIBUTE.fortitude).toBe('CON')
    expect(SAVING_THROW_KEY_ATTRIBUTE.reflexos).toBe('DES')
    expect(SAVING_THROW_KEY_ATTRIBUTE.vontade).toBe('SAB')
  })
})

describe('SAVING_THROW_MODIFIER_APPLIES_TO_ALL — sidebar p119', () => {
  it('bônus/penalidade em resistência afeta as 3', () => {
    expect(SAVING_THROW_MODIFIER_APPLIES_TO_ALL).toBe(true)
    expect(savingThrowModifierAppliesTo('fortitude')).toBe(true)
    expect(savingThrowModifierAppliesTo('reflexos')).toBe(true)
    expect(savingThrowModifierAppliesTo('vontade')).toBe(true)
  })
})

describe('savingThrowKeyAttribute', () => {
  it('fortitude → CON, reflexos → DES, vontade → SAB', () => {
    expect(savingThrowKeyAttribute('fortitude')).toBe('CON')
    expect(savingThrowKeyAttribute('reflexos')).toBe('DES')
    expect(savingThrowKeyAttribute('vontade')).toBe('SAB')
  })
})

describe('Flags Tabela 2-1 — todas as 3', () => {
  it('NENHUMA é somente treinada', () => {
    expect(FORTITUDE_TRAINED_ONLY).toBe(false)
    expect(REFLEXOS_TRAINED_ONLY).toBe(false)
    expect(VONTADE_TRAINED_ONLY).toBe(false)
  })

  it('NENHUMA sofre penalidade de armadura', () => {
    expect(FORTITUDE_ARMOR_PENALTY).toBe(false)
    expect(REFLEXOS_ARMOR_PENALTY).toBe(false)
    expect(VONTADE_ARMOR_PENALTY).toBe(false)
  })
})

// ─── Fortitude ───────────────────────────────────────────────────────
describe('FORTITUDE_USAGES — shape', () => {
  it('2 usos (resistir + folego)', () => {
    expect(FORTITUDE_USAGES.length).toBe(2)
  })

  it('frozen', () => {
    expect(Object.isFrozen(FORTITUDE_USAGES)).toBe(true)
  })

  it('ids canônicos', () => {
    expect(FORTITUDE_USAGES.map((u) => u.id).sort()).toEqual([
      'folego',
      'resistir-efeito',
    ])
  })

  it('todos em p119', () => {
    for (const u of FORTITUDE_USAGES) {
      expect(u.bookPage).toBe(119)
    }
  })
})

describe('Fortitude — Resistir', () => {
  const usage = fortitudeUsageByKind('resistir-efeito')

  it('CD determinada pelo efeito', () => {
    if (usage.kind !== 'resistir-efeito') throw new Error('narrow failed')
    expect(usage.cdSource).toBe('per-effect')
    expect(usage.category).toBe('saving-throw')
  })
})

describe('Fortitude — Fôlego', () => {
  const usage = fortitudeUsageByKind('folego')

  it('CD 15 +1 por teste anterior', () => {
    expect(FORTITUDE_FOLEGO_CD_BASE).toBe(15)
    expect(FORTITUDE_FOLEGO_CD_INCREMENT).toBe(1)
    if (usage.kind !== 'folego') throw new Error('narrow failed')
    expect(usage.cdBase).toBe(15)
    expect(usage.cdIncrementPerPreviousTest).toBe(1)
  })
})

describe('folegoCd', () => {
  it('sem testes anteriores → 15', () => {
    expect(folegoCd(0)).toBe(15)
  })

  it('após 4 testes → 19', () => {
    expect(folegoCd(4)).toBe(19)
  })
})

describe('fortitudeUsageByKind', () => {
  it('throws se inválido', () => {
    expect(() =>
      // @ts-expect-error — invalid kind on purpose
      fortitudeUsageByKind('endurance'),
    ).toThrow(/unknown kind/)
  })
})

// ─── Reflexos ────────────────────────────────────────────────────────
describe('REFLEXOS_USAGES — shape', () => {
  it('2 usos (resistir + evitar-fintas)', () => {
    expect(REFLEXOS_USAGES.length).toBe(2)
  })

  it('frozen', () => {
    expect(Object.isFrozen(REFLEXOS_USAGES)).toBe(true)
  })

  it('ids canônicos', () => {
    expect(REFLEXOS_USAGES.map((u) => u.id).sort()).toEqual([
      'evitar-fintas',
      'resistir-efeito',
    ])
  })

  it('todos em p122', () => {
    for (const u of REFLEXOS_USAGES) {
      expect(u.bookPage).toBe(122)
    }
  })
})

describe('Reflexos — Resistir', () => {
  const usage = reflexosUsageByKind('resistir-efeito')

  it('CD determinada pelo efeito', () => {
    if (usage.kind !== 'resistir-efeito') throw new Error('narrow failed')
    expect(usage.cdSource).toBe('per-effect')
    expect(usage.category).toBe('saving-throw')
  })
})

describe('Reflexos — Evitar Fintas', () => {
  const usage = reflexosUsageByKind('evitar-fintas')

  it('oposto por Enganação (Fintar)', () => {
    if (usage.kind !== 'evitar-fintas') throw new Error('narrow failed')
    expect(usage.opposedBy).toBe('enganacao')
  })
})

// ─── Vontade ─────────────────────────────────────────────────────────
describe('VONTADE_USAGES — shape', () => {
  it('1 uso (resistir)', () => {
    expect(VONTADE_USAGES.length).toBe(1)
  })

  it('frozen', () => {
    expect(Object.isFrozen(VONTADE_USAGES)).toBe(true)
  })

  it('id canônico', () => {
    expect(VONTADE_USAGES.map((u) => u.id)).toEqual(['resistir-efeito'])
  })

  it('todos em p123', () => {
    for (const u of VONTADE_USAGES) {
      expect(u.bookPage).toBe(123)
    }
  })
})

describe('Vontade — Resistir', () => {
  const usage = vontadeUsageByKind('resistir-efeito')

  it('CD determinada pelo efeito', () => {
    if (usage.kind !== 'resistir-efeito') throw new Error('narrow failed')
    expect(usage.cdSource).toBe('per-effect')
    expect(usage.category).toBe('saving-throw')
  })
})

describe('vontadeUsageByKind / reflexosUsageByKind', () => {
  it('throws se inválido', () => {
    expect(() =>
      // @ts-expect-error — invalid kind on purpose
      vontadeUsageByKind('mental-shield'),
    ).toThrow(/unknown kind/)
    expect(() =>
      // @ts-expect-error — invalid kind on purpose
      reflexosUsageByKind('esquivar-tudo'),
    ).toThrow(/unknown kind/)
  })
})
