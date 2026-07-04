import { describe, expect, it } from 'vitest'
import {
  ABRIR_FECHADURA_CD,
  ABRIR_FECHADURA_NO_GAZUA_PENALTY,
  LADINAGEM_ARMOR_PENALTY,
  LADINAGEM_TRAINED_ONLY,
  LADINAGEM_USAGES,
  OCULTAR_BIG_PENALTY,
  OCULTAR_SEARCHER_PERCEPCAO_BONUS,
  OCULTAR_SMALL_BONUS,
  PUNGA_CD,
  SABOTAR_CD_COMPLEXA,
  SABOTAR_CD_SIMPLES,
  SABOTAR_CRITICAL_FAILURE_MARGIN,
  SABOTAR_RUSHED_PENALTY,
  abrirFechaduraCd,
  abrirFechaduraPenalty,
  ladinagemUsageByKind,
  ocultarObserverPercepcaoBonus,
  ocultarSizeModifier,
  pungaOutcome,
  pungaVictimPercepcaoCd,
  sabotarCd,
  sabotarOutcome,
  sabotarPaceModifier,
} from '../ladinagem-skill-usages'

/**
 * PDF livro p120-121 — Perícia Ladinagem (DES, treinada, armor penalty).
 *  1. Abrir Fechadura (p120) — CDs 20/25/30 por qualidade; gazua ou -5
 *  2. Ocultar (p120) — ação padrão vs Percepção; ±5; revistador +10
 *  3. Punga (p120) — CD 20; vítima Percepção CD = seu roll
 *  4. Sabotar (p121) — CDs 20/30; 1d4 rodadas ou completa @ -5; falha 5+ crítica
 */

describe('LADINAGEM_USAGES — shape', () => {
  it('exatamente 4 usos', () => {
    expect(LADINAGEM_USAGES.length).toBe(4)
  })

  it('frozen', () => {
    expect(Object.isFrozen(LADINAGEM_USAGES)).toBe(true)
  })

  it('ids canônicos', () => {
    expect(LADINAGEM_USAGES.map((u) => u.id).sort()).toEqual([
      'abrir-fechadura',
      'ocultar',
      'punga',
      'sabotar',
    ])
  })

  it('Sabotar em p121, resto em p120', () => {
    for (const u of LADINAGEM_USAGES) {
      const expected = u.id === 'sabotar' ? 121 : 120
      expect(u.bookPage).toBe(expected)
    }
  })
})

describe('Perícia flags — Tabela 2-1', () => {
  it('apenas treinada', () => {
    expect(LADINAGEM_TRAINED_ONLY).toBe(true)
  })

  it('com penalidade de armadura', () => {
    expect(LADINAGEM_ARMOR_PENALTY).toBe(true)
  })
})

describe('Abrir Fechadura — p120', () => {
  it('CDs verbatim', () => {
    expect(ABRIR_FECHADURA_CD.simples).toBe(20)
    expect(ABRIR_FECHADURA_CD.media).toBe(25)
    expect(ABRIR_FECHADURA_CD.superior).toBe(30)
  })

  it('CD frozen', () => {
    expect(Object.isFrozen(ABRIR_FECHADURA_CD)).toBe(true)
  })

  it('penalidade sem gazua -5', () => {
    expect(ABRIR_FECHADURA_NO_GAZUA_PENALTY).toBe(-5)
  })

  it('exige gazua', () => {
    const usage = ladinagemUsageByKind('abrir-fechadura')
    if (usage.kind !== 'abrir-fechadura') throw new Error('narrow failed')
    expect(usage.requiresGazua).toBe(true)
  })
})

describe('abrirFechaduraCd', () => {
  it.each([
    ['simples', 20],
    ['media', 25],
    ['superior', 30],
  ] as const)('%s → %s', (q, cd) => {
    expect(abrirFechaduraCd(q)).toBe(cd)
  })
})

describe('abrirFechaduraPenalty', () => {
  it('com gazua → 0', () => {
    expect(abrirFechaduraPenalty(true)).toBe(0)
  })

  it('sem gazua → -5', () => {
    expect(abrirFechaduraPenalty(false)).toBe(-5)
  })
})

describe('Ocultar — p120', () => {
  const usage = ladinagemUsageByKind('ocultar')

  it('oposto por Percepção', () => {
    if (usage.kind !== 'ocultar') throw new Error('narrow failed')
    expect(usage.opposedBy).toBe('percepcao')
  })

  it('constantes verbatim', () => {
    expect(OCULTAR_SMALL_BONUS).toBe(5)
    expect(OCULTAR_BIG_PENALTY).toBe(-5)
    expect(OCULTAR_SEARCHER_PERCEPCAO_BONUS).toBe(10)
  })
})

