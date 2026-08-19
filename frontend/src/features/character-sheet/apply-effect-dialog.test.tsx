import { render, screen, waitFor } from '@solidjs/testing-library'
import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import userEvent from '@testing-library/user-event'
import { For } from 'solid-js'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { makeCharacter } from '@/entities/character/__fixtures__/character'
import { characterQueryOptions } from '@/entities/character/queries'
import { api, type Character } from '@/shared/api/api'
import { ApplyEffectDialog } from './apply-effect-dialog'

/**
 * APLICAR EFEITO DE MAGIA (ALE-197, grupo B).
 *
 * A `VirtualList` mede zero no jsdom e NÃO renderiza linha nenhuma — está
 * escrito no `virtual-list.test.tsx` e no CLAUDE.md do front. Um teste ingênuo
 * deste diálogo passaria verde sobre uma lista vazia, provando nada.
 *
 * Por isso a lista virtual é trocada por uma lista simples aqui: o que este
 * arquivo cobre é a COMPOSIÇÃO do diálogo — buscar, escolher, o que vai para a
 * api, o cache, e o erro inline —, não o virtualizador, que tem teste próprio
 * (altura reservada + guarda de índice) e cobertura de browser no
 * `character-bag.spec.ts`.
 */
vi.mock('@/shared/ui/virtual-list', () => ({
  VirtualList: <T,>(props: {
    items: readonly T[]
    getKey: (item: T, index: number) => string | number
    renderItem: (item: T) => unknown
  }) => <ul>{<For each={props.items}>{(item) => <li>{props.renderItem(item) as never}</li>}</For>}</ul>,
}))

const HEROI = makeCharacter({ id: 1, name: 'Arwen' })

function renderDialog(character: Character = HEROI) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  client.setQueryData(characterQueryOptions(character.id).queryKey, character)
  render(() => (
    <QueryClientProvider client={client}>
      <ApplyEffectDialog character={character} />
    </QueryClientProvider>
  ))
  return { user: userEvent.setup(), client }
}

async function abrir(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByRole('button', { name: 'Aplicar efeito' }))
  await screen.findByRole('dialog')
}

afterEach(() => {
  vi.restoreAllMocks()
  document.body.innerHTML = ''
})

describe('ApplyEffectDialog', () => {
  it('lista só as magias que TÊM efeito aplicável', async () => {
    const { user } = renderDialog()

    await abrir(user)

    // "Escudo da Fé" tem buff; "Bola de Fogo" não é efeito que se aplique a
    // alguém — oferecê-la aqui seria oferecer uma linha que não faz nada.
    expect(await screen.findByRole('button', { name: /Escudo da Fé/ })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Bola de Fogo/ })).not.toBeInTheDocument()
  })

  it('a busca ignora acento', async () => {
    const { user } = renderDialog()
    await abrir(user)

    await user.type(screen.getByLabelText('Buscar magia'), 'escudo da fe')

    // Sem a normalização, quem digita "fe" no telefone não acha "Fé".
    expect(await screen.findByRole('button', { name: /Escudo da Fé/ })).toBeInTheDocument()
  })

  it('busca sem resultado diz que não há nada, em vez de lista vazia', async () => {
    const { user } = renderDialog()
    await abrir(user)

    await user.type(screen.getByLabelText('Buscar magia'), 'zzzz')

    expect(await screen.findByText('Nenhuma magia com efeito aplicável.')).toBeInTheDocument()
  })

  it('escolher aplica a magia escolhida e fecha', async () => {
    const applyEffect = vi
      .spyOn(api.characters, 'applyEffect')
      .mockResolvedValue({ id: 3, catalogId: 'escudo-da-fe', scope: 'scene', modifiers: '[]', createdAt: '' })
    const { user } = renderDialog()
    await abrir(user)

    await user.click(await screen.findByRole('button', { name: /Escudo da Fé/ }))

    await waitFor(() => expect(applyEffect).toHaveBeenCalledWith(1, { spellId: 'escudo-da-fe' }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
  })

  it('a falha aparece DENTRO do diálogo, que continua aberto', async () => {
    vi.spyOn(api.characters, 'applyEffect').mockRejectedValue(new Error('fora do ar'))
    const { user } = renderDialog()
    await abrir(user)

    await user.click(await screen.findByRole('button', { name: /Escudo da Fé/ }))

    // Toast disparado de dentro de um modal NÃO é anunciado: o Kobalte marca os
    // irmãos como aria-hidden e a região do sonner é irmã. E fechar levaria
    // embora a busca que a pessoa já tinha feito.
    expect(await screen.findByText(/Não foi possível aplicar o efeito/)).toBeVisible()
    expect(screen.getByRole('dialog')).toBeInTheDocument()
  })
})
