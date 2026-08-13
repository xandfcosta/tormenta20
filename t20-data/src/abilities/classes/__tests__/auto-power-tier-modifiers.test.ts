import { describe, expect, it } from 'vitest'
import { FLAG_ACTIVATIONS } from '../../../flag-activations'
import {
  computeItemEffects,
  resolveConditionalDisplay,
  statFor,
  type ActiveItem,
} from '../../../items/engine'
import type { Modifier } from '../../../items/types'
import { getActivation, maxStepsForLevel } from '../../../power-activation'
import { classPowerModifiers } from '../index'

/**
 * Auto powers cujo texto do livro dá bônus numérico fixo agora carregam
 * `modifiers` (auditoria "71 AUTO powers sem modifiers"). Cada família de
 * tiers usa o MESMO bonusType para o resolveStack manter só o tier mais alto
 * — espelhando furiaMods/instintoMods do Bárbaro.
 */
function powersAsItem(className: string, level: number): ActiveItem {
  return {
    source: className,
    equipped: 'vested',
    modifiers: classPowerModifiers(className, level, new Set()),
  }
}

/** Armadura pesada mínima — só a flag que o engine lê (armors.ts idem). */
const HEAVY_ARMOR: ActiveItem = {
  source: 'Armadura completa',
  equipped: 'vested',
  modifiers: [
    { target: { k: 'flag', name: 'armadura-pesada' }, amount: 1, bonusType: 'untyped', condition: { c: 'vested' } },
  ] satisfies Modifier[],
}

describe('Bardo — Inspiração como postura (p44)', () => {
  it('FLAG_ACTIVATIONS registra a postura: 2 PM, p44', () => {
    expect(FLAG_ACTIVATIONS.inspiracao).toEqual({
      flag: 'inspiracao',
      name: 'Inspiração',
      pmCost: 2,
      bookPage: 44,
    })
  })

  it('stance spec escala +1 no bônus a cada 4 níveis (L5/9/13/17)', () => {
    const spec = getActivation('class.bardo.inspiracao')!
    expect(spec.kind).toBe('stance')
    expect(spec.pmCost).toBe(2)
    const steps = (level: number) => maxStepsForLevel(spec.scaling!, level)
    expect(steps(4)).toBe(0)
    expect(steps(5)).toBe(1)
    expect(steps(13)).toBe(3)
    expect(steps(17)).toBe(4)
  })

  it('Bardo 17 mostra só o tier +5 em expertiseAll (morale supersede)', () => {
    const effects = computeItemEffects([powersAsItem('Bardo', 17)])
    const rows = effects.conditional.filter((c) => c.flag === 'inspiracao')
    expect(rows).toHaveLength(5)
    const kept = resolveConditionalDisplay(rows)
    expect(kept).toEqual([{ target: { k: 'expertiseAll' }, amount: 5 }])
  })

  it('Bardo 9 mostra só o tier +3', () => {
    const effects = computeItemEffects([powersAsItem('Bardo', 9)])
    const kept = resolveConditionalDisplay(
      effects.conditional.filter((c) => c.flag === 'inspiracao'),
    )
    expect(kept).toEqual([{ target: { k: 'expertiseAll' }, amount: 3 }])
  })
})

describe('Bucaneiro — Esquiva Sagaz (p48)', () => {
  it('Bucaneiro 11 soma +3 (não +1+2+3) em Defesa e Reflexos', () => {
    const effects = computeItemEffects([powersAsItem('Bucaneiro', 11)])
    expect(statFor(effects, { k: 'defense' }).total).toBe(3)
    expect(statFor(effects, { k: 'expertise', name: 'Reflexos' }).total).toBe(3)
  })

  it('desliga sob armadura pesada (flagOff auto-avaliado)', () => {
    const effects = computeItemEffects([powersAsItem('Bucaneiro', 11), HEAVY_ARMOR])
    expect(statFor(effects, { k: 'defense' }).total).toBe(0)
    expect(statFor(effects, { k: 'expertise', name: 'Reflexos' }).total).toBe(0)
  })
})

