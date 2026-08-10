import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { Kbd } from './kbd'

describe('Kbd', () => {
  it('renders the key as a <kbd> element', () => {
    render(<Kbd>⏎</Kbd>)
    const el = screen.getByText('⏎')
    expect(el.tagName).toBe('KBD')
  })

  it('hides below xl (touch has no keyboard) and shows inline at xl', () => {
    render(<Kbd>D</Kbd>)
    const el = screen.getByText('D')
    expect(el.className).toContain('hidden')
    expect(el.className).toContain('xl:inline')
  })

  it('merges caller classes with the base badge styling', () => {
    render(<Kbd className="ml-2">O</Kbd>)
    const el = screen.getByText('O')
    expect(el.className).toContain('ml-2')
    expect(el.className).toContain('xl:inline')
  })
})
