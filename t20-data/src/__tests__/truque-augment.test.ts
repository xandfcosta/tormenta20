import { describe, expect, it } from 'vitest'
import { SPELL_CATALOG } from '../spell-catalog'
import type { SpellAugment } from '../spells'
import {
  TRUQUE_TOTAL_PM,
  assertTruqueRestriction,
  castAsTruque,
  hasTruqueAugment,
  isTruqueAugment,
  spellsWithTruqueAugment,
  truqueAugmentOf,
} from '../truque-augment'

/**
 * PDF Cap 4 p171 verbatim: "Truque. Este aprimoramento transforma
 * a magia em uma versão mais simples e reduz seu custo em PM para
 * zero. Truques não podem ser usados em conjunto com outros
 * aprimoramentos."
 *
 * Pinned:
 *  - Truque = kind 'muda', pmCost 0, descrição começa "Truque:"
 *  - Cast como Truque: PM total = 0 (independente do círculo)
 *  - Não combina com outros aprimoramentos
 */

const TRUQUE_EXAMPLE: SpellAugment = {
  id: 'truque',
  kind: 'muda',
  pmCost: 0,
  description: 'Truque: muda alcance para curto.',
}

const NOT_TRUQUE_MUDA: SpellAugment = {
  id: 'foo',
  kind: 'muda',
  pmCost: 2,
  description: 'Muda alvo para self.',
}

const AUMENTA_ZERO_COST: SpellAugment = {
  id: 'bar',
  kind: 'aumenta',
  pmCost: 0,
  description: 'Adiciona 1d6 fogo.',
}

describe('isTruqueAugment', () => {
  it('matches augmento com kind muda + pmCost 0 + prefixo Truque:', () => {
    expect(isTruqueAugment(TRUQUE_EXAMPLE)).toBe(true)
  })

  it('rejeita muda com pmCost > 0', () => {
    expect(isTruqueAugment(NOT_TRUQUE_MUDA)).toBe(false)
  })

  it('rejeita aumenta mesmo com pmCost 0', () => {
    expect(isTruqueAugment(AUMENTA_ZERO_COST)).toBe(false)
  })

  it('rejeita descrição sem prefixo Truque:', () => {
    const other: SpellAugment = {
      id: 'x',
      kind: 'muda',
      pmCost: 0,
      description: 'Simplifica alvo.',
    }
    expect(isTruqueAugment(other)).toBe(false)
  })
})

describe('TRUQUE_TOTAL_PM', () => {
  it('é sempre 0', () => {
    expect(TRUQUE_TOTAL_PM).toBe(0)
  })
})

describe('castAsTruque', () => {
  it('círculo 1 lançado como Truque = 0 PM', () => {
    expect(castAsTruque(1)).toBe(0)
  })

  it('círculo 5 lançado como Truque = 0 PM (override total)', () => {
    expect(castAsTruque(5)).toBe(0)
  })
})

describe('assertTruqueRestriction', () => {
  it('sem aprimoramentos → OK', () => {
    expect(() => assertTruqueRestriction([])).not.toThrow()
  })

  it('só Truque → OK', () => {
    expect(() => assertTruqueRestriction([TRUQUE_EXAMPLE])).not.toThrow()
  })

  it('outros aprimoramentos sem Truque → OK', () => {
    expect(() =>
      assertTruqueRestriction([NOT_TRUQUE_MUDA, AUMENTA_ZERO_COST]),
    ).not.toThrow()
  })

  it('Truque + outros → throws', () => {
    expect(() =>
      assertTruqueRestriction([TRUQUE_EXAMPLE, NOT_TRUQUE_MUDA]),
    ).toThrow(/não pode combinar/)
  })

  it('2 Truques → throws (só 1 por lançamento)', () => {
    expect(() =>
      assertTruqueRestriction([TRUQUE_EXAMPLE, TRUQUE_EXAMPLE]),
    ).toThrow(/apenas 1 Truque/)
  })
})

describe('spellsWithTruqueAugment — catálogo real', () => {
  const ALL_SPELLS = Object.values(SPELL_CATALOG)
  const truqueSpells = spellsWithTruqueAugment(ALL_SPELLS)

  it('encontra pelo menos 3 magias com Truque no catálogo', () => {
    expect(truqueSpells.length).toBeGreaterThanOrEqual(3)
  })

  it('inclui caminhos-da-natureza', () => {
    const ids = truqueSpells.map((s) => s.id)
    expect(ids).toContain('caminhos-da-natureza')
  })

  it('inclui explosao-de-chamas', () => {
    const ids = truqueSpells.map((s) => s.id)
    expect(ids).toContain('explosao-de-chamas')
  })

  it('cada magia listada retorna truqueAugmentOf definido', () => {
    for (const spell of truqueSpells) {
      expect(truqueAugmentOf(spell)).toBeDefined()
    }
  })
})

describe('hasTruqueAugment — sanity', () => {
  it('magia sem entrada Truque → false', () => {
    const noTruque = Object.values(SPELL_CATALOG).find(
      (s) => !hasTruqueAugment(s),
    )
    expect(noTruque).toBeDefined()
  })
})
