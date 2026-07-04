import { describe, expect, it } from 'vitest'
import {
  AGARRAR_DRAG_SPEED_MULTIPLIER,
  AGARRAR_RENEW_DEALS_UNARMED_DAMAGE,
  DERRUBAR_FLIGHT_FALL_DICE,
  DERRUBAR_LEDGE_REFLEXOS_CD,
  MANEUVER_DEFINITION,
  MANEUVER_RANGED_ALLOWED,
  MANEUVER_TIEBREAK_RULE,
  MANEUVERS,
  QUEBRAR_MOVING_OBJECT_DEFENSE_BONUS,
  RANGED_AT_GRABBED_TARGET_MISS_CHANCE_PERCENT,
  empurrarDistanceMeters,
  quebrarObjectEffectiveDefense,
  rangedAttackAtGrabbedTargetHits,
} from '../maneuvers'

/**
 * Cobertura expandida de manobras (PDF p234 + p239 + p228-229).
 * Camada verbatim (successEffect / fiveOverEffect / escapeRules /
 * weaponRequirement / specialInteractions) + regras globais + helpers.
 */

describe('MANEUVER_DEFINITION — verbatim p234', () => {
  it('menciona "ataque corpo a corpo" e restrição de ataques à distância', () => {
    expect(MANEUVER_DEFINITION).toMatch(/ataque corpo a corpo/)
    expect(MANEUVER_DEFINITION).toMatch(/ataques à distância/)
  })
})

describe('MANEUVER_TIEBREAK_RULE — verbatim p234', () => {
  it('empate = maior bônus vence; empate de bônus = refazer teste', () => {
    expect(MANEUVER_TIEBREAK_RULE).toMatch(/maior bônus vence/)
    expect(MANEUVER_TIEBREAK_RULE).toMatch(/outro teste/)
  })
})

describe('MANEUVER_RANGED_ALLOWED — false por regra geral', () => {
  it('manobras não podem ser feitas com ataques à distância', () => {
    expect(MANEUVER_RANGED_ALLOWED).toBe(false)
  })
})

describe('RANGED_AT_GRABBED_TARGET_MISS_CHANCE_PERCENT — p234 sidebar', () => {
  it('50% de chance de errar alvo agarrado', () => {
    expect(RANGED_AT_GRABBED_TARGET_MISS_CHANCE_PERCENT).toBe(50)
  })
})

describe('Agarrar — mecânica verbatim', () => {
  const agarrar = MANEUVERS.agarrar

  it('successEffect: desprevenida + imóvel + -2 ataque + só armas leves', () => {
    expect(agarrar.successEffect).toMatch(/desprevenida/)
    expect(agarrar.successEffect).toMatch(/imóvel/)
    expect(agarrar.successEffect).toMatch(/-2/)
    expect(agarrar.successEffect).toMatch(/armas leves/)
  })

  it('escapeRules: ação padrão + teste de manobra oposto', () => {
    expect(agarrar.escapeRules).toMatch(/ação padrão/)
    expect(agarrar.escapeRules).toMatch(/oposto/)
  })

  it('weaponRequirement: ataque desarmado ou arma natural + mão ocupada', () => {
    expect(agarrar.weaponRequirement).toMatch(/desarmado|arma natural/)
    expect(agarrar.weaponRequirement).toMatch(/ocupada/)
  })

  it('specialInteractions: arrastar metade + soltar livre + esmagar + 50% erro à distância', () => {
    expect(agarrar.specialInteractions).toMatch(/metade/)
    expect(agarrar.specialInteractions).toMatch(/soltá-la/)
    expect(agarrar.specialInteractions).toMatch(/esmagar|sufocar/)
    expect(agarrar.specialInteractions).toMatch(/50%/)
  })
})

describe('Derrubar — mecânica verbatim', () => {
  const derrubar = MANEUVERS.derrubar

  it('successEffect: alvo caído, sem dano por queda', () => {
    expect(derrubar.successEffect).toMatch(/caído/)
    expect(derrubar.successEffect).toMatch(/sem|não causa/)
  })

  it('fiveOverEffect: empurra um quadrado adicional', () => {
    expect(derrubar.fiveOverEffect).toMatch(/quadrado/)
    expect(derrubar.fiveOverEffect).toMatch(/5/)
  })

  it('specialInteractions: beirada + cair de voo', () => {
    expect(derrubar.specialInteractions).toMatch(/beirada|parapeito|precipício/)
    expect(derrubar.specialInteractions).toMatch(/voo/)
  })
})

describe('Desarmar — mecânica verbatim', () => {
  const desarmar = MANEUVERS.desarmar

  it('fiveOverEffect: derruba item + empurra um quadrado', () => {
    expect(desarmar.fiveOverEffect).toMatch(/quadrado/)
    expect(desarmar.fiveOverEffect).toMatch(/5/)
  })
})

describe('Empurrar — escala contínua', () => {
  const empurrar = MANEUVERS.empurrar

  it('successEffect: 1,5m base + 1,5m por 5 de margem', () => {
    expect(empurrar.successEffect).toMatch(/1,5m/)
    expect(empurrar.successEffect).toMatch(/5/)
  })

  it('specialInteractions: avançar junto com ação de movimento', () => {
    expect(empurrar.specialInteractions).toMatch(/movimento/)
    expect(empurrar.specialInteractions).toMatch(/avançar/)
  })

  it('fiveOverBonus flag = false (escala contínua)', () => {
    expect(empurrar.fiveOverEffect).toBeNull()
    expect(empurrar.hasFiveOverBonus).toBe(false)
  })
})

