import { describe, expect, it } from 'vitest'
import {
  COMBAT_ACTIONS,
  combatActionById,
  combatActionsByKind,
} from '../combat-actions'
import {
  INVESTIDA_ATTACK_BONUS,
  INVESTIDA_DEFENSE_PENALTY,
  isLegalTurn,
  type ActionKind,
} from '../combat'

/**
 * Catálogo de Ações de Combate — PDF Cap 5, p233-236.
 *
 * Pinned:
 *  - 20 named actions (6 padrão / 5 movimento / 4 completa / 4 livre /
 *    1 reação).
 *  - `kind` values match the existing `ActionKind` union in combat.ts.
 *  - `bookPage` lies in the action-economy range 233-236.
 *  - Investida entry text agrees with the INVESTIDA_* constants.
 *  - Investida + Corrida + Golpe de Misericórdia + Lançar Magia
 *    (execução maior) are the only `completa` entries (no Defesa
 *    Total or Atacar e Mover in T20).
 *  - No `provokesOpportunity` field — T20 has no generic AoO rule.
 */

const ALL_KINDS: readonly ActionKind[] = [
  'padrao',
  'movimento',
  'livre',
  'reacao',
  'completa',
]

describe('COMBAT_ACTIONS — shape & invariants', () => {
  it('catalog has exactly 20 entries', () => {
    expect(COMBAT_ACTIONS.length).toBe(20)
  })

  it('all ids unique', () => {
    const ids = COMBAT_ACTIONS.map((a) => a.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('all names unique', () => {
    const names = COMBAT_ACTIONS.map((a) => a.name)
    expect(new Set(names).size).toBe(names.length)
  })

  it('every kind is a known ActionKind', () => {
    for (const a of COMBAT_ACTIONS) {
      expect(ALL_KINDS).toContain(a.kind)
    }
  })

  it('every entry has non-empty effect and bookPage in 233-236', () => {
    for (const a of COMBAT_ACTIONS) {
      expect(a.effect.length).toBeGreaterThan(20)
      expect(a.bookPage).toBeGreaterThanOrEqual(233)
      expect(a.bookPage).toBeLessThanOrEqual(236)
    }
  })

  it('catalog is frozen', () => {
    expect(Object.isFrozen(COMBAT_ACTIONS)).toBe(true)
  })

  it('no entry exposes provokesOpportunity (T20 has no generic AoO)', () => {
    for (const a of COMBAT_ACTIONS) {
      expect(a).not.toHaveProperty('provokesOpportunity')
    }
  })
})

describe('COMBAT_ACTIONS — kind distribution per PDF', () => {
  const counts = (kind: ActionKind) =>
    COMBAT_ACTIONS.filter((a) => a.kind === kind).length

  it('6 ações padrão', () => {
    expect(counts('padrao')).toBe(6)
  })

  it('5 ações de movimento', () => {
    expect(counts('movimento')).toBe(5)
  })

  it('4 ações completas', () => {
    expect(counts('completa')).toBe(4)
  })

  it('4 ações livres', () => {
    expect(counts('livre')).toBe(4)
  })

  it('1 ação de reação', () => {
    expect(counts('reacao')).toBe(1)
  })

  it('counts sum to 20', () => {
    let total = 0
    for (const k of ALL_KINDS) total += counts(k)
    expect(total).toBe(20)
  })
})

describe('COMBAT_ACTIONS — pinned canonical entries', () => {
  it('Agredir: padrão, references "alcance natural" + -5 disparo em CaC', () => {
    const a = combatActionById('agredir')!
    expect(a.kind).toBe('padrao')
    expect(a.effect).toMatch(/alcance natural/)
    expect(a.effect).toMatch(/-5/)
  })

  it('Investida: completa, p235, +2 atk / -2 Def matches combat.ts constants', () => {
    const a = combatActionById('investida')!
    expect(a.kind).toBe('completa')
    expect(a.bookPage).toBe(235)
    expect(a.effect).toMatch(
      new RegExp(`\\+${INVESTIDA_ATTACK_BONUS} no teste de ataque`),
    )
    expect(a.effect).toMatch(
      new RegExp(`${INVESTIDA_DEFENSE_PENALTY} na Defesa`),
    )
  })

  it('Golpe de Misericórdia: completa, acerto crítico automático', () => {
    const a = combatActionById('golpe-de-misericordia')!
    expect(a.kind).toBe('completa')
    expect(a.effect).toMatch(/crítico automático/)
  })

  it('Preparar: padrão, ação preparada disparada como reação', () => {
    const a = combatActionById('preparar')!
    expect(a.kind).toBe('padrao')
    expect(a.effect).toMatch(/reação/)
  })

  it('Fintar: padrão, Enganação vs Reflexos, alvo desprevenido', () => {
    const a = combatActionById('fintar')!
    expect(a.effect).toMatch(/Enganação/)
    expect(a.effect).toMatch(/Reflexos/)
    expect(a.effect).toMatch(/desprevenida/)
  })

  it('Mirar: movimento, anula -5 de Pontaria em CaC', () => {
    const a = combatActionById('mirar')!
    expect(a.kind).toBe('movimento')
    expect(a.effect).toMatch(/Pontaria/)
  })

  it('Atrasar: livre, modifica ordem de Iniciativa', () => {
    const a = combatActionById('atrasar')!
    expect(a.kind).toBe('livre')
    expect(a.effect).toMatch(/Iniciativa/)
  })

  it('Falar: livre, limite de vinte palavras por rodada', () => {
    const a = combatActionById('falar')!
    expect(a.kind).toBe('livre')
    expect(a.effect).toMatch(/vinte palavras/)
  })

  it('Reação: kind=reacao, ocorre fora do turno', () => {
    const a = combatActionById('reacao-preparada')!
    expect(a.kind).toBe('reacao')
    expect(a.effect).toMatch(/fora do seu turno/)
  })
})

describe('COMBAT_ACTIONS — lookups', () => {
  it('combatActionById returns the matching entry', () => {
    expect(combatActionById('investida')?.name).toBe('Investida')
  })

  it('combatActionById returns undefined for unknown id', () => {
    expect(combatActionById('teleporte')).toBeUndefined()
  })

  it('combatActionsByKind("completa") returns the 4 full-round actions', () => {
    const ids = combatActionsByKind('completa')
      .map((a) => a.id)
      .sort()
    expect(ids).toEqual([
      'corrida',
      'golpe-de-misericordia',
      'investida',
      'lancar-uma-magia-completa',
    ])
  })

  it('combatActionsByKind("padrao") returns 6 entries including Agredir', () => {
    const padroes = combatActionsByKind('padrao')
    expect(padroes.length).toBe(6)
    expect(padroes.map((a) => a.id)).toContain('agredir')
  })

  it('combatActionsByKind("reacao") returns exactly one entry', () => {
    expect(combatActionsByKind('reacao').length).toBe(1)
  })
})

describe('COMBAT_ACTIONS — action-economy plausibility vs isLegalTurn', () => {
  it('one padrão + one movimento is a legal turn (Agredir + Movimentar-se)', () => {
    expect(
      isLegalTurn({ padrao: 1, movimento: 1, completa: 0 }),
    ).toBe(true)
  })

  it('one completa (Investida / Corrida) consumes the whole turn', () => {
    expect(
      isLegalTurn({ padrao: 0, movimento: 0, completa: 1 }),
    ).toBe(true)
    expect(
      isLegalTurn({ padrao: 1, movimento: 0, completa: 1 }),
    ).toBe(false)
  })

  it('two movimentos (trading padrão for mov) is legal', () => {
    expect(
      isLegalTurn({ padrao: 0, movimento: 2, completa: 0 }),
    ).toBe(true)
  })
})
