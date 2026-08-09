import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Users2 } from 'lucide-react'
import { describe, expect, it, vi } from 'vitest'
import { GameMenuButton } from './game-menu-button'

describe('GameMenuButton', () => {
  it('renders its label inside a button', () => {
    render(<GameMenuButton>Meus Heróis</GameMenuButton>)
    expect(
      screen.getByRole('button', { name: /Meus Heróis/ }),
    ).toBeInTheDocument()
  })

  it('fires onClick when pressed', async () => {
    const onClick = vi.fn()
    render(<GameMenuButton onClick={onClick}>Crônicas</GameMenuButton>)
    await userEvent.click(screen.getByRole('button'))
    expect(onClick).toHaveBeenCalledOnce()
  })

  it('shows the ► chevron only when hasNext', () => {
    const { rerender } = render(<GameMenuButton>Ir</GameMenuButton>)
    expect(screen.queryByText('►')).not.toBeInTheDocument()
    rerender(<GameMenuButton hasNext>Continuar sessão</GameMenuButton>)
    expect(screen.getByText('►')).toBeInTheDocument()
  })

  it('marks the active destination with aria-current=page', () => {
    render(<GameMenuButton active>Início</GameMenuButton>)
    expect(screen.getByRole('button')).toHaveAttribute('aria-current', 'page')
  })

  it('renders the leading icon when provided', () => {
    const { container } = render(
      <GameMenuButton icon={Users2}>Meus Heróis</GameMenuButton>,
    )
    expect(container.querySelector('svg')).toBeInTheDocument()
  })

  it('defaults to type=button so it never submits a form', () => {
    render(<GameMenuButton>Seguro</GameMenuButton>)
    expect(screen.getByRole('button')).toHaveAttribute('type', 'button')
  })
})
