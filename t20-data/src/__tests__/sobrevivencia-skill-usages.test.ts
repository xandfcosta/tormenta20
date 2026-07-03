import { describe, expect, it } from 'vitest'
import {
  SOBREVIVENCIA_USAGES,
  TERRAIN_CD,
  TERRAIN_HARSH_PENALTY,
  TRACK_CD,
  TRACK_CD_PER_DAY_AGE,
  TRACK_LARGE_GROUP_MIN_INDIVIDUOS,
  TRACK_LARGE_TARGET_CD_REDUCTION,
  TRACK_POOR_VISIBILITY_CD_INCREASE,
  acampamentoCd,
  identificarCriaturaCd,
  orientarSeCd,
  orientarSeOutcome,
  rastrearCd,
  sobrevivenciaUsageByKind,
} from '../sobrevivencia-skill-usages'

/**
 * PDF livro p117 — Perícia Sobrevivência (SAB). 4 usos canônicos:
 *  1. Acampamento — CD por terreno, exige equipamento viagem
 *  2. Identificar Criatura — CD 15+ND
 *  3. Orientar-se — mesma tabela; falha=½ desloc, falha 5+=perdido
 *  4. Rastrear — CD por solo (só treinado), +1 CD/dia, +5 visib ruim, -5 alvo grande
 */

