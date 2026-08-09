import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { SceneShell } from './scene-shell'

/**
 * Named fake for the reduced-motion query — lets each test pin whether the OS
 * is asking to minimize motion without a real browser. Mirrors the fake in
 * use-media-query.test.ts (project I/O-mock rule).
 */
function installReducedMotion(reduce: boolean) {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    configurable: true,
    value: vi.fn((query: string) => ({
      matches: query.includes('prefers-reduced-motion') ? reduce : false,
      media: query,
      addEventListener: () => {},
      removeEventListener: () => {},
    })),
  })
}

afterEach(() => vi.restoreAllMocks())

const shell = () => document.querySelector('[data-slot="scene-shell"]')
const content = () => document.querySelector('[data-slot="scene-content"]')

describe('SceneShell', () => {
  it('renders its children inside the grimório scene scope', () => {
    installReducedMotion(false)
    render(<SceneShell>corpo da cena</SceneShell>)
    expect(screen.getByText('corpo da cena')).toBeInTheDocument()
    expect(shell()).toHaveClass('scene-grimorio')
  })

  it('renders the scene title when provided', () => {
    installReducedMotion(false)
    render(<SceneShell title="Tormenta 20">x</SceneShell>)
    expect(
      screen.getByRole('heading', { name: 'Tormenta 20' }),
    ).toBeInTheDocument()
  })

  it('plays the enter transition when motion is allowed', () => {
    installReducedMotion(false)
    render(<SceneShell>x</SceneShell>)
    expect(content()).toHaveClass('scene-in')
  })

  it('omits the enter transition under prefers-reduced-motion', () => {
    installReducedMotion(true)
    render(<SceneShell>x</SceneShell>)
    expect(content()).not.toHaveClass('scene-in')
  })

  it('shows the back control only when onBack is set, and fires it', async () => {
    installReducedMotion(false)
    const onBack = vi.fn()
    const { rerender } = render(<SceneShell>x</SceneShell>)
    expect(
      screen.queryByRole('button', { name: /Voltar/ }),
    ).not.toBeInTheDocument()
    rerender(<SceneShell onBack={onBack}>x</SceneShell>)
    await userEvent.click(screen.getByRole('button', { name: /Voltar/ }))
    expect(onBack).toHaveBeenCalledOnce()
  })
})
