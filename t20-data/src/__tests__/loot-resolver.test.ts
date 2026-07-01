import { describe, expect, it } from 'vitest'
import { mulberry32 } from '../loot-rng'
import { resolveLootForNd } from '../loot-resolver'
import {
  ARMADURA_ROWS,
  ARMA_ROWS,
  DIVERSOS_ROWS,
  ESOTERICO_ROWS,
} from '../loot-sub-rolls'

/**
 * PDF Cap 8 — loot resolver compõe Tabela 8-1 (kind d%) com
 * Tabelas 8-3/8-4/8-5 (nomes concretos).
 *
 * Invariants:
 *  - items[] tem 0-2 elementos (0 = "none", 2 = row "role duas vezes").
 *  - Nomes de itens vêm dos catálogos das sub-tabelas.
 *  - Superior tem baseName + properties non-empty.
 *  - Determinismo total.
 */

const ALL_NDS = [
  '1/4', '1/2', '1', '2', '3', '4', '5', '6', '7', '8', '9', '10',
  '11', '12', '13', '14', '15', '16', '17', '18', '19', '20',
]

const ALL_EQUIP_NAMES = new Set<string>()
for (const rows of [ARMA_ROWS, ARMADURA_ROWS, ESOTERICO_ROWS]) {
  for (const row of rows) ALL_EQUIP_NAMES.add(row[2])
}
const ALL_DIVERSO_NAMES = new Set(DIVERSOS_ROWS.map((r) => r[2]))

describe('resolveLootForNd — shape', () => {
  it('retorna estrutura { nd, money, items, itemRoll } pra todos NDs', () => {
    const rng = mulberry32(1)
    for (const nd of ALL_NDS) {
      const r = resolveLootForNd(rng, nd)
      expect(r.nd).toBe(nd)
      expect(r.money).toBeDefined()
      expect(Array.isArray(r.items)).toBe(true)
      expect(r.itemRoll).toBeGreaterThanOrEqual(1)
      expect(r.itemRoll).toBeLessThanOrEqual(100)
    }
  })

  it('items[] length em [0, 2]', () => {
    const rng = mulberry32(2)
    for (let i = 0; i < 500; i++) {
      const nd = ALL_NDS[i % ALL_NDS.length]!
      const r = resolveLootForNd(rng, nd)
      expect(r.items.length).toBeGreaterThanOrEqual(0)
      expect(r.items.length).toBeLessThanOrEqual(2)
    }
  })
})

describe('resolveLootForNd — determinismo', () => {
  it('mesma seed produz loot idêntico', () => {
    const a = mulberry32(2026)
    const b = mulberry32(2026)
    for (const nd of ALL_NDS) {
      expect(resolveLootForNd(a, nd)).toEqual(resolveLootForNd(b, nd))
    }
  })
})

describe('resolveLootForNd — nomes concretos', () => {
  it('itens diverso resolvem pra nome do catálogo 8-3', () => {
    const rng = mulberry32(50)
    let hits = 0
    for (let i = 0; i < 2000; i++) {
      const nd = ALL_NDS[i % ALL_NDS.length]!
      const r = resolveLootForNd(rng, nd)
      for (const item of r.items) {
        if (item.kind === 'diverso') {
          hits++
          expect(ALL_DIVERSO_NAMES.has(item.name)).toBe(true)
        }
      }
    }
    expect(hits).toBeGreaterThan(0)
  })

  it('itens equipamento resolvem pra nome dos catálogos 8-4', () => {
    const rng = mulberry32(60)
    let hits = 0
    for (let i = 0; i < 500; i++) {
      const r = resolveLootForNd(rng, '4')
      for (const item of r.items) {
        if (item.kind === 'equipamento') {
          hits++
          expect(ALL_EQUIP_NAMES.has(item.name)).toBe(true)
          expect(['arma', 'armadura', 'esoterico']).toContain(item.equipKind)
        }
      }
    }
    expect(hits).toBeGreaterThan(0)
  })

  it('itens superior tem baseName + properties non-empty', () => {
    const rng = mulberry32(70)
    let hits = 0
    for (let i = 0; i < 500; i++) {
      const r = resolveLootForNd(rng, '8')
      for (const item of r.items) {
        if (item.kind === 'superior') {
          hits++
          expect(ALL_EQUIP_NAMES.has(item.baseName)).toBe(true)
          expect(item.properties.length).toBeGreaterThan(0)
          expect(['arma', 'armadura', 'esoterico']).toContain(item.equipKind)
        }
      }
    }
    expect(hits).toBeGreaterThan(0)
  })

  it('itens potion resolvem pra count numérico > 0', () => {
    const rng = mulberry32(80)
    let hits = 0
    for (let i = 0; i < 500; i++) {
      const r = resolveLootForNd(rng, '3')
      for (const item of r.items) {
        if (item.kind === 'potion') {
          hits++
          expect(item.count).toBeGreaterThan(0)
        }
      }
    }
    expect(hits).toBeGreaterThan(0)
  })

  it('itens magic resolvem via kind router 1d6 (arma/armadura/acessorio)', () => {
    const rng = mulberry32(90)
    let hits = 0
    for (let i = 0; i < 300; i++) {
      const r = resolveLootForNd(rng, '17')
      for (const item of r.items) {
        if (item.kind === 'magic') {
          hits++
          expect(['menor', 'medio', 'maior']).toContain(item.tier)
          expect(['arma', 'armadura', 'acessorio']).toContain(item.magicKind)
          if (item.magicKind === 'arma') {
            expect(typeof item.baseName).toBe('string')
            expect(['encanto', 'specific']).toContain(item.weapon.kind)
          } else if (item.magicKind === 'armadura') {
            expect(typeof item.baseName).toBe('string')
            expect(typeof item.isShield).toBe('boolean')
            expect(['encanto', 'specific']).toContain(item.armor.kind)
          } else {
            expect(typeof item.name).toBe('string')
            expect(item.name.length).toBeGreaterThan(0)
          }
        }
      }
    }
    expect(hits).toBeGreaterThan(0)
  })
})

