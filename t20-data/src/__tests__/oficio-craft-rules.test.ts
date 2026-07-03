import { describe, expect, it } from 'vitest'
import {
  IDENTIFICAR_CD,
  OFICIO_ALLOWS_CUSTOM_CATEGORIES,
  OFICIO_CATEGORIES,
  OFICIO_INSTRUMENT_PENALTY,
  REPAIR_FAIL_LOSES_TIME_AND_MATERIAL,
  REPAIR_RETRY_ALLOWED,
  SUSTENTO_BASE_INCOME_TIBAR,
  SUSTENTO_CD,
  SUSTENTO_DURATION_WEEKS,
  SUSTENTO_INCOME_PER_MARGIN,
  oficioCategoryById,
  oficioInstrumentPenalty,
  sustentoIncome,
} from '../oficio-craft-rules'

/**
 * PDF p121-122 — Perícia Ofício. Gaps além de crafting-rules.ts:
 *  - Instrumentos: -5 sem
 *  - Sustento CD 15 / 1 semana / T$ 1 + T$ 1 por margem
 *  - Identificar CD 20
 *  - Consertar retry permitido
 *  - 5 categorias oficiais + custom permitido
 */

describe('Instrumentos — p122', () => {
  it('penalidade sem instrumentos = -5', () => {
    expect(OFICIO_INSTRUMENT_PENALTY).toBe(-5)
  })

  it('oficioInstrumentPenalty(true) → 0', () => {
    expect(oficioInstrumentPenalty(true)).toBe(0)
  })

  it('oficioInstrumentPenalty(false) → -5', () => {
    expect(oficioInstrumentPenalty(false)).toBe(-5)
  })
})

describe('Sustento — p122', () => {
  it('CD 15', () => {
    expect(SUSTENTO_CD).toBe(15)
  })

  it('duração 1 semana', () => {
    expect(SUSTENTO_DURATION_WEEKS).toBe(1)
  })

  it('renda base T$ 1', () => {
    expect(SUSTENTO_BASE_INCOME_TIBAR).toBe(1)
  })

  it('renda por margem T$ 1', () => {
    expect(SUSTENTO_INCOME_PER_MARGIN).toBe(1)
  })
})

describe('sustentoIncome', () => {
  it('falha (14) → 0', () => {
    expect(sustentoIncome(14)).toBe(0)
  })

  it('exato (15) → T$ 1 (base, margem 0)', () => {
    expect(sustentoIncome(15)).toBe(1)
  })

  it('16 → T$ 2 (base + 1 margem)', () => {
    expect(sustentoIncome(16)).toBe(2)
  })

  it('20 → T$ 6 (base + 5 margem)', () => {
    expect(sustentoIncome(20)).toBe(6)
  })

  it('25 → T$ 11', () => {
    expect(sustentoIncome(25)).toBe(11)
  })

  it('crítico teórico 30 → T$ 16', () => {
    expect(sustentoIncome(30)).toBe(16)
  })
})

describe('Identificar — p122', () => {
  it('CD 20', () => {
    expect(IDENTIFICAR_CD).toBe(20)
  })
})

describe('Consertar retry — p121', () => {
  it('retry permitido', () => {
    expect(REPAIR_RETRY_ALLOWED).toBe(true)
  })

  it('falha perde tempo + material', () => {
    expect(REPAIR_FAIL_LOSES_TIME_AND_MATERIAL).toBe(true)
  })
})

describe('OFICIO_CATEGORIES — 5 oficiais p121', () => {
  it('exatamente 5 categorias', () => {
    expect(OFICIO_CATEGORIES.length).toBe(5)
  })

  it('frozen', () => {
    expect(Object.isFrozen(OFICIO_CATEGORIES)).toBe(true)
  })

  it('ids canônicos', () => {
    const ids = OFICIO_CATEGORIES.map((c) => c.id).sort()
    expect(ids).toEqual([
      'alfaiate',
      'alquimista',
      'armeiro',
      'artesao',
      'cozinheiro',
    ])
  })

  it('Armeiro produz armas + armaduras + escudos', () => {
    const armeiro = oficioCategoryById('armeiro')
    expect([...armeiro.produces].sort()).toEqual([
      'armaduras',
      'armas',
      'escudos',
    ])
  })

  it('Alquimista produz itens alquímicos', () => {
    const alquimista = oficioCategoryById('alquimista')
    expect([...alquimista.produces]).toEqual(['itens alquímicos'])
  })

  it('Artesão produz esotéricos + veículos + ferramentas + equipamento', () => {
    const artesao = oficioCategoryById('artesao')
    expect(artesao.produces).toContain('esotéricos')
    expect(artesao.produces).toContain('veículos')
    expect(artesao.produces).toContain('equipamento de aventura')
    expect(artesao.produces).toContain('ferramentas')
  })

  it('produces frozen em cada categoria', () => {
    for (const c of OFICIO_CATEGORIES) {
      expect(Object.isFrozen(c.produces)).toBe(true)
    }
  })

  it('custom permitido (p121: carpinteiro, ourives, etc)', () => {
    expect(OFICIO_ALLOWS_CUSTOM_CATEGORIES).toBe(true)
  })
})

describe('oficioCategoryById', () => {
  it('throws se id inválido', () => {
    // @ts-expect-error — passing invalid category on purpose
    expect(() => oficioCategoryById('barbeiro')).toThrow(/unknown category/)
  })
})
