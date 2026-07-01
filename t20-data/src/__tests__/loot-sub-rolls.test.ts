import { describe, expect, it } from 'vitest'
import { mulberry32 } from '../loot-rng'
import {
  ARMADURA_ROWS,
  ARMA_ROWS,
  DIVERSOS_ROWS,
  ESOTERICO_ROWS,
  MATERIAL_ROLL_TABLE,
  SUPERIOR_ARMADURA_ROWS,
  SUPERIOR_ARMA_ROWS,
  SUPERIOR_ESOTERICO_ROWS,
  rollDiverso,
  rollEquipKind,
  rollEquipamento,
  rollSuperior,
} from '../loot-sub-rolls'

/**
 * PDF Cap 8 Recompensas, p331-332 (Tabelas 8-3 / 8-4 / 8-5). Pinned:
 *  - Cobertura d% [1, 100] sem gaps em todas 7 sub-tabelas.
 *  - Kind routing 1d6: 1-3 arma, 4-5 armadura, 6 esotérico.
 *  - Superior footnote 1: countsAsTwo com 1 melhoria → reroll.
 *  - Superior footnote 2: "Material especial" → material 1d6.
 *  - Determinismo total: mesma seed → mesmo output.
 */

const ALL_KINDS = ['arma', 'armadura', 'esoterico'] as const

describe('DIVERSOS_ROWS — Tabela 8-3', () => {
  it('cobre [1, 100] sem gaps', () => {
    let cursor = 1
    for (const row of DIVERSOS_ROWS) {
      expect(row[0]).toBe(cursor)
      cursor = row[1] + 1
    }
    expect(cursor).toBe(101)
  })

  it('primeira linha 1-2 = Ácido, última 100 = Veste de seda', () => {
    expect(DIVERSOS_ROWS[0]).toEqual([1, 2, 'Ácido'])
    expect(DIVERSOS_ROWS.at(-1)).toEqual([100, 100, 'Veste de seda'])
  })

  it('frozen', () => {
    expect(Object.isFrozen(DIVERSOS_ROWS)).toBe(true)
  })
})

describe('ARMA_ROWS — Tabela 8-4 arma', () => {
  it('cobre [1, 100] sem gaps', () => {
    let cursor = 1
    for (const row of ARMA_ROWS) {
      expect(row[0]).toBe(cursor)
      cursor = row[1] + 1
    }
    expect(cursor).toBe(101)
  })

  it('adaga em 1-3', () => {
    expect(ARMA_ROWS[0]).toEqual([1, 3, 'Adaga'])
  })
})

describe('ARMADURA_ROWS — Tabela 8-4 armadura/escudo', () => {
  it('cobre [1, 100] sem gaps', () => {
    let cursor = 1
    for (const row of ARMADURA_ROWS) {
      expect(row[0]).toBe(cursor)
      cursor = row[1] + 1
    }
    expect(cursor).toBe(101)
  })

  it('inclui escudo leve e pesado', () => {
    const names = ARMADURA_ROWS.map((r) => r[2])
    expect(names).toContain('Escudo leve')
    expect(names).toContain('Escudo pesado')
  })
})

describe('ESOTERICO_ROWS — Tabela 8-4 esotérico', () => {
  it('cobre [1, 100] sem gaps', () => {
    let cursor = 1
    for (const row of ESOTERICO_ROWS) {
      expect(row[0]).toBe(cursor)
      cursor = row[1] + 1
    }
    expect(cursor).toBe(101)
  })

  it('tem 10 entradas', () => {
    expect(ESOTERICO_ROWS.length).toBe(10)
  })
})

