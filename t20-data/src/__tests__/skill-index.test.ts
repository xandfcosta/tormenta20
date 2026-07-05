import { describe, expect, it } from 'vitest'
import {
  SKILL_IDS,
  SKILL_INDEX,
  TRAINED_BONUS_BASE,
  TRAINED_HALF_LEVEL_MIN,
  skillMetaById,
  skillValue,
  trainedComponent,
} from '../skill-index'

/**
 * Tabela 2-1 (p115) — 29 perícias, atributo-chave, treinada-apenas,
 * penalidade de armadura.
 *
 * Formula T20 Cap 2:
 *   Sem treino: valor = modificador do atributo
 *   Com treino: valor = ½ nível (mín +1) + atributo + 2 (- pen armadura)
 */

describe('SKILL_IDS — 29 perícias canônicas', () => {
  it('exatamente 29 ids', () => {
    expect(SKILL_IDS.length).toBe(29)
  })

  it('sem duplicatas', () => {
    expect(new Set(SKILL_IDS).size).toBe(SKILL_IDS.length)
  })

  it('todas as ids resolvem em SKILL_INDEX', () => {
    for (const id of SKILL_IDS) {
      expect(SKILL_INDEX[id]).toBeDefined()
      expect(SKILL_INDEX[id].id).toBe(id)
    }
  })
})

describe('SKILL_INDEX — Tabela 2-1 verbatim', () => {
  it('Ladinagem: DES, treinada-apenas, sofre armor penalty', () => {
    const ladinagem = SKILL_INDEX.ladinagem
    expect(ladinagem.keyAttribute).toBe('dexterity')
    expect(ladinagem.trainedOnly).toBe(true)
    expect(ladinagem.armorPenalty).toBe(true)
  })

  it('Furtividade: DES, aberta, sofre armor penalty', () => {
    expect(SKILL_INDEX.furtividade.armorPenalty).toBe(true)
    expect(SKILL_INDEX.furtividade.trainedOnly).toBe(false)
  })

  it('Acrobacia: DES, sofre armor penalty', () => {
    expect(SKILL_INDEX.acrobacia.armorPenalty).toBe(true)
    expect(SKILL_INDEX.acrobacia.trainedOnly).toBe(false)
  })

  it('Perícias treinada-apenas: Conhecimento, Guerra, Jogatina, Ladinagem, Misticismo, Nobreza, Ofício, Pilotagem, Religião', () => {
    const trainedOnly = SKILL_IDS.filter((id) => SKILL_INDEX[id].trainedOnly)
    expect(trainedOnly.sort()).toEqual([
      'conhecimento',
      'guerra',
      'jogatina',
      'ladinagem',
      'misticismo',
      'nobreza',
      'oficio',
      'pilotagem',
      'religiao',
    ])
  })

  it('Perícias com armor penalty: Acrobacia, Furtividade, Ladinagem', () => {
    const withPen = SKILL_IDS.filter((id) => SKILL_INDEX[id].armorPenalty)
    expect(withPen.sort()).toEqual(['acrobacia', 'furtividade', 'ladinagem'])
  })

  it('Saves (Fortitude/Reflexos/Vontade) usam CON/DES/SAB', () => {
    expect(SKILL_INDEX.fortitude.keyAttribute).toBe('constitution')
    expect(SKILL_INDEX.reflexos.keyAttribute).toBe('dexterity')
    expect(SKILL_INDEX.vontade.keyAttribute).toBe('wisdom')
  })

  it('Ataques (Luta/Pontaria) usam FOR/DES', () => {
    expect(SKILL_INDEX.luta.keyAttribute).toBe('strength')
    expect(SKILL_INDEX.pontaria.keyAttribute).toBe('dexterity')
  })
})

describe('skillMetaById', () => {
  it('resolve id válido', () => {
    expect(skillMetaById('acrobacia').name).toBe('Acrobacia')
  })

  it('lança em id desconhecido', () => {
    expect(() =>
      // @ts-expect-error — inválido de propósito
      skillMetaById('inexistente'),
    ).toThrow(/unknown skill id/)
  })
})

describe('trainedComponent — ½ nível (mín +1) + 2', () => {
  it('sem treino = 0', () => {
    expect(trainedComponent(10, false)).toBe(0)
  })

  it('L1 treinada = 1 (mín ½) + 2 = 3', () => {
    expect(trainedComponent(1, true)).toBe(TRAINED_HALF_LEVEL_MIN + TRAINED_BONUS_BASE)
  })

  it('L2 treinada = 1 + 2 = 3', () => {
    expect(trainedComponent(2, true)).toBe(3)
  })

  it('L10 treinada = 5 + 2 = 7', () => {
    expect(trainedComponent(10, true)).toBe(7)
  })

  it('L20 treinada = 10 + 2 = 12', () => {
    expect(trainedComponent(20, true)).toBe(12)
  })
})

describe('skillValue — fórmula completa', () => {
  it('Sem treino: valor = atributo puro', () => {
    expect(
      skillValue({
        level: 10,
        attributeValue: 3,
        trained: false,
        armorPenaltyApplies: false,
        armorPenalty: 0,
      }),
    ).toBe(3)
  })

  it('Com treino L10 DES 3 = 5 + 3 + 2 = 10', () => {
    expect(
      skillValue({
        level: 10,
        attributeValue: 3,
        trained: true,
        armorPenaltyApplies: false,
        armorPenalty: 0,
      }),
    ).toBe(10)
  })

  it('Com treino + penalidade de armadura 3 = 10 - 3 = 7', () => {
    expect(
      skillValue({
        level: 10,
        attributeValue: 3,
        trained: true,
        armorPenaltyApplies: true,
        armorPenalty: 3,
      }),
    ).toBe(7)
  })

  it('Penalidade não aplica se armorPenaltyApplies=false (perícia não sensível)', () => {
    expect(
      skillValue({
        level: 10,
        attributeValue: 3,
        trained: true,
        armorPenaltyApplies: false,
        armorPenalty: 5,
      }),
    ).toBe(10)
  })
})
