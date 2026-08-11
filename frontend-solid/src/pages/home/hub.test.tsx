import { render, screen, waitFor } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { Scroll, Users2 } from 'lucide-solid'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { HubFooter } from './hub-footer'
import { HubMenu } from './hub-menu'

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
  vi.clearAllMocks()
  document.body.innerHTML = ''
})

describe('HubMenu', () => {
  const items = [
    { label: 'Meus Heróis', icon: Users2, onSelect: vi.fn() },
    { label: 'Crônicas', icon: Scroll, onSelect: vi.fn() },
  ]

  it('rende cada entrada como um botão de verdade', () => {
    render(() => <HubMenu items={items} />)
    expect(screen.getByRole('button', { name: 'Meus Heróis' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Crônicas' })).toBeInTheDocument()
  })

  it('escolher uma entrada dispara o onSelect dela', async () => {
    render(() => <HubMenu items={items} />)
    await userEvent.setup().click(screen.getByRole('button', { name: 'Crônicas' }))
    expect(items[1].onSelect).toHaveBeenCalledOnce()
  })

  // O driver de navegação lê essas marcas do DOM — sem elas as setas não
  // enxergam o menu, e o teclado morre em silêncio.
  it('declara a região de navegação que as setas usam', () => {
    render(() => <HubMenu items={items} />)
    const nav = screen.getByRole('navigation', { name: 'Menu principal' })
    expect(nav).toHaveAttribute('data-nav-region', 'menu')
    expect(nav).toHaveAttribute('data-nav-layout', 'column')
  })

  it('a deixa de hover dispara ao passar o ponteiro', async () => {
    const onItemHover = vi.fn()
    render(() => <HubMenu items={items} onItemHover={onItemHover} />)
    await userEvent.setup().hover(screen.getByRole('button', { name: 'Meus Heróis' }))
    expect(onItemHover).toHaveBeenCalled()
  })

  it('mostra o chevron de continuar só em quem pediu', () => {
    render(() => <HubMenu items={[{ label: 'Continuar sessão', hasNext: true, onSelect: vi.fn() }]} />)
    expect(screen.getByRole('button', { name: 'Continuar sessão' })).toBeInTheDocument()
  })
})

describe('HubFooter', () => {
  function setup(overrides: Partial<Parameters<typeof HubFooter>[0]> = {}) {
    const props = {
      name: 'Mestre',
      onLogout: vi.fn(),
      sfxEnabled: false,
      onToggleSfx: vi.fn(),
      ...overrides,
    }
    render(() => <HubFooter {...props} />)
    return props
  }

  it('mostra a inicial do jogador e o nome', () => {
    setup()
    expect(screen.getByText('M')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Menu de Mestre' })).toBeInTheDocument()
  })

  it('cai pra ? quando não há nome', () => {
    setup({ name: '   ' })
    expect(screen.getByText('?')).toBeInTheDocument()
  })

  it('o menu rápido só abre no clique', async () => {
    setup()
    expect(screen.queryByText('Sair')).not.toBeInTheDocument()
    await userEvent.setup().click(screen.getByRole('button', { name: 'Menu de Mestre' }))
    expect(await screen.findByText('Sair')).toBeInTheDocument()
  })

  it('alterna o som pelo menu rápido', async () => {
    const props = setup()
    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: 'Menu de Mestre' }))
    await user.click(await screen.findByRole('button', { name: 'Som desligado' }))
    expect(props.onToggleSfx).toHaveBeenCalledOnce()
  })

  it('reflete o som ligado', async () => {
    setup({ sfxEnabled: true })
    await userEvent.setup().click(screen.getByRole('button', { name: 'Menu de Mestre' }))
    expect(await screen.findByRole('button', { name: 'Som ligado' })).toBeInTheDocument()
  })

  it('sair chama o logout', async () => {
    const props = setup()
    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: 'Menu de Mestre' }))
    await user.click(await screen.findByRole('button', { name: 'Sair' }))
    expect(props.onLogout).toHaveBeenCalledOnce()
  })

  it('trava o sair enquanto o logout está em voo, pra não disparar duas vezes', async () => {
    setup({ logoutPending: true })
    await userEvent.setup().click(screen.getByRole('button', { name: 'Menu de Mestre' }))
    await waitFor(() => expect(screen.getByRole('button', { name: 'Sair' })).toBeDisabled())
  })

  // Placeholder até existir tela de configurações — visível, mas inerte.
  it('configurações fica desabilitado', async () => {
    setup()
    await userEvent.setup().click(screen.getByRole('button', { name: 'Menu de Mestre' }))
    expect(await screen.findByRole('button', { name: 'Configurações' })).toBeDisabled()
  })
})
