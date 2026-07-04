import { describe, expect, it } from 'vitest'
import {
  LEFEU_INSANITY_SAMPLES,
  LEFOU_TORMENTA_RESISTANCE_BONUS,
  TORMENTA_AREA_BASE_CD,
  TORMENTA_AREA_CD_PER_PRIOR_DAY,
  TORMENTA_AREA_ENCHANT_LOSS,
  TORMENTA_AREA_ENTRY_CONDITION,
  TORMENTA_AREA_PM_SURCHARGE,
  TORMENTA_AREA_REST_FACTOR,
  TORMENTA_AREA_SAVE_TYPE,
  TORMENTA_NPC_CARISMA_THRESHOLD,
  TORMENTA_POWERS,
  TORMENTA_POWER_IDS,
  canTakePower,
  carismaLossFromPowers,
  escalateTormentaInsanity,
  isTormentaNpc,
  tormentaAreaDailyCd,
} from '../tormenta'

/**
 * Tormenta contamination tests — PDF book p23, p127-128, p135-137,
 * p313-316, p319.
 *
 * Pinned:
 *  - Entry into área: condição `frustrado` automática.
 *  - Vontade CD = 25 + 2 × (dias consecutivos anteriores).
 *  - Escalada: frustrado → esmorecido → confuso → insano-npc.
 *  - Habilidades PM: +2 PM dentro da área.
 *  - Recuperação de descanso: × 0.5 na área.
 *  - Item mágico perde 1 encantamento ao entrar.
 *  - Carisma loss por Poder: incremental "1 + floor((N-1)/2)".
 *  - NPC quando Car < -5 OU estágio = insano-npc.
 *  - 22 Poderes da Tormenta no catálogo.
 *  - Larva Explosiva requer Dentes Afiados.
 */

describe('Tormenta area — constants (book p319)', () => {
  it('entry condition is "frustrado" (auto, sem teste)', () => {
    expect(TORMENTA_AREA_ENTRY_CONDITION).toBe('frustrado')
  })

  it('save type is Vontade', () => {
    expect(TORMENTA_AREA_SAVE_TYPE).toBe('vontade')
  })

  it('base CD is 25', () => {
    expect(TORMENTA_AREA_BASE_CD).toBe(25)
  })

  it('CD escalates by 2 per consecutive prior day', () => {
    expect(TORMENTA_AREA_CD_PER_PRIOR_DAY).toBe(2)
  })

  it('PM cost surcharge is +2 inside the área', () => {
    expect(TORMENTA_AREA_PM_SURCHARGE).toBe(2)
  })

  it('rest recovery is halved', () => {
    expect(TORMENTA_AREA_REST_FACTOR).toBe(0.5)
  })

  it('items lose 1 encantamento on entry', () => {
    expect(TORMENTA_AREA_ENCHANT_LOSS).toBe(1)
  })
})

describe('tormentaAreaDailyCd — CD = 25 + 2 × prior days', () => {
  it('day 1 (no prior) → CD 25', () => {
    expect(tormentaAreaDailyCd(0)).toBe(25)
  })

  it('day 6 (5 prior) → CD 35', () => {
    expect(tormentaAreaDailyCd(5)).toBe(35)
  })

  it('day 11 (10 prior) → CD 45', () => {
    expect(tormentaAreaDailyCd(10)).toBe(45)
  })

  it('rejects negative input', () => {
    expect(() => tormentaAreaDailyCd(-1)).toThrow(
      /consecutivePriorDays must be ≥ 0/,
    )
  })
})

describe('escalateTormentaInsanity — condition ladder', () => {
  it('none → frustrado', () => {
    expect(escalateTormentaInsanity('none')).toBe('frustrado')
  })

  it('frustrado → esmorecido', () => {
    expect(escalateTormentaInsanity('frustrado')).toBe('esmorecido')
  })

  it('esmorecido → confuso', () => {
    expect(escalateTormentaInsanity('esmorecido')).toBe('confuso')
  })

  it('confuso → insano-npc (terminal)', () => {
    expect(escalateTormentaInsanity('confuso')).toBe('insano-npc')
  })

  it('insano-npc is sticky (already an NPC)', () => {
    expect(escalateTormentaInsanity('insano-npc')).toBe('insano-npc')
  })
})

