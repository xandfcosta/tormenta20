import { describe, expect, it } from 'vitest'
import {
  ADAMANTE_ARMADURA_LEVE_RD,
  ADAMANTE_ARMADURA_PESADA_RD,
  CAVALEIRO_BASTIAO_RD,
  CAVALEIRO_ESPECIALIZACAO_RD,
  applicableRd,
  applyDamageReduction,
  applyDamageWithRdIgnore,
  barbaroRdForLevel,
  cavaleiroTotalRd,
  guerreiroRdForLevel,
  type DamageResistanceProfile,
} from '../damage-reduction'

/**
 * PDF Cap 5 p225-230 (RD core + order); p47 (Bárbaro), p54-57
 * (Cavaleiro), p133 (Adamante). Pinned:
 *  - RD 5 vs 8 dano = 3 (exemplo p229)
 *  - RD 10 vs 8 dano = 0 (sem regra de mín 1)
 *  - RD 10/mágico bypassado por tag 'magico'
 *  - Perda de Vida bypassa RD
 *  - Imunidade → 0 dano
 *  - Bárbaro 5º=2, 8º=4, 11º=6, 14º=8, 17º=10
 *  - Cavaleiro Bastião 5º + Especialização 12º = 10 (armor pesada)
 */

describe('applicableRd', () => {
  it('flat RD 5 vs qualquer tipo = 5', () => {
    const rd = applicableRd([{ kind: 'flat', amount: 5 }], 'corte')
    expect(rd).toBe(5)
  })

  it('flat RD 10/mágico bypassed com tag magico', () => {
    const rd = applicableRd(
      [{ kind: 'flat', amount: 10, bypass: ['magico'] }],
      'corte',
      ['magico'],
    )
    expect(rd).toBe(0)
  })

  it('flat RD 10/mágico NÃO bypassed sem tag', () => {
    const rd = applicableRd(
      [{ kind: 'flat', amount: 10, bypass: ['magico'] }],
      'corte',
      [],
    )
    expect(rd).toBe(10)
  })

  it('typed RD só reduz damage type matching', () => {
    const sources = [{ kind: 'typed' as const, amount: 10, damageType: 'fogo' }]
    expect(applicableRd(sources, 'fogo')).toBe(10)
    expect(applicableRd(sources, 'frio')).toBe(0)
  })

  it('múltiplas fontes empilham (soma)', () => {
    const sources = [
      { kind: 'flat' as const, amount: 5 },
      { kind: 'flat' as const, amount: 3 },
      { kind: 'typed' as const, amount: 5, damageType: 'fogo' },
    ]
    expect(applicableRd(sources, 'fogo')).toBe(13)
    expect(applicableRd(sources, 'corte')).toBe(8)
  })
})

describe('applyDamageReduction — exemplo p229', () => {
  const profile: DamageResistanceProfile = {
    rdSources: [{ kind: 'flat', amount: 5 }],
    immunities: [],
  }

  it('RD 5 vs 8 dano corte = 3', () => {
    expect(
      applyDamageReduction({ amount: 8, damageType: 'corte' }, profile),
    ).toBe(3)
  })
})

describe('applyDamageReduction — RD pode zerar hit', () => {
  it('RD 10 vs 8 dano = 0 (sem regra mínimo)', () => {
    const profile: DamageResistanceProfile = {
      rdSources: [{ kind: 'flat', amount: 10 }],
      immunities: [],
    }
    expect(
      applyDamageReduction({ amount: 8, damageType: 'corte' }, profile),
    ).toBe(0)
  })
})

describe('applyDamageReduction — perda de vida', () => {
  it('perda de vida bypassa RD (p228)', () => {
    const profile: DamageResistanceProfile = {
      rdSources: [{ kind: 'flat', amount: 20 }],
      immunities: [],
    }
    expect(
      applyDamageReduction(
        { amount: 15, damageType: 'trevas', isPerdaDeVida: true },
        profile,
      ),
    ).toBe(15)
  })
})

describe('applyDamageReduction — imunidade', () => {
  it('imune a fogo → 0 dano de fogo', () => {
    const profile: DamageResistanceProfile = {
      rdSources: [],
      immunities: ['fogo'],
    }
    expect(
      applyDamageReduction({ amount: 100, damageType: 'fogo' }, profile),
    ).toBe(0)
  })

  it('imune a fogo NÃO afeta frio', () => {
    const profile: DamageResistanceProfile = {
      rdSources: [],
      immunities: ['fogo'],
    }
    expect(
      applyDamageReduction({ amount: 10, damageType: 'frio' }, profile),
    ).toBe(10)
  })
})

