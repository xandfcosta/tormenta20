import { describe, expect, it } from 'vitest'
import type { CharacterInput } from '../character-sheet'
import { computeCharacterSheet } from '../character-sheet'

const BASE_ATTRS = {
  strength: 2,
  dexterity: 1,
  constitution: 1,
  intelligence: 0,
  wisdom: 0,
  charisma: 1,
}

function lefouInput(
  deformidade: CharacterInput['deformidade'],
): CharacterInput {
  return {
    level: 6,
    className: 'Bárbaro',
    raceId: 'lefou',
    raceFloatingPicks: ['strength', 'constitution', 'wisdom'],
    baseAttributes: BASE_ATTRS,
    deformidade,
  }
}

describe('computeCharacterSheet — Deformidade (Lefou p23)', () => {
  it('+2 nas perícias escolhidas, sem contar como treino', () => {
    const sheet = computeCharacterSheet(
      lefouInput({ pericias: ['Furtividade', 'Percepção'] }),
    )
    const noDef = computeCharacterSheet(lefouInput(undefined))
    expect(sheet.skills.furtividade.total).toBe(noDef.skills.furtividade.total + 2)
    expect(sheet.skills.percepcao.total).toBe(noDef.skills.percepcao.total + 2)
    expect(sheet.skills.furtividade.trained).toBe(false)
    // perícias não escolhidas intactas
    expect(sheet.skills.atletismo.total).toBe(noDef.skills.atletismo.total)
  })

  it('bônus de perícia NÃO perde Carisma (p23 "exceto para perda de Carisma")', () => {
    const sheet = computeCharacterSheet(
      lefouInput({ pericias: ['Furtividade', 'Percepção'] }),
    )
    // base 1 + Lefou -1 = 0; sem perda extra
    expect(sheet.attributes.charisma.total).toBe(0)
    expect(sheet.attributes.charisma.tormentaMod).toBeUndefined()
  })

  it('poder trocado perde 1 Carisma (p136, primeiro poder)', () => {
    const sheet = computeCharacterSheet(
      lefouInput({ pericias: ['Furtividade'], tormentaPower: 'dentes-afiados' }),
    )
    expect(sheet.attributes.charisma.tormentaMod).toBe(-1)
    expect(sheet.attributes.charisma.total).toBe(-1) // 1 base − 1 Lefou − 1 poder
    // e a perícia escolhida ainda ganha o +2
    const noDef = computeCharacterSheet(lefouInput(undefined))
    expect(sheet.skills.furtividade.total).toBe(noDef.skills.furtividade.total + 2)
  })

  it('perda de Carisma reflete em perícias de CAR (Intimidação)', () => {
    const withPower = computeCharacterSheet(
      lefouInput({ pericias: [], tormentaPower: 'antenas' }),
    )
    const without = computeCharacterSheet(lefouInput(undefined))
    expect(withPower.skills.intimidacao.total).toBe(
      without.skills.intimidacao.total - 1,
    )
  })

  it('escolha inválida vira warning, não lança', () => {
    const sheet = computeCharacterSheet(
      lefouInput({
        pericias: ['Furtividade', 'Percepção'],
        tormentaPower: 'dentes-afiados', // 3 slots > 2
      }),
    )
    expect(sheet.warnings.some((w) => w.includes('Deformidade'))).toBe(true)
  })

  it('perícia desconhecida vira warning e é ignorada', () => {
    const sheet = computeCharacterSheet(
      lefouInput({ pericias: ['Xadrez' as never] }),
    )
    expect(sheet.warnings.some((w) => w.includes('Xadrez'))).toBe(true)
  })
})

describe('perda de Carisma escala com TODOS os poderes da Tormenta (p136)', () => {
  it('poderes escolhidos no pool também perdem CAR (2 poderes → −2)', () => {
    const sheet = computeCharacterSheet({
      ...lefouInput(undefined),
      powerIds: ['antenas', 'carapaca'],
    })
    // sequência p136: 1º −1, 2º −1 ⇒ total −2
    expect(sheet.attributes.charisma.tormentaMod).toBe(-2)
  })

  it('swap da Deformidade + 2 escolhidos = 3 poderes → −4 (escalada)', () => {
    const sheet = computeCharacterSheet({
      ...lefouInput({ pericias: [], tormentaPower: 'dentes-afiados' }),
      powerIds: ['antenas', 'carapaca'],
    })
    // 1→1, 2→2, 3→4 total
    expect(sheet.attributes.charisma.tormentaMod).toBe(-4)
  })

  it('poder duplicado (Deformidade + pool) conta uma vez + warning', () => {
    const sheet = computeCharacterSheet({
      ...lefouInput({ pericias: [], tormentaPower: 'dentes-afiados' }),
      powerIds: ['dentes-afiados'],
    })
    expect(sheet.attributes.charisma.tormentaMod).toBe(-1)
    expect(sheet.warnings.some((w) => w.includes('duplicado'))).toBe(true)
  })

  it('poderes não-Tormenta em powerIds não perdem CAR', () => {
    const sheet = computeCharacterSheet({
      ...lefouInput(undefined),
      powerIds: ['esquiva', 'class.barbaro.totem-espiritual'],
    })
    expect(sheet.attributes.charisma.tormentaMod).toBeUndefined()
  })
})
