import { describe, expect, it } from 'vitest'
import type { ClassChoices } from '@/shared/api/catalog-types'
import {
  chosenPowerLines,
  classPowerCandidates,
  classSlotCount,
  electiveSlotUsage,
  classChoiceSummary,
  type PowerOption,
  powerBlockedReason,
  powerChoiceOptions,
  tormentaPowerOptions,
  usedSlots,
} from './class-powers'

describe('tormentaPowerOptions', () => {
  it('returns the 22 poderes da Tormenta, all source "tormenta"', () => {
    const opts = tormentaPowerOptions()
    expect(opts).toHaveLength(22)
    expect(opts.every((o) => o.source === 'tormenta')).toBe(true)
  })

  it('encodes a specific-power prereq (Larva Explosiva ← Dentes Afiados)', () => {
    const larva = tormentaPowerOptions().find((o) => o.id === 'larva-explosiva')
    expect(larva?.prerequisites).toContainEqual({
      kind: 'power',
      id: 'dentes-afiados',
    })
  })
})

const opt = (id: string, repeatable?: boolean): PowerOption => ({
  id,
  name: id,
  description: '',
  minLevel: 1,
  prerequisites: [],
  source: 'class',
  choice: repeatable
    ? { kind: 'attribute', label: 'Atributo', repeatable: true }
    : undefined,
})

describe('usedSlots — repeatable powers count per sub-choice', () => {
  const byId = new Map([
    ['rep', opt('rep', true)],
    ['single', opt('single')],
  ])

  it('a plain power uses one slot', () => {
    expect(usedSlots(['single'], {}, byId)).toBe(1)
  })

  it('a repeatable power uses one slot per pick', () => {
    expect(usedSlots(['rep'], { rep: ['strength', 'dexterity', 'wisdom'] }, byId)).toBe(3)
  })

  it('a selected repeatable with no picks still occupies its slot (min 1)', () => {
    // Regression: 0 would let the power read as picked while consuming nothing.
    expect(usedSlots(['rep'], {}, byId)).toBe(1)
  })

  it('sums plain + repeatable', () => {
    expect(usedSlots(['single', 'rep'], { rep: ['strength', 'charisma'] }, byId)).toBe(3)
  })
})

describe('powerChoiceOptions', () => {
  it('returns the inline options for enumerated kinds', () => {
    const opts = powerChoiceOptions({
      kind: 'attribute',
      label: 'Atributo',
      options: [{ id: 'strength', name: 'Força' }],
    })
    expect(opts).toEqual([{ id: 'strength', name: 'Força' }])
  })

  it('sources weapons from the catalog when options are omitted', () => {
    const opts = powerChoiceOptions({ kind: 'weapon', label: 'Arma' })
    expect(opts.length).toBeGreaterThan(0)
    expect(opts[0]).toHaveProperty('name')
  })

  it('attaches a spell effect as desc when a totem note names a spell', () => {
    const [urso] = powerChoiceOptions({
      kind: 'totem',
      label: 'Animal totêmico',
      options: [{ id: 'urso', name: 'Urso', note: 'Vitalidade Fantasma' }],
    })
    expect(urso.note).toBe('Vitalidade Fantasma')
    expect(urso.desc).toBeTruthy()
  })

  it('leaves desc undefined when a note names no catalog spell', () => {
    const [o] = powerChoiceOptions({
      kind: 'totem',
      label: 'X',
      options: [{ id: 'x', name: 'X', note: 'Não Existe Magia' }],
    })
    expect(o.desc).toBeUndefined()
  })
})

describe('chosenPowerLines — Resumo mostra escolhas, não o pool', () => {
  const classes = [{ className: 'Bárbaro', level: 6 }]

  it('resolve nomes por id em todos os pools (classe, geral, tormenta)', () => {
    const lines = chosenPowerLines(
      classes,
      ['class.barbaro.totem-espiritual', 'esquiva', 'dentes-afiados'],
      {},
    )
    expect(lines.map((l) => [l.name, l.source])).toEqual([
      ['Totem Espiritual', 'class'],
      ['Esquiva', 'general'],
      ['Dentes Afiados', 'tormenta'],
    ])
  })

  it('resolve sub-escolhas para nomes (totem lobo → Lobo)', () => {
    const lines = chosenPowerLines(
      classes,
      ['class.barbaro.totem-espiritual'],
      { 'class.barbaro.totem-espiritual': ['lobo'] },
    )
    expect(lines[0].choices).toEqual(['Lobo'])
  })

  it('id desconhecido degrada para o próprio id (não some)', () => {
    const lines = chosenPowerLines(classes, ['nao-existe'], {})
    expect(lines[0].name).toBe('nao-existe')
  })
})

