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
 * PDF Cap 7 Ameaças, livro p282-323. Invariants pinned:
 *  - Baseline: 20 core-T20 monstros cobrindo ND 1/4 → 20.
 *  - Expansão #1 (p286-298): +31 monstros (Masmorras/Ermos/Puristas/
 *    Reino dos Mortos). Total ≥ 51.
 *  - Cada monstro tem tipo + size em unions conhecidos.
 *  - XP de tesouro = ND × 1000 (PDF Cap 8 Recompensas p326).
 *
 * Distribution asserts use "≥ baseline" so future expansions don't
 * force test churn — canonical entries stay pinned so a rewrite of
 * the seed still gets caught.
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
  it('catálogo tem no mínimo 80 monstros (expansão #2)', () => {
    expect(BESTIARY.length).toBeGreaterThanOrEqual(80)
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

  it('ND mínimo é 0.25', () => {
    const min = Math.min(...BESTIARY.map((m) => m.nd))
    expect(min).toBe(0.25)
  })

  it('ND máximo é 20 (Dragão-Rei)', () => {
    const max = Math.max(...BESTIARY.map((m) => m.nd))
    expect(max).toBe(20)
  })

  it('hp positivo em toda entrada', () => {
    for (const m of BESTIARY) expect(m.hp).toBeGreaterThan(0)
  })

  it('defesa >= 10 em toda entrada', () => {
    for (const m of BESTIARY) expect(m.defesa).toBeGreaterThanOrEqual(10)
  })
})

describe('BESTIARY — tipo distribution', () => {
  it('inclui ≥ 3 humanoides base (goblin-salteador, orc-combatente, ogro)', () => {
    const ids = new Set(monstersByTipo('humanoide').map((m) => m.id))
    for (const anchor of ['goblin-salteador', 'orc-combatente', 'ogro']) {
      expect(ids.has(anchor)).toBe(true)
    }
  })

  it('inclui animal base (lobo)', () => {
    const ids = new Set(monstersByTipo('animal').map((m) => m.id))
    expect(ids.has('lobo')).toBe(true)
  })

  it('inclui morto-vivos base (zumbi, esqueleto, vampiro)', () => {
    const ids = new Set(monstersByTipo('morto-vivo').map((m) => m.id))
    for (const anchor of ['zumbi', 'esqueleto', 'vampiro']) {
      expect(ids.has(anchor)).toBe(true)
    }
  })

  it('inclui construtos base (gargula, golem-de-ferro)', () => {
    const ids = new Set(monstersByTipo('construto').map((m) => m.id))
    for (const anchor of ['gargula', 'golem-de-ferro']) {
      expect(ids.has(anchor)).toBe(true)
    }
  })

  it('monstros ≥ 11 (baseline: 4 dragões, Hidra, Mantícora, Troll etc.)', () => {
    expect(monstersByTipo('monstro').length).toBeGreaterThanOrEqual(11)
  })
})