describe('SUPERIOR sub-tables — Tabela 8-5', () => {
  it('SUPERIOR_ARMA_ROWS cobre [1, 100] sem gaps', () => {
    let cursor = 1
    for (const row of SUPERIOR_ARMA_ROWS) {
      expect(row[0]).toBe(cursor)
      cursor = row[1] + 1
    }
    expect(cursor).toBe(101)
  })

  it('SUPERIOR_ARMADURA_ROWS cobre [1, 100] sem gaps', () => {
    let cursor = 1
    for (const row of SUPERIOR_ARMADURA_ROWS) {
      expect(row[0]).toBe(cursor)
      cursor = row[1] + 1
    }
    expect(cursor).toBe(101)
  })

  it('SUPERIOR_ESOTERICO_ROWS cobre [1, 100] sem gaps', () => {
    let cursor = 1
    for (const row of SUPERIOR_ESOTERICO_ROWS) {
      expect(row[0]).toBe(cursor)
      cursor = row[1] + 1
    }
    expect(cursor).toBe(101)
  })

  it('arma: Atroz + Pungente marcados countsAsTwo', () => {
    const atroz = SUPERIOR_ARMA_ROWS.find((r) => r[2] === 'Atroz')!
    const pungente = SUPERIOR_ARMA_ROWS.find((r) => r[2] === 'Pungente')!
    expect(atroz[3]).toBe(true)
    expect(pungente[3]).toBe(true)
  })

  it('armadura: Sob medida marcado countsAsTwo', () => {
    const sm = SUPERIOR_ARMADURA_ROWS.find((r) => r[2] === 'Sob medida')!
    expect(sm[3]).toBe(true)
  })

  it('esotérico: sem propriedades countsAsTwo', () => {
    for (const row of SUPERIOR_ESOTERICO_ROWS) {
      expect(row[3]).toBeUndefined()
    }
  })
})

describe('MATERIAL_ROLL_TABLE — footnote 2', () => {
  it('6 materiais na ordem do PDF (1d6)', () => {
    expect(MATERIAL_ROLL_TABLE).toEqual([
      'aço-rubi',
      'adamante',
      'gelo eterno',
      'madeira Tollon',
      'matéria vermelha',
      'mitral',
    ])
  })

  it('frozen', () => {
    expect(Object.isFrozen(MATERIAL_ROLL_TABLE)).toBe(true)
  })
})

describe('rollEquipKind — 1d6 routing p330', () => {
  it('produz apenas arma/armadura/esoterico', () => {
    const rng = mulberry32(1)
    for (let i = 0; i < 100; i++) {
      const kind = rollEquipKind(rng)
      expect(ALL_KINDS).toContain(kind)
    }
  })

  it('distribuição: arma 50%, armadura 33%, esotérico 17% aprox', () => {
    const rng = mulberry32(42)
    let arma = 0
    let armadura = 0
    let esoterico = 0
    for (let i = 0; i < 6000; i++) {
      const kind = rollEquipKind(rng)
      if (kind === 'arma') arma++
      else if (kind === 'armadura') armadura++
      else esoterico++
    }
    // 1d6: 1-3 arma (50%), 4-5 armadura (33%), 6 esotérico (17%)
    expect(arma / 6000).toBeCloseTo(0.5, 1)
    expect(armadura / 6000).toBeCloseTo(0.333, 1)
    expect(esoterico / 6000).toBeCloseTo(0.167, 1)
  })
})

describe('rollDiverso — Tabela 8-3', () => {
  it('sempre retorna nome válido do catálogo', () => {
    const rng = mulberry32(7)
    const catalog = new Set(DIVERSOS_ROWS.map((r) => r[2]))
    for (let i = 0; i < 500; i++) {
      const name = rollDiverso(rng)
      expect(catalog.has(name)).toBe(true)
    }
  })

  it('determinismo: mesma seed → mesma sequência', () => {
    const a = mulberry32(99)
    const b = mulberry32(99)
    for (let i = 0; i < 20; i++) {
      expect(rollDiverso(a)).toBe(rollDiverso(b))
    }
  })
})

