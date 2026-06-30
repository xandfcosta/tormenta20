import { describe, expect, it } from 'vitest'
import {
  AVERAGE_TREASURE_PER_SCENE,
  COMMON_SERVICES_TABLE,
  DIFFICULTY_TABLE,
  ENCHANTMENT_PRICE_TABLE,
  MAGIC_ITEM_CRAFT_CD,
  MAGIC_ITEM_CRAFT_PM_SACRIFICE,
  REST_RECOVERY_TABLE,
  STARTING_WEALTH_TABLE,
  averageTreasurePerScene,
  difficultyForCd,
  paidMagicCostTibar,
  restRecoveryPerNight,
  startingWealthForLevel,
} from '../reward-tables'

/**
 * PDF Cap 8 (p326-332) + Cap 3 (p140, p157) + Cap 5 (p220) + p106.
 * Pinned:
 *  - Tabela 5-1: salto 30 → 40 sem CD 35.
 *  - Tabela 3-1: L1 = 4d6 T$ (média 14), L20 = T$ 260.000.
 *  - Tabela 8-6: pacing 300 → 72.000 T$/cena.
 *  - Tabela 3-6: 11 serviços + magia paga 1°-3° círculo.
 *  - p106: 4 condições de descanso com multiplicadores 0.5/1/2/3.
 *  - Tabela 8-7: 1/2/3 encantos = 18k/36k/72k T$ + CD +10/+15/+20.
 */

describe('DIFFICULTY_TABLE — Tabela 5-1 p220', () => {
  it('tem exatamente 7 tiers (Fácil → Quase Impossível)', () => {
    expect(DIFFICULTY_TABLE.length).toBe(7)
  })

  it('CD vão de 5 a 40 com salto 30 → 40 (sem CD 35)', () => {
    const cds = DIFFICULTY_TABLE.map((r) => r.cd)
    expect(cds).toEqual([5, 10, 15, 20, 25, 30, 40])
  })

  it('Quase Impossível CD 40 (não 35)', () => {
    const qi = DIFFICULTY_TABLE.find((r) => r.tier === 'quase-impossivel')!
    expect(qi.cd).toBe(40)
  })

  it('é frozen', () => {
    expect(Object.isFrozen(DIFFICULTY_TABLE)).toBe(true)
  })
})

describe('difficultyForCd — resolve tier por CD', () => {
  it('CD 5 → Fácil', () => {
    expect(difficultyForCd(5).tier).toBe('facil')
  })

  it('CD 15 → Difícil', () => {
    expect(difficultyForCd(15).tier).toBe('dificil')
  })

  it('CD 22 → Desafiadora (CD 20 é o último ≤ 22)', () => {
    expect(difficultyForCd(22).tier).toBe('desafiadora')
  })

  it('CD 35 → Heroica (próximo é Quase Impossível CD 40)', () => {
    expect(difficultyForCd(35).tier).toBe('heroica')
  })

  it('CD 100 → Quase Impossível (clamp ao topo)', () => {
    expect(difficultyForCd(100).tier).toBe('quase-impossivel')
  })

  it('CD 0 → Fácil (clamp ao piso)', () => {
    expect(difficultyForCd(0).tier).toBe('facil')
  })
})

