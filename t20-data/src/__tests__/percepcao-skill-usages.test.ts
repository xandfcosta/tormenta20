import { describe, expect, it } from 'vitest'
import {
  OBSERVAR_CD_DIFICIL,
  OBSERVAR_CD_QUASE_INVISIVEL,
  OBSERVAR_LER_LABIOS_CD,
  OUVIR_CD_CONVERSA_CASUAL,
  OUVIR_CD_SUSSURRO,
  OUVIR_CRIATURA_FURTIVIDADE_BONUS,
  OUVIR_CRIATURA_INVISIVEL_BASE_CD,
  OUVIR_SLEEPING_PENALTY,
  OUVIR_THROUGH_DOOR_CD_INCREASE,
  PERCEPCAO_ARMOR_PENALTY,
  PERCEPCAO_TRAINED_ONLY,
  PERCEPCAO_USAGES,
  observarCd,
  ouvirCd,
  ouvirCriaturaInvisivelCd,
  ouvirSleepingPenalty,
  ouvirThroughDoorModifier,
  percepcaoUsageByKind,
} from '../percepcao-skill-usages'

/**
 * PDF livro p122 — Perícia Percepção (SAB, aberta, sem armor penalty).
 *  1. Observar — CDs 15/30, ler lábios 20, escondido = roll de Furt/Lad-Ocultar
 *  2. Ouvir — CD 0 conversa, 15 sussurro, +10 porta, -10 dormindo, criatura invisível CD 20 ou Furt+10 (o maior)
 */

describe('PERCEPCAO_USAGES — shape', () => {
  it('exatamente 2 usos', () => {
    expect(PERCEPCAO_USAGES.length).toBe(2)
  })

  it('frozen', () => {
    expect(Object.isFrozen(PERCEPCAO_USAGES)).toBe(true)
  })

  it('ids canônicos', () => {
    expect(PERCEPCAO_USAGES.map((u) => u.id).sort()).toEqual([
      'observar',
      'ouvir',
    ])
  })

  it('todos em p122', () => {
    for (const u of PERCEPCAO_USAGES) {
      expect(u.bookPage).toBe(122)
    }
  })
})

describe('Perícia flags — Tabela 2-1', () => {
  it('NÃO é somente treinada', () => {
    expect(PERCEPCAO_TRAINED_ONLY).toBe(false)
  })

  it('sem penalidade de armadura', () => {
    expect(PERCEPCAO_ARMOR_PENALTY).toBe(false)
  })
})

describe('Observar — p122', () => {
  const usage = percepcaoUsageByKind('observar')

  it('CDs verbatim', () => {
    expect(OBSERVAR_CD_DIFICIL).toBe(15)
    expect(OBSERVAR_CD_QUASE_INVISIVEL).toBe(30)
    expect(OBSERVAR_LER_LABIOS_CD).toBe(20)
  })

  it('CD escondido = roll do hider', () => {
    if (usage.kind !== 'observar') throw new Error('narrow failed')
    expect(usage.cdHiddenEqualsHiderRoll).toBe(true)
  })
})

describe('observarCd', () => {
  it('dificil → 15', () => {
    expect(observarCd({ kind: 'dificil' })).toBe(15)
  })

  it('quase-invisivel → 30', () => {
    expect(observarCd({ kind: 'quase-invisivel' })).toBe(30)
  })

  it('ler-labios → 20', () => {
    expect(observarCd({ kind: 'ler-labios' })).toBe(20)
  })

  it('escondido-por-furtividade → roll de Furtividade', () => {
    expect(
      observarCd({ kind: 'escondido-por-furtividade', furtividadeRoll: 24 }),
    ).toBe(24)
  })

  it('escondido-por-ladinagem-ocultar → roll de Ladinagem', () => {
    expect(
      observarCd({ kind: 'escondido-por-ladinagem-ocultar', ladinagemRoll: 18 }),
    ).toBe(18)
  })
})

describe('Ouvir — p122', () => {
  const usage = percepcaoUsageByKind('ouvir')

  it('CDs verbatim', () => {
    expect(OUVIR_CD_CONVERSA_CASUAL).toBe(0)
    expect(OUVIR_CD_SUSSURRO).toBe(15)
    expect(OUVIR_THROUGH_DOOR_CD_INCREASE).toBe(10)
    expect(OUVIR_SLEEPING_PENALTY).toBe(-10)
    expect(OUVIR_CRIATURA_INVISIVEL_BASE_CD).toBe(20)
    expect(OUVIR_CRIATURA_FURTIVIDADE_BONUS).toBe(10)
  })

  it('sucesso dormindo acorda', () => {
    if (usage.kind !== 'ouvir') throw new Error('narrow failed')
    expect(usage.sleepingSuccessWakes).toBe(true)
  })

  it('mantém penalidades de lutar sem ver', () => {
    if (usage.kind !== 'ouvir') throw new Error('narrow failed')
    expect(usage.keepsBlindFightPenalties).toBe(true)
  })
})

describe('ouvirThroughDoorModifier', () => {
  it('porta → +10', () => {
    expect(ouvirThroughDoorModifier(true)).toBe(10)
  })

  it('ar livre → 0', () => {
    expect(ouvirThroughDoorModifier(false)).toBe(0)
  })
})

describe('ouvirSleepingPenalty', () => {
  it('dormindo → -10', () => {
    expect(ouvirSleepingPenalty(true)).toBe(-10)
  })

  it('acordado → 0', () => {
    expect(ouvirSleepingPenalty(false)).toBe(0)
  })
})

describe('ouvirCriaturaInvisivelCd', () => {
  it('Furt baixo → CD 20 (base)', () => {
    expect(ouvirCriaturaInvisivelCd(5)).toBe(20)
  })

  it('Furt 10 → Furt+10 = 20 = base (empate)', () => {
    expect(ouvirCriaturaInvisivelCd(10)).toBe(20)
  })

  it('Furt 15 → Furt+10 = 25 (maior que 20)', () => {
    expect(ouvirCriaturaInvisivelCd(15)).toBe(25)
  })

  it('Furt 30 → Furt+10 = 40', () => {
    expect(ouvirCriaturaInvisivelCd(30)).toBe(40)
  })
})

describe('ouvirCd', () => {
  it('conversa casual → 0', () => {
    expect(ouvirCd({ kind: 'conversa-casual' })).toBe(0)
  })

  it('sussurro → 15', () => {
    expect(ouvirCd({ kind: 'sussurro' })).toBe(15)
  })

  it('sussurro atrás de porta → 25', () => {
    expect(ouvirCd({ kind: 'sussurro' }, true)).toBe(25)
  })

  it('conversa casual atrás de porta → 10', () => {
    expect(ouvirCd({ kind: 'conversa-casual' }, true)).toBe(10)
  })

  it('criatura invisível Furt 22 → 32', () => {
    expect(
      ouvirCd({ kind: 'criatura-invisivel', furtividadeRoll: 22 }),
    ).toBe(32)
  })

  it('criatura invisível Furt 5 atrás de porta → base 20 + 10 = 30', () => {
    expect(
      ouvirCd({ kind: 'criatura-invisivel', furtividadeRoll: 5 }, true),
    ).toBe(30)
  })
})

describe('percepcaoUsageByKind', () => {
  it('throws se kind inválido', () => {
    expect(() =>
      // @ts-expect-error — invalid kind on purpose
      percepcaoUsageByKind('cheirar'),
    ).toThrow(/unknown kind/)
  })

  it('resolve todos', () => {
    for (const k of ['observar', 'ouvir'] as const) {
      expect(percepcaoUsageByKind(k).kind).toBe(k)
    }
  })
})
