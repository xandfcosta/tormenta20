import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { DicePill } from './dice-pill'

describe('DicePill', () => {
  it('renders count + sides in Nd<sides> form', () => {
    render(<DicePill count={2} sides={6} />)
    expect(screen.getByText('2d6')).toBeInTheDocument()
  })

  it('appends positive modifier as +N', () => {
    render(<DicePill sides={20} modifier={3} />)
    expect(screen.getByLabelText('1d20 +3')).toBeInTheDocument()
  })

  it('appends negative modifier verbatim', () => {
    render(<DicePill sides={8} modifier={-1} />)
    expect(screen.getByLabelText('1d8 -1')).toBeInTheDocument()
  })

  it('marks the crit variant via data attribute', () => {
    const { container } = render(<DicePill sides={20} variant="crit" />)
    expect(
      container.querySelector('[data-slot="dice-pill"]')?.getAttribute('data-variant'),
    ).toBe('crit')
  })

  it('omits modifier text when 0', () => {
    render(<DicePill sides={4} modifier={0} />)
    expect(screen.queryByText('+0')).not.toBeInTheDocument()
  })
})
