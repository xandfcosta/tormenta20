import { describe, expect, it } from 'vitest'
import { ORIGINS_CATALOG } from '../abilities/origins'
import {
  DEFORMIDADE_PERICIA_BONUS,
  DEFORMIDADE_SLOTS,
  deformidadeAvailablePowers,
  deformidadeSkillIds,
  deformidadeSlotsUsed,
  deformidadeTormentaPowerCount,
  raceWithDeformidade,
  validateDeformidade,
} from '../deformidade'

describe('DeformidadeChoice — slots e limites (Lefou, p23)', () => {
  it('2 perícias preenchem os 2 slots', () => {
    expect(
      deformidadeSlotsUsed({ pericias: ['Furtividade', 'Percepção'] }),
    ).toBe(2)
  })

  it('1 perícia + 1 poder trocado = 2 slots (troca máx. de UM bônus)', () => {
    expect(
      deformidadeSlotsUsed({
        pericias: ['Furtividade'],
        tormentaPower: 'dentes-afiados',
      }),
    ).toBe(2)
  })

  it('sub-preenchimento é válido (homebrew: GM permite menos)', () => {
    expect(validateDeformidade({ pericias: ['Furtividade'] })).toEqual([])
    expect(validateDeformidade({ pericias: [] })).toEqual([])
  })

  it('constantes do livro: 2 slots, bônus +2', () => {
    expect(DEFORMIDADE_SLOTS).toBe(2)
    expect(DEFORMIDADE_PERICIA_BONUS).toBe(2)
  })
})

describe('validateDeformidade — inputs inválidos viram warnings', () => {
  it('mais que 2 slots usados', () => {
    const warnings = validateDeformidade({
      pericias: ['Furtividade', 'Percepção'],
      tormentaPower: 'dentes-afiados',
    })
    expect(warnings.some((w) => w.includes('2'))).toBe(true)
  })

  it('perícia duplicada', () => {
    const warnings = validateDeformidade({
      pericias: ['Furtividade', 'Furtividade'],
    })
    expect(warnings.length).toBeGreaterThan(0)
  })

  it('perícia desconhecida', () => {
    const warnings = validateDeformidade({
      pericias: ['Tocar Flauta' as never],
    })
    expect(warnings.some((w) => w.includes('Tocar Flauta'))).toBe(true)
  })

  it('poder da Tormenta desconhecido', () => {
    const warnings = validateDeformidade({
      pericias: [],
      tormentaPower: 'nao-existe' as never,
    })
    expect(warnings.some((w) => w.includes('nao-existe'))).toBe(true)
  })
})

describe('contam como poder da Tormenta (exceto perda de Carisma)', () => {
  it('bônus de perícia contam para pré-requisitos, não para perda de CAR', () => {
    const choice = {
      pericias: ['Furtividade', 'Percepção'],
    } as const
    // 2 perícias = 2 poderes para pré-requisitos…
    expect(deformidadeTormentaPowerCount(choice)).toBe(2)
  })

  it('poder trocado soma à contagem', () => {
    expect(
      deformidadeTormentaPowerCount({
        pericias: ['Furtividade'],
        tormentaPower: 'antenas',
      }),
    ).toBe(2)
  })
})

describe('deformidadeAvailablePowers — pré-requisitos com perícias contando', () => {
  it('0 perícias → só poderes sem pré-requisito', () => {
    const powers = deformidadeAvailablePowers(0)
    expect(powers.every((p) => p.requiresOtherPowers === 0 && !p.requiresPower)).toBe(
      true,
    )
    expect(powers.some((p) => p.id === 'dentes-afiados')).toBe(true)
    expect(powers.some((p) => p.id === 'asas-insetoides')).toBe(false)
  })

  it('1 perícia → desbloqueia poderes que exigem 1 outro (Armamento Aberrante)', () => {
    const powers = deformidadeAvailablePowers(1)
    expect(powers.some((p) => p.id === 'armamento-aberrante')).toBe(true)
    // requiresPower específico nunca é satisfazível só com perícias
    expect(powers.some((p) => p.id === 'larva-explosiva')).toBe(false)
  })
})

describe('deformidadeSkillIds — nomes de perícia → SkillId', () => {
  it('converte nomes acentuados para slugs', () => {
    expect(
      deformidadeSkillIds({ pericias: ['Furtividade', 'Percepção'] }, []),
    ).toEqual(['furtividade', 'percepcao'])
  })

  it('nome desconhecido vira warning, não lança', () => {
    const warnings: string[] = []
    expect(
      deformidadeSkillIds({ pericias: ['Xadrez' as never] }, warnings),
    ).toEqual([])
    expect(warnings.some((w) => w.includes('Xadrez'))).toBe(true)
  })
})

describe('raceWithDeformidade — só raças com a habilidade (Lefou)', () => {
  it('Lefou tem, Anão não', () => {
    expect(raceWithDeformidade(['Anão', 'Lefou'])).toBe('Lefou')
    expect(raceWithDeformidade(['Anão', 'Humano'])).toBeUndefined()
    expect(raceWithDeformidade([])).toBeUndefined()
  })
})

describe('origem powerPick — benefícios de escolha livre', () => {
  it('5 benefícios flagados (1 tormenta + 4 combate)', () => {
    const flagged = ORIGINS_CATALOG.flatMap((o) =>
      o.benefits.filter((b) => b.powerPick),
    )
    expect(flagged.filter((b) => b.powerPick === 'tormenta')).toHaveLength(1)
    expect(flagged.filter((b) => b.powerPick === 'combate')).toHaveLength(4)
  })
})
