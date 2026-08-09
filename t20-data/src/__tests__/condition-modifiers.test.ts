import { describe, expect, it } from 'vitest'
import { CONDITION_IDS } from '../conditions'
import {
  CONDITION_MODIFIERS,
  conditionEffectSummary,
  conditionModifiers,
} from '../condition-modifiers'

describe('CONDITION_MODIFIERS (ALE-28)', () => {
  it('every encoded modifier uses bonusType "condition" (mais severo, não soma)', () => {
    for (const mods of Object.values(CONDITION_MODIFIERS)) {
      for (const m of mods ?? []) {
        expect(m.bonusType).toBe('condition')
      }
    }
  })

  it('only references real condition ids', () => {
    const ids = new Set<string>(CONDITION_IDS)
    for (const id of Object.keys(CONDITION_MODIFIERS)) {
      expect(ids.has(id)).toBe(true)
    }
  })

  it('Vulnerável = −2 Defesa; Indefeso = −10 Defesa', () => {
    expect(conditionModifiers('vulneravel')).toEqual([
      { target: { k: 'defense' }, amount: -2, bonusType: 'condition' },
    ])
    expect(conditionModifiers('indefeso')).toEqual([
      { target: { k: 'defense' }, amount: -10, bonusType: 'condition' },
    ])
  })

  it('Abalado = −2 em todas as perícias (expertiseAll)', () => {
    expect(conditionModifiers('abalado')).toEqual([
      { target: { k: 'expertiseAll' }, amount: -2, bonusType: 'condition' },
    ])
  })

  it('Fraco = −2 nas perícias de Força/Destreza/Constituição', () => {
    expect(conditionModifiers('fraco')).toEqual([
      { target: { k: 'expertiseByAttribute', attribute: 'strength' }, amount: -2, bonusType: 'condition' },
      { target: { k: 'expertiseByAttribute', attribute: 'dexterity' }, amount: -2, bonusType: 'condition' },
      { target: { k: 'expertiseByAttribute', attribute: 'constitution' }, amount: -2, bonusType: 'condition' },
    ])
  })

  it('Caído penaliza só Luta (corpo-a-corpo), não Pontaria', () => {
    const mods = conditionModifiers('caido')
    expect(mods).toEqual([
      { target: { k: 'expertise', name: 'Luta' }, amount: -5, bonusType: 'condition' },
    ])
  })

  it('condições só-lembrete não têm modificadores', () => {
    expect(conditionModifiers('lento')).toEqual([])
    expect(conditionModifiers('enjoado')).toEqual([])
    expect(conditionModifiers('imovel')).toEqual([])
  })
})

describe('conditionEffectSummary', () => {
  it('resume o efeito numérico para o chip', () => {
    expect(conditionEffectSummary('vulneravel')).toBe('−2 Def')
    expect(conditionEffectSummary('abalado')).toBe('−2 perícias')
    expect(conditionEffectSummary('desprevenido')).toBe('−5 Def · −5 Reflexos')
    expect(conditionEffectSummary('fraco')).toBe('−2 For/Des/Con')
  })

  it('condição só-lembrete → "lembrete"', () => {
    expect(conditionEffectSummary('lento')).toBe('lembrete')
  })
})
