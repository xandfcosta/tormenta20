import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { HubFooter } from './hub-footer'

const baseProps = {
  name: 'Alexandre',
  onLogout: () => {},
  sfxEnabled: false,
  onToggleSfx: () => {},
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

  it('fires onLogout from the menu', async () => {
    const onLogout = vi.fn()
    render(<HubFooter {...baseProps} onLogout={onLogout} />)
    await openMenu()
    await userEvent.click(screen.getByRole('button', { name: 'Sair' }))
    expect(onLogout).toHaveBeenCalledOnce()
  })

  it('labels the sound item by state and toggles it', async () => {
    const onToggleSfx = vi.fn()
    const { rerender } = render(
      <HubFooter {...baseProps} sfxEnabled={false} onToggleSfx={onToggleSfx} />,
    )
    await openMenu()
    await userEvent.click(screen.getByRole('button', { name: 'Som desligado' }))
    expect(onToggleSfx).toHaveBeenCalledOnce()
    rerender(<HubFooter {...baseProps} sfxEnabled onToggleSfx={onToggleSfx} />)
    expect(screen.getByRole('button', { name: 'Som ligado' })).toBeInTheDocument()
  })

  it('disables Configurações (placeholder) and Sair while logging out', async () => {
    render(<HubFooter {...baseProps} logoutPending />)
    await openMenu()
    expect(screen.getByRole('button', { name: 'Configurações' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Sair' })).toBeDisabled()
  })
})
