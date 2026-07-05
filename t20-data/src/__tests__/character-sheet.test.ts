import { describe, expect, it } from 'vitest'
import {
  DEFENSE_BASE,
  computeCharacterSheet,
  halfLevel,
} from '../character-sheet'

/**
 * MVP orchestrator — pinns comportamentos essenciais para uma folha
 * de personagem calculada:
 *   - Atributos totais aplicam mod racial
 *   - PV = pvInicial + (nível-1) × pvPerLevel + CON × nível (por classe)
 *   - PM = mpPerLevel × nível (+ CAR para Paladino L1)
 *   - Defesa = 10 + DES (sem armadura/escudo em v1)
 *   - Saves = ½ nível + atributo-chave (CON/DES/SAB)
 *   - Warnings coletados para inputs inválidos (não lança)
 */

describe('halfLevel — ½ nível arredondado para baixo', () => {
  it('nível 1 → 0', () => {
    expect(halfLevel(1)).toBe(0)
  })

  it('nível 2 → 1', () => {
    expect(halfLevel(2)).toBe(1)
  })

  it('nível 10 → 5', () => {
    expect(halfLevel(10)).toBe(5)
  })

  it('nível 20 → 10', () => {
    expect(halfLevel(20)).toBe(10)
  })
})

describe('computeCharacterSheet — Guerreiro L1 humano', () => {
  const sheet = computeCharacterSheet({
    level: 1,
    className: 'Guerreiro',
    raceId: 'humano',
    raceFloatingPicks: ['strength', 'constitution', 'wisdom'],
    baseAttributes: {
      strength: 3,
      dexterity: 2,
      constitution: 2,
      intelligence: 0,
      wisdom: 1,
      charisma: 0,
    },
  })

  it('atributos totais somam mod racial (Humano +1 em 3 escolhidos)', () => {
    expect(sheet.attributes.strength.total).toBe(4) // 3 base + 1 humano
    expect(sheet.attributes.constitution.total).toBe(3) // 2 + 1
    expect(sheet.attributes.wisdom.total).toBe(2) // 1 + 1
    expect(sheet.attributes.intelligence.total).toBe(0) // 0 + 0
  })

  it('PV = 20 (Guerreiro pvInicial) + 0 (L1) + CON 3 × 1 = 23', () => {
    expect(sheet.vitals.pvMax).toBe(23)
  })

  it('PM = 3 (Guerreiro mpPerLevel) × 1 = 3', () => {
    expect(sheet.vitals.pmMax).toBe(3)
  })

  it('Defesa = 10 + DES 2 = 12', () => {
    expect(sheet.defense.total).toBe(12)
    expect(sheet.defense.base).toBe(DEFENSE_BASE)
    expect(sheet.defense.attribute).toBe(2)
  })

  it('saves = 0 (½ nível) + atributo-chave', () => {
    expect(sheet.saves.fortitude).toBe(3) // 0 + CON 3
    expect(sheet.saves.reflexos).toBe(2) // 0 + DES 2
    expect(sheet.saves.vontade).toBe(2) // 0 + SAB 2
  })

  it('sem warnings', () => {
    expect(sheet.warnings).toEqual([])
  })
})

describe('computeCharacterSheet — Paladino L1 humano (CAR extra em PM)', () => {
  const sheet = computeCharacterSheet({
    level: 1,
    className: 'Paladino',
    raceId: 'humano',
    raceFloatingPicks: ['strength', 'charisma', 'constitution'],
    baseAttributes: {
      strength: 3,
      dexterity: 1,
      constitution: 2,
      intelligence: 0,
      wisdom: 1,
      charisma: 2,
    },
  })

  it('PM = 3 × 1 + CAR 3 (bônus L1 uma vez) = 6', () => {
    expect(sheet.vitals.pmMax).toBe(6)
  })
})

describe('computeCharacterSheet — Guerreiro L10 anão', () => {
  const sheet = computeCharacterSheet({
    level: 10,
    className: 'Guerreiro',
    raceId: 'anao',
    baseAttributes: {
      strength: 4,
      dexterity: 2,
      constitution: 2,
      intelligence: 0,
      wisdom: 1,
      charisma: 0,
    },
  })

  it('Anão: CON +2, SAB +1, DES -1', () => {
    expect(sheet.attributes.constitution.total).toBe(4)
    expect(sheet.attributes.wisdom.total).toBe(2)
    expect(sheet.attributes.dexterity.total).toBe(1)
  })

  it('PV = 20 + 9 × 5 + CON 4 × 10 = 105', () => {
    expect(sheet.vitals.pvMax).toBe(105)
  })

  it('PM = 3 × 10 = 30', () => {
    expect(sheet.vitals.pmMax).toBe(30)
  })

  it('Defesa = 10 + DES 1 (com -1 anão) = 11', () => {
    expect(sheet.defense.total).toBe(11)
  })

  it('saves = 5 (½ 10) + atributo', () => {
    expect(sheet.saves.fortitude).toBe(9)
    expect(sheet.saves.reflexos).toBe(6)
    expect(sheet.saves.vontade).toBe(7)
  })

  it('deslocamento anão = 6m', () => {
    expect(sheet.deslocamento).toBe(6)
  })

  it('tamanho anão = Médio', () => {
    expect(sheet.tamanho).toBe('Médio')
  })
})

