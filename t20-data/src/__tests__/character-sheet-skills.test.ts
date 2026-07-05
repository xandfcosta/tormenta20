import { describe, expect, it } from 'vitest'
import { computeCharacterSheet } from '../character-sheet'

/**
 * v2 orchestrator: perícias com valores totais aplicando fórmula T20
 * (½ nível + atributo + treino - pen armadura).
 */

describe('computeCharacterSheet — perícias sem treino', () => {
  const sheet = computeCharacterSheet({
    level: 10,
    className: 'Guerreiro',
    baseAttributes: {
      strength: 4,
      dexterity: 2,
      constitution: 3,
      intelligence: 1,
      wisdom: 0,
      charisma: 0,
    },
  })

  it('Luta (FOR) sem treino = 4 (só atributo)', () => {
    expect(sheet.skills.luta.total).toBe(4)
    expect(sheet.skills.luta.trained).toBe(false)
  })

  it('Iniciativa (DES) sem treino = 2', () => {
    expect(sheet.skills.iniciativa.total).toBe(2)
  })

  it('Percepção (SAB) sem treino = 0', () => {
    expect(sheet.skills.percepcao.total).toBe(0)
  })
})

describe('computeCharacterSheet — perícias treinadas L10', () => {
  const sheet = computeCharacterSheet({
    level: 10,
    className: 'Ladino',
    baseAttributes: {
      strength: 0,
      dexterity: 4,
      constitution: 2,
      intelligence: 3,
      wisdom: 1,
      charisma: 2,
    },
    trainedSkills: ['ladinagem', 'furtividade', 'percepcao', 'iniciativa'],
  })

  it('Ladinagem treinada L10 DES 4 = 5 + 4 + 2 = 11', () => {
    expect(sheet.skills.ladinagem.total).toBe(11)
    expect(sheet.skills.ladinagem.trained).toBe(true)
    expect(sheet.skills.ladinagem.cannotUse).toBe(false)
  })

  it('Furtividade treinada L10 DES 4 = 11', () => {
    expect(sheet.skills.furtividade.total).toBe(11)
  })

  it('Percepção treinada L10 SAB 1 = 5 + 1 + 2 = 8', () => {
    expect(sheet.skills.percepcao.total).toBe(8)
  })
})

describe('computeCharacterSheet — perícia treinada-apenas sem treino', () => {
  const sheet = computeCharacterSheet({
    level: 5,
    className: 'Guerreiro',
    baseAttributes: {
      strength: 3,
      dexterity: 2,
      constitution: 2,
      intelligence: 1,
      wisdom: 0,
      charisma: 0,
    },
  })

  it('Misticismo (treinada-apenas) sem treino → cannotUse=true, total = INT 1', () => {
    expect(sheet.skills.misticismo.cannotUse).toBe(true)
    expect(sheet.skills.misticismo.trained).toBe(false)
    expect(sheet.skills.misticismo.total).toBe(1)
  })

  it('Ladinagem (treinada-apenas) sem treino → cannotUse=true', () => {
    expect(sheet.skills.ladinagem.cannotUse).toBe(true)
  })

  it('Investigação (não-treinada-apenas) sem treino → cannotUse=false', () => {
    expect(sheet.skills.investigacao.cannotUse).toBe(false)
  })
})

describe('computeCharacterSheet — armor penalty', () => {
  const sheet = computeCharacterSheet({
    level: 10,
    className: 'Ladino',
    baseAttributes: {
      strength: 0,
      dexterity: 4,
      constitution: 2,
      intelligence: 3,
      wisdom: 1,
      charisma: 2,
    },
    trainedSkills: ['ladinagem', 'furtividade'],
    armorPenalty: 3,
  })

  it('Ladinagem (sofre pen) treinada L10 DES 4 pen 3 = 11 - 3 = 8', () => {
    expect(sheet.skills.ladinagem.total).toBe(8)
    expect(sheet.skills.ladinagem.armorPenaltyApplied).toBe(3)
  })

  it('Furtividade (sofre pen) mesma penalidade aplicada', () => {
    expect(sheet.skills.furtividade.total).toBe(8)
  })

  it('Acrobacia (sofre pen) sem treino: DES 4 - 3 = 1', () => {
    expect(sheet.skills.acrobacia.total).toBe(1)
    expect(sheet.skills.acrobacia.armorPenaltyApplied).toBe(3)
  })

  it('Iniciativa (não sofre pen) DES 4 sem treino = 4 (imutado)', () => {
    expect(sheet.skills.iniciativa.total).toBe(4)
    expect(sheet.skills.iniciativa.armorPenaltyApplied).toBe(0)
  })
})

describe('computeCharacterSheet — perícia treinada desconhecida gera warning', () => {
  const sheet = computeCharacterSheet({
    level: 1,
    className: 'Guerreiro',
    baseAttributes: {
      strength: 3,
      dexterity: 2,
      constitution: 2,
      intelligence: 0,
      wisdom: 1,
      charisma: 0,
    },
    // @ts-expect-error — inválido de propósito
    trainedSkills: ['inexistente'],
  })

  it('warning para perícia desconhecida', () => {
    expect(sheet.warnings.some((w) => w.match(/desconhecida/))).toBe(true)
  })
})

describe('computeCharacterSheet — armorPenalty negativa gera warning e clampa a 0', () => {
  const sheet = computeCharacterSheet({
    level: 5,
    className: 'Guerreiro',
    baseAttributes: {
      strength: 3,
      dexterity: 2,
      constitution: 2,
      intelligence: 0,
      wisdom: 1,
      charisma: 0,
    },
    armorPenalty: -5,
  })

  it('warning', () => {
    expect(
      sheet.warnings.some((w) => w.match(/armorPenalty.*não-negativa/)),
    ).toBe(true)
  })

  it('penalty clampado a 0 — Furtividade DES 2 = 2', () => {
    expect(sheet.skills.furtividade.total).toBe(2)
    expect(sheet.skills.furtividade.armorPenaltyApplied).toBe(0)
  })
})

describe('computeCharacterSheet — todas 29 perícias no output', () => {
  const sheet = computeCharacterSheet({
    level: 1,
    className: 'Guerreiro',
    baseAttributes: {
      strength: 2,
      dexterity: 2,
      constitution: 2,
      intelligence: 2,
      wisdom: 2,
      charisma: 2,
    },
  })

  it('29 chaves em skills', () => {
    expect(Object.keys(sheet.skills).length).toBe(29)
  })

  it('cada skill tem keyAttribute + trained + total + cannotUse', () => {
    for (const [id, s] of Object.entries(sheet.skills)) {
      expect(typeof s.total).toBe('number')
      expect(typeof s.trained).toBe('boolean')
      expect(typeof s.cannotUse).toBe('boolean')
      expect(s.keyAttribute).toBeTruthy()
      expect(id).toBeTruthy()
    }
  })
})