describe('classChoiceSummary — caminho/devoto por NOME', () => {
  it('resolve caminho e devoto', () => {
    expect(
      classChoiceSummary('Clérigo', { devoto: 'khalmyr' }),
    ).toBe('devoto de Khalmyr')
    expect(classChoiceSummary('Arcanista', { caminho: 'mago' })).toMatch(
      /^caminho: /,
    )
  })

  it('null sem escolhas', () => {
    expect(classChoiceSummary('Clérigo', {})).toBeNull()
    expect(classChoiceSummary('Clérigo', undefined)).toBeNull()
  })
})

describe('electiveSlotUsage', () => {
  it('conta as vagas ganhas pelo nível e o que já foi gasto', () => {
    const { classPowers } = classPowerCandidates('Bárbaro')
    const picked = classPowers.slice(0, 2).map((p) => p.id)

    const usage = electiveSlotUsage('Bárbaro', 6, picked)

    expect(usage.total).toBe(classSlotCount('Bárbaro', 6))
    expect(usage.used).toBe(2)
    expect(usage.remaining).toBe(usage.total - 2)
  })

  // Multiclasse: poder tirado da lista de OUTRA classe não pode comer a vaga
  // desta — só poderes desta classe ou gerais são gastáveis aqui.
  it('poder de outra classe não gasta vaga desta', () => {
    const { classPowers } = classPowerCandidates('Bardo')
    const doBardo = classPowers[0].id

    expect(electiveSlotUsage('Bárbaro', 6, [doBardo]).used).toBe(0)
  })

  it('poder geral gasta vaga de qualquer classe (p33)', () => {
    const { generalPowers } = classPowerCandidates('Bárbaro')

    expect(electiveSlotUsage('Bárbaro', 6, [generalPowers[0].id]).used).toBe(1)
  })

  it('nunca reporta vaga negativa', () => {
    const { generalPowers } = classPowerCandidates('Bárbaro')
    const demais = generalPowers.slice(0, 10).map((p) => p.id)

    expect(electiveSlotUsage('Bárbaro', 2, demais).remaining).toBe(0)
  })
})

/**
 * O que TRAVA um poder — e não tinha teste em camada nenhuma, nem aqui nem na
 * tela (a lista da Forja é virtualizada: em jsdom nenhuma linha renderiza, então
 * um teste de componente passaria verde sem olhar poder nenhum).
 *
 * O desfecho que o jogador nota: um poder que ele ainda não pode ter precisa
 * estar inalcançável E dizer por quê. Oferecer e recusar depois é pior;
 * oferecer sem explicar manda ele adivinhar.
 */
describe('powerBlockedReason', () => {
  const semNada = { chosenIds: new Set<string>(), classChoices: {} as ClassChoices }

  const doGuerreiro = (name: string) => {
    const { classPowers } = classPowerCandidates('Guerreiro')
    const found = classPowers.find((p) => p.name === name)
    if (!found) throw new Error(`poder ${name} não está no catálogo do Guerreiro`)
    return found
  }

  // "Especialização em Armadura. (...) Pré-requisito: 12º nível de guerreiro" (p65).
  it('poder acima do nível é recusado dizendo o nível que falta', () => {
    const poder = doGuerreiro('Especialização em Armadura')

    expect(powerBlockedReason(poder, 3, semNada)).toBe('≥ Nv 12')
    expect(powerBlockedReason(poder, 12, semNada)).toBeNull()
  })

  // Larva Explosiva pede outro poder da Tormenta antes (p137).
  it('pré-requisito de poder é recusado até o outro estar escolhido', () => {
    const larva = tormentaPowerOptions().find((p) => p.id === 'larva-explosiva')
    if (!larva) throw new Error('larva-explosiva não está no catálogo')

    const bloqueado = powerBlockedReason(larva, 20, semNada)
    expect(bloqueado).not.toBeNull()

    const comOOutro = {
      chosenIds: new Set(larva.prerequisites.flatMap((p) => ('id' in p ? [p.id] : []))),
      classChoices: {} as ClassChoices,
    }
    expect(powerBlockedReason(larva, 20, comOOutro)).toBeNull()
  })
})
