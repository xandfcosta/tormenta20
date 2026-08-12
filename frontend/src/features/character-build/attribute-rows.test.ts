import { describe, expect, it } from 'vitest'
import { attributeRows } from './attribute-rows'
import { wizardDefaults } from './wizard-steps'

const draft = (patch: Partial<typeof wizardDefaults> = {}) => ({
  ...wizardDefaults,
  ...patch,
})

const rowFor = (rows: ReturnType<typeof attributeRows>, key: string) => {
  const row = rows.find((r) => r.key === key)
  if (!row) throw new Error(`atributo ${key} ausente nas colunas`)
  return row
}

describe('attributeRows', () => {
  it('devolve os seis atributos na ordem do livro', () => {
    const rows = attributeRows(draft(), {}, 'free')

    expect(rows.map((r) => r.key)).toEqual([
      'strength',
      'dexterity',
      'constitution',
      'intelligence',
      'wisdom',
      'charisma',
    ])
  })

  it('rotula em pt-BR', () => {
    expect(rowFor(attributeRows(draft(), {}, 'free'), 'strength').label).toBe('Força')
  })

  it('sem raça, o total é a base', () => {
    const rows = attributeRows(draft({ strength: 3 }), {}, 'free')

    expect(rowFor(rows, 'strength')).toMatchObject({ base: 3, raceDelta: 0, total: 3 })
  })

  it('soma o bônus racial ao total sem tocar na base', () => {
    // A base é o que o jogador edita; o bônus de raça entra por fora, senão
    // trocar de raça teria de reescrever o que ele digitou.
    const rows = attributeRows(draft({ races: ['Anão'], constitution: 1 }), {}, 'free')

    expect(rowFor(rows, 'constitution')).toMatchObject({ base: 1, raceDelta: 2, total: 3 })
  })

  it('penalidade racial também aparece', () => {
    const rows = attributeRows(draft({ races: ['Anão'] }), {}, 'free')

    expect(rowFor(rows, 'dexterity').raceDelta).toBe(-1)
  })

  it('raça secundária só conta se o mestre autorizou', () => {
    const values = draft({ races: ['Anão', 'Elfo'] })

    const semOpt = attributeRows(values, {}, 'free')
    const comOpt = attributeRows(values, { Elfo: { applied: true } }, 'free')

    expect(rowFor(semOpt, 'intelligence').raceDelta).toBe(0)
    expect(rowFor(comOpt, 'intelligence').raceDelta).toBe(2)
  })

  it('no modo livre não há custo em pontos', () => {
    const rows = attributeRows(draft({ strength: 4 }), {}, 'free')

    expect(rowFor(rows, 'strength').cost).toBeNull()
  })

  it('na compra de pontos, cada valor custa o da Tabela 1-1', () => {
    const rows = attributeRows(draft({ strength: 4, charisma: -1 }), {}, 'point-buy')

    expect(rowFor(rows, 'strength').cost).toBe(7)
    expect(rowFor(rows, 'charisma').cost).toBe(-1)
  })

  it('valor fora da tabela não tem custo — e não estoura', () => {
    const rows = attributeRows(draft({ strength: 9 }), {}, 'point-buy')

    expect(rowFor(rows, 'strength').cost).toBeNull()
  })

  it('poder da Tormenta cobra Carisma no total, não na base', () => {
    // A perda de CAR (p136) é derivada das escolhas, não digitada — ela some do
    // campo editável e aparece no total, como o bônus racial.
    const values = draft({
      races: ['Lefou'],
      classPowers: [],
    })
    const rows = attributeRows(
      values,
      { Lefou: { applied: true, deformidade: { pericias: [], tormentaPower: 'dentes-afiados' } } },
      'free',
    )

    expect(rowFor(rows, 'charisma').raceDelta).toBeLessThan(0)
    expect(rowFor(rows, 'charisma').base).toBe(0)
  })
})
