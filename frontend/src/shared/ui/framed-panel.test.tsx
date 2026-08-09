import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { FramedPanel } from './framed-panel'

function panel() {
  return screen.getByTestId('panel').querySelector('[data-slot="framed-panel"]')
}

describe('FramedPanel', () => {
  it('renders its children', () => {
    render(<FramedPanel>Conteúdo da cena</FramedPanel>)
    expect(screen.getByText('Conteúdo da cena')).toBeInTheDocument()
  })

  it('defaults to the stone variant', () => {
    render(
      <div data-testid="panel">
        <FramedPanel>x</FramedPanel>
      </div>,
    )
    expect(panel()).toHaveAttribute('data-variant', 'stone')
  })

  it('sets the parchment variant when requested', () => {
    render(
      <div data-testid="panel">
        <FramedPanel variant="parchment">x</FramedPanel>
      </div>,
    )
    expect(panel()).toHaveAttribute('data-variant', 'parchment')
  })
})
