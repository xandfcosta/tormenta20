import { render, screen, waitFor } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { Dialog, DialogContent, DialogTitle, DialogTrigger } from '@/shared/ui/dialog'
import { SceneShell } from './scene-shell'

function mockMotion(reduced: boolean): void {
  window.matchMedia = vi.fn().mockImplementation((media: string) => ({
    matches: media.includes('prefers-reduced-motion') ? reduced : false,
    media,
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))
}

beforeEach(() => mockMotion(false))
afterEach(() => {
  vi.clearAllMocks()
  document.body.innerHTML = ''
})

describe('SceneShell', () => {
  // 'rende o conteúdo que recebe' saiu na ALE-187: passagem de `children`, que o Solid faz
  // e o typechecker garante.

  it('layout padrão mostra o título cinematográfico com kicker', () => {
    render(() => (
      <SceneShell title="Tormenta 20" kicker="— Grimório de Arton —">
        x
      </SceneShell>
    ))
    expect(screen.getByRole('heading', { level: 1, name: 'Tormenta 20' })).toBeInTheDocument()
    expect(screen.getByText('— Grimório de Arton —')).toBeInTheDocument()
  })

  it('layout dense usa o header compacto e aceita controles à direita', () => {
    const { container } = render(() => (
      <SceneShell dense title="Personagens" headerRight={<button type="button">Novo</button>}>
        x
      </SceneShell>
    ))
    expect(container.querySelector('[data-slot=scene-shell]')).toHaveAttribute('data-dense', 'true')
    expect(screen.getByRole('heading', { level: 1, name: 'Personagens' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Novo' })).toBeInTheDocument()
  })

  it('o controle de voltar chama onBack', async () => {
    const onBack = vi.fn()
    render(() => (
      <SceneShell onBack={onBack} backLabel="Voltar ao Hub">
        x
      </SceneShell>
    ))
    await userEvent.setup().click(screen.getByRole('button', { name: 'Voltar ao Hub' }))
    expect(onBack).toHaveBeenCalledOnce()
  })

  it('sem onBack não rende controle nenhum', () => {
    render(() => <SceneShell>x</SceneShell>)
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

  it('onEnter dispara uma vez ao montar', () => {
    const onEnter = vi.fn()
    render(() => <SceneShell onEnter={onEnter}>x</SceneShell>)
    expect(onEnter).toHaveBeenCalledOnce()
  })

  // WCAG 2.3.3: quem pediu menos movimento não ganha nem a animação nem a
  // deixa sonora que anda junto com ela.
  it('respeita prefers-reduced-motion: sem animação e sem onEnter', () => {
    mockMotion(true)
    const onEnter = vi.fn()
    const { container } = render(() => <SceneShell onEnter={onEnter}>x</SceneShell>)
    expect(onEnter).not.toHaveBeenCalled()
    const content = container.querySelector('[data-slot=scene-content]')
    expect(content?.className).not.toContain('scene-in')
    expect(content).not.toHaveAttribute('data-animate')
  })

  // A razão de o container de cena existir: sem isso o dialog abre no body e
  // perde o escopo `.scene-grimorio`, renderizando shadcn puro sobre a cena.
  it('overlays portalam PRA DENTRO da cena, herdando os tokens', async () => {
    const { container } = render(() => (
      <SceneShell>
        <Dialog>
          <DialogTrigger>Abrir</DialogTrigger>
          <DialogContent>
            <DialogTitle>Confirmar</DialogTitle>
          </DialogContent>
        </Dialog>
      </SceneShell>
    ))
    await userEvent.setup().click(screen.getByRole('button', { name: 'Abrir' }))

    const dialog = await screen.findByRole('dialog')
    const scene = container.querySelector('[data-slot=scene-shell]')
    await waitFor(() => expect(scene?.contains(dialog)).toBe(true))
  })
})
