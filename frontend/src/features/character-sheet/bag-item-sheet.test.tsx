import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { CharacterItem } from '@/shared/api/api'
import { BagItemSheet } from './bag-item-sheet'
import { ItemRefused, type ItemActions } from './item-mutations'

function item(overrides: Partial<CharacterItem> = {}): CharacterItem {
  return {
    id: 1,
    catalogId: 'espada-longa',
    name: 'Espada longa',
    quantity: 1,
    slots: 1,
    equipped: null,
    improvements: '[]',
    material: null,
    ...overrides,
  }
}

/** Named fake of the mutation surface — no network, no query client. */
class FakeItemActions implements ItemActions {
  changes: { id: number; input: unknown }[] = []
  removed: number[] = []
  refusal: string | null = null

  add = async () => {}
  change = async (id: number, input: unknown) => {
    if (this.refusal) throw new ItemRefused(this.refusal)
    this.changes.push({ id, input })
  }
  remove = async (id: number) => {
    this.removed.push(id)
  }
  consume = async () => {}
}

afterEach(() => {
  document.body.innerHTML = ''
})

function renderSheet(actions: ItemActions, row = item()) {
  const onOpenChange = vi.fn()
  render(() => (
    <BagItemSheet item={row} proficient open onOpenChange={onOpenChange} actions={actions} />
  ))
  return { user: userEvent.setup(), onOpenChange }
}

describe('BagItemSheet', () => {
  it('oferece só os estados de equipar que faltam', () => {
    renderSheet(new FakeItemActions(), item({ equipped: 'wielded' }))
    // Já empunhada em uma mão: sobram Guardar e as duas mãos.
    expect(screen.getByRole('button', { name: 'Guardar' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Empunhar (1 mão)' })).not.toBeInTheDocument()
  })

  it('equipar fecha a ficha depois que a escrita passa', async () => {
    const actions = new FakeItemActions()
    const { user, onOpenChange } = renderSheet(actions)

    await user.click(screen.getByRole('button', { name: 'Empunhar (1 mão)' }))

    expect(actions.changes).toEqual([{ id: 1, input: { equipped: 'wielded' } }])
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })

  /**
   * A recusa da regra tem de aparecer DENTRO da ficha: o Kobalte marca tudo que
   * é irmão do modal como `aria-hidden`, então um toast disparado daqui não é
   * anunciado por leitor de tela nem é visto por quem olha o diálogo.
   */
  it('recusa da regra fica na ficha, que não fecha', async () => {
    const actions = new FakeItemActions()
    actions.refusal = 'Limite de 2 mãos atingido'
    const { user, onOpenChange } = renderSheet(actions)

    await user.click(screen.getByRole('button', { name: 'Empunhar (1 mão)' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('Limite de 2 mãos atingido')
    expect(onOpenChange).not.toHaveBeenCalledWith(false)
  })

  it('remover pede confirmação antes de apagar', async () => {
    const actions = new FakeItemActions()
    const { user } = renderSheet(actions)

    await user.click(screen.getByRole('button', { name: 'Remover Espada longa' }))
    expect(actions.removed).toEqual([])

    await user.click(screen.getByRole('button', { name: /^Remover$/ }))
    expect(actions.removed).toEqual([1])
  })

  // Poção não aceita melhoria: o rodapé não pode anunciar um botão que não existe.
  it('o rodapé anuncia só as ações presentes', () => {
    renderSheet(new FakeItemActions(), item({ catalogId: 'balsamo-restaurador', name: 'Bálsamo' }))
    expect(screen.getByText('editar · remover')).toBeInTheDocument()
    expect(screen.queryByLabelText('Melhorias e material de Bálsamo')).not.toBeInTheDocument()
  })
})
