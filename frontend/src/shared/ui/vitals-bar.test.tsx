import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { HpBar } from './hp-bar'
import { MpBar } from './mp-bar'

describe('HpBar / Controlled Decay', () => {
  it('shows current / max in the readout', () => {
    render(<HpBar current={30} max={40} />)
    expect(screen.getByText('30')).toBeInTheDocument()
    expect(screen.getByText('/ 40')).toBeInTheDocument()
  })

  it('is state="full" above 50%', () => {
    const { container } = render(<HpBar current={40} max={40} />)
    expect(
      container.querySelector('[data-slot="vitals-bar"]')?.getAttribute('data-state'),
    ).toBe('full')
  })

  it('is state="hurt" between 26% and 50%', () => {
    const { container } = render(<HpBar current={15} max={40} />)
    expect(
      container.querySelector('[data-slot="vitals-bar"]')?.getAttribute('data-state'),
    ).toBe('hurt')
  })

  it('is state="critical" at or below 25%', () => {
    const { container } = render(<HpBar current={5} max={40} />)
    expect(
      container.querySelector('[data-slot="vitals-bar"]')?.getAttribute('data-state'),
    ).toBe('critical')
  })

  it('clamps negative current to 0 and reports the clamp via aria', () => {
    const { container } = render(<HpBar current={-10} max={20} />)
    const bar = container.querySelector('[role="progressbar"]')
    expect(bar?.getAttribute('aria-valuenow')).toBe('0')
  })

  it('exposes label + max via ARIA', () => {
    render(<HpBar current={12} max={20} label="Vida" />)
    const bar = screen.getByRole('progressbar', { name: 'Vida' })
    expect(bar.getAttribute('aria-valuemax')).toBe('20')
  })
})

describe('MpBar', () => {
  it('renders the PM default label + arcane state', () => {
    const { container } = render(<MpBar current={5} max={10} />)
    expect(screen.getByText('PM')).toBeInTheDocument()
    // MpBar never resolves to hurt/critical — always the arcane fill.
    expect(
      container.querySelector('[data-slot="vitals-bar"]')?.getAttribute('data-state'),
    ).toBe('full')
  })
})
