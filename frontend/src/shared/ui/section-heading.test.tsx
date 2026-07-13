import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { SectionHeading } from './section-heading'

describe('SectionHeading', () => {
  it('renders text content in an h2 by default', () => {
    render(<SectionHeading>Perícias</SectionHeading>)
    expect(
      screen.getByRole('heading', { name: 'Perícias', level: 2 }),
    ).toBeInTheDocument()
  })

  it('renders no glyph (flourishes stripped) on any variant', () => {
    const { container } = render(
      <SectionHeading variant="kallyadranoch">Combate</SectionHeading>,
    )
    expect(container.querySelectorAll('svg').length).toBe(0)
  })

  it('honors the `as` prop to render h1/h3', () => {
    render(<SectionHeading as="h1">Título</SectionHeading>)
    expect(screen.getByRole('heading', { level: 1 })).toBeInTheDocument()
  })
})