describe('Cavaleiro — Baluarte e Duelo (p53)', () => {
  it('Cavaleiro 17: toggle de Baluarte mostra só +10 em Defesa e resistências', () => {
    const effects = computeItemEffects([powersAsItem('Cavaleiro', 17)])
    const rows = effects.conditional.filter((c) => c.note.includes('Baluarte'))
    expect(rows).toHaveLength(10) // 5 tiers × (defense + resistance)
    const kept = resolveConditionalDisplay(rows)
    expect(kept).toContainEqual({ target: { k: 'defense' }, amount: 10 })
    expect(kept).toContainEqual({ target: { k: 'resistance' }, amount: 10 })
    expect(kept).toHaveLength(2)
  })

  it('Cavaleiro 12: toggle de Duelo mostra só +4 em ataque e dano', () => {
    const effects = computeItemEffects([powersAsItem('Cavaleiro', 12)])
    const rows = effects.conditional.filter((c) => c.note.includes('Duelo'))
    const kept = resolveConditionalDisplay(rows)
    expect(kept).toContainEqual({ target: { k: 'attack', scope: 'all' }, amount: 4 })
    expect(kept).toContainEqual({ target: { k: 'damage', scope: 'all' }, amount: 4 })
    expect(kept).toHaveLength(2)
  })
})

describe('Lutador — Casca Grossa fixa + críticos desarmados (p77)', () => {
  it('Lutador 19 soma só o +4 fixo na Defesa (tiers não acumulam)', () => {
    const effects = computeItemEffects([powersAsItem('Lutador', 19)])
    expect(statFor(effects, { k: 'defense' }).total).toBe(4)
  })

  it('Golpe Cruel e Golpe Violento viram toggles de crítico desarmado', () => {
    const effects = computeItemEffects([powersAsItem('Lutador', 9)])
    const desarmado = effects.conditional.filter((c) => c.note === 'com ataques desarmados')
    expect(desarmado).toContainEqual(
      expect.objectContaining({ target: { k: 'critRange' }, amount: 1 }),
    )
    expect(desarmado).toContainEqual(
      expect.objectContaining({ target: { k: 'critMult' }, amount: 1 }),
    )
  })
})

describe('Caçador — Rastreador (p50) e Mestre Caçador (p51)', () => {
  it('Rastreador aplica +2 em Sobrevivência desde o nível 1', () => {
    const effects = computeItemEffects([powersAsItem('Caçador', 1)])
    expect(statFor(effects, { k: 'expertise', name: 'Sobrevivência' }).total).toBe(2)
  })

  it('Mestre Caçador (L20) expõe +2 de margem de ameaça como toggle', () => {
    const effects = computeItemEffects([powersAsItem('Caçador', 20)])
    expect(effects.conditional).toContainEqual(
      expect.objectContaining({ target: { k: 'critRange' }, amount: 2 }),
    )
  })
})

describe('Druida — Força da Natureza (p63)', () => {
  it('aplica -2 PM e +2 CD sempre; dobra vira toggle de terreno natural', () => {
    const effects = computeItemEffects([powersAsItem('Druida', 20)])
    expect(statFor(effects, { k: 'pmCost' }).total).toBe(-2)
    expect(statFor(effects, { k: 'spellDC' }).total).toBe(2)
    const terreno = effects.conditional.filter((c) => c.note === 'terreno: natural')
    expect(terreno).toContainEqual(
      expect.objectContaining({ target: { k: 'pmCost' }, amount: -2 }),
    )
    expect(terreno).toContainEqual(
      expect.objectContaining({ target: { k: 'spellDC' }, amount: 2 }),
    )
  })
})

describe('Inventor — Encontrar Fraqueza (p70)', () => {
  it('expõe +2 em ataques contra alvo analisado como toggle', () => {
    const effects = computeItemEffects([powersAsItem('Inventor', 7)])
    expect(effects.conditional).toContainEqual(
      expect.objectContaining({ target: { k: 'attack', scope: 'all' }, amount: 2 }),
    )
  })
})
