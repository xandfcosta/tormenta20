import { describe, expect, it } from 'vitest'
import { missingNotice, overflowNotice, seedFixedExpertises } from './pericia-bands'
import { periciaPlan } from './pericia-helpers'

const plan = (className = 'Guerreiro', intMod = 0, races: string[] = []) => {
  const result = periciaPlan(className, intMod, races)
  if (!result) throw new Error(`classe ${className} sem plano de perícias`)
  return result
}

describe('seedFixedExpertises', () => {
  it('acrescenta as perícias que a classe treina de graça', () => {
    const guerreiro = plan()

    const next = seedFixedExpertises([], guerreiro)

    for (const fixed of guerreiro.fixed) expect(next).toContain(fixed)
  })

  it('não duplica o que já estava treinado', () => {
    const guerreiro = plan()
    const already = [guerreiro.fixed[0]]

    const next = seedFixedExpertises(already, guerreiro)

    expect(next.filter((name) => name === guerreiro.fixed[0])).toHaveLength(1)
  })

  it('preserva escolhas que o jogador já tinha feito', () => {
    const next = seedFixedExpertises(['Acrobacia'], plan())

    expect(next).toContain('Acrobacia')
  })

  it('devolve a MESMA lista quando não há nada a semear (não dispara escrita)', () => {
    const guerreiro = plan()
    const already = seedFixedExpertises([], guerreiro)

    expect(seedFixedExpertises(already, guerreiro)).toBe(already)
  })
})

describe('overflowNotice — a cota da classe transbordando na livre', () => {
  it('cala enquanto o jogador está dentro da cota da classe', () => {
    const guerreiro = plan('Guerreiro', 2)

    expect(overflowNotice(guerreiro, guerreiro.classPool.slice(0, 1))).toBeNull()
  })

  it('fala quando uma escolha da classe passa a gastar ponto livre', () => {
    const guerreiro = plan('Guerreiro', 2)
    const excesso = guerreiro.classPool.slice(0, guerreiro.classCount + 1)

    expect(overflowNotice(guerreiro, excesso)).toContain('1')
  })

  it('conta quantas escolhas transbordaram', () => {
    const guerreiro = plan('Guerreiro', 3)
    const excesso = guerreiro.classPool.slice(0, guerreiro.classCount + 2)

    expect(overflowNotice(guerreiro, excesso)).toContain('2')
  })

  it('cala quando não há cota livre para transbordar', () => {
    // Sem Inteligência nem raça generosa não existe orçamento livre; aí a cota
    // da classe simplesmente trava e não há nada a explicar.
    const guerreiro = plan('Guerreiro', 0)
    const excesso = guerreiro.classPool.slice(0, guerreiro.classCount + 1)

    expect(overflowNotice(guerreiro, excesso)).toBeNull()
  })
})

describe('missingNotice', () => {
  it('concorda no singular', () => {
    expect(missingNotice(1)).toContain('Falta 1 perícia —')
  })

  it('concorda no plural', () => {
    expect(missingNotice(3)).toContain('Faltam 3 perícias')
  })

  it('cala quando não falta nada', () => {
    expect(missingNotice(0)).toBeNull()
  })
})
