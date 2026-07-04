import { describe, expect, it } from 'vitest'
import {
  CATASTROPHIC_FAILURE_MARGIN,
  CIRCUMSTANCE_MIN_MAGNITUDE,
  EASY_TASK_AUTO_PASS_ON_GM_DISCRETION,
  EXTENDED_AID_CONSUMES_SKILL,
  EXTENDED_INTERRUPT_GM_DECIDES,
  EXTENDED_OPEN_REQUIRES_DIFFERENT_SKILL_PER_TEST,
  RETRIES_ALLOWED_BY_DEFAULT,
  applyCircumstance,
  attributeTestResult,
  extendedAidSkillAllowed,
  extendedGroupRoundTally,
  extendedOpenSkillAllowed,
  hybridTestOutcome,
  isCatastrophicFailure,
  multiSkillExtendedComplete,
  skillTestResult,
} from '../test-resolution'

/**
 * Extras do framework de testes de perícia — PDF p220-223.
 * Cobertura de gaps identificados em audit:
 *   - fórmulas 1d20+atributo / 1d20+perícia
 *   - auto-pass Fácil (mestre)
 *   - testes mistos (comum + oposto)
 *   - condições favoráveis/desfavoráveis ±2+
 *   - novas tentativas + falha catastrófica por 5+
 *   - estendidos: multi-perícia / abertos / grupo / ajudar consome
 */

describe('attributeTestResult / skillTestResult — p220 fórmulas', () => {
  it('teste de atributo = 1d20 + atributo', () => {
    expect(attributeTestResult(15, 3)).toBe(18)
    expect(attributeTestResult(1, 0)).toBe(1)
    expect(attributeTestResult(20, -1)).toBe(19)
  })

  it('teste de perícia = 1d20 + valor de perícia', () => {
    expect(skillTestResult(12, 8)).toBe(20)
    expect(skillTestResult(20, 10)).toBe(30)
  })

  it('lança em d20 fora do range 1-20', () => {
    expect(() => attributeTestResult(0, 3)).toThrow(/1-20/)
    expect(() => attributeTestResult(21, 3)).toThrow(/1-20/)
    expect(() => skillTestResult(0, 3)).toThrow(/1-20/)
  })
})

describe('EASY_TASK_AUTO_PASS_ON_GM_DISCRETION — p220 rodapé Tabela 5-1', () => {
  it('true (mestre pode conceder auto-pass em tarefas fáceis)', () => {
    expect(EASY_TASK_AUTO_PASS_ON_GM_DISCRETION).toBe(true)
  })
})

describe('hybridTestOutcome — p220 misturando comum + oposto', () => {
  it('todos passam: winner é o maior', () => {
    const out = hybridTestOutcome([15, 18, 22, 25], 10)
    expect(out.passed).toEqual([0, 1, 2, 3])
    expect(out.winnerIndex).toBe(3)
  })

  it('alguns passam: winner é o maior dentre aprovados', () => {
    const out = hybridTestOutcome([8, 12, 25, 4], 10)
    expect(out.passed).toEqual([1, 2])
    expect(out.winnerIndex).toBe(2)
  })

  it('ninguém passa: winnerIndex null', () => {
    const out = hybridTestOutcome([5, 8, 3], 10)
    expect(out.passed).toEqual([])
    expect(out.winnerIndex).toBeNull()
  })

  it('roll exatamente igual à CD passa', () => {
    const out = hybridTestOutcome([10, 9, 11], 10)
    expect(out.passed).toEqual([0, 2])
  })
})

describe('CIRCUMSTANCE_MIN_MAGNITUDE / applyCircumstance — p221', () => {
  it('magnitude mínima 2', () => {
    expect(CIRCUMSTANCE_MIN_MAGNITUDE).toBe(2)
  })

  it('favorável soma', () => {
    expect(applyCircumstance(15, 'favorable', 2)).toBe(17)
    expect(applyCircumstance(15, 'favorable', 5)).toBe(20)
  })

  it('desfavorável subtrai', () => {
    expect(applyCircumstance(15, 'unfavorable', 2)).toBe(13)
    expect(applyCircumstance(15, 'unfavorable', 4)).toBe(11)
  })

  it('lança se magnitude < 2', () => {
    expect(() => applyCircumstance(15, 'favorable', 1)).toThrow(/must be ≥ 2/)
    expect(() => applyCircumstance(15, 'unfavorable', 0)).toThrow(/must be ≥ 2/)
  })
})

