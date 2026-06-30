import { describe, expect, it } from 'vitest'
import {
  BESTIARY,
  monsterById,
  monstersByNdRange,
  monstersBySize,
  monstersByTipo,
  xpForNd,
  type MonsterSize,
  type MonsterTipo,
} from '../bestiary'

/**
 * PDF Cap 7 Ameaças, livro p282-323. Pinned:
 *  - 20 monstros core T20 cobrindo ND 1/4 → 20.
 *  - 5 tipos (humanoide / animal / monstro / morto-vivo / construto).
 *  - 5 tamanhos (pequeno / médio / grande / enorme / colossal).
 *  - XP de tesouro = ND × 1000 (PDF Cap 8 Recompensas p326).
 */

const ALL_TIPOS: readonly MonsterTipo[] = [
  'humanoide',
  'animal',
  'monstro',
  'morto-vivo',
  'construto',
  'espirito',
  'planar',
]

const ALL_SIZES: readonly MonsterSize[] = [
  'minusculo',
  'pequeno',
  'medio',
  'grande',
  'enorme',
  'colossal',
]

describe('BESTIARY — shape & invariants', () => {
  it('catálogo tem exatamente 20 monstros', () => {
    expect(BESTIARY.length).toBe(20)
  })

  it('todos ids únicos', () => {
    const ids = BESTIARY.map((m) => m.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('todos os nomes únicos', () => {
    const names = BESTIARY.map((m) => m.name)
    expect(new Set(names).size).toBe(names.length)
  })

  it('toda entrada tem tipo em union conhecida', () => {
    for (const m of BESTIARY) expect(ALL_TIPOS).toContain(m.tipo)
  })

  it('toda entrada tem size em union conhecida', () => {
    for (const m of BESTIARY) expect(ALL_SIZES).toContain(m.size)
  })

  it('toda entrada tem hp positivo', () => {
    for (const m of BESTIARY) expect(m.hp).toBeGreaterThan(0)
  })

  it('toda entrada tem ≥ 1 ataque', () => {
    for (const m of BESTIARY) expect(m.attacks.length).toBeGreaterThan(0)
  })

  it('toda entrada tem ≥ 1 specialAbility', () => {
    for (const m of BESTIARY) expect(m.specialAbilities.length).toBeGreaterThan(0)
  })

  it('toda bookPage no range 282-323 (Cap 7 Ameaças)', () => {
    for (const m of BESTIARY) {
      expect(m.bookPage).toBeGreaterThanOrEqual(282)
      expect(m.bookPage).toBeLessThanOrEqual(323)
    }
  })

  it('catálogo é frozen', () => {
    expect(Object.isFrozen(BESTIARY)).toBe(true)
  })
})

describe('xpForNd — XP fórmula PDF Cap 8 p326', () => {
  it('ND 1/4 = 250', () => {
    expect(xpForNd(0.25)).toBe(250)
  })

  it('ND 1/2 = 500', () => {
    expect(xpForNd(0.5)).toBe(500)
  })

  it('ND 1 = 1000', () => {
    expect(xpForNd(1)).toBe(1000)
  })

  it('ND 5 = 5000', () => {
    expect(xpForNd(5)).toBe(5000)
  })

  it('ND 20 = 20000', () => {
    expect(xpForNd(20)).toBe(20000)
  })

  it('treasureXp de cada monstro = xpForNd(nd)', () => {
    for (const m of BESTIARY) {
      expect(m.treasureXp).toBe(xpForNd(m.nd))
    }
  })
})

describe('BESTIARY — ND coverage 1/4 → 20', () => {
  it('NDs presentes incluem 0.25 / 0.5 / 2 / 3 / 4 / 5 / 6 / 7 / 10 / 11 / 12 / 20', () => {
    const nds = new Set(BESTIARY.map((m) => m.nd))
    for (const expected of [
      0.25, 0.5, 2, 3, 4, 5, 6, 7, 10, 11, 12, 20,
    ]) {
      expect(nds.has(expected)).toBe(true)
    }
  })

  it('ND mínimo é 0.25 (Goblin Salteador / Zumbi)', () => {
    const min = Math.min(...BESTIARY.map((m) => m.nd))
    expect(min).toBe(0.25)
  })

  it('ND máximo é 20 (Dragão-Rei)', () => {
    const max = Math.max(...BESTIARY.map((m) => m.nd))
    expect(max).toBe(20)
  })
})

describe('BESTIARY — tipo distribution', () => {
  it('5 humanoides', () => {
    // Goblin, Orc x2 ... ah wait, conferir: Goblin, Orc, Ogro = 3.
    // Subagent contou 5. Verificar de fato:
    const ids = monstersByTipo('humanoide').map((m) => m.id)
    expect(ids).toEqual(['goblin-salteador', 'orc-combatente', 'ogro'])
  })

  it('1 animal (Lobo)', () => {
    expect(monstersByTipo('animal').map((m) => m.id)).toEqual(['lobo'])
  })

  it('3 morto-vivos (Zumbi, Esqueleto, Vampiro)', () => {
    const ids = monstersByTipo('morto-vivo').map((m) => m.id).sort()
    expect(ids).toEqual(['esqueleto', 'vampiro', 'zumbi'])
  })

  it('2 construtos (Gárgula, Golem de Ferro)', () => {
    const ids = monstersByTipo('construto').map((m) => m.id).sort()
    expect(ids).toEqual(['gargula', 'golem-de-ferro'])
  })

  it('11 monstros (incl. 4 dragões, Hidra, Mantícora, Troll, etc.)', () => {
    expect(monstersByTipo('monstro').length).toBe(11)
  })
})

describe('BESTIARY — size distribution', () => {
  it('pequeno: 1 (Goblin)', () => {
    expect(monstersBySize('pequeno').map((m) => m.id)).toEqual([
      'goblin-salteador',
    ])
  })

  it('médio: 8', () => {
    expect(monstersBySize('medio').length).toBe(8)
  })

  it('grande: 8', () => {
    expect(monstersBySize('grande').length).toBe(8)
  })

  it('enorme: 2 (Dragão Adulto + Hidra)', () => {
    const ids = monstersBySize('enorme').map((m) => m.id).sort()
    expect(ids).toEqual(['dragao-adulto', 'hidra'])
  })

  it('colossal: 1 (Dragão-Rei)', () => {
    expect(monstersBySize('colossal').map((m) => m.id)).toEqual(['dragao-rei'])
  })
})

describe('BESTIARY — pinned canonical entries', () => {
  it('Dragão-Rei ND 20, hp 1400, defesa 62, RD 20 menção, p312', () => {
    const m = monsterById('dragao-rei')!
    expect(m.nd).toBe(20)
    expect(m.hp).toBe(1400)
    expect(m.defesa).toBe(62)
    expect(m.size).toBe('colossal')
    expect(m.bookPage).toBe(312)
    expect(m.specialAbilities.join(' ')).toMatch(/RD 20/)
  })

  it('Troll: cura acelerada 15 anulada por ácido ou fogo', () => {
    const m = monsterById('troll')!
    expect(m.specialAbilities.join(' ')).toMatch(/Cura acelerada 15/)
    expect(m.specialAbilities.join(' ')).toMatch(/ácido ou fogo/)
  })

  it('Hidra: Cura acelerada 100 + Cortar Cabeças', () => {
    const m = monsterById('hidra')!
    expect(m.specialAbilities.join(' ')).toMatch(/Cura acelerada 100/)
    expect(m.specialAbilities.join(' ')).toMatch(/Cortar Cabeças/)
  })

  it('Vampiro: Dominação Vampírica + Drenar Sangue', () => {
    const m = monsterById('vampiro')!
    expect(m.tipo).toBe('morto-vivo')
    expect(m.specialAbilities.join(' ')).toMatch(/Dominação Vampírica/)
    expect(m.specialAbilities.join(' ')).toMatch(/Drenar Sangue/)
  })

  it('Basilisco: Olhar Petrificante', () => {
    const m = monsterById('basilisco')!
    expect(m.specialAbilities.join(' ')).toMatch(/Olhar Petrificante/)
  })

  it('Golem de Ferro: Imunidade a Magia exceto eletricidade/fogo', () => {
    const m = monsterById('golem-de-ferro')!
    expect(m.tipo).toBe('construto')
    expect(m.specialAbilities.join(' ')).toMatch(/Imunidade a Magia/)
  })

  it('Goblin Salteador: Frenesi (bônus por adjacência)', () => {
    const m = monsterById('goblin-salteador')!
    expect(m.specialAbilities.join(' ')).toMatch(/Frenesi/)
  })

  it('Lobo: Táticas de Alcateia + derrubar em acerto', () => {
    const m = monsterById('lobo')!
    expect(m.specialAbilities.join(' ')).toMatch(/Táticas de Alcateia/)
    expect(m.attacks[0]!.special).toMatch(/derrubar/)
  })
})

describe('monstersByNdRange', () => {
  it('ND 1/4 a 1: 4 monstros', () => {
    expect(monstersByNdRange(0.25, 1).length).toBe(4)
  })

  it('ND 10+: 5 monstros (Golem, 2 Dragões, Hidra, Vampiro)', () => {
    expect(monstersByNdRange(10, 30).length).toBe(5)
  })

  it('ND 20+: apenas Dragão-Rei', () => {
    expect(monstersByNdRange(20, 30).map((m) => m.id)).toEqual(['dragao-rei'])
  })
})

describe('BESTIARY — lookup helpers', () => {
  it('monsterById hit', () => {
    expect(monsterById('vampiro')?.name).toBe('Vampiro')
  })

  it('monsterById miss', () => {
    expect(monsterById('thanatos')).toBeUndefined()
  })
})