describe('computeCharacterSheet — sem raça (baseline)', () => {
  const sheet = computeCharacterSheet({
    level: 5,
    className: 'Arcanista',
    baseAttributes: {
      strength: 0,
      dexterity: 2,
      constitution: 1,
      intelligence: 4,
      wisdom: 0,
      charisma: 1,
    },
  })

  it('atributos totais = base (raceMod = 0)', () => {
    expect(sheet.attributes.intelligence.total).toBe(4)
    expect(sheet.attributes.intelligence.raceMod).toBe(0)
  })

  it('deslocamento default = 9m', () => {
    expect(sheet.deslocamento).toBe(9)
  })

  it('tamanho default = Médio', () => {
    expect(sheet.tamanho).toBe('Médio')
  })

  it('PV = 8 (Arcanista) + 4 × 2 + CON 1 × 5 = 21', () => {
    expect(sheet.vitals.pvMax).toBe(21)
  })
})

describe('computeCharacterSheet — inputs inválidos geram warnings, não lançam', () => {
  it('classe desconhecida → warning + vitals zerados', () => {
    const sheet = computeCharacterSheet({
      level: 1,
      className: 'NaoExiste',
      baseAttributes: {
        strength: 2,
        dexterity: 2,
        constitution: 2,
        intelligence: 2,
        wisdom: 2,
        charisma: 2,
      },
    })
    expect(sheet.warnings.some((w) => w.match(/classe desconhecida/))).toBe(true)
    expect(sheet.vitals.pvMax).toBe(0)
    expect(sheet.vitals.pmMax).toBe(0)
  })

  it('raça desconhecida → warning + raceMod zero', () => {
    const sheet = computeCharacterSheet({
      level: 1,
      className: 'Guerreiro',
      raceId: 'inexistente',
      baseAttributes: {
        strength: 3,
        dexterity: 2,
        constitution: 2,
        intelligence: 0,
        wisdom: 1,
        charisma: 0,
      },
    })
    expect(sheet.warnings.some((w) => w.match(/raça desconhecida/))).toBe(true)
    expect(sheet.attributes.strength.raceMod).toBe(0)
  })

  it('Humano sem floatingPicks → warning + raceMod zero', () => {
    const sheet = computeCharacterSheet({
      level: 1,
      className: 'Guerreiro',
      raceId: 'humano',
      baseAttributes: {
        strength: 3,
        dexterity: 2,
        constitution: 2,
        intelligence: 0,
        wisdom: 1,
        charisma: 0,
      },
    })
    expect(sheet.warnings.some((w) => w.match(/mod racial/))).toBe(true)
    expect(sheet.attributes.strength.raceMod).toBe(0)
  })

  it('nível fora de range 1-20 → warning', () => {
    const sheet = computeCharacterSheet({
      level: 0,
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
    expect(sheet.warnings.some((w) => w.match(/nível fora do range/))).toBe(true)
  })
})

describe('computeCharacterSheet — currentPv/Pm respeitam o max', () => {
  const sheet = computeCharacterSheet({
    level: 5,
    className: 'Guerreiro',
    baseAttributes: {
      strength: 3,
      dexterity: 2,
      constitution: 3,
      intelligence: 0,
      wisdom: 1,
      charisma: 0,
    },
    currentPv: 999,
    currentPm: 999,
  })

  it('clampa currentPv ao pvMax', () => {
    expect(sheet.vitals.pvCurrent).toBeLessThanOrEqual(sheet.vitals.pvMax)
    expect(sheet.vitals.pvCurrent).toBe(sheet.vitals.pvMax)
  })

  it('clampa currentPm ao pmMax', () => {
    expect(sheet.vitals.pmCurrent).toBeLessThanOrEqual(sheet.vitals.pmMax)
  })

  it('default currentPv = pvMax quando omitido', () => {
    const s = computeCharacterSheet({
      level: 3,
      className: 'Bardo',
      baseAttributes: {
        strength: 0,
        dexterity: 2,
        constitution: 1,
        intelligence: 1,
        wisdom: 0,
        charisma: 3,
      },
    })
    expect(s.vitals.pvCurrent).toBe(s.vitals.pvMax)
    expect(s.vitals.pmCurrent).toBe(s.vitals.pmMax)
  })
})
