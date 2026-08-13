import { render, screen, waitFor } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { makeCharacter } from '@/entities/character/__fixtures__/character'
import type { Character } from '@/shared/api/api'
import { SheetSearch } from './sheet-search'

/** A character with something findable in more than one block. */
function hero(): Character {
  return makeCharacter({
    classes: [{ className: 'Bárbaro', level: 6 }],
    activeConditions: '["caido"]',
    items: [
      {
        id: 1,
        catalogId: null,
        name: 'Adaga Rúnica',
        quantity: 2,
        slots: 1,
        equipped: null,
        improvements: '[]',
        material: null,
      },
    ],
  })
}

function renderSearch(onNavigate = vi.fn()) {
  render(() => <SheetSearch character={hero()} onNavigate={onNavigate} />)
  return { user: userEvent.setup(), onNavigate }
}

beforeEach(() => {
  window.matchMedia = vi.fn().mockImplementation((media: string) => ({
    matches: false,
    media,
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))
})

afterEach(() => {
  vi.restoreAllMocks()
  document.body.innerHTML = ''
})

describe('SheetSearch', () => {
  it('começa fechada', () => {
    renderSearch()
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('abre com "/"', async () => {
    const { user } = renderSearch()

    await user.keyboard('/')

    expect(await screen.findByRole('combobox', { name: 'Buscar na ficha' })).toBeInTheDocument()
  })

  it('abre com Ctrl+K', async () => {
    const { user } = renderSearch()

    await user.keyboard('{Control>}k{/Control}')

    expect(await screen.findByRole('combobox', { name: 'Buscar na ficha' })).toBeInTheDocument()
  })

  // "/" é caractere legítimo dentro de um campo; Ctrl+K não é.
  it('"/" digitado num campo não abre a paleta', async () => {
    const { user } = renderSearch()
    const field = document.createElement('input')
    document.body.append(field)
    field.focus()

    await user.keyboard('/')

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('atravessa os blocos: acha item e condição na mesma busca', async () => {
    const { user } = renderSearch()
    await user.keyboard('/')

    await user.type(await screen.findByRole('combobox', { name: 'Buscar na ficha' }), 'adaga')

    expect(await screen.findByText('Adaga Rúnica')).toBeInTheDocument()
    expect(screen.queryByText('Caído')).not.toBeInTheDocument()
  })

  it('escolher uma linha leva ao bloco dono do fato', async () => {
    const { user, onNavigate } = renderSearch()
    await user.keyboard('/')
    await user.type(await screen.findByRole('combobox', { name: 'Buscar na ficha' }), 'adaga')

    await user.click(await screen.findByRole('option', { name: /Adaga Rúnica/ }))

    await waitFor(() => expect(onNavigate).toHaveBeenCalledWith('bag'))
  })

  it('Enter escolhe a linha sob o cursor', async () => {
    const { user, onNavigate } = renderSearch()
    await user.keyboard('/')
    await user.type(await screen.findByRole('combobox', { name: 'Buscar na ficha' }), 'adaga')
    await screen.findByText('Adaga Rúnica')

    await user.keyboard('{Enter}')

    await waitFor(() => expect(onNavigate).toHaveBeenCalledWith('bag'))
  })

  it('busca sem resultado avisa em vez de sumir', async () => {
    const { user } = renderSearch()
    await user.keyboard('/')

    await user.type(await screen.findByRole('combobox', { name: 'Buscar na ficha' }), 'zzzz')

    expect(await screen.findByText('Nada encontrado.')).toBeInTheDocument()
  })

  // Critério de aceite da ALE-89: sair não pode deixar o cursor do scene-nav
  // órfão — a paleta abre por atalho, sem gatilho pra onde o foco voltar.
  it('fechar devolve o foco de onde veio', async () => {
    const { user } = renderSearch()
    const origin = document.createElement('button')
    document.body.append(origin)
    origin.focus()

    await user.keyboard('{Control>}k{/Control}')
    await screen.findByRole('combobox', { name: 'Buscar na ficha' })
    await user.keyboard('{Escape}')

    await waitFor(() => expect(document.activeElement).toBe(origin))
  })
})