describe('BESTIARY — size distribution', () => {
  it('pequeno inclui goblin-salteador', () => {
    const ids = new Set(monstersBySize('pequeno').map((m) => m.id))
    expect(ids.has('goblin-salteador')).toBe(true)
  })

  it('médio ≥ 8', () => {
    expect(monstersBySize('medio').length).toBeGreaterThanOrEqual(8)
  })

  it('grande ≥ 8', () => {
    expect(monstersBySize('grande').length).toBeGreaterThanOrEqual(8)
  })

  it('enorme inclui dragao-adulto + hidra', () => {
    const ids = new Set(monstersBySize('enorme').map((m) => m.id))
    for (const anchor of ['dragao-adulto', 'hidra']) {
      expect(ids.has(anchor)).toBe(true)
    }
  })

  it('colossal inclui dragao-rei', () => {
    const ids = new Set(monstersBySize('colossal').map((m) => m.id))
    expect(ids.has('dragao-rei')).toBe(true)
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

describe('BESTIARY — expansão #1 pinned entries', () => {
  it('Colosso Supremo ND 14, hp 675, colossal, p296', () => {
    const m = monsterById('colosso-supremo')!
    expect(m.nd).toBe(14)
    expect(m.hp).toBe(675)
    expect(m.size).toBe('colossal')
    expect(m.tipo).toBe('construto')
    expect(m.bookPage).toBe(296)
  })

  it('Necromante ND 7, magias mago 7º (Amedrontar, Crânio Voador)', () => {
    const m = monsterById('necromante')!
    expect(m.nd).toBe(7)
    expect(m.specialAbilities.join(' ')).toMatch(/Amedrontar|Crânio Voador/)
  })

  it('Falange ND 8: bando (dano dobrado em superação de defesa)', () => {
    const m = monsterById('falange')!
    expect(m.nd).toBe(8)
    expect(m.tipo).toBe('morto-vivo')
    expect(m.specialAbilities.join(' ')).toMatch(/[Bb]ando/)
  })

  it('Centopeia-Dragão ND 7: Aura de Calor + Engolir', () => {
    const m = monsterById('centopeia-dragao')!
    expect(m.specialAbilities.join(' ')).toMatch(/Aura de Calor/)
    expect(m.specialAbilities.join(' ')).toMatch(/Engolir/)
  })

  it('Cão do Inferno ND 3: sopro cone fogo + imunidade a fogo', () => {
    const m = monsterById('cao-do-inferno')!
    expect(m.specialAbilities.join(' ')).toMatch(/Sopro.*cone.*fogo/i)
    expect(m.specialAbilities.join(' ')).toMatch(/Imunidade a fogo/i)
  })
})

describe('BESTIARY — expansão #2 pinned entries', () => {
  it('Thuwarokk ND 16: Carapaça + Jato de Ácido + colossal', () => {
    const m = monsterById('thuwarokk')!
    expect(m.nd).toBe(16)
    expect(m.size).toBe('colossal')
    expect(m.specialAbilities.join(' ')).toMatch(/Carapaça/)
    expect(m.specialAbilities.join(' ')).toMatch(/Jato de Ácido/)
  })

  it('Lagash ND 13: Crias de Sszzaas + Cuspe Venenoso', () => {
    const m = monsterById('lagash')!
    expect(m.nd).toBe(13)
    expect(m.specialAbilities.join(' ')).toMatch(/Crias de Sszzaas/)
    expect(m.specialAbilities.join(' ')).toMatch(/Cuspe Venenoso/)
  })

  it('Troll das Cavernas ND 9: cura acelerada 20 anulada por ácido ou fogo', () => {
    const m = monsterById('troll-das-cavernas')!
    expect(m.specialAbilities.join(' ')).toMatch(/Cura acelerada 20/)
    expect(m.specialAbilities.join(' ')).toMatch(/ácido ou fogo/)
  })

  it('Sombra de Thwor ND 9: Assassinar + Ataque Furtivo +7d6', () => {
    const m = monsterById('sombra-de-thwor')!
    expect(m.specialAbilities.join(' ')).toMatch(/Assassinar/)
    expect(m.specialAbilities.join(' ')).toMatch(/\+7d6/)
  })

  it('Engenho de Guerra Goblin ND 6: Arsenal de Engenhocas (6 opções)', () => {
    const m = monsterById('engenho-guerra-goblin')!
    expect(m.tipo).toBe('construto')
    expect(m.specialAbilities.join(' ')).toMatch(/Arsenal de Engenhocas/)
  })

  it('Dragão Venerável ND 15: Aura Aterradora + Sopro cone 16d12 fogo', () => {
    const m = monsterById('dragao-veneravel')!
    expect(m.nd).toBe(15)
    expect(m.specialAbilities.join(' ')).toMatch(/Aura Aterradora/)
    expect(m.specialAbilities.join(' ')).toMatch(/16d12 fogo/)
  })

  it('Enxame Kobold: Unidos Venceremos + immune manobras', () => {
    const m = monsterById('enxame-kobold')!
    expect(m.specialAbilities.join(' ')).toMatch(/Unidos Venceremos/)
    expect(m.specialAbilities.join(' ')).toMatch(/manobras/)
  })
})

describe('monstersByNdRange', () => {
  it('ND ≤ 1: ≥ 4 monstros (baseline coverage)', () => {
    expect(monstersByNdRange(0.25, 1).length).toBeGreaterThanOrEqual(4)
  })

  it('ND 10+: inclui os 3 top-tier canônicos (golem-de-ferro, dragao-adulto, dragao-rei)', () => {
    const ids = new Set(monstersByNdRange(10, 30).map((m) => m.id))
    for (const anchor of ['golem-de-ferro', 'dragao-adulto', 'dragao-rei']) {
      expect(ids.has(anchor)).toBe(true)
    }
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

  it('xpForNd core rule (ND × 1000)', () => {
    expect(xpForNd(1)).toBe(1000)
    expect(xpForNd(0.25)).toBe(250)
    expect(xpForNd(20)).toBe(20000)
  })
})