describe('rollEquipamento — Tabela 8-4', () => {
  it('sempre retorna { kind, name } válido', () => {
    const rng = mulberry32(3)
    for (let i = 0; i < 300; i++) {
      const r = rollEquipamento(rng)
      expect(ALL_KINDS).toContain(r.kind)
      expect(typeof r.name).toBe('string')
      expect(r.name.length).toBeGreaterThan(0)
    }
  })

  it('kind explícito é respeitado', () => {
    const rng = mulberry32(4)
    for (let i = 0; i < 50; i++) {
      expect(rollEquipamento(rng, 'arma').kind).toBe('arma')
      expect(rollEquipamento(rng, 'armadura').kind).toBe('armadura')
      expect(rollEquipamento(rng, 'esoterico').kind).toBe('esoterico')
    }
  })

  it('kind=arma nomes vêm de ARMA_ROWS', () => {
    const rng = mulberry32(5)
    const armas = new Set(ARMA_ROWS.map((r) => r[2]))
    for (let i = 0; i < 100; i++) {
      const r = rollEquipamento(rng, 'arma')
      expect(armas.has(r.name)).toBe(true)
    }
  })

  it('determinismo', () => {
    const a = mulberry32(2026)
    const b = mulberry32(2026)
    for (let i = 0; i < 20; i++) {
      expect(rollEquipamento(a)).toEqual(rollEquipamento(b))
    }
  })
})

describe('rollSuperior — Tabela 8-5', () => {
  it('improvements=2 produz 1 ou 2 properties (depending on countsAsTwo)', () => {
    const rng = mulberry32(50)
    for (let i = 0; i < 100; i++) {
      const r = rollSuperior(rng, 2, 'arma')
      expect(r.properties.length).toBeGreaterThanOrEqual(1)
      expect(r.properties.length).toBeLessThanOrEqual(2)
      const total = r.properties.reduce(
        (n, p) => n + (p.countsAsTwo ? 2 : 1),
        0,
      )
      expect(total).toBeGreaterThanOrEqual(2)
    }
  })

  it('improvements=1 nunca retorna property countsAsTwo (footnote 1)', () => {
    const rng = mulberry32(60)
    for (let i = 0; i < 500; i++) {
      const r = rollSuperior(rng, 1, 'arma')
      expect(r.properties.length).toBe(1)
      expect(r.properties[0]!.countsAsTwo).toBe(false)
    }
  })

  it('improvements=1 armadura nunca retorna Sob medida', () => {
    const rng = mulberry32(70)
    for (let i = 0; i < 500; i++) {
      const r = rollSuperior(rng, 1, 'armadura')
      expect(r.properties[0]!.name).not.toBe('Sob medida')
    }
  })

  it('Material especial preenche field material com valor válido', () => {
    const rng = mulberry32(80)
    const materials = new Set(MATERIAL_ROLL_TABLE)
    let matHits = 0
    for (let i = 0; i < 2000; i++) {
      const r = rollSuperior(rng, 4, 'arma')
      for (const prop of r.properties) {
        if (prop.name === 'Material especial') {
          matHits++
          expect(prop.material).toBeDefined()
          expect(materials.has(prop.material!)).toBe(true)
        } else {
          expect(prop.material).toBeUndefined()
        }
      }
    }
    // 10% chance por roll × 4+ melhorias × 2000 ≈ 800 esperado
    expect(matHits).toBeGreaterThan(50)
  })

  it('improvements=4: soma total de melhorias = 4 exato', () => {
    const rng = mulberry32(90)
    for (let i = 0; i < 200; i++) {
      const r = rollSuperior(rng, 4, 'arma')
      const total = r.properties.reduce(
        (n, p) => n + (p.countsAsTwo ? 2 : 1),
        0,
      )
      expect(total).toBe(4)
    }
  })

  it('kind respeitado', () => {
    const rng = mulberry32(100)
    expect(rollSuperior(rng, 1, 'esoterico').kind).toBe('esoterico')
    expect(rollSuperior(rng, 1, 'armadura').kind).toBe('armadura')
    expect(rollSuperior(rng, 1, 'arma').kind).toBe('arma')
  })

  it('kind=esoterico nomes vêm de SUPERIOR_ESOTERICO_ROWS', () => {
    const rng = mulberry32(110)
    const names = new Set(SUPERIOR_ESOTERICO_ROWS.map((r) => r[2]))
    for (let i = 0; i < 100; i++) {
      const r = rollSuperior(rng, 1, 'esoterico')
      expect(names.has(r.properties[0]!.name)).toBe(true)
    }
  })

  it('determinismo', () => {
    const a = mulberry32(2026)
    const b = mulberry32(2026)
    expect(rollSuperior(a, 3, 'arma')).toEqual(rollSuperior(b, 3, 'arma'))
  })
})