describe('resolveLootForNd — magic integration', () => {
  it('magic armadura: isShield=true quando baseName começa com "Escudo"', () => {
    const rng = mulberry32(200)
    let checked = 0
    for (let i = 0; i < 500; i++) {
      const r = resolveLootForNd(rng, '15')
      for (const item of r.items) {
        if (item.kind === 'magic' && item.magicKind === 'armadura') {
          checked++
          const expectedShield = item.baseName.startsWith('Escudo')
          expect(item.isShield).toBe(expectedShield)
          expect(item.armor.isShield).toBe(expectedShield)
        }
      }
    }
    expect(checked).toBeGreaterThan(0)
  })

  it('magic arma: nomes weapon.kind válidos', () => {
    const rng = mulberry32(210)
    let checked = 0
    for (let i = 0; i < 500; i++) {
      const r = resolveLootForNd(rng, '15')
      for (const item of r.items) {
        if (item.kind === 'magic' && item.magicKind === 'arma') {
          checked++
          expect(item.weapon.tier).toBe(item.tier)
          if (item.weapon.kind === 'encanto') {
            expect(typeof item.weapon.name).toBe('string')
            expect(typeof item.weapon.countsAsTwo).toBe('boolean')
          } else {
            expect(typeof item.weapon.name).toBe('string')
          }
        }
      }
    }
    expect(checked).toBeGreaterThan(0)
  })

  it('potion outcome expande count em potions[] com PotionRoll[]', () => {
    const rng = mulberry32(220)
    let checked = 0
    for (let i = 0; i < 500; i++) {
      const r = resolveLootForNd(rng, '5')
      for (const item of r.items) {
        if (item.kind === 'potion') {
          checked++
          expect(item.potions.length).toBe(item.count)
          for (const p of item.potions) {
            expect(typeof p.name).toBe('string')
            expect(p.priceTs).toBeGreaterThan(0)
            expect(['menor', 'medio', 'maior']).toContain(p.tier)
          }
        }
      }
    }
    expect(checked).toBeGreaterThan(0)
  })
})

describe('resolveLootForNd — "role duas vezes" produz 2 itens', () => {
  it('ND 20 (alto range com double flags) às vezes retorna 2 itens', () => {
    const rng = mulberry32(100)
    let doubleCount = 0
    for (let i = 0; i < 500; i++) {
      const r = resolveLootForNd(rng, '20')
      if (r.items.length === 2) doubleCount++
    }
    expect(doubleCount).toBeGreaterThan(0)
  })
})

describe('resolveLootForNd — "none" retorna items vazio', () => {
  it('ND 1/4 com item outcome "none" → items[] vazio', () => {
    const rng = mulberry32(110)
    let noneCount = 0
    for (let i = 0; i < 500; i++) {
      const r = resolveLootForNd(rng, '1/4')
      if (r.items.length === 0) noneCount++
    }
    expect(noneCount).toBeGreaterThan(0)
  })
})

describe('resolveLootForNd — throws pra ND desconhecido', () => {
  it('throws com mensagem descritiva', () => {
    expect(() => resolveLootForNd(mulberry32(1), '99')).toThrow(/unknown nd/)
  })
})
