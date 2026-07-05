import { describe, expect, it } from 'vitest'
import {
  MAX_CONCURRENT_SUSTAINED_SPELLS,
  OWNER_END_ACTION,
  OWNER_END_REFUNDS_PM,
  SUSTAIN_COST_PM_PER_TURN,
  areaContinuesAfterOwnerDeath,
  canSustainAnotherSpell,
  endsOnOwnerDeath,
  instantaneousConsequencesPersist,
  isAffectedByArea,
  ownerEndAbility,
  paySustainCost,
  resolveDescarregar,
} from '../ability-lifecycle'

/**
 * PDF livro p227 — Ability lifecycle.
 */

describe('Constantes', () => {
  it('sustentar custa 1 PM/turno', () => {
    expect(SUSTAIN_COST_PM_PER_TURN).toBe(1)
  })

  it('apenas 1 MAGIA sustentada por vez', () => {
    expect(MAX_CONCURRENT_SUSTAINED_SPELLS).toBe(1)
  })

  it('encerrar não devolve PM', () => {
    expect(OWNER_END_REFUNDS_PM).toBe(false)
  })

  it('encerrar é ação livre', () => {
    expect(OWNER_END_ACTION).toBe('livre')
  })
})

describe('endsOnOwnerDeath', () => {
  it('sustentada → termina na morte', () => {
    expect(endsOnOwnerDeath('sustentada')).toBe(true)
  })

  it('outras durações → continuam', () => {
    expect(endsOnOwnerDeath('instantanea')).toBe(false)
    expect(endsOnOwnerDeath('cena')).toBe(false)
    expect(endsOnOwnerDeath('definida')).toBe(false)
    expect(endsOnOwnerDeath('permanente')).toBe(false)
    expect(endsOnOwnerDeath('descarregar')).toBe(false)
  })
})

describe('areaContinuesAfterOwnerDeath', () => {
  it('sustentada + instantânea → área não continua', () => {
    expect(areaContinuesAfterOwnerDeath('sustentada')).toBe(false)
    expect(areaContinuesAfterOwnerDeath('instantanea')).toBe(false)
  })

  it('outras → área persiste', () => {
    expect(areaContinuesAfterOwnerDeath('cena')).toBe(true)
    expect(areaContinuesAfterOwnerDeath('definida')).toBe(true)
    expect(areaContinuesAfterOwnerDeath('permanente')).toBe(true)
    expect(areaContinuesAfterOwnerDeath('descarregar')).toBe(true)
  })
})

describe('paySustainCost', () => {
  it('PM suficiente → todas mantidas', () => {
    const r = paySustainCost(5, ['a', 'b', 'c'])
    expect(r.pmAfter).toBe(2)
    expect(r.kept).toEqual(['a', 'b', 'c'])
    expect(r.endedForFailure).toEqual([])
  })

  it('PM insuficiente → primeiras mantidas na ordem', () => {
    const r = paySustainCost(2, ['a', 'b', 'c', 'd'])
    expect(r.pmAfter).toBe(0)
    expect(r.kept).toEqual(['a', 'b'])
    expect(r.endedForFailure).toEqual(['c', 'd'])
  })

  it('PM zero → todas terminam', () => {
    const r = paySustainCost(0, ['a', 'b'])
    expect(r.pmAfter).toBe(0)
    expect(r.kept).toEqual([])
    expect(r.endedForFailure).toEqual(['a', 'b'])
  })

  it('sem habilidades sustentadas → sem mudança', () => {
    const r = paySustainCost(3, [])
    expect(r.pmAfter).toBe(3)
    expect(r.kept).toEqual([])
    expect(r.endedForFailure).toEqual([])
  })

  it('PM negativo lança', () => {
    expect(() => paySustainCost(-1, [])).toThrow(/currentPm must be ≥ 0/)
  })
})

describe('canSustainAnotherSpell', () => {
  it('0 magias ativas → pode', () => {
    expect(canSustainAnotherSpell(0)).toBe(true)
  })

  it('1 magia ativa → não pode', () => {
    expect(canSustainAnotherSpell(1)).toBe(false)
  })
})

describe('resolveDescarregar', () => {
  it('trigger fired → discharged', () => {
    expect(resolveDescarregar(true, false)).toBe('discharged')
  })

  it('trigger + duration → discharged (trigger vence)', () => {
    expect(resolveDescarregar(true, true)).toBe('discharged')
  })

  it('duração expirou sem trigger → expired-without-trigger', () => {
    expect(resolveDescarregar(false, true)).toBe('expired-without-trigger')
  })

  it('sem trigger nem expiração → dormant', () => {
    expect(resolveDescarregar(false, false)).toBe('dormant')
  })
})

describe('ownerEndAbility', () => {
  it('causa e ausência de reembolso', () => {
    const r = ownerEndAbility()
    expect(r.cause).toBe('owner-ended')
    expect(r.refundsPm).toBe(false)
  })
})

describe('isAffectedByArea', () => {
  it('dentro + ativo → afetado', () => {
    expect(isAffectedByArea(true, true)).toBe(true)
  })

  it('dentro mas efeito acabou → não afetado', () => {
    expect(isAffectedByArea(true, false)).toBe(false)
  })

  it('saiu da área → não afetado', () => {
    expect(isAffectedByArea(false, true)).toBe(false)
  })
})

describe('instantaneousConsequencesPersist', () => {
  it('instantânea → consequências persistem (ex.: cura)', () => {
    expect(instantaneousConsequencesPersist('instantanea')).toBe(true)
  })

  it('outras durações → o próprio efeito é o que persiste', () => {
    expect(instantaneousConsequencesPersist('cena')).toBe(false)
    expect(instantaneousConsequencesPersist('sustentada')).toBe(false)
  })
})
