import { describe, expect, it } from 'vitest'
import {
  SITUACOES_ESPECIAIS,
  aggregateSituacaoModifiers,
  applicableSituacoes,
  qualitativeSituacaoNotes,
  situacaoById,
  situacoesBySide,
} from '../combat-situations'

/**
 * PDF Cap 5 (Combate), Tabela 5-3, livro p239. Pinned:
 *  - 10 rows total (6 atacante / 4 alvo; "alvo caído" = 2 rows for
 *    melee vs ranged).
 *  - Atacante cego = 50% miss (qualitative, modifier 0).
 *  - Atacante invisível = -5 na Defesa do alvo (appliesTo defesa).
 *  - Flanqueando = +2 corpo a corpo apenas.
 *  - Alvo caído: -5 vs CaC, +5 vs distância.
 *
 * Cobertura/camuflagem rows NOT here — owned by combat-cover.ts.
 */

describe('SITUACOES_ESPECIAIS — shape & invariants', () => {
  it('catalog has exactly 10 entries', () => {
    expect(SITUACOES_ESPECIAIS.length).toBe(10)
  })

  it('all ids unique', () => {
    const ids = SITUACOES_ESPECIAIS.map((s) => s.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('all bookPage === 239', () => {
    for (const s of SITUACOES_ESPECIAIS) expect(s.bookPage).toBe(239)
  })

  it('catalog is frozen', () => {
    expect(Object.isFrozen(SITUACOES_ESPECIAIS)).toBe(true)
  })

  it('every side is atacante or alvo', () => {
    for (const s of SITUACOES_ESPECIAIS) {
      expect(['atacante', 'alvo']).toContain(s.side)
    }
  })

  it('entries with modifier 0 must carry a qualitativeNote', () => {
    for (const s of SITUACOES_ESPECIAIS) {
      if (s.modifier === 0) expect(s.qualitativeNote).toBeTruthy()
    }
  })

  it('does NOT include cobertura/camuflagem rows (owned by combat-cover)', () => {
    const ids = SITUACOES_ESPECIAIS.map((s) => s.id)
    for (const banned of [
      'alvo-sob-cobertura-leve',
      'alvo-sob-cobertura-total',
      'alvo-sob-camuflagem',
      'alvo-sob-camuflagem-total',
    ]) {
      expect(ids).not.toContain(banned)
    }
  })
})

describe('SITUACOES_ESPECIAIS — pinned rows', () => {
  it('atacante flanqueando = +2 ataque, melee only', () => {
    const s = situacaoById('atacante-flanqueando')!
    expect(s.modifier).toBe(2)
    expect(s.appliesTo).toBe('ataque')
    expect(s.meleeOnly).toBe(true)
  })

  it('atacante cego = 50% miss (qualitative, modifier 0)', () => {
    const s = situacaoById('atacante-cego')!
    expect(s.modifier).toBe(0)
    expect(s.qualitativeNote).toMatch(/50%/)
  })

  it('atacante em posição elevada = +2 ataque (sem restrição)', () => {
    const s = situacaoById('atacante-em-posicao-elevada')!
    expect(s.modifier).toBe(2)
    expect(s.meleeOnly).toBe(false)
    expect(s.rangedOnly).toBe(false)
  })

  it('atacante invisível: -5 na Defesa do alvo (appliesTo defesa)', () => {
    const s = situacaoById('atacante-invisivel')!
    expect(s.modifier).toBe(-5)
    expect(s.appliesTo).toBe('defesa')
    expect(s.qualitativeNote).toMatch(/Defesa/)
  })

  it('atacante caído: -5 em ataque, melee only', () => {
    const s = situacaoById('atacante-caido')!
    expect(s.modifier).toBe(-5)
    expect(s.meleeOnly).toBe(true)
  })

  it('atacante ofuscado: -2 em ataque (qualquer alcance)', () => {
    const s = situacaoById('atacante-ofuscado')!
    expect(s.modifier).toBe(-2)
    expect(s.meleeOnly).toBe(false)
    expect(s.rangedOnly).toBe(false)
  })

  it('alvo caído contra CaC = -5 Defesa', () => {
    const s = situacaoById('alvo-caido-cac')!
    expect(s.modifier).toBe(-5)
    expect(s.meleeOnly).toBe(true)
  })

  it('alvo caído contra distância = +5 Defesa', () => {
    const s = situacaoById('alvo-caido-distancia')!
    expect(s.modifier).toBe(5)
    expect(s.rangedOnly).toBe(true)
  })

  it('alvo cego = -5 Defesa (sem restrição de alcance)', () => {
    const s = situacaoById('alvo-cego')!
    expect(s.modifier).toBe(-5)
    expect(s.meleeOnly).toBe(false)
    expect(s.rangedOnly).toBe(false)
  })

  it('alvo desprevenido = -5 Defesa', () => {
    const s = situacaoById('alvo-desprevenido')!
    expect(s.modifier).toBe(-5)
    expect(s.appliesTo).toBe('defesa')
  })
})

describe('situacoesBySide', () => {
  it('atacante side has 6 entries', () => {
    expect(situacoesBySide('atacante').length).toBe(6)
  })

  it('alvo side has 4 entries (caído split into melee + ranged)', () => {
    expect(situacoesBySide('alvo').length).toBe(4)
  })
})

describe('applicableSituacoes — range gating', () => {
  it('atacante-flanqueando applies em melee, drops em ranged', () => {
    expect(
      applicableSituacoes(['atacante-flanqueando'], 'melee').length,
    ).toBe(1)
    expect(
      applicableSituacoes(['atacante-flanqueando'], 'ranged').length,
    ).toBe(0)
  })

  it('alvo-caido-distancia applies em ranged, drops em melee', () => {
    expect(
      applicableSituacoes(['alvo-caido-distancia'], 'ranged').length,
    ).toBe(1)
    expect(
      applicableSituacoes(['alvo-caido-distancia'], 'melee').length,
    ).toBe(0)
  })

  it('unknown ids são ignored', () => {
    expect(
      applicableSituacoes(['atacante-flanqueando', 'foo-bar'], 'melee').length,
    ).toBe(1)
  })

  it('atacante-ofuscado e atacante-em-posicao-elevada aplicam-se a ambos', () => {
    const ids = ['atacante-ofuscado', 'atacante-em-posicao-elevada']
    expect(applicableSituacoes(ids, 'melee').length).toBe(2)
    expect(applicableSituacoes(ids, 'ranged').length).toBe(2)
  })
})

describe('aggregateSituacaoModifiers', () => {
  it('flanqueando (+2 ataque) + alvo desprevenido (-5 defesa) em melee', () => {
    const out = aggregateSituacaoModifiers(
      ['atacante-flanqueando', 'alvo-desprevenido'],
      'melee',
    )
    expect(out.ataque).toBe(2)
    expect(out.defesa).toBe(-5)
  })

  it('alvo caído: -5 em melee, +5 em ranged (separates corretamente)', () => {
    const ids = ['alvo-caido-cac', 'alvo-caido-distancia']
    expect(aggregateSituacaoModifiers(ids, 'melee').defesa).toBe(-5)
    expect(aggregateSituacaoModifiers(ids, 'ranged').defesa).toBe(5)
  })

  it('atacante cego (qualitative) não soma em ataque', () => {
    const out = aggregateSituacaoModifiers(['atacante-cego'], 'ranged')
    expect(out.ataque).toBe(0)
    expect(out.defesa).toBe(0)
  })

  it('atacante invisível desconta -5 na defesa do alvo, não no ataque', () => {
    const out = aggregateSituacaoModifiers(
      ['atacante-invisivel'],
      'melee',
    )
    expect(out.ataque).toBe(0)
    expect(out.defesa).toBe(-5)
  })

  it('múltiplos atacante mods empilham (ofuscado + posição elevada)', () => {
    const out = aggregateSituacaoModifiers(
      ['atacante-ofuscado', 'atacante-em-posicao-elevada'],
      'ranged',
    )
    expect(out.ataque).toBe(0) // -2 + 2
  })
})

describe('qualitativeSituacaoNotes', () => {
  it('atacante cego retorna nota "50% de chance de falha"', () => {
    const notes = qualitativeSituacaoNotes(['atacante-cego'], 'melee')
    expect(notes).toEqual(['50% de chance de falha'])
  })

  it('atacante invisível inclui sua nota qualitativa', () => {
    const notes = qualitativeSituacaoNotes(['atacante-invisivel'], 'melee')
    expect(notes[0]).toMatch(/Defesa/)
  })

  it('entradas sem nota qualitativa não aparecem na lista', () => {
    const notes = qualitativeSituacaoNotes(
      ['atacante-flanqueando', 'alvo-desprevenido'],
      'melee',
    )
    expect(notes).toEqual([])
  })

  it('lookup helper situacaoById funciona', () => {
    expect(situacaoById('atacante-flanqueando')?.name).toBe(
      'Flanqueando o alvo',
    )
    expect(situacaoById('foo')).toBeUndefined()
  })
})