describe('ocultarSizeModifier', () => {
  it('discreto/pequeno → +5', () => {
    expect(ocultarSizeModifier('discreto-pequeno')).toBe(5)
  })

  it('normal → 0', () => {
    expect(ocultarSizeModifier('normal')).toBe(0)
  })

  it('desajeitado/grande → -5', () => {
    expect(ocultarSizeModifier('desajeitado-grande')).toBe(-5)
  })
})

describe('ocultarObserverPercepcaoBonus', () => {
  it('revistando → +10', () => {
    expect(ocultarObserverPercepcaoBonus(true)).toBe(10)
  })

  it('passivo → 0', () => {
    expect(ocultarObserverPercepcaoBonus(false)).toBe(0)
  })
})

describe('Punga — p120', () => {
  const usage = ladinagemUsageByKind('punga')

  it('CD 20', () => {
    expect(PUNGA_CD).toBe(20)
    if (usage.kind !== 'punga') throw new Error('narrow failed')
    expect(usage.dc).toBe(20)
  })

  it('pode plantar objeto', () => {
    if (usage.kind !== 'punga') throw new Error('narrow failed')
    expect(usage.canPlantObject).toBe(true)
  })
})

describe('pungaVictimPercepcaoCd', () => {
  it('CD Percepção = roll de Ladinagem', () => {
    expect(pungaVictimPercepcaoCd(23)).toBe(23)
  })
})

describe('pungaOutcome', () => {
  it('rolagem 25 vs perc 15 → success-unseen', () => {
    expect(pungaOutcome(25, 15)).toBe('success-unseen')
  })

  it('rolagem 25 vs perc 25 → success-seen (empate percebido)', () => {
    expect(pungaOutcome(25, 25)).toBe('success-seen')
  })

  it('rolagem 25 vs perc 30 → success-seen (percebe mesmo com sucesso)', () => {
    expect(pungaOutcome(25, 30)).toBe('success-seen')
  })

  it('rolagem 19 vs perc 15 → failed-unseen (fail CD 20 + não percebido)', () => {
    expect(pungaOutcome(19, 15)).toBe('failed-unseen')
  })

  it('rolagem 19 vs perc 19 → failed-seen (fail + percebido)', () => {
    expect(pungaOutcome(19, 19)).toBe('failed-seen')
  })
})

describe('Sabotar — p121', () => {
  it('CDs verbatim', () => {
    expect(SABOTAR_CD_SIMPLES).toBe(20)
    expect(SABOTAR_CD_COMPLEXA).toBe(30)
  })

  it('rushed penalty -5', () => {
    expect(SABOTAR_RUSHED_PENALTY).toBe(-5)
  })

  it('critical failure margin 5', () => {
    expect(SABOTAR_CRITICAL_FAILURE_MARGIN).toBe(5)
  })
})

describe('sabotarCd', () => {
  it('simples → 20', () => {
    expect(sabotarCd('simples')).toBe(20)
  })

  it('complexa → 30', () => {
    expect(sabotarCd('complexa')).toBe(30)
  })
})

describe('sabotarPaceModifier', () => {
  it('1d4 rodadas → 0', () => {
    expect(sabotarPaceModifier('padrao-1d4-rodadas')).toBe(0)
  })

  it('ação completa → -5', () => {
    expect(sabotarPaceModifier('ação-completa-com-penalidade')).toBe(-5)
  })
})

describe('sabotarOutcome', () => {
  it('sucesso (CD 20 vs 20) → success', () => {
    expect(sabotarOutcome(20, 20)).toBe('success')
  })

  it('sucesso alto (CD 20 vs 30) → success', () => {
    expect(sabotarOutcome(30, 20)).toBe('success')
  })

  it('falha por 4 (CD 20 vs 16) → failed', () => {
    expect(sabotarOutcome(16, 20)).toBe('failed')
  })

  it('falha por 5 (CD 20 vs 15) → critical-failure', () => {
    expect(sabotarOutcome(15, 20)).toBe('critical-failure')
  })

  it('falha por 10 (CD 30 vs 20) → critical-failure', () => {
    expect(sabotarOutcome(20, 30)).toBe('critical-failure')
  })
})

describe('ladinagemUsageByKind', () => {
  it('throws se kind inválido', () => {
    expect(() =>
      // @ts-expect-error — invalid kind on purpose
      ladinagemUsageByKind('escapismo'),
    ).toThrow(/unknown kind/)
  })

  it('resolve todos', () => {
    for (const k of ['abrir-fechadura', 'ocultar', 'punga', 'sabotar'] as const) {
      expect(ladinagemUsageByKind(k).kind).toBe(k)
    }
  })
})
