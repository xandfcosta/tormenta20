import { describe, expect, it } from 'vitest'
import { SPELL_SCHOOLS } from '../spells'
import {
  DAMAGE_META,
  DAMAGE_TYPES,
  EFFECT_META,
  EFFECT_TYPES,
  SCHOOL_META,
  damageMeta,
  effectMeta,
  intrinsicEffectsForSchool,
  schoolMeta,
} from '../spell-taxonomy'

/**
 * PDF Cap 4 p172-173 (escolas), Cap 5 p227-228 (efeitos), p230 (dano).
 * Pinned:
 *  - 18 tipos de efeito (Tabela de tipos p227-228).
 *  - 11 tipos de dano (p230).
 *  - 8 escolas com metadata.
 *  - Encantamento/Ilusão intrinsic 'mental'; Necromancia intrinsic 'trevas'.
 */

describe('EFFECT_TYPES — p227-228', () => {
  it('tem 18 tipos', () => {
    expect(EFFECT_TYPES.length).toBe(18)
  })

  it('inclui os tipos principais', () => {
    const set = new Set(EFFECT_TYPES)
    for (const t of [
      'arcano',
      'divino',
      'mental',
      'medo',
      'cura',
      'trevas',
      'luz',
      'metabolismo',
      'veneno',
    ]) {
      expect(set.has(t as (typeof EFFECT_TYPES)[number])).toBe(true)
    }
  })

  it('cada tipo tem metadata completa', () => {
    for (const t of EFFECT_TYPES) {
      const m = EFFECT_META[t]
      expect(m.type).toBe(t)
      expect(m.category).toBeDefined()
      expect(Array.isArray(m.immuneCreatures)).toBe(true)
      expect(m.description.length).toBeGreaterThan(0)
    }
  })

  it('cansaço/metabolismo/veneno imunes a construto+morto-vivo', () => {
    for (const t of ['cansaco', 'metabolismo', 'veneno'] as const) {
      const imm = EFFECT_META[t].immuneCreatures
      expect(imm).toContain('construto')
      expect(imm).toContain('morto-vivo')
    }
  })

  it('medo/mental imunes a inteligência nula', () => {
    for (const t of ['medo', 'mental'] as const) {
      expect(EFFECT_META[t].immuneCreatures).toContain('inteligencia-nula')
    }
  })

  it('EFFECT_META frozen', () => {
    expect(Object.isFrozen(EFFECT_META)).toBe(true)
  })
})

describe('DAMAGE_TYPES — p230', () => {
  it('tem 11 tipos', () => {
    expect(DAMAGE_TYPES.length).toBe(11)
  })

  it('físicos: corte/impacto/perfuração', () => {
    for (const t of ['corte', 'impacto', 'perfuracao'] as const) {
      expect(DAMAGE_META[t].class).toBe('fisico')
    }
  })

  it('energia: fogo/frio/eletricidade/ácido/luz/trevas/essência/psíquico', () => {
    for (const t of [
      'fogo',
      'frio',
      'eletricidade',
      'acido',
      'luz',
      'trevas',
      'essencia',
      'psiquico',
    ] as const) {
      expect(DAMAGE_META[t].class).toBe('energia')
    }
  })

  it('elementos ligados: fogo=fogo, frio=água, elétrico=ar, ácido=terra', () => {
    expect(DAMAGE_META.fogo.element).toBe('fogo')
    expect(DAMAGE_META.frio.element).toBe('agua')
    expect(DAMAGE_META.eletricidade.element).toBe('ar')
    expect(DAMAGE_META.acido.element).toBe('terra')
  })

  it('físicos não têm elemento associado', () => {
    for (const t of ['corte', 'impacto', 'perfuracao'] as const) {
      expect(DAMAGE_META[t].element).toBeNull()
    }
  })
})

describe('SCHOOL_META — p172', () => {
  it('cobre todas as 8 escolas', () => {
    for (const s of SPELL_SCHOOLS) {
      expect(SCHOOL_META[s]).toBeDefined()
      expect(SCHOOL_META[s].school).toBe(s)
    }
  })

  it('Encantamento intrinsic mental', () => {
    expect(SCHOOL_META.encantamento.intrinsicEffects).toEqual(['mental'])
  })

  it('Ilusão intrinsic mental', () => {
    expect(SCHOOL_META.ilusao.intrinsicEffects).toEqual(['mental'])
  })

  it('Necromancia intrinsic trevas', () => {
    expect(SCHOOL_META.necromancia.intrinsicEffects).toEqual(['trevas'])
  })

  it('demais escolas sem intrinsic', () => {
    for (const s of [
      'abjuracao',
      'adivinhacao',
      'convocacao',
      'evocacao',
      'transmutacao',
    ] as const) {
      expect(SCHOOL_META[s].intrinsicEffects).toEqual([])
    }
  })

  it('abrevs padronizados', () => {
    expect(SCHOOL_META.abjuracao.abbrev).toBe('Abjur')
    expect(SCHOOL_META.necromancia.abbrev).toBe('Necro')
  })
})

describe('helpers', () => {
  it('effectMeta retorna entry', () => {
    expect(effectMeta('mental').category).toBe('mental')
  })

  it('damageMeta retorna entry', () => {
    expect(damageMeta('fogo').class).toBe('energia')
  })

  it('schoolMeta retorna entry', () => {
    expect(schoolMeta('evocacao').abbrev).toBe('Evoc')
  })

  it('intrinsicEffectsForSchool para encantamento = [mental]', () => {
    expect(intrinsicEffectsForSchool('encantamento')).toEqual(['mental'])
  })

  it('intrinsicEffectsForSchool para abjuração = []', () => {
    expect(intrinsicEffectsForSchool('abjuracao')).toEqual([])
  })
})
