import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from './sheet'

function Harness({ side }: { side?: 'left' | 'right' | 'top' | 'bottom' }) {
  return (
    <Sheet>
      <SheetTrigger>open</SheetTrigger>
      <SheetContent side={side}>
        <SheetHeader>
          <SheetTitle>Menu</SheetTitle>
          <SheetDescription>Navegação lateral</SheetDescription>
        </SheetHeader>
        <p>drawer body</p>
      </SheetContent>
    </Sheet>
  )
}

describe('Sheet', () => {
  it('opens the drawer content on trigger click', async () => {
    const user = userEvent.setup()
    render(<Harness />)
    expect(screen.queryByText('drawer body')).not.toBeInTheDocument()
    await user.click(screen.getByText('open'))
    expect(screen.getByText('drawer body')).toBeInTheDocument()
    expect(screen.getByRole('dialog')).toBeInTheDocument()
  })

  it('exposes the requested side via data-side', async () => {
    const user = userEvent.setup()
    render(<Harness side="bottom" />)
    await user.click(screen.getByText('open'))
    const content = document.querySelector('[data-slot="sheet-content"]')
    expect(content?.getAttribute('data-side')).toBe('bottom')
  })

  it('renders title + description with the correct slots', async () => {
    const user = userEvent.setup()
    render(<Harness />)
    await user.click(screen.getByText('open'))
    expect(document.querySelector('[data-slot="sheet-title"]')?.textContent).toBe('Menu')
    expect(document.querySelector('[data-slot="sheet-description"]')?.textContent).toBe(
      'Navegação lateral',
    )
  })

  it('exposes an accessible Close button', async () => {
    const user = userEvent.setup()
    render(<Harness />)
    await user.click(screen.getByText('open'))
    expect(screen.getByRole('button', { name: 'Fechar' })).toBeInTheDocument()
  })
})
