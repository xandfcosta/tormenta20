import { describe, expect, it } from 'vitest'
import {
  AUTO_STABILIZATION_CONSTITUICAO_CD,
  COUP_DE_GRACE_ACTION,
  COUP_DE_GRACE_AUTO_CRITICAL,
  COUP_DE_GRACE_INSTANT_DEATH_CHANCE,
  DEATH_ABSOLUTE_THRESHOLD,
  EXTERNAL_STABILIZATION_CURA_CD,
  NONLETHAL_CONVERSION_ATTACK_PENALTY,
  applyHealingPriority,
  autoStabilizationSucceeds,
  coupDeGraceInstantDeathChance,
  coupDeGraceKills,
  deathThreshold,
  externalStabilizationSucceeds,
  healingRestoresConsciousness,
  healthState,
  isDead,
  nonlethalKnocksOut,
} from '../ferimentos-e-morte'

/**
 * PDF livro p235-236 — Ferimentos & Morte, Dano Não Letal, Golpe de Misericórdia.
 */

describe('Constantes', () => {
  it('valores verbatim livro', () => {
    expect(DEATH_ABSOLUTE_THRESHOLD).toBe(-10)
    expect(AUTO_STABILIZATION_CONSTITUICAO_CD).toBe(15)
    expect(EXTERNAL_STABILIZATION_CURA_CD).toBe(15)
    expect(NONLETHAL_CONVERSION_ATTACK_PENALTY).toBe(-5)
    expect(COUP_DE_GRACE_ACTION).toBe('completa')
    expect(COUP_DE_GRACE_AUTO_CRITICAL).toBe(true)
  })

  it('COUP_DE_GRACE_INSTANT_DEATH_CHANCE frozen + valores', () => {
    expect(Object.isFrozen(COUP_DE_GRACE_INSTANT_DEATH_CHANCE)).toBe(true)
    expect(COUP_DE_GRACE_INSTANT_DEATH_CHANCE['pc-ou-npc-importante']).toBe(
      0.25,
    )
    expect(COUP_DE_GRACE_INSTANT_DEATH_CHANCE['npc-secundario']).toBe(0.75)
  })
})

describe('deathThreshold — exemplos do livro p236', () => {
  it('12 PV → -10 (metade seria -6, mais raso; usa -10)', () => {
    expect(deathThreshold(12)).toBe(-10)
  })

  it('30 PV → -15 (metade -15, mais negativo que -10)', () => {
    expect(deathThreshold(30)).toBe(-15)
  })

  it('20 PV → -10 (metade seria -10; empate = -10)', () => {
    expect(deathThreshold(20)).toBe(-10)
  })

  it('21 PV → -10 (metade seria -10.5 → floor -10)', () => {
    expect(deathThreshold(21)).toBe(-10)
  })

  it('22 PV → -11 (metade -11)', () => {
    expect(deathThreshold(22)).toBe(-11)
  })

  it('100 PV → -50', () => {
    expect(deathThreshold(100)).toBe(-50)
  })

  it('maxPv 0 lança', () => {
    expect(() => deathThreshold(0)).toThrow(/maxPv must be > 0/)
  })
})

describe('isDead', () => {
  it('12 PV, -11 → dead (abaixo de -10)', () => {
    expect(isDead(-11, 12)).toBe(true)
  })

  it('12 PV, -10 → dead (limite exato)', () => {
    expect(isDead(-10, 12)).toBe(true)
  })

  it('12 PV, -9 → alive', () => {
    expect(isDead(-9, 12)).toBe(false)
  })

  it('30 PV, -14 → alive (limiar é -15)', () => {
    expect(isDead(-14, 30)).toBe(false)
  })

  it('30 PV, -15 → dead', () => {
    expect(isDead(-15, 30)).toBe(true)
  })
})

describe('healthState', () => {
  it('PV positivo → saudavel', () => {
    expect(healthState(10, 20)).toBe('saudavel')
    expect(healthState(1, 20)).toBe('saudavel')
  })

  it('PV entre 0 e death threshold, não estabilizado → sangrando', () => {
    expect(healthState(0, 20)).toBe('inconsciente-sangrando')
    expect(healthState(-5, 20)).toBe('inconsciente-sangrando')
  })

  it('PV entre 0 e death threshold, estabilizado → estabilizado', () => {
    expect(healthState(0, 20, true)).toBe('estabilizado')
    expect(healthState(-5, 20, true)).toBe('estabilizado')
  })

  it('PV no death threshold → morto', () => {
    expect(healthState(-10, 20)).toBe('morto')
    expect(healthState(-15, 30)).toBe('morto')
  })

  it('morto ignora flag stabilized', () => {
    expect(healthState(-11, 20, true)).toBe('morto')
  })
})