describe('Quebrar — objeto (p239 cross-ref)', () => {
  const quebrar = MANEUVERS.quebrar

  it('specialInteractions: Defesa do objeto + +5 em movimento', () => {
    expect(quebrar.specialInteractions).toMatch(/Defesa/)
    expect(quebrar.specialInteractions).toMatch(/\+5/)
    expect(quebrar.specialInteractions).toMatch(/movimento/)
  })
})

describe('DERRUBAR_LEDGE_REFLEXOS_CD — p234', () => {
  it('CD 20 para se agarrar em beirada', () => {
    expect(DERRUBAR_LEDGE_REFLEXOS_CD).toBe(20)
  })
})

describe('DERRUBAR_FLIGHT_FALL_DICE — p229 cross-ref', () => {
  it('1d6 x 1,5m', () => {
    expect(DERRUBAR_FLIGHT_FALL_DICE).toBe('1d6 x 1,5m')
  })
})

describe('AGARRAR_DRAG_SPEED_MULTIPLIER — p234', () => {
  it('metade do deslocamento', () => {
    expect(AGARRAR_DRAG_SPEED_MULTIPLIER).toBe(0.5)
  })
})

describe('AGARRAR_RENEW_DEALS_UNARMED_DAMAGE — p234', () => {
  it('true (renovar agarrar causa dano de impacto = desarmado/natural)', () => {
    expect(AGARRAR_RENEW_DEALS_UNARMED_DAMAGE).toBe(true)
  })
})

describe('QUEBRAR_MOVING_OBJECT_DEFENSE_BONUS — p239', () => {
  it('+5 na Defesa do objeto em movimento', () => {
    expect(QUEBRAR_MOVING_OBJECT_DEFENSE_BONUS).toBe(5)
  })
})

describe('empurrarDistanceMeters — escala 1,5m + 1,5m por 5 margem', () => {
  it('margin 0 → 1,5m', () => {
    expect(empurrarDistanceMeters(0)).toBe(1.5)
  })

  it('margin 4 → 1,5m (aquém do próximo breakpoint)', () => {
    expect(empurrarDistanceMeters(4)).toBe(1.5)
  })

  it('margin 5 → 3m', () => {
    expect(empurrarDistanceMeters(5)).toBe(3)
  })

  it('margin 10 → 4,5m', () => {
    expect(empurrarDistanceMeters(10)).toBeCloseTo(4.5)
  })

  it('margin 15 → 6m', () => {
    expect(empurrarDistanceMeters(15)).toBe(6)
  })

  it('lança em margin negativa (falha na manobra)', () => {
    expect(() => empurrarDistanceMeters(-1)).toThrow(/must be >= 0/)
  })
})

describe('quebrarObjectEffectiveDefense — +5 em movimento', () => {
  it('parado retorna base', () => {
    expect(quebrarObjectEffectiveDefense(10, false)).toBe(10)
  })

  it('em movimento retorna base + 5', () => {
    expect(quebrarObjectEffectiveDefense(10, true)).toBe(15)
  })
})

describe('rangedAttackAtGrabbedTargetHits — 50/50 por 1d100', () => {
  it('roll ≤ 50 = erra', () => {
    expect(rangedAttackAtGrabbedTargetHits(50)).toBe(false)
    expect(rangedAttackAtGrabbedTargetHits(1)).toBe(false)
  })

  it('roll > 50 = acerta', () => {
    expect(rangedAttackAtGrabbedTargetHits(51)).toBe(true)
    expect(rangedAttackAtGrabbedTargetHits(100)).toBe(true)
  })

  it('lança em rolagem inválida', () => {
    expect(() => rangedAttackAtGrabbedTargetHits(0)).toThrow(/1-100/)
    expect(() => rangedAttackAtGrabbedTargetHits(101)).toThrow(/1-100/)
  })
})

describe('MANEUVERS — todas as 5 têm bookPage: 234', () => {
  it('todas apontam para p234', () => {
    for (const id of ['agarrar', 'derrubar', 'desarmar', 'empurrar', 'quebrar'] as const) {
      expect(MANEUVERS[id].bookPage).toBe(234)
    }
  })
})

describe('MANEUVERS — fiveOverEffect null coerente com hasFiveOverBonus', () => {
  it('Agarrar/Empurrar/Quebrar: fiveOverEffect null e hasFiveOverBonus false', () => {
    for (const id of ['agarrar', 'empurrar', 'quebrar'] as const) {
      expect(MANEUVERS[id].fiveOverEffect).toBeNull()
      expect(MANEUVERS[id].hasFiveOverBonus).toBe(false)
    }
  })

  it('Derrubar/Desarmar: fiveOverEffect verbatim e hasFiveOverBonus true', () => {
    for (const id of ['derrubar', 'desarmar'] as const) {
      expect(MANEUVERS[id].fiveOverEffect).toBeTruthy()
      expect(MANEUVERS[id].hasFiveOverBonus).toBe(true)
    }
  })
})
