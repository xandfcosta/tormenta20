import { describe, expect, it } from 'vitest'
import {
  PENITENCIA_SACRIFICE_TIBAR_PER_LEVEL,
  RELIGIAO_ARMOR_PENALTY,
  RELIGIAO_IDENTIFICAR_CRIATURA_CD_BASE,
  RELIGIAO_INFORMACAO_CD_COMPLEXA,
  RELIGIAO_INFORMACAO_CD_MISTERIO,
  RELIGIAO_TRAINED_ONLY,
  RELIGIAO_USAGES,
  RITO_CD,
  penitenciaSacrificeTibar,
  religiaoIdentificarCriaturaCd,
  religiaoIdentificarItemMagicoCd,
  religiaoIdentificarItemMagicoRushedPenalty,
  religiaoInformacaoCd,
  religiaoUsageByKind,
  ritoCd,
} from '../religiao-skill-usages'

/**
 * PDF livro p122 — Perícia Religião (SAB, treinada).
 *  1. Identificar Criatura — CD 15 + ND (delega Misticismo, criatura divina)
 *  2. Identificar Item Mágico — delega Misticismo (item mágico divino)
 *  3. Informação — sem teste / CD 20 / CD 30
 *  4. Rito — CD 20; penitência T$ 100 × nível ou missão
 */

describe('RELIGIAO_USAGES — shape', () => {
  it('exatamente 4 usos', () => {
    expect(RELIGIAO_USAGES.length).toBe(4)
  })

  it('frozen', () => {
    expect(Object.isFrozen(RELIGIAO_USAGES)).toBe(true)
  })

  it('ids canônicos', () => {
    expect(RELIGIAO_USAGES.map((u) => u.id).sort()).toEqual([
      'identificar-criatura',
      'identificar-item-magico',
      'informacao',
      'rito',
    ])
  })

  it('todos em p122', () => {
    for (const u of RELIGIAO_USAGES) {
      expect(u.bookPage).toBe(122)
    }
  })
})

describe('Perícia flags — Tabela 2-1', () => {
  it('apenas treinada', () => {
    expect(RELIGIAO_TRAINED_ONLY).toBe(true)
  })

  it('sem penalidade de armadura', () => {
    expect(RELIGIAO_ARMOR_PENALTY).toBe(false)
  })
})

describe('Identificar Criatura — p122', () => {
  const usage = religiaoUsageByKind('identificar-criatura')

  it('CD base 15', () => {
    expect(RELIGIAO_IDENTIFICAR_CRIATURA_CD_BASE).toBe(15)
    if (usage.kind !== 'identificar-criatura') throw new Error('narrow failed')
    expect(usage.cdBase).toBe(15)
    expect(usage.cdIncludesNd).toBe(true)
  })

  it('só origem divina; delega Misticismo', () => {
    if (usage.kind !== 'identificar-criatura') throw new Error('narrow failed')
    expect(usage.divineOriginOnly).toBe(true)
    expect(usage.delegatesTo).toBe('misticismo')
  })
})

describe('religiaoIdentificarCriaturaCd', () => {
  it('ND 0 → 15', () => {
    expect(religiaoIdentificarCriaturaCd(0)).toBe(15)
  })

  it('ND 8 → 23', () => {
    expect(religiaoIdentificarCriaturaCd(8)).toBe(23)
  })
})

describe('Identificar Item Mágico — p122 (delega Misticismo)', () => {
  const usage = religiaoUsageByKind('identificar-item-magico')

  it('só origem divina; delega Misticismo', () => {
    if (usage.kind !== 'identificar-item-magico') throw new Error('narrow failed')
    expect(usage.divineOriginOnly).toBe(true)
    expect(usage.delegatesTo).toBe('misticismo')
  })
})

describe('religiaoIdentificarItemMagicoCd', () => {
  it.each([
    ['menor', 20],
    ['medio', 25],
    ['maior', 30],
  ] as const)('%s → %s (delegado)', (c, cd) => {
    expect(religiaoIdentificarItemMagicoCd(c)).toBe(cd)
  })
})

describe('religiaoIdentificarItemMagicoRushedPenalty', () => {
  it('rushed → -10 (delegado)', () => {
    expect(religiaoIdentificarItemMagicoRushedPenalty(true)).toBe(-10)
  })

  it('1 hora → 0', () => {
    expect(religiaoIdentificarItemMagicoRushedPenalty(false)).toBe(0)
  })
})

describe('Informação — p122', () => {
  it('CDs verbatim', () => {
    expect(RELIGIAO_INFORMACAO_CD_COMPLEXA).toBe(20)
    expect(RELIGIAO_INFORMACAO_CD_MISTERIO).toBe(30)
  })

  it('simples não exige teste', () => {
    const usage = religiaoUsageByKind('informacao')
    if (usage.kind !== 'informacao') throw new Error('narrow failed')
    expect(usage.simplesRequiresNoTest).toBe(true)
  })
})

describe('religiaoInformacaoCd', () => {
  it('simples → null', () => {
    expect(religiaoInformacaoCd('simples')).toBeNull()
  })

  it('complexa → 20', () => {
    expect(religiaoInformacaoCd('complexa')).toBe(20)
  })

  it('mistério/enigma → 30', () => {
    expect(religiaoInformacaoCd('misterio-ou-enigma')).toBe(30)
  })
})

describe('Rito — p122', () => {
  const usage = religiaoUsageByKind('rito')

  it('CD 20', () => {
    expect(RITO_CD).toBe(20)
    if (usage.kind !== 'rito') throw new Error('narrow failed')
    expect(usage.dc).toBe(20)
  })

  it('penitência: T$ 100/nível ou missão', () => {
    expect(PENITENCIA_SACRIFICE_TIBAR_PER_LEVEL).toBe(100)
    if (usage.kind !== 'rito') throw new Error('narrow failed')
    expect(usage.penitenciaSacrificeTibarPerLevel).toBe(100)
    expect(usage.penitenciaMissionAlternative).toBe(true)
  })
})

describe('ritoCd', () => {
  it('sempre 20', () => {
    expect(ritoCd()).toBe(20)
  })
})

describe('penitenciaSacrificeTibar', () => {
  it('nível 1 → T$ 100', () => {
    expect(penitenciaSacrificeTibar(1)).toBe(100)
  })

  it('nível 10 → T$ 1000', () => {
    expect(penitenciaSacrificeTibar(10)).toBe(1000)
  })

  it('nível 0 lança', () => {
    expect(() => penitenciaSacrificeTibar(0)).toThrow(/devotoLevel must be ≥ 1/)
  })

  it('nível negativo lança', () => {
    expect(() => penitenciaSacrificeTibar(-3)).toThrow(/devotoLevel must be ≥ 1/)
  })
})

describe('religiaoUsageByKind', () => {
  it('throws se kind inválido', () => {
    expect(() =>
      // @ts-expect-error — invalid kind on purpose
      religiaoUsageByKind('exorcizar'),
    ).toThrow(/unknown kind/)
  })

  it('resolve todos', () => {
    for (const k of [
      'identificar-criatura',
      'identificar-item-magico',
      'informacao',
      'rito',
    ] as const) {
      expect(religiaoUsageByKind(k).kind).toBe(k)
    }
  })
})
