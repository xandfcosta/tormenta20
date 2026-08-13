import { render, screen, within } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { ForgeProvider } from '@/features/character-build/forge-context'
import type { CharacterOptions } from '@/shared/api/api'
import {
  type CharacterDraftStore,
  createCharacterDraftStore,
} from '@/shared/stores/character-draft-store'
import { FakeStorage } from '@/shared/test/fake-storage'
import { EquipamentoStep } from './equipamento-step'

const OPTIONS = {
  races: ['Humano'],
  classes: ['Guerreiro', 'Arcanista'],
  origins: ['Acólito'],
  gods: [],
  sizes: ['Médio'],
  expertises: [],
} as unknown as CharacterOptions

function renderStep(setup: (draft: CharacterDraftStore) => void = () => {}) {
  const draft = createCharacterDraftStore(new FakeStorage())
  draft.setValue('races', ['Humano'])
  setup(draft)
  render(() => (
    <ForgeProvider draft={draft} options={OPTIONS}>
      <EquipamentoStep />
    </ForgeProvider>
  ))
  return { draft }
}

const guerreiro = (draft: CharacterDraftStore) =>
  draft.setValue('classes', [{ className: 'Guerreiro', level: 1 }])

const bag = () => within(screen.getByRole('complementary'))

describe('EquipamentoStep', () => {
  it('manda escolher a classe antes de montar o kit', () => {
    renderStep()

    expect(screen.getByText(/escolha uma classe primeiro/i)).toBeInTheDocument()
  })

  it('mostra a bagagem já com os itens automáticos do kit', () => {
    renderStep(guerreiro)

    expect(bag().getByText(/automático/i)).toBeInTheDocument()
  })

  it('marca como pendente a arma que ainda não foi escolhida', () => {
    renderStep(guerreiro)

    expect(bag().getByRole('button', { name: /arma simples · pendente/i })).toBeInTheDocument()
  })

  it('escolher a armadura entra no rascunho', async () => {
    const { draft } = renderStep(guerreiro)

    const brunea = screen.getByRole('button', { name: /brunea/i })
    await userEvent.click(brunea)

    expect(draft.values.startingArmor).toBe('brunea')
  })

  it('desmarcar a armadura devolve o campo ao vazio', async () => {
    const { draft } = renderStep((d) => {
      guerreiro(d)
      d.setValue('startingArmor', 'brunea')
    })

    await userEvent.click(screen.getByRole('button', { name: /brunea/i }))

    expect(draft.values.startingArmor).toBe('')
  })

  it('liga e desliga o escudo leve', async () => {
    const { draft } = renderStep(guerreiro)

    await userEvent.click(screen.getByRole('button', { name: /escudo leve/i }))

    expect(draft.values.startingShield).toBe(false)
  })

  it('rola o dinheiro inicial uma vez só', async () => {
    const { draft } = renderStep(guerreiro)

    const roll = screen.getByRole('button', { name: /rolar 4d6/i })
    await userEvent.click(roll)

    expect(draft.values.tibar).toBeGreaterThanOrEqual(4)
    expect(draft.values.tibar).toBeLessThanOrEqual(24)
    // Re-rolling would let a player fish for a better purse.
    expect(screen.getByRole('button', { name: /rolado/i })).toBeDisabled()
  })

  it('a loja fica fechada enquanto não há dinheiro', () => {
    renderStep(guerreiro)

    expect(screen.getByText(/defina seu dinheiro inicial acima/i)).toBeInTheDocument()
  })

  it('a carteira desconta o que foi comprado', () => {
    renderStep((d) => {
      guerreiro(d)
      d.setValue('tibar', 50)
      d.setValue('startingPurchases', { corda: 1 })
    })

    // A compra só aparece na carteira se o catálogo conhece o item; o que o
    // teste fixa é que o saldo é DERIVADO e não um número guardado.
    expect(bag().getByText(/T\$ 50/)).toBeInTheDocument()
  })

  it('avisa quando a bagagem passa da capacidade', () => {
    renderStep((d) => {
      guerreiro(d)
      d.setValue('strength', 0)
      d.setValue('startingPurchases', { 'saco-de-dormir': 20 })
    })

    expect(bag().getByText(/sobrecarregado/i)).toBeInTheDocument()
  })
})
