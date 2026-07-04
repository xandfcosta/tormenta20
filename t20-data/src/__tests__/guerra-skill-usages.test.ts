import { describe, expect, it } from 'vitest'
import {
  ANALISAR_TERRENO_CD,
  GUERRA_ARMOR_PENALTY,
  GUERRA_TRAINED_ONLY,
  GUERRA_USAGES,
  PLANO_DE_ACAO_CD,
  PLANO_DE_ACAO_INICIATIVA_BONUS,
  analisarTerrenoCd,
  guerraUsageByKind,
  planoDeAcaoCd,
  planoDeAcaoIniciativaBonus,
  planoDeAcaoRoundOrder,
} from '../guerra-skill-usages'

/**
 * PDF livro p119 — Perícia Guerra (INT, treinada).
 *  1. Analisar Terreno — CD 20, ação de movimento, revela vantagem
 *  2. Plano de Ação — CD 20, ação padrão, +5 Iniciativa, reordenação condicional
 */

describe('GUERRA_USAGES — shape', () => {
  it('exatamente 2 usos', () => {
    expect(GUERRA_USAGES.length).toBe(2)
  })

  it('frozen', () => {
    expect(Object.isFrozen(GUERRA_USAGES)).toBe(true)
  })

  it('ids canônicos', () => {
    expect(GUERRA_USAGES.map((u) => u.id).sort()).toEqual([
      'analisar-terreno',
      'plano-de-acao',
    ])
  })

  it('todos em p119', () => {
    for (const u of GUERRA_USAGES) {
      expect(u.bookPage).toBe(119)
    }
  })
})

describe('Perícia flags — Tabela 2-1', () => {
  it('apenas treinada', () => {
    expect(GUERRA_TRAINED_ONLY).toBe(true)
  })

  it('sem penalidade de armadura', () => {
    expect(GUERRA_ARMOR_PENALTY).toBe(false)
  })
})

describe('Analisar Terreno — p119', () => {
  const usage = guerraUsageByKind('analisar-terreno')

  it('CD 20 ação de movimento', () => {
    expect(ANALISAR_TERRENO_CD).toBe(20)
    if (usage.kind !== 'analisar-terreno') throw new Error('narrow failed')
    expect(usage.dc).toBe(20)
    expect(usage.action).toBe('movimento')
  })

  it('só revela vantagem existente', () => {
    if (usage.kind !== 'analisar-terreno') throw new Error('narrow failed')
    expect(usage.revealsAdvantage).toBe(true)
  })
})

describe('analisarTerrenoCd', () => {
  it('sempre 20', () => {
    expect(analisarTerrenoCd()).toBe(20)
  })
})

describe('Plano de Ação — p119', () => {
  const usage = guerraUsageByKind('plano-de-acao')

  it('CD 20 ação padrão', () => {
    expect(PLANO_DE_ACAO_CD).toBe(20)
    if (usage.kind !== 'plano-de-acao') throw new Error('narrow failed')
    expect(usage.dc).toBe(20)
    expect(usage.action).toBe('padrao')
  })

  it('alvo em alcance médio', () => {
    if (usage.kind !== 'plano-de-acao') throw new Error('narrow failed')
    expect(usage.targetRange).toBe('medio')
  })

  it('+5 na Iniciativa', () => {
    expect(PLANO_DE_ACAO_INICIATIVA_BONUS).toBe(5)
    if (usage.kind !== 'plano-de-acao') throw new Error('narrow failed')
    expect(usage.iniciativaBonus).toBe(5)
  })
})

describe('planoDeAcaoCd / planoDeAcaoIniciativaBonus', () => {
  it('CD 20 e bônus 5', () => {
    expect(planoDeAcaoCd()).toBe(20)
    expect(planoDeAcaoIniciativaBonus()).toBe(5)
  })
})

describe('planoDeAcaoRoundOrder', () => {
  it('nova > caller e não agiu → acts-immediately-after-caster', () => {
    // caller 15, aliado com 12 base → 17 novo, ainda não agiu.
    expect(planoDeAcaoRoundOrder(15, 17, false)).toBe(
      'acts-immediately-after-caster',
    )
  })

  it('nova > caller mas já agiu → new-order-next-round', () => {
    expect(planoDeAcaoRoundOrder(15, 17, true)).toBe('new-order-next-round')
  })

  it('nova = caller (empate) e não agiu → new-order-next-round (usa "estritamente maior")', () => {
    expect(planoDeAcaoRoundOrder(15, 15, false)).toBe('new-order-next-round')
  })

  it('nova < caller → new-order-next-round', () => {
    expect(planoDeAcaoRoundOrder(20, 18, false)).toBe('new-order-next-round')
  })
})

describe('guerraUsageByKind', () => {
  it('throws se kind inválido', () => {
    expect(() =>
      // @ts-expect-error — invalid kind on purpose
      guerraUsageByKind('inspirar-tropas'),
    ).toThrow(/unknown kind/)
  })

  it('resolve todos', () => {
    for (const k of ['analisar-terreno', 'plano-de-acao'] as const) {
      expect(guerraUsageByKind(k).kind).toBe(k)
    }
  })
})
