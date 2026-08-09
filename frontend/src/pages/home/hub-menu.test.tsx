import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { HubMenu, type HubMenuItem } from './hub-menu'

describe('HubMenu', () => {
  it('renders a button for every item', () => {
    const items: HubMenuItem[] = [
      { label: 'Meus Heróis', onSelect: vi.fn() },
      { label: 'Crônicas', onSelect: vi.fn() },
    ]
    render(<HubMenu items={items} />)
    expect(screen.getByRole('button', { name: /Meus Heróis/ })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Crônicas/ })).toBeInTheDocument()
  })

  it('fires the chosen item onSelect', async () => {
    const onSelect = vi.fn()
    render(<HubMenu items={[{ label: 'Crônicas', onSelect }]} />)
    await userEvent.click(screen.getByRole('button', { name: /Crônicas/ }))
    expect(onSelect).toHaveBeenCalledOnce()
  })

  it('fires onItemHover when the pointer enters an item', async () => {
    const onItemHover = vi.fn()
    render(
      <HubMenu
        items={[{ label: 'Crônicas', onSelect: vi.fn() }]}
        onItemHover={onItemHover}
      />,
    )
    await userEvent.hover(screen.getByRole('button', { name: /Crônicas/ }))
    expect(onItemHover).toHaveBeenCalled()
  })

  it('shows the ► chevron on a hasNext item', () => {
    render(
      <HubMenu
        items={[{ label: 'Continuar sessão', onSelect: vi.fn(), hasNext: true }]}
      />,
    )
    expect(screen.getByText('►')).toBeInTheDocument()
  })
})
