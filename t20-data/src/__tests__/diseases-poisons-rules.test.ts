import { describe, expect, it } from 'vitest'
import {
  CURA_NO_MALETA_PENALTY,
  CURA_SELF_TREATMENT_PENALTY,
  CURA_TREATMENT_ACTION,
  CURA_TREATMENT_BONUS_NEXT_FORT,
  CURA_TREATMENT_TRAINED_ONLY,
  DISEASE_EXPOSICAO_CUMULATIVA,
  POISON_CONDITION_NAME,
  POISON_CURE_RESTORES_LOST_HP,
  POISON_SAVES_PER_EXPOSURE,
  POISON_SAVE_TYPE,
  POISON_WEAPON_APPLY_ACTION,
  POISON_WEAPON_APPLY_DIE,
  POISON_WEAPON_DURATION,
  POISON_WEAPON_SELF_CONTAM_ROLL,
  curaTreatmentPenalty,
  isDiseaseReexposicaoNoOp,
  isPoisonSelfContamination,
  poisonAppliesEnvenenadaCondition,
} from '../diseases-poisons-rules'

/**
 * PDF p117 (Cura Tratamento) + p161 (Venenos) + p318 (Doenças).
 */

describe('Cura Tratamento — constantes p117', () => {
  it('+5 no próximo Fort da vítima', () => {
    expect(CURA_TREATMENT_BONUS_NEXT_FORT).toBe(5)
  })

  it('ação completa', () => {
    expect(CURA_TREATMENT_ACTION).toBe('completa')
  })

  it('apenas treinado', () => {
    expect(CURA_TREATMENT_TRAINED_ONLY).toBe(true)
  })

  it('penalidade sem maleta = -5', () => {
    expect(CURA_NO_MALETA_PENALTY).toBe(-5)
  })

  it('penalidade auto-tratamento = -5', () => {
    expect(CURA_SELF_TREATMENT_PENALTY).toBe(-5)
  })
})

describe('curaTreatmentPenalty', () => {
  it('sem penalidades → 0', () => {
    expect(curaTreatmentPenalty({})).toBe(0)
  })

  it('sem maleta → -5', () => {
    expect(curaTreatmentPenalty({ withoutKit: true })).toBe(-5)
  })

  it('auto-tratamento → -5', () => {
    expect(curaTreatmentPenalty({ selfTreatment: true })).toBe(-5)
  })

  it('ambos cumulativos → -10', () => {
    expect(
      curaTreatmentPenalty({ withoutKit: true, selfTreatment: true }),
    ).toBe(-10)
  })
})

describe('Doenças — exposição não cumulativa p318', () => {
  it('flag = false', () => {
    expect(DISEASE_EXPOSICAO_CUMULATIVA).toBe(false)
  })

  it('re-exposição em vítima infectada é no-op', () => {
    expect(isDiseaseReexposicaoNoOp(true)).toBe(true)
  })

  it('exposição em vítima limpa não é no-op', () => {
    expect(isDiseaseReexposicaoNoOp(false)).toBe(false)
  })
})

describe('Venenos — aplicação em arma p161', () => {
  it('ação de movimento', () => {
    expect(POISON_WEAPON_APPLY_ACTION).toBe('movimento')
  })

  it('rolagem d6', () => {
    expect(POISON_WEAPON_APPLY_DIE).toBe('d6')
  })

  it('face 1 = autocontaminação', () => {
    expect(POISON_WEAPON_SELF_CONTAM_ROLL).toBe(1)
  })

  it('duração: até acerto OU fim de cena', () => {
    expect(POISON_WEAPON_DURATION).toBe('ate-acerto-ou-fim-cena')
  })
})

describe('isPoisonSelfContamination', () => {
  it('rolagem 1 → true', () => {
    expect(isPoisonSelfContamination(1)).toBe(true)
  })

  it.each([2, 3, 4, 5, 6])('rolagem %s → false', (roll) => {
    expect(isPoisonSelfContamination(roll)).toBe(false)
  })

  it('throws se rolagem fora de [1,6]', () => {
    expect(() => isPoisonSelfContamination(0)).toThrow(/d6Roll/)
    expect(() => isPoisonSelfContamination(7)).toThrow(/d6Roll/)
  })
})

describe('Venenos — save model p161', () => {
  it('save type: fortitude', () => {
    expect(POISON_SAVE_TYPE).toBe('fortitude')
  })

  it('1 save por exposição (sem re-roll por rodada)', () => {
    expect(POISON_SAVES_PER_EXPOSURE).toBe(1)
  })

  it('condição aplicada: envenenada', () => {
    expect(POISON_CONDITION_NAME).toBe('envenenada')
  })

  it('curar condição NÃO recupera PV perdidos', () => {
    expect(POISON_CURE_RESTORES_LOST_HP).toBe(false)
  })
})

describe('poisonAppliesEnvenenadaCondition — p161', () => {
  it('efeito instantâneo → NÃO aplica condição', () => {
    expect(poisonAppliesEnvenenadaCondition('instantaneo')).toBe(false)
  })

  it('efeito recorrente (PV/rodada) → aplica', () => {
    expect(poisonAppliesEnvenenadaCondition('recorrente')).toBe(true)
  })

  it('efeito de condição → aplica', () => {
    expect(poisonAppliesEnvenenadaCondition('condicao')).toBe(true)
  })
})