describe('isTormentaNpc — NPC trigger', () => {
  it('stage insano-npc triggers NPC regardless of Carisma', () => {
    expect(
      isTormentaNpc({ insanityStage: 'insano-npc', carismaAfterLoss: 10 }),
    ).toBe(true)
  })

  it('Carisma below -5 triggers NPC even when sane', () => {
    expect(
      isTormentaNpc({ insanityStage: 'none', carismaAfterLoss: -6 }),
    ).toBe(true)
  })

  it('Carisma exactly -5 does NOT trigger (threshold is "menos que -5")', () => {
    expect(
      isTormentaNpc({ insanityStage: 'none', carismaAfterLoss: -5 }),
    ).toBe(false)
  })

  it('sane character above threshold remains a PC', () => {
    expect(
      isTormentaNpc({ insanityStage: 'frustrado', carismaAfterLoss: 0 }),
    ).toBe(false)
  })
})

describe('TORMENTA_POWERS — catalog', () => {
  it('catalog lists 22 powers (book p128 table)', () => {
    expect(TORMENTA_POWER_IDS.length).toBe(22)
  })

  it('every id has a non-empty name', () => {
    for (const id of TORMENTA_POWER_IDS) {
      expect(TORMENTA_POWERS[id].name).toBeTruthy()
    }
  })

  it('requiresOtherPowers is a non-negative integer', () => {
    for (const id of TORMENTA_POWER_IDS) {
      const n = TORMENTA_POWERS[id].requiresOtherPowers
      expect(n).toBeGreaterThanOrEqual(0)
      expect(Number.isInteger(n)).toBe(true)
    }
  })

  it('Larva Explosiva requires Dentes Afiados specifically', () => {
    expect(TORMENTA_POWERS['larva-explosiva'].requiresPower).toBe(
      'dentes-afiados',
    )
  })

  it('Legião Aberrante requires Anatomia Insana + 3 others', () => {
    expect(TORMENTA_POWERS['legiao-aberrante'].requiresPower).toBe(
      'anatomia-insana',
    )
    expect(TORMENTA_POWERS['legiao-aberrante'].requiresOtherPowers).toBe(3)
  })

  it('Asas Insetoides / Membros Extras / Desprezar a Realidade need 4 others', () => {
    expect(TORMENTA_POWERS['asas-insetoides'].requiresOtherPowers).toBe(4)
    expect(TORMENTA_POWERS['membros-extras'].requiresOtherPowers).toBe(4)
    expect(TORMENTA_POWERS['desprezar-a-realidade'].requiresOtherPowers).toBe(4)
  })
})

describe('canTakePower — prerequisite gating', () => {
  it('cannot retake a power already held', () => {
    expect(canTakePower('dentes-afiados', ['dentes-afiados'])).toBe(false)
  })

  it('Larva Explosiva needs Dentes Afiados; refuses without it', () => {
    expect(canTakePower('larva-explosiva', [])).toBe(false)
    expect(canTakePower('larva-explosiva', ['carapaca'])).toBe(false)
  })

  it('Larva Explosiva allowed once Dentes Afiados is held', () => {
    expect(canTakePower('larva-explosiva', ['dentes-afiados'])).toBe(true)
  })

  it('Legião Aberrante refuses without Anatomia Insana (even with 3+ others)', () => {
    expect(
      canTakePower('legiao-aberrante', [
        'carapaca',
        'dentes-afiados',
        'antenas',
        'olhos-vermelhos',
      ]),
    ).toBe(false)
  })

  it('Legião Aberrante refuses with Anatomia Insana but only 2 others', () => {
    expect(
      canTakePower('legiao-aberrante', ['anatomia-insana', 'carapaca', 'antenas']),
    ).toBe(false)
  })

  it('Legião Aberrante allowed with Anatomia Insana + 3 others', () => {
    expect(
      canTakePower('legiao-aberrante', [
        'anatomia-insana',
        'carapaca',
        'antenas',
        'olhos-vermelhos',
      ]),
    ).toBe(true)
  })

  it('Asas Insetoides requires 4 other powers', () => {
    expect(
      canTakePower('asas-insetoides', ['carapaca', 'dentes-afiados', 'antenas']),
    ).toBe(false)
    expect(
      canTakePower('asas-insetoides', [
        'carapaca',
        'dentes-afiados',
        'antenas',
        'olhos-vermelhos',
      ]),
    ).toBe(true)
  })

  it('zero-prereq power always allowed when not held', () => {
    expect(canTakePower('carapaca', [])).toBe(true)
  })
})

