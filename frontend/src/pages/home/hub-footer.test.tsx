import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { HubFooter } from './hub-footer'

const baseProps = {
  name: 'Alexandre',
  theme: 'dark' as const,
  onToggleTheme: () => {},
  onLogout: () => {},
}

const openMenu = async () =>
  userEvent.click(screen.getByRole('button', { name: 'Menu de Alexandre' }))

describe('HubFooter', () => {
  it('shows the name and its initial on the trigger', () => {
    render(<HubFooter {...baseProps} />)
    expect(screen.getByText('Alexandre')).toBeInTheDocument()
    expect(screen.getByText('A')).toBeInTheDocument()
  })

  it('keeps the quick menu closed until the portrait is clicked', async () => {
    render(<HubFooter {...baseProps} />)
    expect(screen.queryByRole('button', { name: 'Sair' })).not.toBeInTheDocument()
    await openMenu()
    expect(screen.getByRole('button', { name: 'Sair' })).toBeInTheDocument()
  })

  it('labels the theme item by the target mode', async () => {
    const { rerender } = render(<HubFooter {...baseProps} theme="dark" />)
    await openMenu()
    expect(screen.getByRole('button', { name: 'Modo claro' })).toBeInTheDocument()
    rerender(<HubFooter {...baseProps} theme="light" />)
    expect(screen.getByRole('button', { name: 'Modo escuro' })).toBeInTheDocument()
  })

  it('fires onToggleTheme and onLogout from the menu', async () => {
    const onToggleTheme = vi.fn()
    const onLogout = vi.fn()
    render(
      <HubFooter
        {...baseProps}
        onToggleTheme={onToggleTheme}
        onLogout={onLogout}
      />,
    )
    await openMenu()
    await userEvent.click(screen.getByRole('button', { name: /Modo/ }))
    await userEvent.click(screen.getByRole('button', { name: 'Sair' }))
    expect(onToggleTheme).toHaveBeenCalledOnce()
    expect(onLogout).toHaveBeenCalledOnce()
  })

  it('disables Configurações (placeholder) and Sair while logging out', async () => {
    render(<HubFooter {...baseProps} logoutPending />)
    await openMenu()
    expect(screen.getByRole('button', { name: 'Configurações' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Sair' })).toBeDisabled()
  })
})
