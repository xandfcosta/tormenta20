import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { SceneTitle } from './scene-title'

describe('SceneTitle', () => {
  it('renders the title as an h1 by default', () => {
    render(<SceneTitle>Tormenta 20</SceneTitle>)
    expect(
      screen.getByRole('heading', { name: 'Tormenta 20', level: 1 }),
    ).toBeInTheDocument()
  })

  it('renders the kicker when provided', () => {
    render(<SceneTitle kicker="— Grimório de Arton —">Tormenta 20</SceneTitle>)
    expect(screen.getByText('— Grimório de Arton —')).toBeInTheDocument()
  })

  it('omits the kicker element when not provided', () => {
    const { container } = render(<SceneTitle>Só título</SceneTitle>)
    expect(container.querySelector('p')).toBeNull()
  })

  it('honors the `as` prop to render an h2', () => {
    render(<SceneTitle as="h2">Cena</SceneTitle>)
    expect(screen.getByRole('heading', { level: 2 })).toBeInTheDocument()
  })
})
