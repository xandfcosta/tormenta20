import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { BackgroundTexture } from './background-texture'

function texture(container: HTMLElement) {
  return container.querySelector('[data-slot="background-texture"]')
}

describe('BackgroundTexture', () => {
  it('is decorative — hidden from the accessibility tree', () => {
    const { container } = render(<BackgroundTexture />)
    expect(texture(container)).toHaveAttribute('aria-hidden')
  })

  it('defaults to the stone variant', () => {
    const { container } = render(<BackgroundTexture />)
    expect(texture(container)).toHaveAttribute('data-variant', 'stone')
    expect(texture(container)).toHaveClass('grimorio-stone')
  })

  it('renders the parchment variant when requested', () => {
    const { container } = render(<BackgroundTexture variant="parchment" />)
    expect(texture(container)).toHaveAttribute('data-variant', 'parchment')
    expect(texture(container)).toHaveClass('grimorio-parchment-bg')
  })

  it('adds the vignette only when asked', () => {
    const { container: plain } = render(<BackgroundTexture />)
    expect(texture(plain)).not.toHaveClass('grimorio-vignette')
    const { container: veiled } = render(<BackgroundTexture vignette />)
    expect(texture(veiled)).toHaveClass('grimorio-vignette')
  })
})
