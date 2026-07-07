import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { PageChrome } from './page-chrome'

describe('PageChrome', () => {
  it('renders children inside a section with the page-chrome slot', () => {
    const { container, getByText } = render(
      <PageChrome>
        <p>hello</p>
      </PageChrome>,
    )
    expect(getByText('hello')).toBeInTheDocument()
    expect(container.querySelector('[data-slot="page-chrome"]')).toBeTruthy()
  })

  it('applies the wide width variant', () => {
    const { container } = render(<PageChrome width="wide">wide</PageChrome>)
    const node = container.querySelector('[data-slot="page-chrome"]')
    expect(node?.getAttribute('data-width')).toBe('wide')
  })

  it('marks bordered via data attribute when opted in', () => {
    const { container } = render(<PageChrome bordered>b</PageChrome>)
    const node = container.querySelector('[data-slot="page-chrome"]')
    expect(node?.getAttribute('data-bordered')).toBe('true')
  })
})
