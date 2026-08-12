import { createRoot } from 'solid-js'
import { describe, expect, it } from 'vitest'
import { createEncounterDraft } from './encounter-draft'

const withDraft = <T>(fn: (draft: ReturnType<typeof createEncounterDraft>) => T): T =>
  createRoot((dispose) => {
    const result = fn(createEncounterDraft())
    dispose()
    return result
  })

describe('createEncounterDraft', () => {
  it('começa vazio, num grupo de 4 no nível 1', () => {
    withDraft((draft) => {
      expect(draft.entries()).toEqual([])
      expect(draft.partyLevel()).toBe(1)
      expect(draft.partySize()).toBe(4)
    })
  })

  it('adicionar de novo SOMA na quantidade, não cria segunda linha', () => {
    withDraft((draft) => {
      draft.add('goblin')
      draft.add('goblin')

      // Duas linhas do mesmo monstro calculariam dois NDs de grupo, e a regra
      // de dobra do p282 só significa alguma coisa sobre UM grupo.
      expect(draft.entries()).toEqual([{ monsterId: 'goblin', quantity: 2 }])
    })
  })

  it('mantém criaturas diferentes em linhas próprias', () => {
    withDraft((draft) => {
      draft.add('goblin')
      draft.add('ogro')

      expect(draft.entries().map((e) => e.monsterId)).toEqual(['goblin', 'ogro'])
    })
  })

  it('quantidade nunca desce abaixo de 1 — some pelo remover', () => {
    withDraft((draft) => {
      draft.add('goblin')
      draft.setQuantity('goblin', 0)

      expect(draft.entries()[0].quantity).toBe(1)
    })
  })

  it('remove tira a linha inteira', () => {
    withDraft((draft) => {
      draft.add('goblin')
      draft.add('ogro')
      draft.remove('goblin')

      expect(draft.entries().map((e) => e.monsterId)).toEqual(['ogro'])
    })
  })

  it('prende o grupo nas faixas do livro', () => {
    withDraft((draft) => {
      draft.setPartyLevel(99)
      draft.setPartySize(0)

      expect(draft.partyLevel()).toBe(20)
      expect(draft.partySize()).toBe(1)
    })
  })

  it('um NaN vindo do campo não vira nível NaN', () => {
    withDraft((draft) => {
      draft.setPartyLevel(Number.NaN)

      expect(draft.partyLevel()).toBe(1)
    })
  })

  it('limpar esvazia a composição sem mexer no grupo', () => {
    withDraft((draft) => {
      draft.setPartyLevel(5)
      draft.add('goblin')
      draft.clear()

      expect(draft.entries()).toEqual([])
      expect(draft.partyLevel()).toBe(5)
    })
  })
})