describe('healingRestoresConsciousness', () => {
  it('PV ≥ 1 → recupera consciência', () => {
    expect(healingRestoresConsciousness(1)).toBe(true)
    expect(healingRestoresConsciousness(5)).toBe(true)
  })

  it('PV 0 ou negativo → segue inconsciente', () => {
    expect(healingRestoresConsciousness(0)).toBe(false)
    expect(healingRestoresConsciousness(-3)).toBe(false)
  })
})

describe('autoStabilizationSucceeds — Con CD 15', () => {
  it('roll ≥ 15 → passa', () => {
    expect(autoStabilizationSucceeds(15)).toBe(true)
    expect(autoStabilizationSucceeds(20)).toBe(true)
  })

  it('roll < 15 → falha (perde 1d6 no turno)', () => {
    expect(autoStabilizationSucceeds(14)).toBe(false)
  })
})

describe('externalStabilizationSucceeds — Cura CD 15', () => {
  it('roll ≥ 15 → passa', () => {
    expect(externalStabilizationSucceeds(15)).toBe(true)
  })

  it('roll < 15 → falha', () => {
    expect(externalStabilizationSucceeds(14)).toBe(false)
  })
})

describe('applyHealingPriority — letal primeiro, não-letal depois', () => {
  it('cura menor que letal → só letal', () => {
    expect(applyHealingPriority(10, 5, 4)).toEqual({
      lethalHealed: 4,
      nonlethalHealed: 0,
    })
  })

  it('cura exata do letal → cura só letal', () => {
    expect(applyHealingPriority(10, 5, 10)).toEqual({
      lethalHealed: 10,
      nonlethalHealed: 0,
    })
  })

  it('cura excede letal → resto vai para não-letal', () => {
    expect(applyHealingPriority(10, 5, 12)).toEqual({
      lethalHealed: 10,
      nonlethalHealed: 2,
    })
  })

  it('cura excede letal + não-letal → resto perdido', () => {
    expect(applyHealingPriority(10, 5, 30)).toEqual({
      lethalHealed: 10,
      nonlethalHealed: 5,
    })
  })

  it('sem letal → cura vai direto para não-letal', () => {
    expect(applyHealingPriority(0, 8, 5)).toEqual({
      lethalHealed: 0,
      nonlethalHealed: 5,
    })
  })

  it('valores negativos lançam', () => {
    expect(() => applyHealingPriority(-1, 5, 3)).toThrow(/values must be ≥ 0/)
    expect(() => applyHealingPriority(5, 5, -1)).toThrow(/values must be ≥ 0/)
  })
})

describe('nonlethalKnocksOut', () => {
  it('não-letal ≥ PV atual → derruba', () => {
    expect(nonlethalKnocksOut(10, 10)).toBe(true)
    expect(nonlethalKnocksOut(15, 10)).toBe(true)
  })

  it('não-letal < PV → só machuca', () => {
    expect(nonlethalKnocksOut(5, 10)).toBe(false)
  })
})

describe('coupDeGraceInstantDeathChance', () => {
  it('PC/importante → 25%', () => {
    expect(coupDeGraceInstantDeathChance('pc-ou-npc-importante')).toBe(0.25)
  })

  it('NPC secundário → 75%', () => {
    expect(coupDeGraceInstantDeathChance('npc-secundario')).toBe(0.75)
  })
})

describe('coupDeGraceKills — 1d4 vs categoria', () => {
  it('PC/importante: 1 mata, 2-4 não', () => {
    expect(coupDeGraceKills(1, 'pc-ou-npc-importante')).toBe(true)
    expect(coupDeGraceKills(2, 'pc-ou-npc-importante')).toBe(false)
    expect(coupDeGraceKills(3, 'pc-ou-npc-importante')).toBe(false)
    expect(coupDeGraceKills(4, 'pc-ou-npc-importante')).toBe(false)
  })

  it('NPC secundário: 1-3 matam, 4 não', () => {
    expect(coupDeGraceKills(1, 'npc-secundario')).toBe(true)
    expect(coupDeGraceKills(2, 'npc-secundario')).toBe(true)
    expect(coupDeGraceKills(3, 'npc-secundario')).toBe(true)
    expect(coupDeGraceKills(4, 'npc-secundario')).toBe(false)
  })
})
