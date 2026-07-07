import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { StatBlock } from './stat-block'

describe('StatBlock', () => {
  it('renders label + value + optional sublabel', () => {
    render(<StatBlock label="FOR" value={3} sublabel="+1 raça" />)
    expect(screen.getByText('FOR')).toBeInTheDocument()
    expect(screen.getByText('3')).toBeInTheDocument()
    expect(screen.getByText('+1 raça')).toBeInTheDocument()
  })

  it('renders no trend arrow by default', () => {
    const { container } = render(<StatBlock label="DES" value={2} />)
    const block = container.querySelector('[data-slot="stat-block"]')
    expect(block?.getAttribute('data-trend')).toBe('none')
  })

  it('renders an up arrow with an aria label when trend is up', () => {
    render(<StatBlock label="FOR" value={5} trend="up" />)
    expect(screen.getByLabelText('tendência positiva')).toBeInTheDocument()
  })

  it('renders a down arrow with an aria label when trend is down', () => {
    render(<StatBlock label="FOR" value={1} trend="down" />)
    expect(screen.getByLabelText('tendência negativa')).toBeInTheDocument()
  })
})