describe('RETRIES_ALLOWED_BY_DEFAULT — p221', () => {
  it('true (default) — refazer testes é permitido por padrão', () => {
    expect(RETRIES_ALLOWED_BY_DEFAULT).toBe(true)
  })
})

describe('isCatastrophicFailure — p221 falha por 5+', () => {
  it('margem exatamente 5 = catastrófica', () => {
    expect(isCatastrophicFailure(10, 15)).toBe(true)
    expect(CATASTROPHIC_FAILURE_MARGIN).toBe(5)
  })

  it('margem 4 = falha simples (não catastrófica)', () => {
    expect(isCatastrophicFailure(11, 15)).toBe(false)
  })

  it('sucesso nunca é catastrófico', () => {
    expect(isCatastrophicFailure(20, 15)).toBe(false)
    expect(isCatastrophicFailure(15, 15)).toBe(false)
  })

  it('margem muito grande = catastrófica', () => {
    expect(isCatastrophicFailure(3, 20)).toBe(true)
  })
})

describe('Estendidos: extensões p223', () => {
  it('EXTENDED_OPEN_REQUIRES_DIFFERENT_SKILL_PER_TEST = true', () => {
    expect(EXTENDED_OPEN_REQUIRES_DIFFERENT_SKILL_PER_TEST).toBe(true)
  })

  it('EXTENDED_AID_CONSUMES_SKILL = true', () => {
    expect(EXTENDED_AID_CONSUMES_SKILL).toBe(true)
  })

  it('EXTENDED_INTERRUPT_GM_DECIDES = true', () => {
    expect(EXTENDED_INTERRUPT_GM_DECIDES).toBe(true)
  })
})

describe('extendedOpenSkillAllowed — p223', () => {
  it('permite perícia ainda não usada', () => {
    expect(extendedOpenSkillAllowed('enganacao', ['diplomacia', 'nobreza'])).toBe(true)
  })

  it('bloqueia perícia já usada', () => {
    expect(extendedOpenSkillAllowed('enganacao', ['enganacao'])).toBe(false)
  })

  it('lista vazia permite qualquer', () => {
    expect(extendedOpenSkillAllowed('atletismo', [])).toBe(true)
  })
})

describe('extendedAidSkillAllowed — p223', () => {
  it('permite perícia se ainda não consumida', () => {
    expect(extendedAidSkillAllowed('intuicao', ['nobreza'])).toBe(true)
  })

  it('bloqueia perícia já consumida (por ajuda ou teste)', () => {
    expect(extendedAidSkillAllowed('intuicao', ['intuicao', 'nobreza'])).toBe(false)
  })
})

describe('extendedGroupRoundTally — p223 grupo', () => {
  it('soma sucessos e falhas do pool', () => {
    const tally = extendedGroupRoundTally([
      'success',
      'failure',
      'success',
      'success',
      'failure',
    ])
    expect(tally.successes).toBe(3)
    expect(tally.failures).toBe(2)
  })

  it('rodada vazia = zeros', () => {
    const tally = extendedGroupRoundTally([])
    expect(tally.successes).toBe(0)
    expect(tally.failures).toBe(0)
  })
})

describe('multiSkillExtendedComplete — p223 multi-perícia', () => {
  it('completo quando cada perícia atinge o mínimo', () => {
    const req = { atletismo: 1, furtividade: 2 }
    expect(multiSkillExtendedComplete(req, { atletismo: 1, furtividade: 2 })).toBe(
      true,
    )
    expect(multiSkillExtendedComplete(req, { atletismo: 5, furtividade: 3 })).toBe(
      true,
    )
  })

  it('incompleto quando qualquer perícia está abaixo do mínimo', () => {
    const req = { nobreza: 2, diplomacia: 3 }
    expect(multiSkillExtendedComplete(req, { nobreza: 2, diplomacia: 2 })).toBe(
      false,
    )
    expect(multiSkillExtendedComplete(req, { nobreza: 1, diplomacia: 3 })).toBe(
      false,
    )
  })

  it('perícia ausente conta como 0', () => {
    const req = { atletismo: 1 }
    expect(multiSkillExtendedComplete(req, {})).toBe(false)
    expect(multiSkillExtendedComplete(req, { furtividade: 5 })).toBe(false)
  })
})