describe('applyDamageReduction — RD 10/luz (Vampiro)', () => {
  const vampireProfile: DamageResistanceProfile = {
    rdSources: [{ kind: 'flat', amount: 10, bypass: ['luz'] }],
    immunities: [],
  }

  it('espada normal vs vampiro (RD 10/luz) → 5 dano de 15', () => {
    expect(
      applyDamageReduction(
        { amount: 15, damageType: 'corte' },
        vampireProfile,
      ),
    ).toBe(5)
  })

  it('magia de luz (tag luz) bypassa RD → 15 dano integral', () => {
    expect(
      applyDamageReduction(
        { amount: 15, damageType: 'luz', tags: ['luz'] },
        vampireProfile,
      ),
    ).toBe(15)
  })
})

describe('applyDamageWithRdIgnore — Guerreiro Romper Resistências', () => {
  const profile: DamageResistanceProfile = {
    rdSources: [{ kind: 'flat', amount: 15 }],
    immunities: [],
  }

  it('RD 15, ignora 10 → RD efetiva 5, dano 8 = 3', () => {
    expect(
      applyDamageWithRdIgnore({ amount: 8, damageType: 'corte' }, profile, 10),
    ).toBe(3)
  })

  it('RD 15, ignora Infinity (Ladino Encontrar Fraqueza) → dano integral', () => {
    expect(
      applyDamageWithRdIgnore(
        { amount: 8, damageType: 'corte' },
        profile,
        Infinity,
      ),
    ).toBe(8)
  })

  it('throws se rdIgnored negativo', () => {
    expect(() =>
      applyDamageWithRdIgnore({ amount: 8, damageType: 'corte' }, profile, -1),
    ).toThrow(/rdIgnored/)
  })
})

describe('barbaroRdForLevel — p47', () => {
  it('nível 1-4: RD 0', () => {
    expect(barbaroRdForLevel(1)).toBe(0)
    expect(barbaroRdForLevel(4)).toBe(0)
  })

  it('nível 5-7: RD 2', () => {
    expect(barbaroRdForLevel(5)).toBe(2)
    expect(barbaroRdForLevel(7)).toBe(2)
  })

  it('nível 8-10: RD 4', () => {
    expect(barbaroRdForLevel(8)).toBe(4)
    expect(barbaroRdForLevel(10)).toBe(4)
  })

  it('nível 11-13: RD 6', () => {
    expect(barbaroRdForLevel(11)).toBe(6)
    expect(barbaroRdForLevel(13)).toBe(6)
  })

  it('nível 14-16: RD 8', () => {
    expect(barbaroRdForLevel(14)).toBe(8)
    expect(barbaroRdForLevel(16)).toBe(8)
  })

  it('nível 17+: RD 10', () => {
    expect(barbaroRdForLevel(17)).toBe(10)
    expect(barbaroRdForLevel(20)).toBe(10)
  })

  it('throws se nível < 1', () => {
    expect(() => barbaroRdForLevel(0)).toThrow(/level/)
  })
})

describe('guerreiroRdForLevel', () => {
  it('sem armadura pesada → 0', () => {
    expect(guerreiroRdForLevel(17, false)).toBe(0)
  })

  it('com armadura pesada nível 5+: RD 2', () => {
    expect(guerreiroRdForLevel(5, true)).toBe(2)
  })

  it('escala até 10 no 17º', () => {
    expect(guerreiroRdForLevel(17, true)).toBe(10)
  })
})

describe('cavaleiroTotalRd', () => {
  it('sem armadura pesada → 0', () => {
    expect(cavaleiroTotalRd(20, false)).toBe(0)
  })

  it('nível 5 (só Bastião) = 5', () => {
    expect(cavaleiroTotalRd(5, true)).toBe(CAVALEIRO_BASTIAO_RD)
  })

  it('nível 12 (Bastião + Especialização) = 10', () => {
    expect(cavaleiroTotalRd(12, true)).toBe(
      CAVALEIRO_BASTIAO_RD + CAVALEIRO_ESPECIALIZACAO_RD,
    )
  })

  it('nível 4 = 0 (sem Bastião ainda)', () => {
    expect(cavaleiroTotalRd(4, true)).toBe(0)
  })
})

describe('Adamante constants — p133', () => {
  it('armadura leve/escudo = RD 2', () => {
    expect(ADAMANTE_ARMADURA_LEVE_RD).toBe(2)
  })

  it('armadura pesada = RD 5', () => {
    expect(ADAMANTE_ARMADURA_PESADA_RD).toBe(5)
  })
})

describe('applyDamageReduction — validation', () => {
  it('throws se amount negativo', () => {
    const profile: DamageResistanceProfile = { rdSources: [], immunities: [] }
    expect(() =>
      applyDamageReduction({ amount: -1, damageType: 'corte' }, profile),
    ).toThrow(/amount/)
  })

  it('dano 0 = 0 sempre', () => {
    const profile: DamageResistanceProfile = { rdSources: [], immunities: [] }
    expect(
      applyDamageReduction({ amount: 0, damageType: 'corte' }, profile),
    ).toBe(0)
  })
})
