import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { ForgeProvider } from '@/features/character-build/forge-context'
import type { CharacterOptions } from '@/shared/api/api'
import {
  type CharacterDraftStore,
  createCharacterDraftStore,
} from '@/shared/stores/character-draft-store'
import { FakeStorage } from '@/shared/test/fake-storage'
import { AtributosStep } from './atributos-step'

const OPTIONS = {
  races: ['Anão'],
  classes: ['Guerreiro'],
  origins: [],
  gods: [],
  sizes: ['Médio'],
  expertises: [],
} as unknown as CharacterOptions

function renderStep(setup: (draft: CharacterDraftStore) => void = () => {}) {
  const draft = createCharacterDraftStore(new FakeStorage())
  setup(draft)
  render(() => (
    <ForgeProvider draft={draft} options={OPTIONS}>
      <AtributosStep />
    </ForgeProvider>
  ))
  return { draft }
}

describe('AtributosStep — as seis colunas', () => {
  it('cada atributo tem seu campo, alcançável pelo nome', () => {
    renderStep()

    expect(screen.getByLabelText('Força')).toHaveValue(0)
    expect(screen.getByLabelText('Carisma')).toHaveValue(0)
  })

  it('a seta sobe um ponto', async () => {
    const { draft } = renderStep()

    await userEvent.click(screen.getByRole('button', { name: 'Aumentar Força' }))

    expect(draft.values.strength).toBe(1)
  })

  it('a seta desce um ponto', async () => {
    const { draft } = renderStep((d) => d.setValue('wisdom', 2))

    await userEvent.click(screen.getByRole('button', { name: 'Diminuir Sabedoria' }))

    expect(draft.values.wisdom).toBe(1)
  })

  it('dá para digitar direto, sem seis toques para chegar em −1', async () => {
    const { draft } = renderStep()

    const field = screen.getByLabelText('Carisma')
    await userEvent.clear(field)
    await userEvent.type(field, '-1')

    expect(draft.values.charisma).toBe(-1)
    expect(field).toHaveFocus()
  })

  it('o bônus de raça entra no total, não na base', () => {
    renderStep((d) => {
      d.setValue('races', ['Anão'])
      d.setValue('constitution', 1)
    })

    expect(screen.getByLabelText('Constituição')).toHaveValue(1)
    expect(screen.getByText('Constituição total 3')).toBeInTheDocument()
  })

  it('avisa enquanto a raça deve uma escolha de atributo', () => {
    renderStep((d) => d.setValue('races', ['Humano']))

    expect(screen.getByText(/escolhas de atributo de raça pendentes/)).toBeInTheDocument()
  })
})

describe('AtributosStep — compra de pontos (p17)', () => {
  it('no modo livre não existe medidor de pontos', () => {
    renderStep()

    expect(screen.queryByRole('progressbar')).not.toBeInTheDocument()
  })

  it('trocar para compra de pontos abre o medidor', async () => {
    const { draft } = renderStep((d) => d.setValue('strength', 2))

    await userEvent.click(screen.getByRole('button', { name: /Compra de pontos/ }))

    expect(draft.attributeMode()).toBe('point-buy')
    expect(screen.getByRole('progressbar')).toHaveAttribute('aria-valuenow', '2')
  })

  it('mostra o preço de cada valor pela Tabela 1-1', async () => {
    renderStep((d) => {
      d.setAttributeMode('point-buy')
      d.setValue('strength', 4)
    })

    expect(screen.getByText('7 pts')).toBeInTheDocument()
  })

  it('estourar o orçamento vira aviso, não bloqueio', async () => {
    renderStep((d) => {
      d.setAttributeMode('point-buy')
      d.setValue('strength', 4)
      d.setValue('dexterity', 4)
    })

    expect(screen.getByText(/excedem o limite/)).toBeInTheDocument()
    expect(screen.getByLabelText('Força')).toHaveValue(4)
  })

  it('na compra de pontos as setas param nos limites da tabela', async () => {
    const { draft } = renderStep((d) => {
      d.setAttributeMode('point-buy')
      d.setValue('strength', 4)
    })

    expect(screen.getByRole('button', { name: 'Aumentar Força' })).toBeDisabled()
    expect(draft.values.strength).toBe(4)
  })

  it('no modo livre o teto é bem mais alto', () => {
    renderStep((d) => d.setValue('strength', 4))

    expect(screen.getByRole('button', { name: 'Aumentar Força' })).toBeEnabled()
  })
})
