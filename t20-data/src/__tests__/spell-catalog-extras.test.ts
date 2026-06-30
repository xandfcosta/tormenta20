import { describe, expect, it } from 'vitest'
import {
  SPELL_IDS,
  spellById,
  spellsByCircle,
  spellsBySchool,
} from '../spell-catalog'
import { SPELL_SCHOOLS } from '../spells'

/**
 * Lote +19 magias (PDF Cap 7, p179-211). Garantia:
 *  - Catálogo cresceu de 14 → 33 (>= 30).
 *  - Toda das 8 escolas tem ≥ 1 entrada.
 *  - Círculos 1-5 todos populados.
 *  - Cada nova magia pinned por id existe e bate na escola + círculo.
 */

describe('spell-catalog — lote +19 magias adicionado', () => {
  it('SPELL_IDS tem ≥ 33 entradas (14 originais + 19 novas)', () => {
    expect(SPELL_IDS.length).toBeGreaterThanOrEqual(33)
  })

  it('toda das 8 escolas tem ≥ 1 magia', () => {
    for (const school of SPELL_SCHOOLS) {
      const count = spellsBySchool(school).length
      expect(count, `escola ${school} sem magia`).toBeGreaterThan(0)
    }
  })

  it('círculos 1-5 todos populados', () => {
    for (const circle of [1, 2, 3, 4, 5] as const) {
      expect(spellsByCircle(circle).length).toBeGreaterThan(0)
    }
  })
})

describe('spell-catalog — pinned canonical entries do lote', () => {
  const PINS: Array<[string, string, number]> = [
    ['dissipar-magia', 'abjuracao', 2],
    ['escudo-da-fe', 'abjuracao', 1],
    ['invulnerabilidade', 'abjuracao', 5],
    ['tranca-arcana', 'abjuracao', 1],
    ['detectar-ameacas', 'adivinhacao', 1],
    ['voz-divina', 'adivinhacao', 2],
    ['videncia', 'adivinhacao', 3],
    ['caminhos-da-natureza', 'convocacao', 1],
    ['teletransporte', 'convocacao', 3],
    ['amedrontar', 'necromancia', 1],
    ['marionete', 'encantamento', 4],
    ['explosao-de-chamas', 'evocacao', 1],
    ['tempestade-divina', 'evocacao', 2],
    ['sopro-das-uivantes', 'evocacao', 2],
    ['manto-de-sombras', 'ilusao', 3],
    ['imagem-espelhada', 'ilusao', 1],
    ['conjurar-mortos-vivos', 'necromancia', 2],
    ['velocidade', 'transmutacao', 2],
    ['voo', 'transmutacao', 3],
  ]

  it.each(PINS)('%s: escola %s, círculo %i', (id, school, circle) => {
    const spell = spellById(id)
    expect(spell.school).toBe(school)
    expect(spell.circle).toBe(circle)
    expect(spell.bookPage).toBeGreaterThanOrEqual(179)
    expect(spell.bookPage).toBeLessThanOrEqual(211)
  })
})

describe('spell-catalog — pinned mecânicas faithful to PDF', () => {
  it('Vidência: ilimitado, sustentada, vontade/anula, foco', () => {
    const s = spellById('videncia')
    expect(s.range).toBe('ilimitado')
    expect(s.duration).toBe('sustentada')
    expect(s.saveType).toBe('vontade')
    expect(s.resistance).toBe('anula')
    expect(s.components).toContain('foco')
  })

  it('Escudo da Fé: reação, +2 Defesa, círculo 1, paladino+clérigo', () => {
    const s = spellById('escudo-da-fe')
    expect(s.execution).toBe('reacao')
    expect(s.baseEffect).toMatch(/\+2 na Defesa/)
    expect(s.classes).toEqual(expect.arrayContaining(['Clérigo', 'Paladino']))
  })

  it('Invulnerabilidade: círculo 5 — pinned alto', () => {
    expect(spellById('invulnerabilidade').circle).toBe(5)
  })

  it('Tranca Arcana: componente material (chave T$ 25)', () => {
    const s = spellById('tranca-arcana')
    expect(s.components).toContain('material')
    expect(s.baseEffect).toMatch(/T\$ 25/)
  })

  it('Voo: deslocamento de voo 12m', () => {
    expect(spellById('voo').baseEffect).toMatch(/voo 12m/)
  })

  it('Sopro das Uivantes: cone 9m, 4d6 frio', () => {
    const s = spellById('sopro-das-uivantes')
    expect(s.baseEffect).toMatch(/cone de 9m/)
    expect(s.baseEffect).toMatch(/4d6 dano de frio/)
  })

  it('Marionete: sustentada, Fortitude anula', () => {
    const s = spellById('marionete')
    expect(s.duration).toBe('sustentada')
    expect(s.saveType).toBe('fortitude')
    expect(s.resistance).toBe('anula')
  })

  it('Teletransporte: alcance toque, instantânea, sem resistance', () => {
    const s = spellById('teletransporte')
    expect(s.range).toBe('toque')
    expect(s.duration).toBe('instantanea')
    expect(s.resistance).toBeNull()
  })
})

describe('spell-catalog — augments com gating de círculo / classe', () => {
  it('Dissipar Magia: augment 5° círculo (área 9m)', () => {
    const s = spellById('dissipar-magia')
    const gated = s.augments.find((a) => a.requiresCircle === 5)
    expect(gated).toBeDefined()
  })

  it('Escudo da Fé: augment 2° círculo (transferência de dano)', () => {
    const s = spellById('escudo-da-fe')
    const gated = s.augments.find((a) => a.requiresCircle === 2)
    expect(gated).toBeDefined()
  })

  it('Voo: dois augments 4° círculo (duração dia / 10 criaturas)', () => {
    const s = spellById('voo')
    const c4 = s.augments.filter((a) => a.requiresCircle === 4)
    expect(c4.length).toBe(2)
  })

  it('Velocidade: dois augments 4° círculo', () => {
    const s = spellById('velocidade')
    const c4 = s.augments.filter((a) => a.requiresCircle === 4)
    expect(c4.length).toBe(2)
  })

  it('Conjurar Mortos-Vivos: gating 3° (carniçais) + 4° (sombras)', () => {
    const s = spellById('conjurar-mortos-vivos')
    expect(s.augments.find((a) => a.requiresCircle === 3)).toBeDefined()
    expect(s.augments.find((a) => a.requiresCircle === 4)).toBeDefined()
  })
})