describe('carismaLossFromPowers — incremental cost', () => {
  it('0 powers → 0 Carisma loss', () => {
    expect(carismaLossFromPowers(0)).toBe(0)
  })

  it('1 power → 1 Carisma', () => {
    expect(carismaLossFromPowers(1)).toBe(1)
  })

  it('2 powers → 2 Carisma (no bonus tier yet)', () => {
    expect(carismaLossFromPowers(2)).toBe(2)
  })

  it('3 powers → 4 (3rd pick has 2 others, +1 bonus)', () => {
    expect(carismaLossFromPowers(3)).toBe(4)
  })

  it('4 powers → 6', () => {
    expect(carismaLossFromPowers(4)).toBe(6)
  })

  it('5 powers → 9 (5th pick adds 1 + floor(4/2) = 3)', () => {
    expect(carismaLossFromPowers(5)).toBe(9)
  })

  it('7 powers → 16', () => {
    expect(carismaLossFromPowers(7)).toBe(16)
  })

  it('rejects negative powerCount', () => {
    expect(() => carismaLossFromPowers(-1)).toThrow(/powerCount must be ≥ 0/)
  })
})

describe('NPC threshold — Carisma loss reaches -5', () => {
  it('Carisma 0 + 5 powers (loss 9) → Car -9 → NPC', () => {
    const carisma = 0 - carismaLossFromPowers(5)
    expect(isTormentaNpc({ insanityStage: 'none', carismaAfterLoss: carisma })).toBe(
      true,
    )
  })

  it('Carisma 4 + 5 powers (loss 9) → Car -5 → NOT yet NPC (strict <)', () => {
    const carisma = 4 - carismaLossFromPowers(5)
    expect(carisma).toBe(TORMENTA_NPC_CARISMA_THRESHOLD)
    expect(isTormentaNpc({ insanityStage: 'none', carismaAfterLoss: carisma })).toBe(
      false,
    )
  })
})

describe('LEFEU_INSANITY_SAMPLES — pinned lefeu CDs (book p313-316)', () => {
  it('uktril: ND 3, CD 17, 1d6 PM', () => {
    const lefeu = LEFEU_INSANITY_SAMPLES.uktril
    expect(lefeu.nd).toBe(3)
    expect(lefeu.saveCd).toBe(17)
    expect(lefeu.pmDrainDice).toBe('1d6')
  })

  it('thuwarokk: ND 16, CD 42, 2d12 PM (apex lefeu)', () => {
    const lefeu = LEFEU_INSANITY_SAMPLES.thuwarokk
    expect(lefeu.nd).toBe(16)
    expect(lefeu.saveCd).toBe(42)
    expect(lefeu.pmDrainDice).toBe('2d12')
  })

  it('CDs monotonically scale with ND across the 4 samples', () => {
    const ordered = ['uktril', 'geraktril', 'reishid', 'thuwarokk'] as const
    for (let i = 1; i < ordered.length; i++) {
      expect(LEFEU_INSANITY_SAMPLES[ordered[i]].saveCd).toBeGreaterThan(
        LEFEU_INSANITY_SAMPLES[ordered[i - 1]].saveCd,
      )
    }
  })
})

describe('LEFOU_TORMENTA_RESISTANCE_BONUS — Cria da Tormenta (book p23)', () => {
  it('lefou gain +5 em testes de resistência vs Tormenta', () => {
    expect(LEFOU_TORMENTA_RESISTANCE_BONUS).toBe(5)
  })
})