describe('SOBREVIVENCIA_USAGES — shape', () => {
  it('exatamente 4 usos (p117)', () => {
    expect(SOBREVIVENCIA_USAGES.length).toBe(4)
  })

  it('frozen', () => {
    expect(Object.isFrozen(SOBREVIVENCIA_USAGES)).toBe(true)
  })

  it('ids únicos', () => {
    const ids = SOBREVIVENCIA_USAGES.map((u) => u.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('todos bookPage 117', () => {
    for (const u of SOBREVIVENCIA_USAGES) expect(u.bookPage).toBe(117)
  })

  it('só Rastrear é treinado', () => {
    const treinados = SOBREVIVENCIA_USAGES.filter((u) => u.trainedOnly).map(
      (u) => u.id,
    )
    expect(treinados).toEqual(['rastrear'])
  })
})

describe('TERRAIN_CD — tabela p117', () => {
  it.each([
    ['planicies-colinas', 15],
    ['florestas-pantanos', 20],
    ['desertos-montanhas', 25],
    ['planares-tormenta', 30],
  ] as const)('%s → %s', (terrain, cd) => {
    expect(TERRAIN_CD[terrain]).toBe(cd)
  })

  it('frozen', () => {
    expect(Object.isFrozen(TERRAIN_CD)).toBe(true)
  })

  it('penalidade árido/clima ruim = +5', () => {
    expect(TERRAIN_HARSH_PENALTY).toBe(5)
  })
})

describe('acampamentoCd', () => {
  it('planícies sem penalidade → 15', () => {
    expect(acampamentoCd('planicies-colinas')).toBe(15)
  })

  it('florestas + clima ruim → 25', () => {
    expect(
      acampamentoCd('florestas-pantanos', { harshWeatherOrTerrain: true }),
    ).toBe(25)
  })

  it('planares/Tormenta + clima ruim → 35', () => {
    expect(
      acampamentoCd('planares-tormenta', { harshWeatherOrTerrain: true }),
    ).toBe(35)
  })
})

describe('Acampamento — requer equipamento viagem', () => {
  const usage = sobrevivenciaUsageByKind('acampamento')

  it('requiresTravelKit true', () => {
    if (usage.kind !== 'acampamento') throw new Error('narrow failed')
    expect(usage.requiresTravelKit).toBe(true)
  })

  it('ação por-dia', () => {
    expect(usage.action).toBe('por-dia')
  })
})

describe('identificarCriaturaCd', () => {
  it('ND 0 → 15', () => {
    expect(identificarCriaturaCd(0)).toBe(15)
  })

  it('ND 5 → 20', () => {
    expect(identificarCriaturaCd(5)).toBe(20)
  })

  it('ND 20 → 35', () => {
    expect(identificarCriaturaCd(20)).toBe(35)
  })

  it('throws se ND negativo', () => {
    expect(() => identificarCriaturaCd(-1)).toThrow(/nd/)
  })
})

describe('orientarSeCd — reutiliza tabela terreno', () => {
  it('mesma CD que Acampamento p/ florestas', () => {
    expect(orientarSeCd('florestas-pantanos')).toBe(
      acampamentoCd('florestas-pantanos'),
    )
  })

  it('planares/Tormenta → 30', () => {
    expect(orientarSeCd('planares-tormenta')).toBe(30)
  })
})

describe('orientarSeOutcome — falha ladder', () => {
  it('sucesso exato → ok', () => {
    expect(orientarSeOutcome(15, 15)).toBe('ok')
  })

  it('sucesso alto → ok', () => {
    expect(orientarSeOutcome(25, 15)).toBe('ok')
  })

  it('falha por 1 → metade', () => {
    expect(orientarSeOutcome(14, 15)).toBe('metade')
  })

  it('falha por 4 → metade', () => {
    expect(orientarSeOutcome(11, 15)).toBe('metade')
  })

  it('falha por 5 → perdido', () => {
    expect(orientarSeOutcome(10, 15)).toBe('perdido')
  })

  it('falha por 20 → perdido', () => {
    expect(orientarSeOutcome(-5, 15)).toBe('perdido')
  })
})

describe('TRACK_CD — tabela p117', () => {
  it.each([
    ['macio', 15],
    ['comum', 20],
    ['duro', 25],
  ] as const)('%s → %s', (soil, cd) => {
    expect(TRACK_CD[soil]).toBe(cd)
  })

  it('frozen', () => {
    expect(Object.isFrozen(TRACK_CD)).toBe(true)
  })

  it('+1 CD por dia de idade', () => {
    expect(TRACK_CD_PER_DAY_AGE).toBe(1)
  })

  it('-5 CD alvo grande', () => {
    expect(TRACK_LARGE_TARGET_CD_REDUCTION).toBe(5)
  })

  it('+5 CD visibilidade precária', () => {
    expect(TRACK_POOR_VISIBILITY_CD_INCREASE).toBe(5)
  })

  it('grupo grande = 10+ indivíduos', () => {
    expect(TRACK_LARGE_GROUP_MIN_INDIVIDUOS).toBe(10)
  })
})

describe('rastrearCd', () => {
  it('solo comum sem contexto → 20', () => {
    expect(rastrearCd('comum')).toBe(20)
  })

  it('solo duro + 3 dias idade → 28', () => {
    expect(rastrearCd('duro', { ageInDays: 3 })).toBe(28)
  })

  it('solo macio + alvo grande → 10', () => {
    expect(rastrearCd('macio', { largeGroupOrHugeCreature: true })).toBe(10)
  })

  it('solo comum + noite/chuva → 25', () => {
    expect(rastrearCd('comum', { poorVisibility: true })).toBe(25)
  })

  it('solo duro + 7 dias + grupo grande + visib ruim → 25+7-5+5=32', () => {
    expect(
      rastrearCd('duro', {
        ageInDays: 7,
        largeGroupOrHugeCreature: true,
        poorVisibility: true,
      }),
    ).toBe(32)
  })

  it('throws se idade negativa', () => {
    expect(() => rastrearCd('comum', { ageInDays: -1 })).toThrow(/ageInDays/)
  })
})

describe('Rastrear — usage flags', () => {
  const usage = sobrevivenciaUsageByKind('rastrear')

  it('treinado', () => {
    expect(usage.trainedOnly).toBe(true)
  })

  it('deslocamento à metade', () => {
    if (usage.kind !== 'rastrear') throw new Error('narrow failed')
    expect(usage.movementFraction).toBe(0.5)
  })

  it('retry permitido', () => {
    if (usage.kind !== 'rastrear') throw new Error('narrow failed')
    expect(usage.retryAllowed).toBe(true)
  })
})