describe('STARTING_WEALTH_TABLE — Tabela 3-1 p140', () => {
  it('20 linhas (L1-L20)', () => {
    expect(STARTING_WEALTH_TABLE.length).toBe(20)
  })

  it('L1 = 14 T$ (média 4d6) e isDiced=true', () => {
    const l1 = STARTING_WEALTH_TABLE[0]!
    expect(l1.level).toBe(1)
    expect(l1.startingTibar).toBe(14)
    expect(l1.isDiced).toBe(true)
  })

  it('apenas L1 tem isDiced=true', () => {
    for (const row of STARTING_WEALTH_TABLE) {
      if (row.level === 1) expect(row.isDiced).toBe(true)
      else expect(row.isDiced).toBe(false)
    }
  })

  it('L20 = T$ 260.000 (máximo)', () => {
    const l20 = STARTING_WEALTH_TABLE[19]!
    expect(l20.level).toBe(20)
    expect(l20.startingTibar).toBe(260000)
  })

  it('progressão monotônica', () => {
    for (let i = 1; i < STARTING_WEALTH_TABLE.length; i++) {
      expect(STARTING_WEALTH_TABLE[i]!.startingTibar).toBeGreaterThan(
        STARTING_WEALTH_TABLE[i - 1]!.startingTibar,
      )
    }
  })

  it('startingWealthForLevel clamp [1, 20]', () => {
    expect(startingWealthForLevel(1)).toBe(14)
    expect(startingWealthForLevel(10)).toBe(13000)
    expect(startingWealthForLevel(20)).toBe(260000)
    expect(startingWealthForLevel(0)).toBe(14)
    expect(startingWealthForLevel(25)).toBe(260000)
  })
})

describe('AVERAGE_TREASURE_PER_SCENE — Tabela 8-6 p332', () => {
  it('20 linhas (L1-L20)', () => {
    expect(AVERAGE_TREASURE_PER_SCENE.length).toBe(20)
  })

  it('L1 = T$ 300', () => {
    expect(AVERAGE_TREASURE_PER_SCENE[0]!.treasureTibar).toBe(300)
  })

  it('L20 = T$ 72.000', () => {
    expect(AVERAGE_TREASURE_PER_SCENE[19]!.treasureTibar).toBe(72000)
  })

  it('L15 e L16 ambos T$ 22.000 (PDF repete valor)', () => {
    expect(AVERAGE_TREASURE_PER_SCENE[14]!.treasureTibar).toBe(22000)
    expect(AVERAGE_TREASURE_PER_SCENE[15]!.treasureTibar).toBe(22000)
  })

  it('averageTreasurePerScene clamp [1, 20]', () => {
    expect(averageTreasurePerScene(5)).toBe(1000)
    expect(averageTreasurePerScene(0)).toBe(300)
    expect(averageTreasurePerScene(99)).toBe(72000)
  })
})

describe('COMMON_SERVICES_TABLE — Tabela 3-6 p157', () => {
  it('tem 11 entradas', () => {
    expect(COMMON_SERVICES_TABLE.length).toBe(11)
  })

  it('Estadia comum T$ 0,5/noite', () => {
    const e = COMMON_SERVICES_TABLE.find(
      (s) => s.service === 'Estadia comum',
    )!
    expect(e.priceTibar).toBe(0.5)
    expect(e.unit).toBe('por-noite')
  })

  it('Estadia luxuosa T$ 20/noite', () => {
    const e = COMMON_SERVICES_TABLE.find(
      (s) => s.service === 'Estadia luxuosa',
    )!
    expect(e.priceTibar).toBe(20)
  })

  it('Condução aérea T$ 10/km (mais cara)', () => {
    const a = COMMON_SERVICES_TABLE.find(
      (s) => s.service === 'Condução aérea',
    )!
    expect(a.priceTibar).toBe(10)
  })

  it('Magia 3° círculo: T$ 360/uso', () => {
    expect(paidMagicCostTibar(3)).toBe(360)
  })

  it('Magia 1° círculo: T$ 10/uso', () => {
    expect(paidMagicCostTibar(1)).toBe(10)
  })

  it('Magia 4°-5° círculo: null (não tabulado)', () => {
    expect(paidMagicCostTibar(4)).toBeNull()
    expect(paidMagicCostTibar(5)).toBeNull()
  })

  it('Magia 0 (truque): null', () => {
    expect(paidMagicCostTibar(0)).toBeNull()
  })
})

