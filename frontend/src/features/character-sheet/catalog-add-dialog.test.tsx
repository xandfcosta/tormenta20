import { render, screen, waitFor } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { For } from 'solid-js'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { CreateItemInput } from '@/shared/api/api'
import { CatalogAddDialog } from './catalog-add-dialog'
import { ItemRefused } from './item-mutations'

/**
 * ADICIONAR DO CATÁLOGO (ALE-197, grupo B).
 *
 * O `filterCatalog` já é testado como função. O que nunca foi montado é o
 * diálogo: escolher a entrada, o que aparece DEPOIS de escolher (o seletor de
 * "Equipar" só existe para o que se equipa) e o que sai no `onAdd`.
 *
 * A `VirtualList` é trocada por lista simples pelo mesmo motivo do
 * `apply-effect-dialog.test.tsx`: no jsdom ela mede zero e não renderiza linha
 * nenhuma, e um teste sobre lista vazia não prova nada. O virtualizador tem
 * teste próprio e cobertura de browser.
 */
vi.mock('@/shared/ui/virtual-list', () => ({
  VirtualList: <T,>(props: {
    items: readonly T[]
    getKey: (item: T, index: number) => string | number
    renderItem: (item: T) => unknown
  }) => (
    <ul>
      <For each={props.items}>{(item) => <li>{props.renderItem(item) as never}</li>}</For>
    </ul>
  ),
}))

function renderDialog(onAdd = vi.fn<(input: CreateItemInput) => Promise<void>>()) {
  render(() => <CatalogAddDialog onAdd={onAdd} />)
  return { user: userEvent.setup(), onAdd }
}

async function abrir(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByRole('button', { name: 'Adicionar do catálogo' }))
  await screen.findByRole('dialog')
}

/** Busca e escolhe a entrada pelo nome — o caminho de quem usa a tela. */
async function escolher(user: ReturnType<typeof userEvent.setup>, busca: string, nome: RegExp) {
  await user.type(screen.getByLabelText('Buscar no catálogo'), busca)
  await user.click(await screen.findByRole('button', { name: nome }))
}

afterEach(() => {
  vi.restoreAllMocks()
  document.body.innerHTML = ''
})

describe('CatalogAddDialog', () => {
  it('sem escolher nada, não dá para adicionar', async () => {
    const { user } = renderDialog()

    await abrir(user)

    // "Adicionar" travado é o que impede um item sem catálogo de nascer.
    expect(screen.getByRole('button', { name: 'Adicionar' })).toBeDisabled()
  })

  it('escolher a entrada manda o id, a quantidade e o slot', async () => {
    const onAdd = vi.fn().mockResolvedValue(undefined)
    const { user } = renderDialog(onAdd)
    await abrir(user)

    await escolher(user, 'escudo leve', /Escudo leve/)
    await user.selectOptions(screen.getByLabelText('Equipar'), 'wielded')
    await user.click(screen.getByRole('button', { name: 'Adicionar' }))

    await waitFor(() =>
      expect(onAdd).toHaveBeenCalledWith({
        catalogId: 'escudo-leve',
        quantity: 1,
        equipped: 'wielded',
      }),
    )
  })

  it('o que não se equipa não oferece onde equipar', async () => {
    const { user } = renderDialog()
    await abrir(user)

    await escolher(user, 'bálsamo', /Bálsamo restaurador/)

    // Uma poção só tem "—": oferecer um seletor de uma escolha morta é pior
    // que não oferecer nada.
    expect(screen.queryByLabelText('Equipar')).not.toBeInTheDocument()
  })

  it('a busca acha por CATEGORIA, não só por nome', async () => {
    const { user } = renderDialog()
    await abrir(user)

    await user.type(screen.getByLabelText('Buscar no catálogo'), 'armor')

    // Quem procura pelo tipo não sabe o nome de cada armadura do livro.
    expect(await screen.findByRole('button', { name: /Cota de malha/ })).toBeInTheDocument()
  })

  it('busca sem resultado diz "Nenhum item" em vez de lista vazia', async () => {
    const { user } = renderDialog()
    await abrir(user)

    await user.type(screen.getByLabelText('Buscar no catálogo'), 'zzzz')

    expect(await screen.findByText('Nenhum item.')).toBeInTheDocument()
  })

  it('a recusa da REGRA chega ao diálogo com a frase dela', async () => {
    const onAdd = vi.fn().mockRejectedValue(new ItemRefused('Limite de 2 mãos atingido'))
    const { user } = renderDialog(onAdd)
    await abrir(user)

    await escolher(user, 'escudo leve', /Escudo leve/)
    await user.click(screen.getByRole('button', { name: 'Adicionar' }))

    // Verbatim: a regra já escreveu a frase certa, e trocá-la pelo texto de
    // último recurso ("não foi possível") esconderia o motivo de quem pode agir
    // sobre ele — largar o escudo.
    expect(await screen.findByText('Limite de 2 mãos atingido')).toBeVisible()
    expect(screen.getByRole('dialog')).toBeInTheDocument()
  })
})
