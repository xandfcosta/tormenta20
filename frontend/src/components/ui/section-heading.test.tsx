import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { SectionHeading } from './section-heading'

describe('SectionHeading', () => {
  it('renders text content in an h2 by default', () => {
    render(<SectionHeading>Perícias</SectionHeading>)
    const heading = screen.getByRole('heading', { name: 'Perícias', level: 2 })
    expect(heading).toBeInTheDocument()
    expect(heading.getAttribute('data-variant')).toBe('default')
  })

  it('renders no glyph on the default variant', () => {
    const { container } = render(<SectionHeading>Nada</SectionHeading>)
    expect(container.querySelectorAll('svg').length).toBe(0)
  })

  it('renders the aharadak glyph for occult sections', () => {
    const { container } = render(
      <SectionHeading variant="aharadak">Magias</SectionHeading>,
    )
    expect(container.querySelectorAll('svg').length).toBe(1)
    expect(
      container.querySelector('[data-slot="section-heading"]')?.getAttribute('data-variant'),
    ).toBe('aharadak')
  })

  it('renders the kallyadranoch glyph for combat sections', () => {
    const { container } = render(
      <SectionHeading variant="kallyadranoch">Combate</SectionHeading>,
    )
    expect(
      container.querySelector('[data-slot="section-heading"]')?.getAttribute('data-variant'),
    ).toBe('kallyadranoch')
  })

  it('honors the `as` prop to render h1/h3', () => {
    render(<SectionHeading as="h1">Título</SectionHeading>)
    expect(screen.getByRole('heading', { level: 1 })).toBeInTheDocument()
  })
})
