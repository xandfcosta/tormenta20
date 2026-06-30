import { describe, expect, it } from 'vitest'
import {
  CAMUFLAGEM,
  CAMUFLAGEM_DIE_SIDES,
  CAMUFLAGEM_LEVE_MISS_THRESHOLD,
  CAMUFLAGEM_TOTAL_MISS_THRESHOLD,
  COBERTURA,
  COBERTURA_LEVE_DEFENSE_BONUS,
  camuflagemCausesMiss,
  canBeAttacked,
  defesaComCobertura,
} from '../combat-cover'

/**
 * PDF Cap 5 (Combate), p238-239, Tabela 5-3. Pinned:
 *  - Only 2 cobertura tiers (Leve / Total) — no ¼/½/¾.
 *  - Cobertura Leve: +5 Defesa; NO Reflex bonus (diverges from D&D 3.5).
 *  - Cobertura Total: alvo não pode ser atacado.
 *  - Camuflagem Leve / Total: 20% / 50% miss via parallel 1d10 (NOT a
 *    reroll of the d20).
 */

describe('COBERTURA — pinned values', () => {
  it('cobertura leve = +5 Defesa', () => {
    expect(COBERTURA.leve.defenseBonus).toBe(5)
    expect(COBERTURA_LEVE_DEFENSE_BONUS).toBe(5)
  })

  it('cobertura leve still allows attacks (attackable=true)', () => {
    expect(COBERTURA.leve.attackable).toBe(true)
  })

  it('cobertura total: attackable=false (alvo não pode ser atacado)', () => {
    expect(COBERTURA.total.attackable).toBe(false)
    expect(COBERTURA.total.defenseBonus).toBe(0)
  })

  it('cobertura none: zero bonus, attackable', () => {
    expect(COBERTURA.none.defenseBonus).toBe(0)
    expect(COBERTURA.none.attackable).toBe(true)
  })

  it('every cobertura entry has bookPage 239', () => {
    for (const def of Object.values(COBERTURA)) {
      expect(def.bookPage).toBe(239)
    }
  })

  it('record is frozen', () => {
    expect(Object.isFrozen(COBERTURA)).toBe(true)
  })

  it('descriptions reference the canto-a-canto tracing rule', () => {
    expect(COBERTURA.leve.description).toMatch(/canto/)
  })
})

describe('CAMUFLAGEM — pinned values', () => {
  it('camuflagem leve = 20% miss / d10 ≤ 2', () => {
    expect(CAMUFLAGEM.leve.missChancePct).toBe(20)
    expect(CAMUFLAGEM.leve.d10MissThreshold).toBe(2)
    expect(CAMUFLAGEM_LEVE_MISS_THRESHOLD).toBe(2)
  })

  it('camuflagem total = 50% miss / d10 ≤ 5', () => {
    expect(CAMUFLAGEM.total.missChancePct).toBe(50)
    expect(CAMUFLAGEM.total.d10MissThreshold).toBe(5)
    expect(CAMUFLAGEM_TOTAL_MISS_THRESHOLD).toBe(5)
  })

  it('camuflagem none: zero miss chance', () => {
    expect(CAMUFLAGEM.none.missChancePct).toBe(0)
    expect(CAMUFLAGEM.none.d10MissThreshold).toBe(0)
  })

  it('CAMUFLAGEM_DIE_SIDES === 10 (paralelo, não 1d100)', () => {
    expect(CAMUFLAGEM_DIE_SIDES).toBe(10)
  })

  it('bookPages within 238-239', () => {
    for (const def of Object.values(CAMUFLAGEM)) {
      expect(def.bookPage).toBeGreaterThanOrEqual(238)
      expect(def.bookPage).toBeLessThanOrEqual(239)
    }
  })

  it('record is frozen', () => {
    expect(Object.isFrozen(CAMUFLAGEM)).toBe(true)
  })
})

describe('defesaComCobertura', () => {
  it('cobertura leve adiciona +5 à Defesa base', () => {
    expect(defesaComCobertura(15, 'leve')).toBe(20)
  })

  it('cobertura none deixa Defesa inalterada', () => {
    expect(defesaComCobertura(15, 'none')).toBe(15)
  })

  it('cobertura total NÃO soma bônus à Defesa (uso isolado de canBeAttacked)', () => {
    // Total significa intocável; o caller deve curto-circuitar com
    // canBeAttacked. defesaComCobertura por si só não fabrica número.
    expect(defesaComCobertura(15, 'total')).toBe(15)
  })
})

describe('canBeAttacked', () => {
  it('true para none e leve', () => {
    expect(canBeAttacked('none')).toBe(true)
    expect(canBeAttacked('leve')).toBe(true)
  })

  it('false para total', () => {
    expect(canBeAttacked('total')).toBe(false)
  })
})

describe('camuflagemCausesMiss — paralelo 1d10', () => {
  it('camuflagem none nunca causa miss', () => {
    for (let d10 = 1; d10 <= 10; d10++) {
      expect(camuflagemCausesMiss(d10, 'none')).toBe(false)
    }
  })

  it('camuflagem leve: d10 1-2 = erra, 3-10 = passa', () => {
    expect(camuflagemCausesMiss(1, 'leve')).toBe(true)
    expect(camuflagemCausesMiss(2, 'leve')).toBe(true)
    expect(camuflagemCausesMiss(3, 'leve')).toBe(false)
    expect(camuflagemCausesMiss(10, 'leve')).toBe(false)
  })

  it('camuflagem total: d10 1-5 = erra, 6-10 = passa', () => {
    for (let d10 = 1; d10 <= 5; d10++) {
      expect(camuflagemCausesMiss(d10, 'total')).toBe(true)
    }
    for (let d10 = 6; d10 <= 10; d10++) {
      expect(camuflagemCausesMiss(d10, 'total')).toBe(false)
    }
  })

  it('jogada total de 10 d10 produz percentual esperado', () => {
    let levHits = 0
    let totHits = 0
    for (let d10 = 1; d10 <= 10; d10++) {
      if (camuflagemCausesMiss(d10, 'leve')) levHits++
      if (camuflagemCausesMiss(d10, 'total')) totHits++
    }
    // 2 em 10 = 20% / 5 em 10 = 50%, conforme PDF.
    expect(levHits).toBe(2)
    expect(totHits).toBe(5)
  })

  it('lança Error para d10 fora de [1, 10]', () => {
    expect(() => camuflagemCausesMiss(0, 'leve')).toThrow(/got 0/)
    expect(() => camuflagemCausesMiss(11, 'leve')).toThrow(/got 11/)
    expect(() => camuflagemCausesMiss(-1, 'total')).toThrow(/got -1/)
  })
})

describe('Cross-references — cobertura + camuflagem são canais independentes', () => {
  it('alvo pode acumular cobertura leve E camuflagem total (efeitos somam)', () => {
    // A Defesa sobe +5 e o atacante ainda rola 1d10 paralelo.
    expect(defesaComCobertura(15, 'leve')).toBe(20)
    expect(camuflagemCausesMiss(3, 'total')).toBe(true)
  })

  it('cobertura total prevalece sobre camuflagem (ataque nem rolado)', () => {
    expect(canBeAttacked('total')).toBe(false)
    // Caller deve curto-circuitar antes de chegar à camuflagem.
  })
})
