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

describe('HubFooter', () => {
  it('shows the name and its initial', () => {
    render(<HubFooter {...baseProps} />)
    expect(screen.getByText('Alexandre')).toBeInTheDocument()
    expect(screen.getByText('A')).toBeInTheDocument()
  })

  it('labels the theme toggle by the target mode', () => {
    const { rerender } = render(<HubFooter {...baseProps} theme="dark" />)
    expect(screen.getByRole('button', { name: 'Modo claro' })).toBeInTheDocument()
    rerender(<HubFooter {...baseProps} theme="light" />)
    expect(screen.getByRole('button', { name: 'Modo escuro' })).toBeInTheDocument()
  })

  it('fires onToggleTheme and onLogout', async () => {
    const onToggleTheme = vi.fn()
    const onLogout = vi.fn()
    render(
      <HubFooter
        {...baseProps}
        onToggleTheme={onToggleTheme}
        onLogout={onLogout}
      />,
    )
    await userEvent.click(screen.getByRole('button', { name: /Modo/ }))
    await userEvent.click(screen.getByRole('button', { name: 'Sair' }))
    expect(onToggleTheme).toHaveBeenCalledOnce()
    expect(onLogout).toHaveBeenCalledOnce()
  })

  it('disables logout while pending', () => {
    render(<HubFooter {...baseProps} logoutPending />)
    expect(screen.getByRole('button', { name: 'Sair' })).toBeDisabled()
  })
})