describe('REST_RECOVERY_TABLE — p106', () => {
  it('4 qualidades (ruim, normal, confortavel, luxuosa)', () => {
    expect(REST_RECOVERY_TABLE.length).toBe(4)
    const qs = REST_RECOVERY_TABLE.map((r) => r.quality)
    expect(qs).toEqual(['ruim', 'normal', 'confortavel', 'luxuosa'])
  })

  it('multiplicadores 0.5 / 1.0 / 2.0 / 3.0', () => {
    expect(REST_RECOVERY_TABLE.map((r) => r.recoveryMultiplier)).toEqual([
      0.5, 1.0, 2.0, 3.0,
    ])
  })

  it('Ruim: dormir ao relento, sem estadia paga', () => {
    expect(REST_RECOVERY_TABLE[0]!.pairedEstadiaTibar).toBeNull()
  })

  it('Luxuosa: T$ 20/noite (bate com Estadia Luxuosa)', () => {
    expect(REST_RECOVERY_TABLE[3]!.pairedEstadiaTibar).toBe(20)
  })

  it('restRecoveryPerNight: ruim L4 = 2 PV (floor de 0.5 × 4)', () => {
    expect(restRecoveryPerNight('ruim', 4)).toBe(2)
  })

  it('restRecoveryPerNight: normal L5 = 5', () => {
    expect(restRecoveryPerNight('normal', 5)).toBe(5)
  })

  it('restRecoveryPerNight: confortável L10 = 20', () => {
    expect(restRecoveryPerNight('confortavel', 10)).toBe(20)
  })

  it('restRecoveryPerNight: luxuosa L20 = 60', () => {
    expect(restRecoveryPerNight('luxuosa', 20)).toBe(60)
  })
})

describe('ENCHANTMENT_PRICE_TABLE — Tabela 8-7 p334', () => {
  it('3 linhas (1, 2, 3 encantos)', () => {
    expect(ENCHANTMENT_PRICE_TABLE.length).toBe(3)
  })

  it('1 encanto = T$ 18.000 + CD +10', () => {
    const row = ENCHANTMENT_PRICE_TABLE[0]!
    expect(row.enchantmentCount).toBe(1)
    expect(row.priceIncreaseTibar).toBe(18000)
    expect(row.cdIncrease).toBe(10)
  })

  it('2 encantos = T$ 36.000 + CD +15', () => {
    const row = ENCHANTMENT_PRICE_TABLE[1]!
    expect(row.priceIncreaseTibar).toBe(36000)
    expect(row.cdIncrease).toBe(15)
  })

  it('3 encantos = T$ 72.000 + CD +20', () => {
    const row = ENCHANTMENT_PRICE_TABLE[2]!
    expect(row.priceIncreaseTibar).toBe(72000)
    expect(row.cdIncrease).toBe(20)
  })

  it('preço dobra de 1→2 encantos (18k → 36k)', () => {
    expect(ENCHANTMENT_PRICE_TABLE[1]!.priceIncreaseTibar).toBe(
      ENCHANTMENT_PRICE_TABLE[0]!.priceIncreaseTibar * 2,
    )
  })

  it('preço quadruplica de 1→3 encantos (18k → 72k)', () => {
    expect(ENCHANTMENT_PRICE_TABLE[2]!.priceIncreaseTibar).toBe(
      ENCHANTMENT_PRICE_TABLE[0]!.priceIncreaseTibar * 4,
    )
  })
})

describe('MAGIC_ITEM_CRAFT_CD / PM_SACRIFICE — p334', () => {
  it('CD escalona 30 / 40 / 50 por tier', () => {
    expect(MAGIC_ITEM_CRAFT_CD.menor).toBe(30)
    expect(MAGIC_ITEM_CRAFT_CD.medio).toBe(40)
    expect(MAGIC_ITEM_CRAFT_CD.maior).toBe(50)
  })

  it('PM sacrificado escalona 1 / 2 / 3', () => {
    expect(MAGIC_ITEM_CRAFT_PM_SACRIFICE.menor).toBe(1)
    expect(MAGIC_ITEM_CRAFT_PM_SACRIFICE.medio).toBe(2)
    expect(MAGIC_ITEM_CRAFT_PM_SACRIFICE.maior).toBe(3)
  })
})
